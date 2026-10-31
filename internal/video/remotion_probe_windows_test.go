//go:build windows

// remotion_probe_windows_test.go 锁住 npx 探针在上下文结束时会返回，而不是等孙进程松开管道。

package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNpxProbeReturnsWhenGrandchildHoldsPipes 用一个假的 npx.cmd 拉起不退出的 node。探针必须在上下文结束后返回，并且不留下那个 node。
func TestNpxProbeReturnsWhenGrandchildHoldsPipes(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not installed")
	}
	token := fmt.Sprintf("yoyoProbeHold%d", time.Now().UnixNano())
	dir := t.TempDir()
	script := "@echo off\r\nnode -e \"global['" + token + "']=1;setInterval(()=>{},1e9)\"\r\n"
	path := filepath.Join(dir, "npx.cmd")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopProbeNodes(token) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	_, err := probeNpxRemotion(ctx, path)
	elapsed := time.Since(start)
	if elapsed > 6*time.Second {
		t.Fatalf("probe blocked for %s: %v", elapsed, err)
	}
	if pids := probeNodePIDs(token); len(pids) > 0 {
		t.Fatalf("probe left node running: %s", strings.Join(pids, ","))
	}
}

// probeNodePIDs 找出命令行里带有 token 的 node 进程号。
func probeNodePIDs(token string) []string {
	cmd := quiet(exec.Command("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_Process -Filter \"Name='node.exe'\" | Where-Object { $_.CommandLine -like '*"+token+"*' } | ForEach-Object { $_.ProcessId }"))
	b, err := cmd.Output()
	if err != nil {
		return nil
	}
	var pids []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			pids = append(pids, line)
		}
	}
	return pids
}

// stopProbeNodes 结束命令行里带有 token 的 node，避免失败的探针把进程留在机器上。
func stopProbeNodes(token string) {
	for _, pid := range probeNodePIDs(token) {
		kill := quiet(exec.Command("taskkill", "/PID", pid, "/T", "/F"))
		_ = kill.Run()
	}
}
