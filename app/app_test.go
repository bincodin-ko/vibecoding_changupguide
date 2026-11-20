package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bp(b bool) *bool { return &b }

func strongEvidence(n int) []Evidence {
	var out []Evidence
	for i := 0; i < n; i++ {
		out = append(out, Evidence{ID: newID(), Title: "t", Experienced: bp(true), Tried: bp(true)})
	}
	return out
}

func TestCustomerTooBroad(t *testing.T) {
	if len(customerProblems(Customer{Who: "할인 정보를 찾는 사람들", Situation: "쇼핑할 때마다", Problem: "정보가 흩어져 있다"})) == 0 {
		t.Fatal("넓은 고객 문장을 잡아내지 못했다")
	}
	if p := customerProblems(Customer{Who: "돌 전 아기를 키우는 부모", Situation: "기저귀와 분유를 살 때마다", Problem: "최저가를 서너 곳 비교해야 한다"}); len(p) != 0 {
		t.Fatalf("좁은 고객 문장을 넓다고 했다: %v", p)
	}
}

func TestJudgeVerdicts(t *testing.T) {
	th := defaultThresholds()
	p := &Project{}
	normalize(p)
	if v := judge(p, th).Verdict; v != "근거 부족" {
		t.Fatalf("빈 프로젝트: %s", v)
	}
	// 글을 충분히 봤는데 겪는 사람이 적다
	for i := 0; i < 12; i++ {
		p.Evidence = append(p.Evidence, Evidence{ID: newID(), Experienced: bp(false), Tried: bp(false)})
	}
	if v := judge(p, th).Verdict; v != "수정: 고객 또는 문제" {
		t.Fatalf("겪는 사람이 적을 때: %s", v)
	}
	p.Evidence = strongEvidence(10)
	if v := judge(p, th).Verdict; v != "근거 부족" {
		t.Fatalf("글을 아직 안 올렸을 때: %s", v)
	}
	p.Posts = []Post{{Posted: true, Visits: 5}}
	if v := judge(p, th).Verdict; v != "수정: 닿는 길" {
		t.Fatalf("방문이 적을 때: %s", v)
	}
	p.Posts = []Post{{Posted: true, Visits: 20}, {Posted: true, Visits: 15}, {Posted: false, Visits: 999}}
	if v := judge(p, th).Verdict; v != "수정: 서비스" {
		t.Fatalf("남지 않았을 때: %s", v)
	}
	p.Survey.Helpful = 3
	if v := judge(p, th).Verdict; v != "계속" {
		t.Fatalf("모두 통과: %s", v)
	}
}

func TestStagesUnlockInOrder(t *testing.T) {
	th := defaultThresholds()
	p := &Project{}
	normalize(p)
	st := computeStages(p, th)
	if st[0].Status != "now" || st[1].Status != "locked" {
		t.Fatalf("처음 상태가 다르다: %s %s", st[0].Status, st[1].Status)
	}
	p.Customer = Customer{Who: "돌 전 아기를 키우는 부모", Situation: "기저귀와 분유를 살 때마다", Problem: "최저가 비교가 번거롭다", Solution: "최저가 페이지"}
	p.Evidence = strongEvidence(10)
	st = computeStages(p, th)
	if st[0].Status != "done" || st[1].Status != "done" || st[2].Status != "now" {
		t.Fatalf("0·1단계 통과 뒤: %s %s %s", st[0].Status, st[1].Status, st[2].Status)
	}
}

func TestParsePasted(t *testing.T) {
	items := parsePasted("기저귀 어디가 제일 싼가요\n매번 세 군데 들어가서 비교하는데 너무 번거로워요\nhttps://example.com/a\n\n\n분유 핫딜 놓쳤어요 ㅠㅠ")
	if len(items) != 2 {
		t.Fatalf("2건이어야 하는데 %d건", len(items))
	}
	if items[0].Link != "https://example.com/a" || !strings.Contains(items[0].Snippet, "번거로워요") || strings.Contains(items[0].Snippet, "http") {
		t.Fatalf("첫 글을 잘못 읽었다: %+v", items[0])
	}
	p := &Project{}
	if addEvidence(p, items) != 2 || addEvidence(p, items) != 0 {
		t.Fatal("같은 글을 두 번 넣었다")
	}
}

