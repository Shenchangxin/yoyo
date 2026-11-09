// uninstall_test.go 锁住卸载托管安装时的顺序：先等正在跑的探测停下再删目录，卸载中新的探测直接跳过，删完记成未安装。

package buddy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// managedGemini 在 fixture 里放一份假的托管 Gemini，返回它的安装目录。
func managedGemini(t *testing.T, f *fixture) string {
	t.Helper()
	dir := filepath.Join(f.m.opts.Root, "gemini", "pkg-1")
	entry := filepath.Join(dir, "gemini.js")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(entry, []byte("#!/usr/bin/env node\n"), 0o644)
	node := filepath.Join(f.m.opts.RuntimeDir, "node", nodeVersion, "bin", "node")
	_ = os.MkdirAll(filepath.Dir(node), 0o755)
	_ = os.WriteFile(node, []byte("bin"), 0o755)
	if err := f.m.writeInstall("gemini", &InstallRecord{Kind: "npm", Version: "1", Command: node, Entry: entry, Args: []string{entry, "--acp"}, Runtime: "node", Dir: dir}); err != nil {
		t.Fatal(err)
	}
	f.m.detectAll(context.Background())
	return f.m.installDir("gemini")
}

// TestUninstallWaitsForProbe 锁住有探测在跑时卸载先等着、不删目录，探测结束后才删干净并记成未安装。
func TestUninstallWaitsForProbe(t *testing.T) {
	f := newFixture(t, nil)
	dir := managedGemini(t, f)
	f.m.mu.Lock()
	f.m.probing["gemini"] = 1
	f.m.mu.Unlock()
	done := make(chan error, 1)
	go func() {
		_, err := f.m.Uninstall(context.Background(), "gemini")
		done <- err
	}()
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("removed while a probe was running: %v", err)
	}
	f.m.mu.Lock()
	removing := f.m.removing["gemini"]
	f.m.mu.Unlock()
	if !removing {
		t.Fatal("uninstall did not mark the entry")
	}
	if err := f.m.probe(context.Background(), "gemini", true); err != nil {
		t.Fatalf("probe during uninstall: %v", err)
	}
	f.m.mu.Lock()
	if f.m.probing["gemini"] != 1 {
		t.Fatalf("probe ran during uninstall: %d", f.m.probing["gemini"])
	}
	delete(f.m.probing, "gemini")
	f.m.mu.Unlock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("uninstall did not finish")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("install dir left behind: %v", err)
	}
	v, _ := f.m.Get("gemini")
	if v.State != StateNotInstalled || v.Install.Kind != "none" {
		t.Fatalf("view %+v %+v", v.State, v.Install)
	}
	f.m.mu.Lock()
	defer f.m.mu.Unlock()
	if f.m.removing["gemini"] {
		t.Fatal("removing flag left set")
	}
}

// TestInstallLogClosesTwice 锁住安装日志可以重复关，关掉之后的写入不出错。
func TestInstallLogClosesTwice(t *testing.T) {
	f := newFixture(t, nil)
	l := f.m.openInstallLog("gemini", "managed")
	if l.f == nil {
		t.Fatal("log not opened")
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Write([]byte("late line\n")); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(f.m.installDir("gemini")); err != nil {
		t.Fatalf("closed log still held: %v", err)
	}
}
