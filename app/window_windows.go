//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
)

// Windows에 들어 있는 WebView2로 창을 띄운다.
// WebView2가 없는 컴퓨터에서는 기본 브라우저로 열고, 프로그램은 안내 창을 닫을 때까지 켜 둔다.
func openWindow(url, title string) {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  title,
			Width:  1180,
			Height: 820,
			Center: true,
		},
	})
	if w == nil {
		openBrowser(url)
		messageBox(title, "브라우저에서 프로그램 화면을 열었습니다.\n\n이 창을 닫으면 프로그램이 종료됩니다.", 0x40)
		return
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
}

// 주소를 기본 브라우저로 연다. 시험할 때 바꿔 끼울 수 있게 변수로 둔다.
var openBrowser = func(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func showError(msg string) { messageBox("바이브 창업 코치", msg, 0x10) }

func messageBox(title, text string, flags uintptr) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}
