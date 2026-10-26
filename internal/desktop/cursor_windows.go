//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

var (
	user32Cursor        = syscall.NewLazyDLL("user32.dll")
	procGetCursorPos    = user32Cursor.NewProc("GetCursorPos")
	procGetDpiForSystem = user32Cursor.NewProc("GetDpiForSystem")
)

type cursorPoint struct {
	x, y int32
}

// desktopCursorCSS 读取屏幕光标，并按该点 DPI 除以 96 换成 CSS 像素。GetCursorPos 失败时返回假。
func desktopCursorCSS() (x, y float64, ok bool) {
	var pt cursorPoint
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r == 0 {
		return 0, 0, false
	}
	dpi := dpiAt(pt.x, pt.y)
	if dpi < 96 {
		dpi = 96
	}
	scale := float64(dpi) / 96
	return float64(pt.x) / scale, float64(pt.y) / scale, true
}
