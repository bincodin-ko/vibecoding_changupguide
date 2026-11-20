package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// 프로젝트 하나의 모든 기록. data.json 파일 하나에 저장된다.
type Project struct {
	Name       string            `json:"name"`
	ServiceURL string            `json:"serviceUrl"`
	Customer   Customer          `json:"customer"`
	Keywords   []string          `json:"keywords"`
	Evidence   []Evidence        `json:"evidence"`
	Plan       string            `json:"plan"`
	Checks     map[string]bool   `json:"checks"`
	SiteCheck  *SiteCheck        `json:"siteCheck"`
	Places     []Place           `json:"places"`
	Posts      []Post            `json:"posts"`
	Survey     Survey            `json:"survey"`
	Returning  int               `json:"returning"`
	Revenue    Revenue           `json:"revenue"`
	Log        []LogEntry        `json:"log"`
	Drafts     map[string]string `json:"drafts"`
}

type Customer struct {
	Who       string `json:"who"`
	Situation string `json:"situation"`
	Problem   string `json:"problem"`
	Solution  string `json:"solution"`
}

// 고객이 이미 쓴 글 한 건. Experienced/Tried가 nil이면 아직 분류하지 않은 글이다.
type Evidence struct {
	ID          string `json:"id"`
	Source      string `json:"source"` // 붙여넣기
	Keyword     string `json:"keyword"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet"`
	Link        string `json:"link"`
	Date        string `json:"date"`
	Experienced *bool  `json:"experienced"` // 글쓴이가 이 문제를 실제로 겪었나
	Tried       *bool  `json:"tried"`       // 스스로 해결하려 한 흔적이 있나
	Problem     string `json:"problem"`     // 어떤 문제인지 한 줄
	Quote       string `json:"quote"`       // 글쓴이의 표현 그대로
	By          string `json:"by"`          // 분류한 주체: ai, me
}

type SiteCheck struct {
	At      string      `json:"at"`
	URL     string      `json:"url"`
	Error   string      `json:"error"`
	Results []CheckItem `json:"results"`
}

type CheckItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
	Fix    string `json:"fix"`  // 통과하지 못했을 때 코딩 도구에 줄 지시
	Must   bool   `json:"must"` // 이게 안 되면 단계를 통과할 수 없는 항목
}

type Place struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Rules       string `json:"rules"`
	LinkAllowed bool   `json:"linkAllowed"`
	RulesRead   bool   `json:"rulesRead"`
}

type Post struct {
	ID       string `json:"id"`
	PlaceID  string `json:"placeId"`
	Date     string `json:"date"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Posted   bool   `json:"posted"`
	Views    int    `json:"views"`
	Comments int    `json:"comments"`
	Visits   int    `json:"visits"`
	Deleted  bool   `json:"deleted"`
}

type Survey struct {
	Helpful    int      `json:"helpful"`
	NotHelpful int      `json:"notHelpful"`
	Comments   []string `json:"comments"`
}

type Revenue struct {
	MonthlyVisitors float64 `json:"monthlyVisitors"`
	PagesPerVisit   float64 `json:"pagesPerVisit"`
	RPM             float64 `json:"rpm"`    // 1,000번 노출당 수익(원). 사용자가 직접 넣는다.
	Target          float64 `json:"target"` // 목표 월 수익(원)
}

type LogEntry struct {
	Date string `json:"date"`
	Text string `json:"text"`
}

// 프로그램 설정. 키는 화면으로 다시 내보내지 않는다.
type Settings struct {
	AIKey      string     `json:"aiKey"`
	AIModel    string     `json:"aiModel"`
	Thresholds Thresholds `json:"thresholds"`
}

// 판정 기준. 출발점으로 잡은 값이며 설정에서 바꿀 수 있다.
type Thresholds struct {
	Evidence int `json:"evidence"` // 실제로 겪고 해결하려 한 글 수
	Visits   int `json:"visits"`   // 올린 글을 보고 들어온 방문 수
	Helpful  int `json:"helpful"`  // 익명 설문 "도움 됨" 수
	Return   int `json:"return"`   // 다시 온 사람 수
}

func defaultThresholds() Thresholds {
	return Thresholds{Evidence: 10, Visits: 30, Helpful: 3, Return: 5}
}

const defaultModel = "claude-opus-5-5"

type Store struct {
	mu  sync.Mutex
	dir string
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(base, "VibeCoach")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) read(name string, v any) error {
	b, err := os.ReadFile(filepath.Join(s.dir, name))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// 쓰다가 프로그램이 꺼져도 기존 파일이 깨지지 않게 임시 파일에 쓴 뒤 바꿔치기한다.
func (s *Store) write(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) LoadProject() (*Project, error) {
	p := &Project{}
	if err := s.read("project.json", p); err != nil {
		return nil, err
	}
	normalize(p)
	return p, nil
}

func (s *Store) SaveProject(p *Project) error {
	normalize(p)
	return s.write("project.json", p)
}

func (s *Store) LoadSettings() (*Settings, error) {
	st := &Settings{}
	if err := s.read("settings.json", st); err != nil {
		return nil, err
	}
	if st.AIModel == "" {
		st.AIModel = defaultModel
	}
	d := defaultThresholds()
	if st.Thresholds.Evidence <= 0 {
		st.Thresholds.Evidence = d.Evidence
	}
	if st.Thresholds.Visits <= 0 {
		st.Thresholds.Visits = d.Visits
	}
	if st.Thresholds.Helpful <= 0 {
		st.Thresholds.Helpful = d.Helpful
	}
	if st.Thresholds.Return <= 0 {
		st.Thresholds.Return = d.Return
	}
	return st, nil
}

func (s *Store) SaveSettings(st *Settings) error { return s.write("settings.json", st) }

// 화면이 빈 값(null)을 다루지 않아도 되게 목록과 맵을 항상 채워 둔다.
func normalize(p *Project) {
	if p.Keywords == nil {
		p.Keywords = []string{}
	}
	if p.Evidence == nil {
		p.Evidence = []Evidence{}
	}
	if p.Checks == nil {
		p.Checks = map[string]bool{}
	}
	if p.Places == nil {
		p.Places = []Place{}
	}
	if p.Posts == nil {
		p.Posts = []Post{}
	}
	if p.Survey.Comments == nil {
		p.Survey.Comments = []string{}
	}
	if p.Log == nil {
		p.Log = []LogEntry{}
	}
	if p.Drafts == nil {
		p.Drafts = map[string]string{}
	}
}
