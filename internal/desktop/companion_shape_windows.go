//go:build windows

// companion_shape_windows.go 拥有 Windows 上宠物窗口的区域裁剪、组合样式和光标换算。命中几何在 companion_hit.go。companion.go 在显示和调整时调用 applyCompanionShape。

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

// procReady 在过程已加载且 Find 成功时返回真。
func procReady(p *syscall.LazyProc) bool {
	return p != nil && p.Find() == nil
}

// enumCompanionShape 是 EnumChildWindows 的回调。它按 companionChildRegion 决定清掉子窗口区域、跳过，或给子窗口贴宠物区域。返回 1 以继续枚举。
// hwnd 传给 GetWindowRect。要清掉区域时传给 clearWindowRgn，要贴区域时转成 HWND 传给 setWindowCompanionRgn。
// lparam 只在贴宠物区域时转成布尔值传给 setWindowCompanionRgn，非 0 为真。
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
	// 嵌套的 Chromium 窗口常常只有几十像素，或者父窗口已经改了大小而子窗口还是旧尺寸。把宠物裁剪缩放到它们上面会挡住通话页。通话中丢掉这份旧裁剪。
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

// clearWindowRgn 调用 SetWindowRgn 把区域设成空，从而去掉旧裁剪。句柄为 0 或函数没加载时返回。
// hwnd 为 0 或 SetWindowRgn 没加载时返回。否则传给 SetWindowRgn。
func clearWindowRgn(hwnd w32.HWND) {
	if hwnd == 0 || !procReady(procSetWindowRgn) {
		return
	}
	_, _, _ = procSetWindowRgn.Call(uintptr(hwnd), 0, 1)
}

// setWindowCompanionRgn 按窗口矩形创建宠物区域并 SetWindowRgn。创建失败则返回。SetWindowRgn 失败时删掉区域对象。
// hwnd 为 0，或 SetWindowRgn、GetWindowRect 没加载时返回。否则用来取窗口矩形，再把创建的区域传给 SetWindowRgn。
// caption 和窗口宽高一起传给 createCompanionRgn。
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

// companionHWND 从 WebviewWindow 取出原生句柄。类型不对或窗口为空时返回 0。
// win 断言成 *application.WebviewWindow。类型不对或为空时返回 0。否则取 NativeWindow。
func companionHWND(win application.Window) w32.HWND {
	wv, ok := win.(*application.WebviewWindow)
	if !ok || wv == nil {
		return 0
	}
	return w32.HWND(uintptr(wv.NativeWindow()))
}

// restoreCompanionComposition 给宠物窗口加上 WS_EX_NOREDIRECTIONBITMAP。专注穿透开着时再加上 WS_EX_LAYERED 和 WS_EX_TRANSPARENT。否则去掉这两项。样式有变化时用 SetWindowPos 刷框架。
// win 传给 companionHWND。
func restoreCompanionComposition(win application.Window) {
	hwnd := companionHWND(win)
	if hwnd == 0 {
		return
	}
	style := w32.GetWindowLongPtr(hwnd, w32.GWL_EXSTYLE)
	next := style | wsExNoRedirectionBitmap
	if companionClickThrough.Load() {
		// 专注模式用稳定的点击穿透。悬停时来回切换 WS_EX_TRANSPARENT 会让 WebView2 的宠物不再绘制；标志保持不动则不会。
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

// applyCompanionShape 先恢复组合样式，再给父窗口和每个子窗口贴区域。caption 为真时枚举回调收到非 0 的 lparam。
// win 传给 restoreCompanionComposition 和 companionHWND。句柄非 0 时再传给 setWindowCompanionRgn 和 EnumChildWindows。
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

// setCompanionPassthrough 在 Windows 上是空操作。悬停切换 WS_EX_TRANSPARENT 既穿不透 Chromium 窗口，也会让宠物不再绘制。点击穿透靠 SetWindowRgn。
// bool 没有使用。
func setCompanionPassthrough(application.Window, bool) {}

// createCompanionRgn 在通话中创建整窗圆角区域，半径 24。空闲时按宠物占位做圆角矩形，字幕打开时再并上字幕胶囊。GDI 的直径用圆角的两倍，避免裁成圆。
// winW 在通话中作为圆角区域的宽。空闲时转成 float64 传给 companionPetRegionRect，并参与 companionRegionCornerRadius 的缩放。caption 为真且 CombineRgn 已加载时再传给 companionCaptionRect 和 companionScale。
// winH 在通话中作为圆角区域的高。空闲时转成 float64 传给 companionPetRegionRect，并参与 companionRegionCornerRadius 的缩放。caption 为真且 CombineRgn 已加载时再传给 companionCaptionRect 和 companionScale。
// caption 传给 companionPetRegionRect。为真且 CombineRgn 已加载时，再传给 companionCaptionRect 和 companionScale，并把字幕区域并进宠物区域。
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
	// GDI 要的是椭圆直径。直径取宽高的较小值会裁成圆，宠物胯部会落在圆外面。
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

// companionCursorLocalNative 用 GetCursorPos 和 GetWindowRect 算出光标在窗口里的坐标，以及窗口宽高。句柄或调用失败时返回假。
// win 传给 companionHWND。
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

// leftMouseDown 用 GetAsyncKeyState 看左键高位是否按下。函数没加载时返回假。
func leftMouseDown() bool {
	if !procReady(procGetAsyncKeyState) {
		return false
	}
	v, _, _ := procGetAsyncKeyState.Call(vkLButton)
	return v&0x8000 != 0
}

// dpiAt 用 MonitorFromPoint 和 GetDpiForMonitor 取该点的有效 DPI。取不到或低于 96 时用 GetDpiForSystem，仍低于 96 则返回 96。
// x 和 y 传给 MonitorFromPoint。
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
