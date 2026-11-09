//go:build windows

// procgroup_windows.go 拥有 Windows 上的进程组：启动标志和带 KILL_ON_JOB_CLOSE 的 Job Object。
// Kill 调 TerminateJobObject；主进程退出后关掉作业句柄，残留的子进程随之结束。

package procgroup

import (
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// group 是一个 Job Object。job 为 0 表示没建成或已经关掉，只能杀主进程。
type group struct {
	mu   sync.Mutex
	job  windows.Handle
	proc *exec.Cmd
}

// prepare 按 opts 设置窗口标志。
func prepare(cmd *exec.Cmd, opts Options) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	switch {
	case opts.NewConsole:
		cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_CONSOLE
	case opts.Hide:
		cmd.SysProcAttr.HideWindow = true
		cmd.SysProcAttr.CreationFlags |= windows.CREATE_NO_WINDOW
	}
}

// startGroup 建作业、启动 cmd，再把进程放进作业。作业建不成时进程照常启动，Kill 退回到只杀主进程。
func startGroup(cmd *exec.Cmd) (*group, error) {
	job, _ := newJob()
	if err := cmd.Start(); err != nil {
		if job != 0 {
			_ = windows.CloseHandle(job)
		}
		return nil, err
	}
	if job != 0 {
		h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
		if err != nil || windows.AssignProcessToJobObject(job, h) != nil {
			_ = windows.CloseHandle(job)
			job = 0
		}
		if err == nil {
			_ = windows.CloseHandle(h)
		}
	}
	return &group{job: job, proc: cmd}, nil
}

// newJob 建一个句柄关闭时结束全部成员的作业。
func newJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

// kill 结束作业里的全部进程。没有作业时只杀主进程；主进程已经退出时什么都不做。
func (g *group) kill(done <-chan struct{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.job != 0 {
		return windows.TerminateJobObject(g.job, 1)
	}
	select {
	case <-done:
		return nil
	default:
	}
	if g.proc != nil && g.proc.Process != nil {
		return g.proc.Process.Kill()
	}
	return nil
}

// exited 在主进程退出后关掉作业句柄，留下的子进程一起结束。
func (g *group) exited() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.job != 0 {
		_ = windows.CloseHandle(g.job)
		g.job = 0
	}
}
