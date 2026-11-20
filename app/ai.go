package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// 시험할 때 가짜 서버를 가리키게 하려고 둔 값. 평소에는 비어 있다.
var aiBaseURL = ""

const aiSystem = `당신은 혼자 서비스를 만드는 초보 창업자를 돕는 조수입니다. 사용자는 개발자가 아니며, 낯선 사람에게 직접 연락하거나 인터뷰하는 일을 어려워합니다.
지킬 것:
- 사용자가 준 정보에 없는 숫자, 사례, 경험을 지어내지 않습니다.
- 수익이나 성공을 약속하지 않습니다. 과장하는 말(최고, 혁신, 완벽)을 쓰지 않습니다.
- 사람에게 1:1로 연락하거나 인터뷰하라는 제안은 하지 않습니다.
- 쉬운 한국어로 짧게 씁니다. 전문 용어를 써야 하면 바로 풀어 씁니다.`

func customerSentence(c Customer) string {
	return fmt.Sprintf("%s은(는) %s에서 %s 문제를 겪는다. 그래서 %s을(를) 만들었다(만들려 한다).",
		orBlank(c.Who), orBlank(c.Situation), orBlank(c.Problem), orBlank(c.Solution))
}

func orBlank(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(아직 안 적음)"
	}
	return strings.TrimSpace(s)
}

