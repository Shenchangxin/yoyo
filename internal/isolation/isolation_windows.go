//go:build windows

// 本文件拥有 Windows 上把 shell 子进程放进 Job Object、隐藏控制台，以及 Authenticode 探针。
// 不拥有文件系统白名单，也不是 AppContainer；等待循环在 isolation.go。
// Status 调用 probe，WaitNotify 调用 apply、assign、release，SigningStatus 调用 signingProbe。

package isolation

import (
	"debug/pe"
	"os"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// jobByPID 按进程号保存作业句柄，供 release 拆树或关句柄。
var jobByPID sync.Map

// probe 把 Kind 设为 job_object，Sandbox 设为 false，Available 设为 true。
// 取消时拆进程树；工具正常返回后后台子进程还在。这不是 AppContainer，Job 没有文件系统白名单。
func probe() Report {
	return Report{
		Kind:      "job_object",
		Sandbox:   false,
		Available: true,
		Note:      "shell children join a job object. cancel terminates the tree; a finished tool call leaves background children. not AppContainer.",
	}
}

// apply 隐藏控制台窗口，并在 CreationFlags 上加上 CREATE_NEW_PROCESS_GROUP。
// cmd 为 nil 时直接返回；否则交给 osutil.HideConsole，并在 CreationFlags 上或上 CREATE_NEW_PROCESS_GROUP。
func apply(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	osutil.HideConsole(cmd)
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
}

// assign 在 Start 之后把进程放进新作业，并把句柄按 pid 放进 jobByPID。
// 建作业失败或打开进程失败时直接返回，这次命令没有作业。
// cmd 为 nil 或 Process 为 nil 时直接返回；否则按 Pid 打开进程、放进新作业，并把句柄以 Pid 放进 jobByPID。
func assign(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	job, err := createJob()
	if err != nil || job == 0 {
		return
	}
	proc, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return
	}
	_ = windows.AssignProcessToJobObject(job, proc)
	_ = windows.CloseHandle(proc)
	jobByPID.Store(cmd.Process.Pid, job)
}

// release 按 pid 取出作业。kill 为 true 时用 TerminateJobObject 拆掉进程树，再关闭句柄。
// kill 为 false 时只关闭句柄，后台子进程还在。
// cmd 为 nil 或 Process 为 nil 时直接返回；否则按 Pid 从 jobByPID 取出作业。
func release(cmd *exec.Cmd, kill bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	h, ok := jobByPID.LoadAndDelete(cmd.Process.Pid)
	if !ok {
		return
	}
	job := h.(windows.Handle)
	if kill {
		_ = windows.TerminateJobObject(job, 1)
	}
	_ = windows.CloseHandle(job)
}

// createJob 创建匿名作业，LimitFlags 只有 JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION，没有文件系统白名单。
// 设置失败时关掉句柄并返回错误。
func createJob() (windows.Handle, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION,
		},
	}
	if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(h)
		return 0, err
	}
	return h, nil
}

// signingProbe 读取当前可执行文件。安全数据目录有体积时返回 authenticode；该路径接上 .signed 且文件存在时返回 sidecar；否则返回 unsigned。
func signingProbe() string {
	exe, err := os.Executable()
	if err != nil {
		return "unsigned"
	}
	if peAuthenticode(exe) {
		return "authenticode"
	}
	if _, err := os.Stat(exe + ".signed"); err == nil {
		return "sidecar"
	}
	return "unsigned"
}

// peAuthenticode 在 PE32 或 PE32+ 的安全数据目录大小大于 0 时返回 true。只看目录项有没有体积，不校验证书链。
// path 交给 pe.Open，打开失败返回 false；数据目录下标 4 的 Size 大于 0 时返回 true。
func peAuthenticode(path string) bool {
	f, err := pe.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	const certDir = 4 // IMAGE_DIRECTORY_ENTRY_SECURITY 的数据目录下标
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		if int(oh.NumberOfRvaAndSizes) > certDir && oh.DataDirectory[certDir].Size > 0 {
			return true
		}
	case *pe.OptionalHeader32:
		if int(oh.NumberOfRvaAndSizes) > certDir && oh.DataDirectory[certDir].Size > 0 {
			return true
		}
	}
	return false
}
