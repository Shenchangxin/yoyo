//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32               = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteExW   = shell32.NewProc("ShellExecuteExW")
	seeMaskNoCloseProcess = uint32(0x00000040)
)

type shellExecuteInfoW struct {
	CbSize       uint32
	FMask        uint32
	Hwnd         windows.HWND
	LpVerb       *uint16
	LpFile       *uint16
	LpParameters *uint16
	LpDirectory  *uint16
	NShow        int32
	HInstApp     windows.Handle
	LpIDList     uintptr
	LpClass      *uint16
	HKeyClass    windows.Handle
	DwHotKey     uint32
	HIcon        windows.Handle
	HProcess     windows.Handle
}

func hideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: uint32(windows.CREATE_NO_WINDOW | windows.CREATE_NEW_PROCESS_GROUP),
	}
}

func startElevatedApply(exe, spec string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	params, err := windows.UTF16PtrFromString(nsisApplyArg + ` "` + spec + `"`)
	if err != nil {
		return err
	}
	info := shellExecuteInfoW{
		FMask:        seeMaskNoCloseProcess,
		LpVerb:       verb,
		LpFile:       file,
		LpParameters: params,
		NShow:        windows.SW_HIDE,
	}
	info.CbSize = uint32(unsafe.Sizeof(info))
	r1, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return fmt.Errorf("update: administrator approval failed: %w", callErr)
	}
	if info.HProcess != 0 {
		_ = windows.CloseHandle(info.HProcess)
	}
	return nil
}

func waitForPID(pid int, timeout time.Duration) {
	if pid <= 0 {
		time.Sleep(3 * time.Second)
		return
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	ms := uint32(timeout / time.Millisecond)
	if ms == 0 {
		ms = 3000
	}
	_, _ = windows.WaitForSingleObject(h, ms)
}

func runNSISInstaller(installer, destDir string) error {
	cmd := exec.Command(installer)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		CmdLine:    nsisCmdLine(installer, destDir),
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	err := cmd.Wait()
	launchInstalledYoyo(destDir)
	return err
}

func launchInstalledYoyo(destDir string) {
	var candidates []string
	if destDir != "" {
		candidates = append(candidates, filepath.Join(destDir, "Yoyo.exe"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates,
			filepath.Join(pf, "Yoyo", "Yoyo", "Yoyo.exe"),
			filepath.Join(pf, "Yoyo", "Yoyo.exe"),
		)
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		candidates = append(candidates, filepath.Join(local, "Programs", "Yoyo", "Yoyo.exe"))
	}
	for _, p := range candidates {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		cmd := exec.Command(p)
		cmd.Dir = filepath.Dir(p)
		if err := cmd.Start(); err == nil {
			_ = cmd.Process.Release()
			return
		}
	}
}
