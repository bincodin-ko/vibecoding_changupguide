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
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
	select {}
}

func showError(msg string) { fmt.Fprintln(os.Stderr, msg) }