func TestCollectNaver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Naver-Client-Id") != "id" {
			w.WriteHeader(401)
			io.WriteString(w, `{"errorMessage":"bad key"}`)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/search/") || r.URL.Query().Get("query") != "기저귀 최저가" {
			t.Errorf("요청이 다르다: %s", r.URL)
		}
		io.WriteString(w, `{"items":[{"title":"<b>기저귀</b> 최저가 &amp; 비교","link":"https://x/1","description":"매번 <b>비교</b>해요","postdate":"20261101"}]}`)
	}))
	defer srv.Close()
	old := naverBase
	naverBase = srv.URL
	defer func() { naverBase = old }()

	items, err := collectNaver(context.Background(), &Settings{NaverID: "id", NaverSecret: "s"}, "기저귀 최저가", []string{"blog"})
	if err != nil || len(items) != 1 {
		t.Fatalf("수집 실패: %v %d", err, len(items))
	}
	if items[0].Title != "기저귀 최저가 & 비교" || items[0].Date != "2026-11-01" || items[0].Source != "블로그" {
		t.Fatalf("글을 잘못 읽었다: %+v", items[0])
	}
	if _, err := collectNaver(context.Background(), &Settings{NaverID: "wrong", NaverSecret: "s"}, "기저귀 최저가", []string{"blog"}); err == nil || !strings.Contains(err.Error(), "키가 맞지") {
		t.Fatalf("틀린 키 안내가 없다: %v", err)
	}
	if _, err := collectNaver(context.Background(), &Settings{}, "x", nil); err == nil {
		t.Fatal("키가 없는데 오류가 없다")
	}
}

func TestSiteCheck(t *testing.T) {
	good := `<html><head><title>돌 전 아기 부모를 위한 기저귀·분유 오늘 최저가</title>
<meta name="description" content="기저귀와 분유의 오늘 최저가만 모아 보여 드립니다. 가입 없이 바로 봅니다.">
<meta name="viewport" content="width=device-width, initial-scale=1"><meta property="og:title" content="t"><meta property="og:image" content="i">
<script>var x = "이 글자는 세지 않는다";</script></head><body><h1>오늘 최저가</h1><p>` + strings.Repeat("기저귀 값을 매번 비교하기 번거로운 분을 위한 페이지입니다. ", 12) + `</p><a href="https://open.kakao.com/o/abc">의견 보내기</a></body></html>`
	bad := `<html><head><title>Vite App</title><meta name="robots" content="noindex"></head><body><div id="root"></div><script>render()</script></body></html>`
	page := good
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, page) }))
	defer srv.Close()

	res := map[string]bool{}
	for _, r := range runSiteCheck(context.Background(), srv.URL).Results {
		res[r.ID] = r.Pass
	}
	for _, id := range []string{"status", "speed", "title", "description", "viewport", "index", "content", "share", "contact"} {
		if !res[id] {
			t.Errorf("좋은 페이지인데 %s 가 통과하지 못했다", id)
		}
	}
	if res["https"] {
		t.Error("http 주소인데 https 통과로 나왔다")
	}

	page = bad
	sc := runSiteCheck(context.Background(), srv.URL)
	for _, r := range sc.Results {
		switch r.ID {
		case "title", "description", "viewport", "index", "content", "share", "contact":
			if r.Pass {
				t.Errorf("나쁜 페이지인데 %s 가 통과했다", r.ID)
			}
			if r.Fix == "" {
				t.Errorf("%s 에 고칠 지시가 없다", r.ID)
			}
		}
	}
	// 꼭 고칠 것이 남아 있으면 출시 준비 단계를 통과할 수 없다
	p := &Project{ServiceURL: srv.URL, SiteCheck: sc}
	normalize(p)
	for _, id := range stageChecks[4] {
		p.Checks[id] = true
	}
	if st := computeStages(p, defaultThresholds())[4]; st.Status == "done" || !strings.Contains(st.Next, "꼭 고칠 것") {
		t.Errorf("꼭 고칠 것이 남았는데 통과했다: %+v", st)
	}
	page = good
	p.SiteCheck = runSiteCheck(context.Background(), srv.URL)
	if st := computeStages(p, defaultThresholds())[4]; st.Status != "done" {
		t.Errorf("꼭 고칠 것이 없는데 통과하지 못했다: %+v", st)
	}
	if sc := runSiteCheck(context.Background(), "example.com"); sc.Error == "" {
		t.Error("잘못된 주소인데 오류가 없다")
	}
}

