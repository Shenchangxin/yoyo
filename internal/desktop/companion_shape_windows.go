//go:build windows

package desktop

import (
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

const (
	monitorDefaultNearest   = 2
	mdtEffectiveDPI         = 0
	vkLButton               = 0x01
	wsExLayered             = 0x00080000
	wsExTransparent         = 0x00000020
	wsExNoRedirectionBitmap = 0x00200000
)

type winRect struct {
	left, top, right, bottom int32
}

var (
	shcoreShape          = syscall.NewLazyDLL("shcore.dll")
	procSetWindowRgn     = user32Cursor.NewProc("SetWindowRgn")
	procGetWindowRect    = user32Cursor.NewProc("GetWindowRect")
	procGetAsyncKeyState = user32Cursor.NewProc("GetAsyncKeyState")
	procMonitorFromPoint = user32Cursor.NewProc("MonitorFromPoint")
	procGetDpiForMonitor = shcoreShape.NewProc("GetDpiForMonitor")
)

func procReady(p *syscall.LazyProc) bool {
	return p != nil && p.Find() == nil
}

func companionHWND(win application.Window) w32.HWND {
	wv, ok := win.(*application.WebviewWindow)
	if !ok || wv == nil {
		return 0
	}
	return w32.HWND(uintptr(wv.NativeWindow()))
}

func restoreCompanionComposition(win application.Window) {
	hwnd := companionHWND(win)
	if hwnd == 0 {
		return
	}
	style := w32.GetWindowLongPtr(hwnd, w32.GWL_EXSTYLE)
	next := (style | wsExNoRedirectionBitmap) &^ wsExLayered
	if next != style {
		w32.SetWindowLongPtr(hwnd, w32.GWL_EXSTYLE, next)
		w32.SetWindowPos(hwnd, 0, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE|w32.SWP_FRAMECHANGED)
	}
	if procReady(procSetWindowRgn) {
		_, _, _ = procSetWindowRgn.Call(uintptr(hwnd), 0, 1)
	}
}

func applyCompanionShape(win application.Window, _ bool) {
	restoreCompanionComposition(win)
}

func setCompanionPassthrough(win application.Window, ignore bool) {
	hwnd := companionHWND(win)
	if hwnd == 0 {
		return
	}
	style := w32.GetWindowLongPtr(hwnd, w32.GWL_EXSTYLE)
	next := (style | wsExNoRedirectionBitmap) &^ wsExLayered
	if ignore {
		next |= wsExTransparent
	} else {
		next &^= wsExTransparent
	}
	if next == style {
		return
	}
	w32.SetWindowLongPtr(hwnd, w32.GWL_EXSTYLE, next)
}

func companionCursorLocalNative(win application.Window) (lx, ly, ww, wh float64, ok bool) {
	hwnd := companionHWND(win)
	if hwnd == 0 || !procReady(procGetCursorPos) || !procReady(procGetWindowRect) {
		return 0, 0, 0, 0, false
	}
	var pt cursorPoint
	okr, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if okr == 0 {
		return 0, 0, 0, 0, false
	}
	var wr winRect
	okr, _, _ = procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
	if okr == 0 {
		return 0, 0, 0, 0, false
	}
	return float64(pt.x - wr.left), float64(pt.y - wr.top), float64(wr.right - wr.left), float64(wr.bottom - wr.top), true
}

func leftMouseDown() bool {
	if !procReady(procGetAsyncKeyState) {
		return false
	}
	v, _, _ := procGetAsyncKeyState.Call(vkLButton)
	return v&0x8000 != 0
}

func dpiAt(x, y int32) int {
	hmon, _, _ := procMonitorFromPoint.Call(uintptr(int32(x)), uintptr(int32(y)), monitorDefaultNearest)
	if hmon != 0 && procGetDpiForMonitor.Find() == nil {
		var dpiX, dpiY uint32
		hr, _, _ := procGetDpiForMonitor.Call(hmon, mdtEffectiveDPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
		if hr == 0 && dpiX >= 96 {
			return int(dpiX)
		}
	}
	dpi, _, _ := procGetDpiForSystem.Call()
	if dpi < 96 {
		return 96
	}
	return int(dpi)
}
