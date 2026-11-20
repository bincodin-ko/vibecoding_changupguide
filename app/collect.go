package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// 네이버 검색 API 주소. 시험할 때 가짜 서버로 바꿀 수 있게 변수로 둔다.
var naverBase = "https://openapi.naver.com"

var naverKinds = map[string]string{"blog": "블로그", "cafearticle": "카페", "kin": "지식iN"}

var tagRe = regexp.MustCompile(`<[^>]+>`)

func cleanText(s string) string {
	return strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(s, "")))
}

// 네이버 검색 API로 키워드에 대한 글을 찾아 온다. 제목과 요약만 가져오며 본문은 읽지 않는다.
func collectNaver(ctx context.Context, st *Settings, keyword string, kinds []string) ([]Evidence, error) {
	if st.NaverID == "" || st.NaverSecret == "" {
		return nil, fmt.Errorf("네이버 검색 키가 없습니다. 설정에서 넣거나, 찾은 글을 직접 붙여넣어 주세요.")
	}
	if len(kinds) == 0 {
		kinds = []string{"kin", "cafearticle", "blog"}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	var out []Evidence
	for _, kind := range kinds {
		label, ok := naverKinds[kind]
		if !ok {
			continue
		}
		u := fmt.Sprintf("%s/v1/search/%s.json?display=30&sort=sim&query=%s", naverBase, kind, url.QueryEscape(keyword))
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("X-Naver-Client-Id", st.NaverID)
		req.Header.Set("X-Naver-Client-Secret", st.NaverSecret)
		resp, err := client.Do(req)
		if err != nil {
			return out, fmt.Errorf("네이버 검색에 연결하지 못했습니다. 인터넷 연결을 확인해 주세요. (%v)", err)
		}
		var body struct {
			ErrorMessage string `json:"errorMessage"`
			Items        []struct {
				Title       string `json:"title"`
				Link        string `json:"link"`
				Description string `json:"description"`
				Postdate    string `json:"postdate"`
			} `json:"items"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return out, fmt.Errorf("네이버 검색 키가 맞지 않습니다. 설정에서 Client ID와 Secret을 다시 확인해 주세요.")
		}
		if resp.StatusCode == 429 {
			return out, fmt.Errorf("네이버 검색의 하루 사용량을 넘었습니다. 내일 다시 시도해 주세요.")
		}
		if resp.StatusCode != 200 || err != nil {
			return out, fmt.Errorf("네이버 검색이 실패했습니다. (%d %s)", resp.StatusCode, body.ErrorMessage)
		}
		for _, it := range body.Items {
			date := it.Postdate
			if len(date) == 8 {
				date = date[:4] + "-" + date[4:6] + "-" + date[6:]
			}
			out = append(out, Evidence{ID: newID(), Source: label, Keyword: keyword,
				Title: cleanText(it.Title), Snippet: cleanText(it.Description), Link: it.Link, Date: date})
		}
	}
	return out, nil
}

var urlRe = regexp.MustCompile(`https?://\S+`)

// 사용자가 붙여넣은 글을 빈 줄 기준으로 나눠 한 건씩 근거로 만든다.
// 첫 줄은 제목, 나머지는 내용으로 보고, 안에 주소가 있으면 링크로 뺀다.
func parsePasted(text string) []Evidence {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []Evidence
	for _, block := range regexp.MustCompile(`\n\s*\n`).Split(text, -1) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		link := urlRe.FindString(block)
		body := strings.TrimSpace(urlRe.ReplaceAllString(block, ""))
		if body == "" {
			continue
		}
		lines := strings.SplitN(body, "\n", 2)
		title := strings.TrimSpace(lines[0])
		snippet := ""
		if len(lines) > 1 {
			snippet = strings.TrimSpace(lines[1])
		}
		if r := []rune(title); len(r) > 80 && snippet == "" {
			snippet = title
			title = string(r[:60]) + "…"
		}
		out = append(out, Evidence{ID: newID(), Source: "붙여넣기", Title: title, Snippet: snippet, Link: link})
	}
	return out
}

// 이미 있는 글(같은 링크, 또는 링크가 없으면 같은 제목)은 다시 넣지 않는다.
func addEvidence(p *Project, items []Evidence) int {
	seen := map[string]bool{}
	for _, e := range p.Evidence {
		seen[evidenceKey(e)] = true
	}
	added := 0
	for _, e := range items {
		k := evidenceKey(e)
		if seen[k] {
			continue
		}
		seen[k] = true
		p.Evidence = append(p.Evidence, e)
		added++
	}
	return added
}

func evidenceKey(e Evidence) string {
	if e.Link != "" {
		return e.Link
	}
	return e.Title + "|" + e.Snippet
}
