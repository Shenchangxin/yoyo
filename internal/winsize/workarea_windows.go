//go:build windows

// 本文件用 user32 读主显示器工作区和全部显示器的并集，再换成 DIP。尺寸公式在 size.go。

package winsize

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// spiGetWorkArea 是 SystemParametersInfoW 的 SPI_GETWORKAREA，值 0x0030。
	spiGetWorkArea = 0x0030
	// monitorDefaultToPrimary 是 MonitorFromPoint 的 MONITOR_DEFAULTTOPRIMARY，值 1。
	monitorDefaultToPrimary = 1
	// mdtEffectiveDPI 是 GetDpiForMonitor 的 MDT_EFFECTIVE_DPI，值 0。
	mdtEffectiveDPI = 0
	// smXVIRTUALSCREEN 是 GetSystemMetrics 的 SM_XVIRTUALSCREEN，值 76。
	smXVIRTUALSCREEN = 76
	// smYVIRTUALSCREEN 是 GetSystemMetrics 的 SM_YVIRTUALSCREEN，值 77。
	smYVIRTUALSCREEN = 77
	// smCXVIRTUALSCREEN 是 GetSystemMetrics 的 SM_CXVIRTUALSCREEN，值 78。
	smCXVIRTUALSCREEN = 78
	// smCYVIRTUALSCREEN 是 GetSystemMetrics 的 SM_CYVIRTUALSCREEN，值 79。
	smCYVIRTUALSCREEN = 79
)

// point 是传给 MonitorFromPoint 的点。本文件实际把中心坐标拆成两个 uintptr，没有使用这个类型的值。
type point struct {
	X, Y int32
}

// PlatformWorkArea 返回主显示器工作区的 DIP 宽高。SystemParametersInfoW 失败，或宽高小于等于 0 时返回 0, 0。
func PlatformWorkArea() (int, int) {
	var r windows.Rect
	user32 := windows.NewLazySystemDLL("user32.dll")
	spi := user32.NewProc("SystemParametersInfoW")
	ok, _, _ := spi.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if ok == 0 {
		return 0, 0
	}
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	dpi := primaryDPI(user32, r)
	return ToDIP(w, dpi), ToDIP(h, dpi)
}

// PlatformVirtualScreen 返回全部显示器并集的 DIP 原点、宽和高。宽或高小于等于 0 时四个值都是 0。DPI 来自 GetDpiForSystem，找不到或小于 96 时用 96。
func PlatformVirtualScreen() (x, y, w, h int) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	get := user32.NewProc("GetSystemMetrics")
	sx := metric(get, smXVIRTUALSCREEN)
	sy := metric(get, smYVIRTUALSCREEN)
	sw := metric(get, smCXVIRTUALSCREEN)
	sh := metric(get, smCYVIRTUALSCREEN)
	if sw <= 0 || sh <= 0 {
		return 0, 0, 0, 0
	}
	dpi := 96
	if getDpi := user32.NewProc("GetDpiForSystem"); getDpi.Find() == nil {
		if v, _, _ := getDpi.Call(); v >= 96 {
			dpi = int(v)
		}
	}
	return ToDIP(sx, dpi), ToDIP(sy, dpi), ToDIP(sw, dpi), ToDIP(sh, dpi)
}

// metric 调用 GetSystemMetrics，并把返回值按 int32 再转成 int，从而保留负的屏幕原点。
// get 以 idx 调用 Call，返回值经 int32 再转成 int。
// idx 作为 get.Call 的参数。
func metric(get *windows.LazyProc, idx uintptr) int {
	r, _, _ := get.Call(idx)
	return int(int32(r))
}

// primaryDPI 取工作区中心所在显示器的有效 DPI。GetDpiForMonitor 失败时改用 GetDpiForSystem。都不可用或小于 96 时返回 96。
// user32 用来取 MonitorFromPoint 和 GetDpiForSystem。
// work 的左右中点和上下中点作为 MonitorFromPoint 的坐标。
func primaryDPI(user32 *windows.LazyDLL, work windows.Rect) int {
	shcore := windows.NewLazySystemDLL("shcore.dll")
	getDpiForMonitor := shcore.NewProc("GetDpiForMonitor")
	monitorFromPoint := user32.NewProc("MonitorFromPoint")
	cx := int((work.Left + work.Right) / 2)
	cy := int((work.Top + work.Bottom) / 2)
	hmon, _, _ := monitorFromPoint.Call(uintptr(cx), uintptr(cy), monitorDefaultToPrimary)
	if hmon != 0 && getDpiForMonitor.Find() == nil {
		var dpiX, dpiY uint32
		hr, _, _ := getDpiForMonitor.Call(hmon, mdtEffectiveDPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
		if hr == 0 && dpiX >= 96 {
			return int(dpiX)
		}
	}
	getDpiForSystem := user32.NewProc("GetDpiForSystem")
	if getDpiForSystem.Find() == nil {
		if dpi, _, _ := getDpiForSystem.Call(); dpi >= 96 {
			return int(dpi)
		}
	}
	return 96
}
