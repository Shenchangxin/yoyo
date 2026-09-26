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
