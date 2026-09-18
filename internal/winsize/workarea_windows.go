//go:build windows

package winsize

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	spiGetWorkArea          = 0x0030
	monitorDefaultToPrimary = 1
	mdtEffectiveDPI         = 0
	smXVIRTUALSCREEN        = 76
	smYVIRTUALSCREEN        = 77
	smCXVIRTUALSCREEN       = 78
	smCYVIRTUALSCREEN       = 79
)

type point struct {
	X, Y int32
}

// PlatformWorkArea is the primary monitor work area in DIP, matching Wails SetSize.
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

// PlatformVirtualScreen is the union of all monitors in DIP.
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

func metric(get *windows.LazyProc, idx uintptr) int {
	r, _, _ := get.Call(idx)
	return int(int32(r))
}

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
