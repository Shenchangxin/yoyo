//go:build windows

// 本文件锁住 Windows 的 apply 会藏起控制台，以及 Run 正常返回后后台子进程仍能写标记文件。
// 不覆盖其他 GOOS 的探针。由 go test 调用。

package isolation

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TestApplyHidesConsole 要求 apply 之后 HideWindow 为真，且带有 CREATE_NO_WINDOW 和 CREATE_NEW_PROCESS_GROUP。
func TestApplyHidesConsole(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "echo hi")
	apply(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("console window not hidden")
	}
	flags := cmd.SysProcAttr.CreationFlags
	if flags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatal("CREATE_NO_WINDOW not set")
	}
	if flags&windows.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatal("CREATE_NEW_PROCESS_GROUP not set")
	}
}

// TestSuccessfulRunLeavesBackgroundChild 要求 Run 返回后不拆后台子进程。
// 返回若早于 1 秒则标记尚未出现。子进程用 ping 拖大约 2 秒再写标记，之后 20 秒内标记必须出现。
func TestSuccessfulRunLeavesBackgroundChild(t *testing.T) {
	dir := t.TempDir()
	flag := filepath.Join(dir, "alive.txt")
	cmd := backgroundWriteCmd(t, dir, flag)
	start := time.Now()
	out, err := Run(context.Background(), cmd)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	t.Logf("shell returned in %s out=%q", elapsed, out)
	if elapsed < time.Second {
		if _, err := os.Stat(flag); err == nil {
			t.Fatal("flag appeared before the background child could have written it")
		}
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(flag); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("background child was killed when the job closed")
}

// backgroundWriteCmd 让 powershell 再启动一个隐藏的 cmd。cmd 先 ping 大约 2 秒，再把标记写到 flag。
// 不用第二个 powershell：运行器上它的冷启动会超过原来的等待。
func backgroundWriteCmd(t *testing.T, dir, flag string) *exec.Cmd {
	t.Helper()
	script := filepath.Join(dir, "child.cmd")
	body := "@echo off\r\nping -n 3 127.0.0.1 >nul\r\necho ready> \"" + flag + "\"\r\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptPS := strings.ReplaceAll(script, `'`, `''`)
	ps := "Start-Process -WindowStyle Hidden -FilePath '" + scriptPS + "'"
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.Dir = dir
	return cmd
}

// TestBackgroundChildUnderKillOnCloseAncestor 把子测试放进带 JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE 的父作业。
// 父作业在子测试结束前一直开着，用来确认正常返回后后台子进程还在。
func TestBackgroundChildUnderKillOnCloseAncestor(t *testing.T) {
	if os.Getenv("YOYO_JOB_CHILD") == "1" {
		t.Skip()
	}
	parent, err := windows.CreateJobObject(nil, nil)
	if err != nil || parent == 0 {
		t.Fatal(err)
	}
	defer windows.CloseHandle(parent)
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK,
		},
	}
	if _, err := windows.SetInformationJobObject(parent, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^TestSuccessfulRunLeavesBackgroundChild$", "-test.count=1", "-test.timeout", "40s", "-test.v")
	cmd.Env = append(os.Environ(), "YOYO_JOB_CHILD=1")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	proc, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = cmd.Process.Kill()
		t.Fatal(err)
	}
	assignErr := windows.AssignProcessToJobObject(parent, proc)
	_ = windows.CloseHandle(proc)
	if assignErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Skipf("parent job rejected the process: %v", assignErr)
	}
	err = cmd.Wait()
	if err != nil {
		t.Fatalf("inner: %v\n%s", err, buf.String())
	}
}
