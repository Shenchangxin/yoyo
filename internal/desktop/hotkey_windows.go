//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

const (
	modControl = 0x0002
	modShift   = 0x0004
	wmHotkey   = 0x0312
)

func registerWakeHotkey(s *Service) {
	user32 := syscall.NewLazyDLL("user32.dll")
	reg := user32.NewProc("RegisterHotKey")
	getMsg := user32.NewProc("GetMessageW")
	r, _, _ := reg.Call(0, 1, modControl|modShift, uintptr('Y'))
	if r == 0 {
		return
	}
	type msg struct {
		hwnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      struct{ x, y int32 }
	}
	var m msg
	for {
		r, _, _ = getMsg.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		if m.message == wmHotkey {
			s.RaiseWindow()
		}
	}
}
