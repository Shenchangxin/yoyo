//go:build windows

// 本文件锁住 Windows 的 apply 会藏起控制台，以及 Run 正常返回后后台子进程仍能写标记文件。
// 不覆盖其他 GOOS 的探针。由 go test 调用。

package isolation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestSuccessfulRunLeavesBackgroundChild 要求 Run 返回后不拆后台子进程：返回若早于 1 秒则标记尚未出现，之后 6 秒内标记必须出现。
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
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(flag); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("background child was killed when the job closed")
}

// backgroundWriteCmd 让 powershell 再启动一个隐藏的 powershell，睡 2 秒后把 1 写到 flag。
func backgroundWriteCmd(t *testing.T, dir, flag string) *exec.Cmd {
	t.Helper()
	flagPS := strings.ReplaceAll(flag, `'`, `''`)
	ps := "Start-Process -WindowStyle Hidden -FilePath powershell -ArgumentList '-NoProfile','-Command','Start-Sleep -Seconds 2; Set-Content -LiteralPath ''" + flagPS + "'' 1'"
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.Dir = dir
	return cmd
}
