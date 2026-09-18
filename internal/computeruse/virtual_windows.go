//go:build windows

package computeruse

import (
	"syscall"
	"unsafe"
)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procCreateDesktop = user32.NewProc("CreateDesktopW")
	procCloseDesktop  = user32.NewProc("CloseDesktop")
)

func (h *Host) ensureVirtual() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hwnd != 0 {
		return
	}
	name, err := syscall.UTF16PtrFromString("YoyoVirtual")
	if err != nil {
		return
	}
	r, _, _ := procCreateDesktop.Call(uintptr(unsafe.Pointer(name)), 0, 0, 0, 0x01FF, 0)
	if r != 0 {
		h.hwnd = r
		h.display = "virtual-desktop:YoyoVirtual"
	}
}

func (h *Host) closeVirtual() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hwnd != 0 {
		_, _, _ = procCloseDesktop.Call(h.hwnd)
		h.hwnd = 0
	}
}
