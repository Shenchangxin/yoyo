//go:build windows

// 用 CreateDesktopW 建立名为 YoyoVirtual 的桌面。句柄已有则直接返回。

package computeruse

import (
	"syscall"
	"unsafe"
)

var (
	// user32 是延迟加载的 user32.dll。
	user32 = syscall.NewLazyDLL("user32.dll")
	// procCreateDesktop 是 user32.CreateDesktopW。
	procCreateDesktop = user32.NewProc("CreateDesktopW")
	// procCloseDesktop 是 user32.CloseDesktop。
	procCloseDesktop = user32.NewProc("CloseDesktop")
)

// ensureVirtual 在句柄已非 0 时返回。否则 CreateDesktopW 建立 YoyoVirtual，成功则记下句柄，并把 display 设为 virtual-desktop:YoyoVirtual。
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

// closeVirtual 在句柄非 0 时 CloseDesktop，并把句柄清 0。
func (h *Host) closeVirtual() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hwnd != 0 {
		_, _, _ = procCloseDesktop.Call(h.hwnd)
		h.hwnd = 0
	}
}
