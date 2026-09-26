package isolation

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWaitIdleSilenceReturns(t *testing.T) {
	cmd := sleeper(t, 5*time.Second)
	start := time.Now()
	o := Wait(context.Background(), cmd, 200*time.Millisecond, 0)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("idle fuse blocked %s", elapsed)
	}
	if !o.Background {
		t.Fatalf("want background err=%v out=%q", o.Err, o.Output)
	}
	killProcess(t, cmd)
}

func TestWaitBlockReturnsWhileAlive(t *testing.T) {
	cmd := pingLoop(t)
	start := time.Now()
	o := Wait(context.Background(), cmd, 0, 300*time.Millisecond)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("block fuse blocked %s", elapsed)
	}
	if !o.Background {
		t.Fatalf("want background err=%v out=%q", o.Err, o.Output)
	}
	killProcess(t, cmd)
}

func TestWaitNotifyChunks(t *testing.T) {
	cmd := echoHi()
	var n int
	o := WaitNotify(context.Background(), cmd, 0, 0, func(p []byte) {
		if len(p) > 0 {
			n++
		}
	})
	if o.Err != nil {
		t.Fatal(o.Err)
	}
	if n == 0 {
		t.Fatal("no stdout chunks")
	}
	if !strings.Contains(strings.ToLower(string(o.Output)), "hi") {
		t.Fatalf("output %q", o.Output)
	}
}

func TestWaitEchoFinishes(t *testing.T) {
	cmd := echoHi()
	o := Wait(context.Background(), cmd, 2*time.Second, 2*time.Second)
	if o.Background {
		t.Fatalf("echo still running pid=%d out=%q", o.PID, o.Output)
	}
	if o.Err != nil {
		t.Fatal(o.Err)
	}
	if !strings.Contains(strings.ToLower(string(o.Output)), "hi") {
		t.Fatalf("output %q", o.Output)
	}
}

func TestRunStillKillsOnCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	cmd := sleeper(t, 5*time.Second)
	start := time.Now()
	_, err := Run(ctx, cmd)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancel blocked %s", elapsed)
	}
	if err == nil {
		t.Fatal("want ctx error")
	}
}

func sleeper(t *testing.T, d time.Duration) *exec.Cmd {
	t.Helper()
	sec := int(d / time.Second)
	if sec < 1 {
		sec = 1
	}
	if runtime.GOOS == "windows" {
		return exec.Command("powershell", "-NoProfile", "-Command", "Start-Sleep -Seconds "+strconv.Itoa(sec))
	}
	return exec.Command("sleep", strconv.Itoa(sec))
}

func pingLoop(t *testing.T) *exec.Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		return exec.Command("ping", "-n", "20", "127.0.0.1")
	}
	return exec.Command("ping", "-c", "20", "127.0.0.1")
}

func echoHi() *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", "echo hi")
	}
	return exec.Command("echo", "hi")
}

func killProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
