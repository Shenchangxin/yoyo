//go:build windows

package isolation

import (
	"debug/pe"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var jobByPID sync.Map

func probe() Report {
	return Report{
		Kind:      "job_object",
		Sandbox:   false,
		Available: true,
		Note:      "shell children join a kill-on-close job object. this is process-tree containment, not AppContainer.",
	}
}

func apply(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_NEW_PROCESS_GROUP
}

func assign(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	job, err := createKillJob()
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

func release(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if h, ok := jobByPID.LoadAndDelete(cmd.Process.Pid); ok {
		_ = windows.CloseHandle(h.(windows.Handle))
	}
}

func createKillJob() (windows.Handle, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION,
		},
	}
	if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(h)
		return 0, err
	}
	return h, nil
}

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

func peAuthenticode(path string) bool {
	f, err := pe.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	const certDir = 4 // IMAGE_DIRECTORY_ENTRY_SECURITY
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
