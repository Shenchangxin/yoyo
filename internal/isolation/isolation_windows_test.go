//go:build windows

package isolation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func backgroundWriteCmd(t *testing.T, dir, flag string) *exec.Cmd {
	t.Helper()
	flagPS := strings.ReplaceAll(flag, `'`, `''`)
	ps := "Start-Process -WindowStyle Hidden -FilePath powershell -ArgumentList '-NoProfile','-Command','Start-Sleep -Seconds 2; Set-Content -LiteralPath ''" + flagPS + "'' 1'"
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.Dir = dir
	return cmd
}
