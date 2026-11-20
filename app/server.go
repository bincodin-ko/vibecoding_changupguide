package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

type Server struct {
	store *Store
	token string // 이 프로그램의 화면만 기능을 부를 수 있게 하는 값
	host  string // 127.0.0.1:포트
}

func newToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type stateResponse struct {
	Project  *Project   `json:"project"`
	Settings settingsUI `json:"settings"`
	Stages   []Stage    `json:"stages"`
	Judge    Judgement  `json:"judge"`
	Customer []string   `json:"customerProblems"`
	DataDir  string     `json:"dataDir"`
}

// 화면으로 내보내는 설정. 키 자체는 넣지 않고 있는지만 알린다.
type settingsUI struct {
	HasAIKey   bool       `json:"hasAiKey"`
	AIModel    string     `json:"aiModel"`
	Thresholds Thresholds `json:"thresholds"`
}

func (s *Server) state() (*stateResponse, error) {
	p, err := s.store.LoadProject()
	if err != nil {
		return nil, err
	}
	st, err := s.store.LoadSettings()
	if err != nil {
		return nil, err
	}
	return &stateResponse{
		Project:  p,
		Settings: settingsUI{HasAIKey: st.AIKey != "", AIModel: st.AIModel, Thresholds: st.Thresholds},
		Stages:   computeStages(p, st.Thresholds),
		Judge:    judge(p, st.Thresholds),
		Customer: customerProblems(p.Customer),
		DataDir:  s.store.dir,
	}, nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) replyState(w http.ResponseWriter, extra map[string]any) {
	st, err := s.state()
	if err != nil {
		fail(w, 500, "기록 파일을 읽지 못했습니다: "+err.Error())
		return
	}
	out := map[string]any{"state": st}
	for k, v := range extra {
		out[k] = v
	}
	writeJSON(w, 200, out)
}

