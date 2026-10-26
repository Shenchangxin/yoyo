// 本文件锁住 Wait 与 Run 的等待：空闲熔断、墙钟熔断、实时片段、短命令结束，以及取消时杀掉进程。
// 不检查各平台 probe 的 Sandbox。由 go test 调用。

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

// TestWaitIdleSilenceReturns 要求静默达到 idle 后返回，进程仍在后台，且耗时不超过 2 秒。
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

// TestWaitBlockReturnsWhileAlive 要求墙钟达到 block 后返回，即使 ping 仍在输出，且耗时不超过 2 秒。
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

// TestWaitNotifyChunks 要求 onChunk 至少收到一块非空输出，且完整结果里仍有 hi。
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

// TestWaitEchoFinishes 要求短命令在 idle 和 block 都是 2 秒时仍然正常结束，Background 为 false，输出含 hi。
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

// TestRunStillKillsOnCancel 要求 ctx 在 200 毫秒超时后 Run 返回错误，且不会堵过 2 秒。
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

// sleeper 启动一个睡够 d 的进程；不足 1 秒按 1 秒算。Windows 用 powershell 的 Start-Sleep，其他平台用 sleep。
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

// pingLoop 向 127.0.0.1 发 20 次 ping，用来在墙钟熔断返回时进程仍有输出。
func pingLoop(t *testing.T) *exec.Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		return exec.Command("ping", "-n", "20", "127.0.0.1")
	}
	return exec.Command("ping", "-c", "20", "127.0.0.1")
}

// echoHi 启动一条马上打印 hi 并退出的命令。
func echoHi() *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", "echo hi")
	}
	return exec.Command("echo", "hi")
}

// killProcess 结束熔断测试留下的进程。
func killProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
