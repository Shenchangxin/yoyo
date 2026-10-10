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
	wsExNoRedirectionBitmap = 0x00200000
	rgnOr                   = 2
)

type winRect struct {
	left, top, right, bottom int32
}

var (
	gdi32Shape             = syscall.NewLazyDLL("gdi32.dll")
	shcoreShape            = syscall.NewLazyDLL("shcore.dll")
	procSetWindowRgn       = user32Cursor.NewProc("SetWindowRgn")
	procGetWindowRect      = user32Cursor.NewProc("GetWindowRect")
	procGetClientRect      = user32Cursor.NewProc("GetClientRect")
	procGetAsyncKeyState   = user32Cursor.NewProc("GetAsyncKeyState")
	procMonitorFromPoint   = user32Cursor.NewProc("MonitorFromPoint")
	procGetDpiForMonitor   = shcoreShape.NewProc("GetDpiForMonitor")
	procEnumChildWindows   = user32Cursor.NewProc("EnumChildWindows")
	procCreateEllipticRgn  = gdi32Shape.NewProc("CreateEllipticRgn")
	procCreateRoundRectRgn = gdi32Shape.NewProc("CreateRoundRectRgn")
	procCombineRgn         = gdi32Shape.NewProc("CombineRgn")
	procDeleteObject       = gdi32Shape.NewProc("DeleteObject")
	applyShapeChild        = syscall.NewCallback(enumCompanionShape)
)

func procReady(p *syscall.LazyProc) bool {
	return p != nil && p.Find() == nil
}

func enumCompanionShape(hwnd uintptr, lparam uintptr) uintptr {
	setWindowCompanionRgn(w32.HWND(hwnd), true, lparam != 0)
	return 1
}

func setWindowCompanionRgn(hwnd w32.HWND, client, caption bool) {
	if hwnd == 0 || !procReady(procSetWindowRgn) {
		return
	}
	w, h := 0, 0
	if client && procReady(procGetClientRect) {
		var cr winRect
		okr, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&cr)))
		if okr == 0 {
			return
		}
		w = int(cr.right - cr.left)
		h = int(cr.bottom - cr.top)
	} else if procReady(procGetWindowRect) {
		var wr winRect
		okr, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
		if okr == 0 {
			return
		}
		w = int(wr.right - wr.left)
		h = int(wr.bottom - wr.top)
	}
	if w < 8 || h < 8 {
		return
	}
	rgn := createCompanionRgn(w, h, caption)
	if rgn == 0 {
		return
	}
	ok, _, _ := procSetWindowRgn.Call(uintptr(hwnd), rgn, 1)
	if ok == 0 && procReady(procDeleteObject) {
		_, _, _ = procDeleteObject.Call(rgn)
	}
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
}

func applyCompanionShape(win application.Window, caption bool) {
	restoreCompanionComposition(win)
	hwnd := companionHWND(win)
	if hwnd == 0 {
		return
	}
	setWindowCompanionRgn(hwnd, false, caption)
	if procReady(procEnumChildWindows) {
		var lp uintptr
		if caption {
			lp = 1
		}
		_, _, _ = procEnumChildWindows.Call(uintptr(hwnd), applyShapeChild, lp)
	}
}

// setCompanionPassthrough is a no-op on Windows. Toggling WS_EX_TRANSPARENT on
// the WebView2 child tree both fails to punch through the Chromium HWND and
// un-paints the blob on hover. Click-through comes from SetWindowRgn instead.
func setCompanionPassthrough(application.Window, bool) {}

func createCompanionRgn(winW, winH int, caption bool) uintptr {
	if companionInteractive.Load() && procReady(procCreateRoundRectRgn) {
		rad := int32(24)
		rgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(winW), uintptr(winH), uintptr(rad), uintptr(rad))
		return rgn
	}
	if !procReady(procCreateEllipticRgn) {
		return 0
	}
	cx, cy, rx, ry := companionBlobEllipse(float64(winW), float64(winH))
	blob, _, _ := procCreateEllipticRgn.Call(
		uintptr(int32(cx-rx)),
		uintptr(int32(cy-ry)),
		uintptr(int32(cx+rx+0.5)),
		uintptr(int32(cy+ry+0.5)),
	)
	if blob == 0 {
		return 0
	}
	if !caption || !procReady(procCreateRoundRectRgn) || !procReady(procCombineRgn) {
		return blob
	}
	x, y, w, h := companionCaptionRect(float64(winW), float64(winH))
	rad := int32(h)
	if rad < 8 {
		rad = 8
	}
	pill, _, _ := procCreateRoundRectRgn.Call(
		uintptr(int32(x)),
		uintptr(int32(y)),
		uintptr(int32(x+w+0.5)),
		uintptr(int32(y+h+0.5)),
		uintptr(rad),
		uintptr(rad),
	)
	if pill == 0 {
		return blob
	}
	_, _, _ = procCombineRgn.Call(blob, blob, pill, rgnOr)
	if procReady(procDeleteObject) {
		_, _, _ = procDeleteObject.Call(pill)
	}
	return blob
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
