//go:build windows

package desktop

import (
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

const clsctxInproc = 1

var procCoCreateInstance = syscall.NewLazyDLL("ole32.dll").NewProc("CoCreateInstance")

type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

type iTaskbarListVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	hrInit         uintptr
	addTab         uintptr
	deleteTab      uintptr
	activateTab    uintptr
	setActiveAlt   uintptr
}

type iTaskbarList struct {
	vtbl *iTaskbarListVtbl
}

var (
	clsidTaskbarList = guid{0x56FDF344, 0xFD6D, 0x11d0, [8]byte{0x95, 0x8A, 0x00, 0x60, 0x97, 0xC9, 0xA0, 0x90}}
	iidITaskbarList  = guid{0x56FDF342, 0xFD6D, 0x11d0, [8]byte{0x95, 0x8A, 0x00, 0x60, 0x97, 0xC9, 0xA0, 0x90}}
)

func hideCompanionTaskbar(win application.Window) {
	wv, ok := win.(*application.WebviewWindow)
	if !ok || wv == nil {
		return
	}
	hwnd := w32.HWND(uintptr(wv.NativeWindow()))
	if hwnd == 0 {
		return
	}
	style := w32.GetWindowLongPtr(hwnd, w32.GWL_EXSTYLE)
	next := (style | w32.WS_EX_TOOLWINDOW) &^ w32.WS_EX_APPWINDOW
	if next != style {
		w32.SetWindowLongPtr(hwnd, w32.GWL_EXSTYLE, next)
		w32.SetWindowPos(hwnd, 0, 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE|w32.SWP_FRAMECHANGED)
	}
	deleteTaskbarTab(uintptr(hwnd))
}

func deleteTaskbarTab(hwnd uintptr) {
	var obj *iTaskbarList
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidTaskbarList)),
		0,
		clsctxInproc,
		uintptr(unsafe.Pointer(&iidITaskbarList)),
		uintptr(unsafe.Pointer(&obj)),
	)
	if hr != 0 || obj == nil || obj.vtbl == nil {
		return
	}
	_, _, _ = syscall.SyscallN(obj.vtbl.hrInit, uintptr(unsafe.Pointer(obj)))
	_, _, _ = syscall.SyscallN(obj.vtbl.deleteTab, uintptr(unsafe.Pointer(obj)), hwnd)
	_, _, _ = syscall.SyscallN(obj.vtbl.release, uintptr(unsafe.Pointer(obj)))
}
