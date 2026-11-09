//go:build windows

// alive_windows_test.go 给测试判断进程是否还活着：能打开且退出码是 STILL_ACTIVE。

package procgroup

import "golang.org/x/sys/windows"

// stillActive 是 GetExitCodeProcess 对还在跑的进程返回的值。
const stillActive = 259

// alive 在 pid 对应的进程还在跑时为真。
func alive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}
