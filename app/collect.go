package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

var urlRe = regexp.MustCompile(`https?://\S+`)

// 검색어로 네이버의 검색 화면 주소를 만든다. 프로그램이 글을 가져오지 않고,
// 사용자의 브라우저에서 네이버 검색을 열어 줄 뿐이다.
// (네이버 검색 API는 약관상 결과를 저장하거나 AI에 넣을 수 없어 쓰지 않는다.)
var searchPlaces = map[string]string{"kin": "kin", "cafe": "cafearticle", "blog": "blog"}

func searchURL(place, keyword string) (string, error) {
	where, ok := searchPlaces[place]
	if !ok || strings.TrimSpace(keyword) == "" {
		return "", fmt.Errorf("검색어를 넣어 주세요.")
	}
	return "https://search.naver.com/search.naver?where=" + where + "&query=" + url.QueryEscape(strings.TrimSpace(keyword)), nil
}

// 브라우저로 열어도 되는 주소인지 본다. 웹 주소만 연다.
func checkOpenURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("열 수 없는 주소입니다.")
	}
	return nil
}

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