// 다른 웹사이트가 내 컴퓨터의 이 프로그램을 몰래 부르지 못하게, 주소와 토큰을 모두 확인한다.
func (s *Server) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.host {
			fail(w, 403, "허용되지 않은 접근입니다.")
			return
		}
		if r.Header.Get("X-App-Token") != s.token {
			fail(w, 403, "허용되지 않은 접근입니다.")
			return
		}
		s.store.mu.Lock()
		defer s.store.mu.Unlock()
		next(w, r)
	}
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 8<<20)).Decode(v)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")

	// 첫 화면에는 토큰을 심어서 내보낸다.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.host {
			http.Error(w, "forbidden", 403)
			return
		}
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			w.Header().Set("Cache-Control", "no-store")
			http.FileServer(http.FS(sub)).ServeHTTP(w, r)
			return
		}
		b, _ := fs.ReadFile(sub, "index.html")
		page := strings.Replace(string(b), "__APP_TOKEN__", s.token, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(page))
	})

	mux.HandleFunc("GET /api/state", s.guard(func(w http.ResponseWriter, r *http.Request) {
		s.replyState(w, nil)
	}))

	// 화면에서 고친 프로젝트 전체를 저장한다.
	mux.HandleFunc("PUT /api/project", s.guard(func(w http.ResponseWriter, r *http.Request) {
		var p Project
		if err := decode(r, &p); err != nil {
			fail(w, 400, "보낸 내용을 읽지 못했습니다.")
			return
		}
		if err := s.store.SaveProject(&p); err != nil {
			fail(w, 500, "저장하지 못했습니다: "+err.Error())
			return
		}
		s.replyState(w, nil)
	}))

	mux.HandleFunc("POST /api/settings", s.guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			AIKey      *string     `json:"aiKey"`
			AIModel    *string     `json:"aiModel"`
			Thresholds *Thresholds `json:"thresholds"`
		}
		if err := decode(r, &in); err != nil {
			fail(w, 400, "보낸 내용을 읽지 못했습니다.")
			return
		}
		st, err := s.store.LoadSettings()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		// 빈 칸으로 보내면 기존 값을 그대로 둔다. 지우려면 "-"를 보낸다.
		set := func(dst *string, v *string) {
			if v == nil || strings.TrimSpace(*v) == "" {
				return
			}
			if strings.TrimSpace(*v) == "-" {
				*dst = ""
				return
			}
			*dst = strings.TrimSpace(*v)
		}
		set(&st.AIKey, in.AIKey)
		set(&st.AIModel, in.AIModel)
		if in.Thresholds != nil {
			st.Thresholds = *in.Thresholds
		}
		if err := s.store.SaveSettings(st); err != nil {
			fail(w, 500, "설정을 저장하지 못했습니다: "+err.Error())
			return
		}
		s.replyState(w, nil)
	}))

	// 주소를 사용자의 기본 브라우저로 연다. 검색 바로가기와 모은 글의 링크에 쓴다.
	mux.HandleFunc("POST /api/open", s.guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			URL     string `json:"url"`
			Place   string `json:"place"`
			Keyword string `json:"keyword"`
		}
		if err := decode(r, &in); err != nil {
			fail(w, 400, "보낸 내용을 읽지 못했습니다.")
			return
		}
		target := in.URL
		if in.Place != "" {
			u, err := searchURL(in.Place, in.Keyword)
			if err != nil {
				fail(w, 400, err.Error())
				return
			}
			target = u
			// 어떤 검색어로 찾아봤는지는 사용자가 넣은 말이므로 기록해 둔다.
			p, _ := s.store.LoadProject()
			kw := strings.TrimSpace(in.Keyword)
			has := false
			for _, k := range p.Keywords {
				has = has || k == kw
			}
			if !has {
				p.Keywords = append(p.Keywords, kw)
				s.store.SaveProject(p)
			}
		}
		if err := checkOpenURL(target); err != nil {
			fail(w, 400, err.Error())
			return
		}
		if err := openBrowser(target); err != nil {
			fail(w, 500, "브라우저를 열지 못했습니다. 주소를 직접 복사해 여세요: "+target)
			return
		}
		s.replyState(w, map[string]any{"opened": target})
	}))

	mux.HandleFunc("POST /api/collect/paste", s.guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Text string `json:"text"`
		}
		if err := decode(r, &in); err != nil || strings.TrimSpace(in.Text) == "" {
			fail(w, 400, "붙여넣은 내용이 없습니다.")
			return
		}
		p, _ := s.store.LoadProject()
		items := parsePasted(in.Text)
		added := addEvidence(p, items)
		if err := s.store.SaveProject(p); err != nil {
			fail(w, 500, err.Error())
			return
		}
		s.replyState(w, map[string]any{"found": len(items), "added": added})
	}))

	mux.HandleFunc("POST /api/ai/classify", s.guard(func(w http.ResponseWriter, r *http.Request) {
		st, _ := s.store.LoadSettings()
		p, _ := s.store.LoadProject()
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()
		done, remaining, err := classifyEvidence(ctx, st, p)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		if err := s.store.SaveProject(p); err != nil {
			fail(w, 500, err.Error())
			return
		}
		s.replyState(w, map[string]any{"done": done, "remaining": remaining})
	}))

	// 지시문만 돌려준다. 키가 없을 때 사용자가 복사해 자기 AI에 붙여넣는다.
	mux.HandleFunc("POST /api/prompt", s.guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Kind   string            `json:"kind"` // ai, tool
			Task   string            `json:"task"`
			Params map[string]string `json:"params"`
		}
		if err := decode(r, &in); err != nil {
			fail(w, 400, "보낸 내용을 읽지 못했습니다.")
			return
		}
		p, _ := s.store.LoadProject()
		var text string
		var err error
		if in.Kind == "tool" {
			text, err = buildToolPrompt(in.Task, p)
		} else {
			text, err = buildPrompt(in.Task, p, in.Params)
		}
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"text": text})
	}))

	mux.HandleFunc("POST /api/ai/run", s.guard(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Task   string            `json:"task"`
			Params map[string]string `json:"params"`
		}
		if err := decode(r, &in); err != nil {
			fail(w, 400, "보낸 내용을 읽지 못했습니다.")
			return
		}
		st, _ := s.store.LoadSettings()
		p, _ := s.store.LoadProject()
		prompt, err := buildPrompt(in.Task, p, in.Params)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()
		text, err := callAI(ctx, st, prompt, nil)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]string{"text": text})
	}))

	mux.HandleFunc("POST /api/sitecheck", s.guard(func(w http.ResponseWriter, r *http.Request) {
		p, _ := s.store.LoadProject()
		if strings.TrimSpace(p.ServiceURL) == "" {
			fail(w, 400, "먼저 서비스 주소를 넣어 주세요.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		p.SiteCheck = runSiteCheck(ctx, strings.TrimSpace(p.ServiceURL))
		if err := s.store.SaveProject(p); err != nil {
			fail(w, 500, err.Error())
			return
		}
		s.replyState(w, nil)
	}))

	return mux
}
