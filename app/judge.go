package main

import (
	"fmt"
	"strings"
)

// 단계 하나의 상태. 화면 왼쪽 목록에 그대로 쓰인다.
type Stage struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // done, now, locked
	Note   string `json:"note"`   // 지금 상태 한 줄
	Next   string `json:"next"`   // 넘어가려면 필요한 것
}

type Judgement struct {
	Questions []JudgeQ `json:"questions"`
	Verdict   string   `json:"verdict"` // 근거 부족, 계속, 수정: 닿는 길, 수정: 서비스, 수정: 고객 또는 문제
	NextStep  string   `json:"nextStep"`
}

type JudgeQ struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Have   int    `json:"have"`
	Need   int    `json:"need"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

var broadWords = []string{"누구나", "모든 사람", "모든사람", "모두", "사람들", "일반인", "직장인", "대학생", "2030", "20~40대", "20-40대", "남녀노소", "현대인", "사용자"}

// 고객 한 문장이 너무 넓은지 본다. 넓으면 그 이유를 돌려준다.
func customerProblems(c Customer) []string {
	out := []string{}
	if strings.TrimSpace(c.Who) == "" || strings.TrimSpace(c.Situation) == "" || strings.TrimSpace(c.Problem) == "" {
		out = append(out, "누구, 어떤 상황, 어떤 문제 세 칸을 모두 채워야 합니다.")
		return out
	}
	for _, w := range broadWords {
		if strings.Contains(c.Who, w) {
			out = append(out, fmt.Sprintf("\"누구\"에 들어간 \"%s\"은(는) 너무 넓습니다. 그 사람들이 모이는 곳을 한 군데 떠올릴 수 있을 만큼 좁혀 주세요.", w))
			break
		}
	}
	if len([]rune(strings.TrimSpace(c.Who))) < 6 {
		out = append(out, "\"누구\"가 너무 짧습니다. 나이대가 아니라 처한 상황으로 적어 보세요. (예: 돌 전 아기를 키우는 부모)")
	}
	if len([]rune(strings.TrimSpace(c.Situation))) < 6 {
		out = append(out, "\"어떤 상황\"이 너무 짧습니다. 그 문제가 생기는 구체적인 때를 적어 주세요.")
	}
	return out
}

func countEvidence(p *Project) (total, classified, strong int) {
	for _, e := range p.Evidence {
		total++
		if e.Experienced != nil {
			classified++
			if *e.Experienced && e.Tried != nil && *e.Tried {
				strong++
			}
		}
	}
	return
}

func postStats(p *Project) (posted, visits, deleted int) {
	for _, x := range p.Posts {
		if x.Posted {
			posted++
			visits += x.Visits
			if x.Deleted {
				deleted++
			}
		}
	}
	return
}

var stageTitles = []string{
	"고객·문제 정하기", "만들기 전 검증", "기획·AI 지시", "만들기", "출시 준비",
	"알리기·첫 사용자", "검증·판정", "고치기·다시 오게 하기", "수익 계산", "운영·기록",
}

var stageChecks = map[int][]string{
	3: {"build.small", "build.save", "build.secret", "build.mobile"},
	4: {"launch.analytics", "launch.survey", "launch.policy"},
	7: {"improve.fixone", "improve.return"},
	9: {"ops.cost", "ops.backup"},
}

func allChecked(p *Project, ids []string) (int, bool) {
	n := 0
	for _, id := range ids {
		if p.Checks[id] {
			n++
		}
	}
	return n, n == len(ids)
}

// 각 단계가 끝났는지 프로젝트 기록만 보고 계산한다. 사용자가 직접 "완료"를 누르는 버튼은 없다.
func computeStages(p *Project, th Thresholds) []Stage {
	total, classified, strong := countEvidence(p)
	posted, visits, _ := postStats(p)
	j := judge(p, th)

	done := make([]bool, 10)
	note := make([]string, 10)
	next := make([]string, 10)

	probs := customerProblems(p.Customer)
	done[0] = len(probs) == 0
	if done[0] {
		note[0] = "고객 한 문장을 정했습니다."
	} else {
		note[0] = "고객 한 문장을 다듬는 중입니다."
		next[0] = probs[0]
	}

	done[1] = strong >= th.Evidence
	note[1] = fmt.Sprintf("모은 글 %d건, 분류 %d건, 실제로 겪고 해결하려 한 글 %d건", total, classified, strong)
	if !done[1] {
		next[1] = fmt.Sprintf("실제로 겪고 해결하려 한 글이 %d건 더 필요합니다.", th.Evidence-strong)
	}

	done[2] = len(strings.TrimSpace(p.Plan)) >= 80
	if done[2] {
		note[2] = "한 장 기획서가 있습니다."
	} else {
		note[2] = "한 장 기획서가 아직 없습니다."
		next[2] = "한 장 기획서를 만들어 저장하세요."
	}

	n3, ok3 := allChecked(p, stageChecks[3])
	done[3] = ok3 && strings.HasPrefix(p.ServiceURL, "http")
	note[3] = fmt.Sprintf("점검 %d/%d, 서비스 주소 %s", n3, len(stageChecks[3]), yesNo(strings.HasPrefix(p.ServiceURL, "http"), "있음", "없음"))
	if !done[3] {
		next[3] = "점검 항목을 모두 확인하고 서비스 주소를 넣으세요."
	}

	n4, ok4 := allChecked(p, stageChecks[4])
	siteRan := p.SiteCheck != nil && p.SiteCheck.Error == "" && len(p.SiteCheck.Results) > 0
	mustLeft, niceLeft := 0, 0
	if siteRan {
		for _, r := range p.SiteCheck.Results {
			if !r.Pass && r.Must {
				mustLeft++
			} else if !r.Pass {
				niceLeft++
			}
		}
	}
	done[4] = ok4 && siteRan && mustLeft == 0
	switch {
	case !siteRan:
		note[4] = fmt.Sprintf("준비 %d/%d, 첫 화면 점검은 아직 안 함", n4, len(stageChecks[4]))
		next[4] = "첫 화면 점검을 실행하세요."
	case mustLeft > 0:
		note[4] = fmt.Sprintf("준비 %d/%d, 첫 화면에서 꼭 고칠 것 %d개, 고치면 좋은 것 %d개", n4, len(stageChecks[4]), mustLeft, niceLeft)
		next[4] = fmt.Sprintf("첫 화면에서 꼭 고칠 것 %d개를 고치고 다시 점검하세요.", mustLeft)
	default:
		note[4] = fmt.Sprintf("준비 %d/%d, 첫 화면에서 꼭 고칠 것 없음, 고치면 좋은 것 %d개", n4, len(stageChecks[4]), niceLeft)
		next[4] = "준비 항목을 모두 확인하세요."
	}

	done[5] = posted >= 2
	note[5] = fmt.Sprintf("올린 글 %d건, 글을 보고 들어온 방문 %d", posted, visits)
	if !done[5] {
		next[5] = fmt.Sprintf("글을 %d건 더 올리고 결과를 기록하세요.", 2-posted)
	}

	done[6] = j.Verdict == "계속"
	note[6] = "판정: " + j.Verdict
	if !done[6] {
		next[6] = j.NextStep
	}

	_, done[7] = allChecked(p, stageChecks[7])
	note[7] = yesNo(done[7], "고칠 점 하나와 다시 오게 할 장치를 정했습니다.", "아직 정하지 않았습니다.")
	if !done[7] {
		next[7] = "가장 많이 나온 불편 하나를 고치고, 다시 오게 할 장치 하나를 정하세요."
	}

	done[8] = p.Revenue.MonthlyVisitors > 0 && p.Revenue.RPM > 0 && p.Revenue.Target > 0
	note[8] = yesNo(done[8], "수익 계산을 해 봤습니다.", "아직 계산하지 않았습니다.")
	if !done[8] {
		next[8] = "방문자 수, 1,000번 노출당 수익, 목표 수익을 넣어 계산해 보세요."
	}

	_, ok9 := allChecked(p, stageChecks[9])
	done[9] = ok9 && len(p.Log) > 0
	note[9] = fmt.Sprintf("주간 기록 %d건", len(p.Log))
	if !done[9] {
		next[9] = "비용 상한과 백업을 확인하고 이번 주 기록을 한 줄 남기세요."
	}

	// 순서대로 풀린다. 앞 단계가 끝나야 다음 단계가 "지금 할 일"이 된다.
	// 단, 이미 서비스가 있는 사람이 0~2단계에 묶이지 않도록 서비스 주소가 있으면 3단계부터는 열어 둔다.
	stages := make([]Stage, 10)
	foundNow := false
	hasService := strings.HasPrefix(p.ServiceURL, "http")
	for i := range stages {
		st := Stage{ID: i, Title: stageTitles[i], Note: note[i], Next: next[i]}
		switch {
		case done[i]:
			st.Status = "done"
		case !foundNow:
			st.Status = "now"
			foundNow = true
		case hasService && i >= 3 && i <= 6:
			st.Status = "open"
		default:
			st.Status = "locked"
		}
		stages[i] = st
	}
	return stages
}

func yesNo(b bool, y, n string) string {
	if b {
		return y
	}
	return n
}

// 세 질문으로 계속·수정·중단을 정한다. 사람에게 직접 묻지 않고 얻을 수 있는 숫자만 쓴다.
func judge(p *Project, th Thresholds) Judgement {
	_, classified, strong := countEvidence(p)
	posted, visits, _ := postStats(p)

	qa := JudgeQ{Key: "가", Label: "문제가 진짜인가", Have: strong, Need: th.Evidence, Pass: strong >= th.Evidence,
		Detail: fmt.Sprintf("모은 글 중 글쓴이가 이 문제를 실제로 겪었고 스스로 해결하려 한 글 (분류한 글 %d건 중)", classified)}
	qb := JudgeQ{Key: "나", Label: "닿을 수 있는가", Have: visits, Need: th.Visits, Pass: visits >= th.Visits,
		Detail: fmt.Sprintf("내가 올린 글을 보고 서비스에 들어온 방문 수 (올린 글 %d건)", posted)}
	back := p.Survey.Helpful >= th.Helpful || p.Returning >= th.Return
	qc := JudgeQ{Key: "다", Label: "다시 오는가", Have: p.Survey.Helpful, Need: th.Helpful, Pass: back,
		Detail: fmt.Sprintf("익명 설문 \"도움 됨\" %d명(기준 %d) 또는 다시 온 사람 %d명(기준 %d)", p.Survey.Helpful, th.Helpful, p.Returning, th.Return)}

	j := Judgement{Questions: []JudgeQ{qa, qb, qc}}
	switch {
	case classified < th.Evidence && !qa.Pass:
		j.Verdict = "근거 부족"
		j.NextStep = fmt.Sprintf("아직 판정할 만큼 글을 보지 않았습니다. 글을 더 모아 최소 %d건을 분류하세요.", th.Evidence)
	case !qa.Pass:
		j.Verdict = "수정: 고객 또는 문제"
		j.NextStep = "글을 충분히 봤는데 이 문제를 겪는 사람이 적습니다. 서비스는 손대지 말고, 고객 한 문장을 바꿔 글을 다시 모으세요."
	case posted == 0:
		j.Verdict = "근거 부족"
		j.NextStep = "문제는 확인됐습니다. 이제 글을 올려 사람들이 들어오는지 보세요."
	case !qb.Pass:
		j.Verdict = "수정: 닿는 길"
		j.NextStep = "문제는 진짜인데 사람들이 들어오지 않았습니다. 올릴 곳을 바꾸거나 글을 고쳐 다시 올리세요. 서비스는 손대지 않습니다."
	case !qc.Pass:
		j.Verdict = "수정: 서비스"
		j.NextStep = "들어왔지만 남지 않았습니다. 익명 설문에 적힌 불편 가운데 가장 많은 것 하나를 고치세요."
	default:
		j.Verdict = "계속"
		j.NextStep = "반응이 나온 곳에서 같은 방식으로 2주 더 하세요. 기능은 요청받은 것 하나만 더합니다."
	}
	return j
}