// 가짜 AI 서버로 요청 모양과 답 처리를 확인한다. 실제 AI를 부르지 않는다.
func TestClassifyWithFakeAI(t *testing.T) {
	p := &Project{Customer: Customer{Who: "돌 전 아기 부모", Situation: "기저귀 살 때", Problem: "비교가 번거롭다"}}
	p.Evidence = []Evidence{{ID: "a1", Title: "기저귀 어디가 싼가요", Snippet: "매번 세 군데 비교해요"}, {ID: "b2", Title: "기저귀 특가 광고", Snippet: "지금 구매하세요"}, {ID: "c3", Title: "이미 분류함", Experienced: bp(false), Tried: bp(false)}}
	var got map[string]any
	var headers http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		json.NewDecoder(r.Body).Decode(&got)
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid key"}}`)
			return
		}
		answer := `{"items":[{"id":"a1","experienced":true,"tried":true,"problem":"최저가 비교가 번거롭다","quote":"매번 세 군데 비교해요"},{"id":"b2","experienced":false,"tried":true,"problem":"","quote":""},{"id":"zz","experienced":true,"tried":true,"problem":"없는 글","quote":""}]}`
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-opus-5-5",
			"content":     []map[string]any{{"type": "text", "text": answer}},
			"stop_reason": "end_turn", "usage": map[string]any{"input_tokens": 10, "output_tokens": 10},
		})
	}))
	defer srv.Close()
	aiBaseURL = srv.URL
	defer func() { aiBaseURL = "" }()

	done, remaining, err := classifyEvidence(context.Background(), &Settings{AIKey: "sk-test", AIModel: defaultModel}, p)
	if err != nil || done != 2 || remaining != 0 {
		t.Fatalf("분류 결과가 다르다: done=%d remaining=%d err=%v", done, remaining, err)
	}
	if !*p.Evidence[0].Experienced || !*p.Evidence[0].Tried || p.Evidence[0].Quote == "" || p.Evidence[0].By != "ai" {
		t.Fatalf("첫 글 분류가 다르다: %+v", p.Evidence[0])
	}
	if *p.Evidence[1].Experienced || *p.Evidence[1].Tried {
		t.Fatal("겪지 않은 글에 해결 시도가 표시됐다")
	}
	if got["model"] != "claude-opus-5-5" || got["output_config"] == nil || got["fallbacks"] == nil {
		t.Fatalf("요청에 빠진 것이 있다: %v", got)
	}
	if !strings.Contains(headers.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Fatalf("베타 헤더가 없다: %v", headers.Get("Anthropic-Beta"))
	}
	if strings.Contains(got["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string), "이미 분류함") {
		t.Fatal("이미 분류한 글을 다시 보냈다")
	}

	p.Evidence = append(p.Evidence, Evidence{ID: "d4", Title: "새 글"})
	if _, _, err := classifyEvidence(context.Background(), &Settings{AIKey: "wrong", AIModel: defaultModel}, p); err == nil || !strings.Contains(err.Error(), "AI 키가 맞지") {
		t.Fatalf("틀린 키 안내가 없다: %v", err)
	}
	if _, err := callAI(context.Background(), &Settings{}, "x", nil); err == nil || !strings.Contains(err.Error(), "AI 키가 없습니다") {
		t.Fatalf("키 없음 안내가 없다: %v", err)
	}
}

func TestPrompts(t *testing.T) {
	p := &Project{Customer: Customer{Who: "돌 전 아기 부모", Situation: "기저귀 살 때", Problem: "비교가 번거롭다", Solution: "최저가 페이지"}}
	normalize(p)
	p.Places = []Place{{ID: "p1", Name: "육아 카페", Rules: "홍보 금지", LinkAllowed: false}}
	text, err := buildPrompt("post", p, map[string]string{"placeId": "p1"})
	if err != nil || !strings.Contains(text, "링크와 서비스 이름을 넣을 수 없는 곳") || !strings.Contains(text, "홍보 금지") {
		t.Fatalf("링크 금지 곳의 글 지시문이 다르다: %v\n%s", err, text)
	}
	if _, err := buildPrompt("post", p, map[string]string{}); err == nil {
		t.Fatal("곳을 고르지 않았는데 오류가 없다")
	}
	for _, task := range []string{"narrow", "summary", "plan"} {
		if s, err := buildPrompt(task, p, nil); err != nil || !strings.Contains(s, "돌 전 아기 부모") {
			t.Fatalf("%s 지시문에 고객 문장이 없다: %v", task, err)
		}
	}
	for _, s := range []string{aiSystem} {
		if !strings.Contains(s, "1:1로 연락하거나 인터뷰하라는 제안은 하지 않습니다") {
			t.Fatal("접촉을 요구하지 않는다는 원칙이 빠졌다")
		}
	}
	if _, err := buildToolPrompt("sitefix", p); err == nil {
		t.Fatal("점검 전인데 고칠 점 지시문이 나왔다")
	}
}
