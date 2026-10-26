//go:build windows

// 本文件拥有 Windows 上启动 Chrome 的进程标志，以及无头进程的 Job Object。
// prepareChromeCmd 在无头时隐藏窗口。CreationFlags 总是加上 CREATE_NEW_PROCESS_GROUP 和 CREATE_NO_WINDOW。
// bindChromeJob 只由无头启动调用。作业带 JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE，返回的函数会 TerminateJobObject。

package browser

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// prepareChromeCmd 按 headed 设置 HideWindow。CreationFlags 再加上 CREATE_NEW_PROCESS_GROUP 和 CREATE_NO_WINDOW。
func prepareChromeCmd(cmd *exec.Cmd, headed bool) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = !headed
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW
}

// bindChromeJob 把进程放进关闭时杀掉整棵树的作业。失败时返回 nil。返回的函数调用 TerminateJobObject 并关掉作业句柄。
func bindChromeJob(cmd *exec.Cmd) func() {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil || job == 0 {
		return nil
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	proc, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil
	}
	_ = windows.AssignProcessToJobObject(job, proc)
	_ = windows.CloseHandle(proc)
	return func() {
		_ = windows.TerminateJobObject(job, 1)
		_ = windows.CloseHandle(job)
	}
}