// 분류가 끝난 글 가운데 "실제로 겪은" 글에서 문제와 표현을 뽑아 글감으로 쓴다.
func evidenceDigest(p *Project, max int) string {
	var b strings.Builder
	n := 0
	for _, e := range p.Evidence {
		if e.Experienced == nil || !*e.Experienced {
			continue
		}
		fmt.Fprintf(&b, "- 문제: %s / 글쓴이 표현: \"%s\" (%s)\n", orBlank(e.Problem), firstNonEmpty(e.Quote, e.Title), e.Source)
		n++
		if n >= max {
			break
		}
	}
	if n == 0 {
		return "(아직 분류한 글이 없음)\n"
	}
	return b.String()
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func findPlace(p *Project, id string) Place {
	for _, pl := range p.Places {
		if pl.ID == id {
			return pl
		}
	}
	return Place{}
}

// AI에게 보낼 지시문을 만든다. 키가 없을 때는 같은 글을 사용자가 복사해 자기 AI에 붙여넣는다.
func buildPrompt(task string, p *Project, params map[string]string) (string, error) {
	cs := customerSentence(p.Customer)
	switch task {
	case "narrow":
		return fmt.Sprintf(`아래는 내가 적은 고객 한 문장입니다.

%s

이 문장의 "누구"와 "어떤 상황"이 너무 넓으면, 더 좁힌 문장 3개를 제안해 주세요.
각 제안에는 (1) 좁힌 문장, (2) 그 사람들이 온라인에서 모일 만한 곳 한두 군데(네이버 카페, 커뮤니티 게시판 종류), (3) 그 사람들이 이 문제를 검색할 때 쓸 만한 검색어 3개를 붙여 주세요.
이미 충분히 좁다면 그렇다고 말하고 검색어 5개만 제안해 주세요.`, cs), nil

	case "summary":
		return fmt.Sprintf(`아래는 사람들이 인터넷에 직접 쓴 글에서 뽑은 문제와 표현입니다. 내 고객 한 문장은 이렇습니다.

%s

모은 글:
%s
이 글들만 근거로 다음을 정리해 주세요. 글에 없는 내용은 넣지 않습니다.
1. 자주 나온 문제 (많이 나온 순서, 각각 몇 건인지)
2. 사람들이 지금 쓰고 있는 해결 방법이나 서비스
3. 사람들이 쓰는 표현 가운데 내 서비스 첫 화면 문구로 쓸 만한 것 3개
4. 내 고객 한 문장과 어긋나는 점이 있으면 한 줄`, cs, evidenceDigest(p, 60)), nil

	case "plan":
		return fmt.Sprintf(`아래 정보로 "한 장 기획서"를 써 주세요. 이 기획서는 AI 코딩 도구에 그대로 붙여넣어 서비스를 만들게 할 용도입니다.

고객 한 문장: %s

사람들이 실제로 쓴 글에서 확인한 문제:
%s
형식:
# 한 줄 소개
# 누구를 위한 것인가
# 풀려는 문제 (확인한 근거 요약)
# 첫 버전에 넣는 것 (3개 이하)
# 첫 버전에 넣지 않는 것
# 화면 (첫 화면, 핵심 화면 한두 개. 화면마다 보이는 것과 누를 수 있는 것)
# 가입 없이 바로 써 볼 수 있는 길
# 익명 한 줄 설문 (화면 어디에, 어떤 질문으로)
# 만들 때 지킬 것 (휴대폰 화면 우선, 비밀 키를 코드에 넣지 않기, 개인정보를 받지 않기)

기능은 가장 작게 잡습니다. 근거에 없는 기능은 넣지 않습니다.`, cs, evidenceDigest(p, 40)), nil

	case "post":
		pl := findPlace(p, params["placeId"])
		if pl.Name == "" {
			return "", fmt.Errorf("글을 올릴 곳을 먼저 고르세요.")
		}
		linkRule := "링크를 넣을 수 있는 곳입니다. 서비스 이야기는 마지막 두세 문장에만, 만든 사람이 나라는 것을 밝히며 씁니다."
		if !pl.LinkAllowed {
			linkRule = "링크와 서비스 이름을 넣을 수 없는 곳입니다. 서비스 이야기는 쓰지 말고, 같은 문제를 겪는 사람들에게 어떻게 하고 있는지 묻는 질문 글로 씁니다."
		}
		return fmt.Sprintf(`아래 내용으로 커뮤니티에 올릴 글 초안을 써 주세요.

내 고객: %s
올릴 곳: %s
그곳의 규칙(내가 읽고 적은 것): %s
%s

사람들이 실제로 쓴 표현과 문제:
%s
조건:
1. 서비스 소개가 아니라 이 문제에 대한 이야기로 씁니다. 내 경험은 내가 준 내용만 쓰고, 없는 경험을 지어내지 않습니다. 내 경험이 필요한 자리는 [여기에 내 경험 한두 문장]처럼 비워 둡니다.
2. 그곳 사람들이 쓰는 말투에 맞추고, 광고 문구처럼 쓰지 않습니다.
3. 끝에 읽는 사람에게 묻는 질문 하나를 넣습니다.
4. 그곳의 규칙에 어긋나는 부분이 있으면 글보다 먼저 한 줄로 알려 줍니다.
5. 제목 1개와 본문을 줍니다. 본문은 400자 안팎입니다.`, cs, pl.Name, orBlank(pl.Rules), linkRule, evidenceDigest(p, 20)), nil

	case "reply":
		if strings.TrimSpace(params["comment"]) == "" {
			return "", fmt.Errorf("답할 댓글 내용을 붙여넣어 주세요.")
		}
		return fmt.Sprintf(`내가 올린 글에 아래 댓글이 달렸습니다. 답글 초안을 2개 써 주세요.

내 고객: %s
댓글: %s

조건:
1. 먼저 댓글 내용에 고마움을 표하거나 답합니다. 변명하지 않습니다.
2. 비판이면 받아들이고, 구체적으로 무엇이 불편했는지 한 가지를 되묻습니다.
3. 서비스를 다시 홍보하지 않습니다. 두세 문장으로 짧게 씁니다.`, cs, params["comment"]), nil
	}
	return "", fmt.Errorf("알 수 없는 작업입니다: %s", task)
}

// AI가 아니라 코딩 도구(서비스를 만든 도구)에 붙여넣는 지시문. 키가 없어도 항상 쓸 수 있다.
func buildToolPrompt(task string, p *Project) (string, error) {
	switch task {
	case "analytics":
		return `내 서비스에 방문 기록을 볼 수 있는 무료 도구를 하나 붙여 줘.

조건:
1. 하루 방문자 수, 다시 온 방문자 수, 방문자가 어느 사이트에서 왔는지 볼 수 있어야 해.
2. 주소 끝에 ?utm_source=이름 을 붙인 링크로 들어오면 그 이름별로 따로 보여야 해.
3. 나는 개발자가 아니야. 내가 직접 해야 하는 일(가입, 코드 복사 등)은 화면에서 무엇을 누르는지까지 순서대로 알려 줘.
4. 다 붙인 뒤 제대로 기록되는지 확인하는 방법도 알려 줘.
5. 방문자의 개인정보를 수집하게 된다면, 사이트에 어떤 안내를 적어야 하는지도 알려 줘.`, nil
	case "survey":
		return `내 서비스 화면에 익명 한 줄 설문을 붙여 줘.

조건:
1. 핵심 기능을 한 번 쓴 뒤에 화면 아래쪽에 작게 보인다. 처음 들어오자마자 뜨지 않는다.
2. 질문은 "이게 도움이 됐나요?" 하나. 버튼은 "도움 됐어요"와 "아니요" 두 개.
3. 버튼을 누르면 "어떤 점이 불편했나요? (선택)" 한 줄 입력칸이 나온다. 안 적어도 된다.
4. 이름, 이메일, 연락처는 받지 않는다. 로그인도 요구하지 않는다.
5. 한 사람이 한 번 답하면 다시 보이지 않는다.
6. 응답은 내가 볼 수 있는 곳(서비스의 데이터 저장소)에 날짜와 함께 쌓고, "도움 됐어요" 수, "아니요" 수, 적힌 한 줄들을 볼 수 있는 간단한 관리자 화면을 만들어 줘. 관리자 화면은 나만 볼 수 있게 해 줘.`, nil
	case "sitefix":
		if p.SiteCheck == nil {
			return "", fmt.Errorf("먼저 첫 화면 점검을 실행하세요.")
		}
		var b strings.Builder
		b.WriteString("내 서비스 첫 화면을 아래 항목만 고쳐 줘. 기능은 건드리지 마.\n\n내 고객: " + customerSentence(p.Customer) + "\n\n고칠 것:\n")
		n := 0
		for _, r := range p.SiteCheck.Results {
			if !r.Pass && r.Fix != "" {
				n++
				fmt.Fprintf(&b, "%d. %s\n", n, r.Fix)
			}
		}
		if n == 0 {
			return "", fmt.Errorf("고칠 항목이 없습니다. 모든 점검을 통과했습니다.")
		}
		b.WriteString("\n다 고친 뒤 무엇을 어떻게 바꿨는지 항목별로 한 줄씩 알려 줘.")
		return b.String(), nil
	case "plan":
		if strings.TrimSpace(p.Plan) == "" {
			return "", fmt.Errorf("먼저 한 장 기획서를 만들어 저장하세요.")
		}
		return "아래 기획서대로 서비스의 첫 버전을 만들어 줘. 기획서에 없는 기능은 넣지 마. 한 번에 다 만들지 말고, 먼저 만들 순서를 5단계 이하로 나눠서 보여 주고 내가 좋다고 하면 1단계부터 시작해 줘.\n\n" + p.Plan, nil
	}
	return "", fmt.Errorf("알 수 없는 지시문입니다: %s", task)
}

// AI 연결을 만든다. 이 컴퓨터에 따로 저장된 로그인 정보나 환경 변수는 쓰지 않고,
// 설정에 넣은 키만 쓴다. (기본 연결 방식은 주변의 로그인 정보를 먼저 집어 쓴다.)
func newAIService(st *Settings) anthropic.BetaMessageService {
	base := option.WithEnvironmentProduction()
	if aiBaseURL != "" {
		base = option.WithBaseURL(aiBaseURL)
	}
	return anthropic.NewBetaMessageService(base, option.WithAPIKey(st.AIKey))
}

// AI 호출 오류를 사용자가 읽고 조치할 수 있는 말로 바꾼다.
func friendlyAIError(err error) error {
	var apierr *anthropic.Error
	if errors.As(err, &apierr) {
		switch {
		case apierr.StatusCode == 401:
			return fmt.Errorf("AI 키가 맞지 않습니다. 설정에서 키를 다시 확인해 주세요.")
		case apierr.StatusCode == 403:
			return fmt.Errorf("이 AI 키로는 이 모델을 쓸 수 없습니다. 설정에서 모델 이름을 확인해 주세요.")
		case apierr.StatusCode == 404:
			return fmt.Errorf("설정에 적힌 모델 이름을 찾지 못했습니다. 설정에서 모델 이름을 확인해 주세요.")
		case apierr.StatusCode == 429:
			return fmt.Errorf("AI 사용 한도에 걸렸습니다. 잠시 뒤 다시 시도하거나, AI 서비스의 사용량과 결제 상태를 확인해 주세요.")
		case apierr.StatusCode == 400:
			return fmt.Errorf("AI가 요청을 받지 못했습니다. (%s)", apierr.Error())
		case apierr.StatusCode >= 500:
			return fmt.Errorf("AI 서비스가 지금 응답하지 않습니다. 잠시 뒤 다시 시도해 주세요.")
		}
	}
	return fmt.Errorf("AI에 연결하지 못했습니다. 인터넷 연결을 확인해 주세요. (%v)", err)
}

func callAI(ctx context.Context, st *Settings, prompt string, schema map[string]any) (string, error) {
	if st.AIKey == "" {
		return "", fmt.Errorf("AI 키가 없습니다. 설정에서 키를 넣거나, 지시문을 복사해 쓰던 AI에 붙여넣어 주세요.")
	}
	svc := newAIService(st)
	params := anthropic.BetaMessageNewParams{
		Model:     st.AIModel,
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: aiSystem}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt))},
		// 모델이 요청을 거절했을 때 다른 모델이 같은 요청을 이어받게 한다.
		Betas:     []anthropic.AnthropicBeta{"server-side-fallback-2026-07-01"},
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
	}
	if schema != nil {
		params.OutputConfig = anthropic.BetaOutputConfigParam{Format: anthropic.BetaJSONSchemaOutputFormat(schema)}
	}
	resp, err := svc.New(ctx, params)
	if err != nil {
		return "", friendlyAIError(err)
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return "", fmt.Errorf("AI가 이 요청에 답하지 않았습니다. 내용을 바꿔 다시 시도해 주세요.")
	}
	var out strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			out.WriteString(t.Text)
		}
	}
	if resp.StopReason == anthropic.BetaStopReasonMaxTokens {
		return "", fmt.Errorf("AI의 답이 너무 길어 중간에 끊겼습니다. 글을 나눠서 다시 시도해 주세요.")
	}
	if strings.TrimSpace(out.String()) == "" {
		return "", fmt.Errorf("AI가 빈 답을 보냈습니다. 다시 시도해 주세요.")
	}
	return out.String(), nil
}

var classifySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":          map[string]any{"type": "string"},
					"experienced": map[string]any{"type": "boolean"},
					"tried":       map[string]any{"type": "boolean"},
					"problem":     map[string]any{"type": "string"},
					"quote":       map[string]any{"type": "string"},
				},
				"required":             []string{"id", "experienced", "tried", "problem", "quote"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"items"},
	"additionalProperties": false,
}

const classifyBatch = 30

func classifyPrompt(p *Project, items []Evidence) string {
	var b strings.Builder
	b.WriteString("내 고객 한 문장: " + customerSentence(p.Customer) + "\n\n")
	b.WriteString(`아래는 인터넷에서 찾은 글의 제목과 요약입니다. 글마다 다음을 판단해 주세요.
- experienced: 글쓴이 본인이 위 문제(또는 아주 가까운 문제)를 실제로 겪고 있다고 볼 수 있으면 true. 광고, 업체 홍보, 뉴스, 일반 정보 글, 문제와 무관한 글은 false.
- tried: 글쓴이가 스스로 해결하려 한 흔적(방법을 묻는다, 써 본 방법을 말한다, 돈이나 시간을 썼다)이 있으면 true.
- problem: 글쓴이가 겪는 문제를 한 줄로. experienced가 false면 빈 문자열.
- quote: 글에서 문제를 드러내는 표현을 고치지 않고 그대로 옮긴 것(40자 이내). 없으면 빈 문자열.
제목과 요약에 없는 내용은 추측하지 않습니다. 판단하기 어려우면 false로 둡니다.

`)
	for _, e := range items {
		fmt.Fprintf(&b, "[id: %s] (%s) %s\n%s\n\n", e.ID, e.Source, e.Title, e.Snippet)
	}
	return b.String()
}

