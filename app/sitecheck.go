package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// 만들기 도구가 처음에 넣어 두는 제목들. 이런 제목이 남아 있으면 고치지 않은 것이다.
var placeholderTitles = []string{"vite app", "vite + react", "react app", "create next app", "next.js", "document", "untitled", "lovable", "bolt", "my app", "home", "index", "v0 app", "replit"}

type pageFacts struct {
	title, description, viewport, robots, ogTitle, ogImage string
	textLen                                                int
	hasContact                                             bool
}

func readPage(r io.Reader) pageFacts {
	var f pageFacts
	z := html.NewTokenizer(r)
	skip := 0 // script, style 안의 글자는 세지 않는다
	inTitle := false
	for {
		switch z.Next() {
		case html.ErrorToken:
			return f
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			attr := func(k string) string {
				for _, a := range t.Attr {
					if strings.EqualFold(a.Key, k) {
						return a.Val
					}
				}
				return ""
			}
			switch t.Data {
			case "script", "style", "noscript":
				if t.Type == html.StartTagToken {
					skip++
				}
			case "title":
				inTitle = true
			case "meta":
				name := strings.ToLower(attr("name"))
				prop := strings.ToLower(attr("property"))
				switch {
				case name == "description":
					f.description = strings.TrimSpace(attr("content"))
				case name == "viewport":
					f.viewport = attr("content")
				case name == "robots":
					f.robots = strings.ToLower(attr("content"))
				case prop == "og:title":
					f.ogTitle = attr("content")
				case prop == "og:image":
					f.ogImage = attr("content")
				}
			case "a":
				h := strings.ToLower(attr("href"))
				if strings.HasPrefix(h, "mailto:") || strings.Contains(h, "open.kakao.com") || strings.Contains(h, "forms.gle") || strings.Contains(h, "docs.google.com/forms") || strings.Contains(h, "tally.so") || strings.Contains(h, "form.naver.com") {
					f.hasContact = true
				}
			}
		case html.EndTagToken:
			t := z.Token()
			switch t.Data {
			case "script", "style", "noscript":
				if skip > 0 {
					skip--
				}
			case "title":
				inTitle = false
			}
		case html.TextToken:
			txt := strings.TrimSpace(string(z.Text()))
			if inTitle {
				f.title += txt
				continue
			}
			if skip == 0 && txt != "" {
				f.textLen += len([]rune(txt))
				for _, w := range []string{"문의", "의견 보내기", "피드백", "만든 사람", "연락처"} {
					if strings.Contains(txt, w) {
						f.hasContact = true
					}
				}
			}
		}
	}
}

// 서비스 첫 화면을 한 번 열어 보고, 처음 온 사람과 검색·공유에 필요한 기본을 갖췄는지 본다.
func runSiteCheck(ctx context.Context, rawURL string) *SiteCheck {
	sc := &SiteCheck{At: time.Now().Format("2006-01-02 15:04"), URL: rawURL}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		sc.Error = "주소는 http:// 또는 https:// 로 시작해야 합니다."
		return sc
	}
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		sc.Error = "주소 형식이 올바르지 않습니다."
		return sc
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; VibeCoach site check)")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		sc.Error = "주소를 열지 못했습니다. 주소가 맞는지, 서비스가 켜져 있는지 확인해 주세요."
		return sc
	}
	defer resp.Body.Close()
	facts := readPage(io.LimitReader(resp.Body, 3<<20))
	elapsed := time.Since(start)

	add := func(id, label string, pass bool, detail, fix string) {
		if pass {
			fix = ""
		}
		sc.Results = append(sc.Results, CheckItem{ID: id, Label: label, Pass: pass, Detail: detail, Fix: fix})
	}

	add("status", "주소가 정상으로 열린다", resp.StatusCode == 200,
		fmt.Sprintf("응답 코드 %d", resp.StatusCode),
		"첫 화면 주소가 오류 없이 열리게 해 줘. 지금은 정상 응답이 오지 않아.")

	add("https", "안전한 주소(https)를 쓴다", resp.Request.URL.Scheme == "https",
		"최종 주소: "+resp.Request.URL.String(),
		"사이트가 https 주소로 열리게 설정해 줘. http로 들어와도 https로 넘어가게 해 줘.")

	add("speed", "3초 안에 열린다", elapsed < 3*time.Second,
		fmt.Sprintf("%.1f초 걸림", elapsed.Seconds()),
		"첫 화면이 3초 안에 열리게 해 줘. 큰 이미지와 처음에 필요 없는 스크립트를 줄여 줘.")

	title := strings.TrimSpace(facts.title)
	titleOK := len([]rune(title)) >= 8
	for _, ph := range placeholderTitles {
		if strings.EqualFold(title, ph) {
			titleOK = false
		}
	}
	add("title", "제목에 누구를 위한 무엇인지 적혀 있다", titleOK,
		"지금 제목: "+quoteOrNone(title),
		"브라우저 탭과 검색 결과에 보이는 페이지 제목(title)을 \"누구를 위한 무엇\"이 드러나는 한 문장으로 바꿔 줘. 지금은 "+quoteOrNone(title)+" 이야.")

	add("description", "검색 결과에 나올 설명이 있다", len([]rune(facts.description)) >= 20,
		"지금 설명: "+quoteOrNone(facts.description),
		"검색 결과에 보이는 설명(meta description)을 고객이 쓰는 말로 두 문장 넣어 줘.")

	add("viewport", "휴대폰 화면에 맞춰진다", strings.Contains(facts.viewport, "width=device-width"),
		yesNo(facts.viewport != "", "화면 맞춤 설정: "+facts.viewport, "화면 맞춤 설정이 없음"),
		"휴대폰에서 글자가 작게 보이지 않도록 화면 맞춤 설정(viewport)을 넣고, 휴대폰 너비에서 가로로 넘치는 곳이 없게 해 줘.")

	add("index", "검색에 나오지 않게 막혀 있지 않다", !strings.Contains(facts.robots, "noindex"),
		yesNo(facts.robots != "", "검색 설정: "+facts.robots, "막는 설정 없음"),
		"검색 엔진이 이 사이트를 싣지 못하게 막는 설정(noindex)을 지워 줘.")

	add("content", "화면을 그리기 전에도 읽을 글이 있다", facts.textLen >= 200,
		fmt.Sprintf("처음 받아 온 화면의 글자 수 %d자", facts.textLen),
		"첫 화면의 핵심 문구가 스크립트 실행 전에도 페이지에 들어 있게 해 줘. 지금은 처음 받아 온 화면에 글이 거의 없어서 검색과 링크 미리보기에 불리해.")

	add("share", "링크를 공유하면 미리보기가 뜬다", facts.ogTitle != "" && facts.ogImage != "",
		fmt.Sprintf("미리보기 제목 %s · 미리보기 그림 %s", yesNo(facts.ogTitle != "", "있음", "없음"), yesNo(facts.ogImage != "", "있음", "없음")),
		"카카오톡이나 카페에 링크를 올렸을 때 제목과 그림이 보이도록 공유 미리보기(og:title, og:description, og:image)를 넣어 줘.")

	add("contact", "의견을 보낼 곳이 있다", facts.hasContact,
		yesNo(facts.hasContact, "의견 창구로 보이는 링크나 문구가 있음", "의견 창구를 찾지 못함"),
		"화면 아래쪽에 \"의견 보내기\" 링크를 넣어 줘. 누르면 내가 준 오픈채팅방 또는 설문 주소가 열리게 해 줘.")

	return sc
}

func quoteOrNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(없음)"
	}
	return "\"" + s + "\""
}
