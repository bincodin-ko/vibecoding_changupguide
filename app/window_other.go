//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Windows가 아닌 곳(개발·시험용)에서는 기본 브라우저로 연다.
func openWindow(url, title string) {
	fmt.Println(title + ": " + url)
	openBrowser(url)
	select {}
}

// 주소를 기본 브라우저로 연다. 시험할 때 바꿔 끼울 수 있게 변수로 둔다.
var openBrowser = func(url string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}

func showError(msg string) { fmt.Fprintln(os.Stderr, msg) }