// 아직 분류하지 않은 글을 한 묶음(최대 30건) 분류한다. 남은 건수를 함께 돌려준다.
func classifyEvidence(ctx context.Context, st *Settings, p *Project) (done, remaining int, err error) {
	var batch []Evidence
	pending := 0
	for _, e := range p.Evidence {
		if e.Experienced == nil {
			pending++
			if len(batch) < classifyBatch {
				batch = append(batch, e)
			}
		}
	}
	if len(batch) == 0 {
		return 0, 0, nil
	}
	text, err := callAI(ctx, st, classifyPrompt(p, batch), classifySchema)
	if err != nil {
		return 0, pending, err
	}
	var parsed struct {
		Items []struct {
			ID          string `json:"id"`
			Experienced bool   `json:"experienced"`
			Tried       bool   `json:"tried"`
			Problem     string `json:"problem"`
			Quote       string `json:"quote"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(extractJSON(text)), &parsed); err != nil {
		return 0, pending, fmt.Errorf("AI의 답을 읽지 못했습니다. 다시 시도해 주세요.")
	}
	byID := map[string]int{}
	for i, e := range p.Evidence {
		byID[e.ID] = i
	}
	for _, it := range parsed.Items {
		i, ok := byID[it.ID]
		if !ok || p.Evidence[i].Experienced != nil {
			continue
		}
		exp, tried := it.Experienced, it.Experienced && it.Tried
		p.Evidence[i].Experienced = &exp
		p.Evidence[i].Tried = &tried
		p.Evidence[i].Problem = it.Problem
		p.Evidence[i].Quote = it.Quote
		p.Evidence[i].By = "ai"
		done++
	}
	if done == 0 {
		return 0, pending, fmt.Errorf("AI가 글을 분류하지 못했습니다. 다시 시도해 주세요.")
	}
	return done, pending - done, nil
}

// 답 앞뒤에 다른 글이 섞여 와도 JSON 부분만 꺼낸다.
func extractJSON(s string) string {
	i, j := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if i >= 0 && j > i {
		return s[i : j+1]
	}
	return s
}
