// 바이브 창업 코치: AI로 서비스를 만드는 1인 초보 창업자가
// 고객 정하기부터 수익 계산까지 단계별로 진행하도록 돕는 데스크톱 프로그램.
//
// 구조: 프로그램이 내 컴퓨터 안에서만 열리는 작은 서버를 띄우고,
// 그 화면을 창(Windows에서는 WebView2)으로 보여 준다.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	dataDir := flag.String("data", "", "기록을 저장할 폴더 (비우면 사용자 설정 폴더)")
	port := flag.Int("port", 0, "쓸 포트 (0이면 비어 있는 포트를 고름)")
	noWindow := flag.Bool("no-window", false, "창을 띄우지 않고 주소만 출력")
	flag.Parse()

	store, err := NewStore(*dataDir)
	if err != nil {
		fatal("기록 폴더를 만들지 못했습니다: " + err.Error())
	}

	// 127.0.0.1에만 연다. 같은 네트워크의 다른 컴퓨터에서는 접속할 수 없다.
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fatal("프로그램을 시작하지 못했습니다: " + err.Error())
	}
	srv := &Server{store: store, token: newToken(), host: ln.Addr().String()}
	url := "http://" + srv.host + "/"

	go func() {
		if err := http.Serve(ln, srv.routes()); err != nil {
			log.Println(err)
		}
	}()

	if *noWindow {
		fmt.Println(url)
		select {}
	}
	openWindow(url, "바이브 창업 코치")
}

func fatal(msg string) {
	showError(msg)
	os.Exit(1)
}
