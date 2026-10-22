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
	wsExTransparent         = 0x00000020
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
	procGetAsyncKeyState   = user32Cursor.NewProc("GetAsyncKeyState")
	procMonitorFromPoint   = user32Cursor.NewProc("MonitorFromPoint")
	procGetDpiForMonitor   = shcoreShape.NewProc("GetDpiForMonitor")
	procEnumChildWindows   = user32Cursor.NewProc("EnumChildWindows")
	procCreateRoundRectRgn = gdi32Shape.NewProc("CreateRoundRectRgn")
	procCombineRgn         = gdi32Shape.NewProc("CombineRgn")
	procDeleteObject       = gdi32Shape.NewProc("DeleteObject")
	applyShapeChild        = syscall.NewCallback(enumCompanionShape)
	shapeParentW           int
	shapeParentH           int
)

func procReady(p *syscall.LazyProc) bool {
	return p != nil && p.Find() == nil
}

func enumCompanionShape(hwnd uintptr, lparam uintptr) uintptr {
	if !procReady(procGetWindowRect) {
		return 1
	}
	var wr winRect
	okr, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	if okr == 0 {
		return 1
	}
	w := int(wr.right - wr.left)
	h := int(wr.bottom - wr.top)
	// Nested Chromium widgets are often a few dozen pixels, or still the
	// previous size while the parent has already resized. Scaling the pet
	// clip onto them hides the call page. During a call, drop that clip.
	switch companionChildRegion(w, h, shapeParentW, shapeParentH, companionInteractive.Load()) {
	case companionChildClear:
		clearWindowRgn(w32.HWND(hwnd))
	case companionChildSkip:
		return 1
	default:
		setWindowCompanionRgn(w32.HWND(hwnd), lparam != 0)
	}
	return 1
}

func clearWindowRgn(hwnd w32.HWND) {
	if hwnd == 0 || !procReady(procSetWindowRgn) {
		return
	}
	_, _, _ = procSetWindowRgn.Call(uintptr(hwnd), 0, 1)
}

func setWindowCompanionRgn(hwnd w32.HWND, caption bool) {
	if hwnd == 0 || !procReady(procSetWindowRgn) || !procReady(procGetWindowRect) {
		return
	}
	var wr winRect
	okr, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
	if okr == 0 {
		return
	}
	w := int(wr.right - wr.left)
	h := int(wr.bottom - wr.top)
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
	next := style | wsExNoRedirectionBitmap
	if companionClickThrough.Load() {
		// Persistent click-through for Focus Mode. Hover-toggling WS_EX_TRANSPARENT
		// un-paints the WebView2 blob; a stable flag does not.
		next |= wsExLayered | wsExTransparent
	} else {
		next &^= wsExLayered
		next &^= wsExTransparent
	}
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
	if procReady(procGetWindowRect) {
		var wr winRect
		if okr, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr))); okr != 0 {
			shapeParentW = int(wr.right - wr.left)
			shapeParentH = int(wr.bottom - wr.top)
		}
	}
	setWindowCompanionRgn(hwnd, caption)
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
	if !procReady(procCreateRoundRectRgn) {
		return 0
	}
	if companionInteractive.Load() {
		rad := int32(24)
		rgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(winW), uintptr(winH), uintptr(rad), uintptr(rad))
		return rgn
	}
	x, y, w, h := companionPetRegionRect(float64(winW), float64(winH), caption)
	// GDI wants the ellipse diameter. A diameter of min(w,h) is a circle
	// clip — the blob hips sit outside that disk.
	rad := companionRegionCornerRadius(w, h, min(float64(winW)/float64(companionW), float64(winH)/companionDesignH(caption)))
	dia := int32(rad*2 + 0.5)
	if dia < 16 {
		dia = 16
	}
	blob, _, _ := procCreateRoundRectRgn.Call(
		uintptr(int32(x)),
		uintptr(int32(y)),
		uintptr(int32(x+w+0.5)),
		uintptr(int32(y+h+0.5)),
		uintptr(dia),
		uintptr(dia),
	)
	if blob == 0 {
		return 0
	}
	if !caption || !procReady(procCombineRgn) {
		return blob
	}
	cx, cy, cw, ch := companionCaptionRect(float64(winW), float64(winH), caption)
	_, _, s := companionScale(float64(winW), float64(winH), caption)
	cssR := float64(companionCaptionCorner) * s
	pillDia := int32(cssR*2 + 0.5)
	short := cw
	if ch < short {
		short = ch
	}
	if float64(pillDia) > short {
		pillDia = int32(short)
	}
	if pillDia < 12 {
		pillDia = 12
	}
	pill, _, _ := procCreateRoundRectRgn.Call(
		uintptr(int32(cx)),
		uintptr(int32(cy)),
		uintptr(int32(cx+cw+0.5)),
		uintptr(int32(cy+ch+0.5)),
		uintptr(pillDia),
		uintptr(pillDia),
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
