//go:build windows

package initiative

import (
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procGetLastInputInfo         = user32.NewProc("GetLastInputInfo")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetTickCount             = kernel32.NewProc("GetTickCount")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procQueryImage               = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procPower                    = kernel32.NewProc("GetSystemPowerStatus")
)

func readMachine() Machine {
	var out Machine
	var power struct {
		ACLineStatus        byte
		BatteryFlag         byte
		BatteryLifePercent  byte
		SystemStatusFlag    byte
		BatteryLifeTime     uint32
		BatteryFullLifeTime uint32
	}
	r, _, _ := procPower.Call(uintptr(unsafe.Pointer(&power)))
	if r != 0 && power.BatteryLifePercent <= 100 {
		out.PowerKnown = true
		out.Percent = int(power.BatteryLifePercent)
		out.Charging = power.ACLineStatus == 1
	}
	var last struct {
		CbSize uint32
		DwTime uint32
	}
	last.CbSize = uint32(unsafe.Sizeof(last))
	r, _, _ = procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&last)))
	if r != 0 {
		tick, _, _ := procGetTickCount.Call()
		delta := uint32(tick) - last.DwTime
		out.Idle = time.Duration(delta) * time.Millisecond
		out.IdleKnown = true
	}
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd != 0 {
		var pid uint32
		_, _, _ = procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != 0 {
			const query = 0x1000
			handle, _, _ := procOpenProcess.Call(query, 0, uintptr(pid))
			if handle != 0 {
				buf := make([]uint16, 260)
				n := uint32(len(buf))
				ok, _, _ := procQueryImage.Call(handle, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
				_, _, _ = procCloseHandle.Call(handle)
				if ok != 0 && n > 0 && int(n) <= len(buf) {
					name := filepath.Base(syscall.UTF16ToString(buf[:n]))
					if scrub(name) != "" {
						out.Focus = name
					}
				}
			}
		}
	}
	return out
}
