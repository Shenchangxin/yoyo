// detect_test.go 锁住检测规则：PATH 上找到 ACP 命令算系统安装、只有 CLI 时标出缺适配器、托管安装优先、第一次检测做完前报告检测中，
// npm 垫片的解析和查找器的扩展名规则；以及性能目标：全量检测不等 --version、检测完以后 List 只读缓存。

package buddy

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// TestDetectingUntilFirstScan 锁住第一次检测做完之前 Detecting 为真，第一次列出之后很快变成假，并且已经认出系统安装。
func TestDetectingUntilFirstScan(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.LookPath = func(name string) (string, error) {
			if name == "codex-acp" {
				return "/usr/bin/codex-acp", nil
			}
			return "", errNotFound
		}
	})
	if !f.m.Detecting() {
		t.Fatal("not detecting before the first list")
	}
	_, _ = f.m.List()
	deadline := time.Now().Add(5 * time.Second)
	for f.m.Detecting() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if f.m.Detecting() {
		t.Fatal("still detecting after the first scan")
	}
	if v, _ := f.m.Get("codex"); v.Install.Kind != "system" {
		t.Fatalf("codex after scan: %+v", v.Install)
	}
}

// TestDetectionDoesNotWaitForVersions 锁住全量检测不等 --version：每个版本命令都要 5 秒时，「检测中」也在 3 秒内结束，
// 找到的系统安装先列出来，版本之后再补。
func TestDetectionDoesNotWaitForVersions(t *testing.T) {
	bin := t.TempDir()
	f := newFixture(t, func(o *Options) {
		o.LookPath = func(name string) (string, error) {
			p := filepath.Join(bin, name)
			if err := os.WriteFile(p, nil, 0o755); err != nil {
				return "", err
			}
			return p, nil
		}
		o.Run = func(ctx context.Context, _ Cmd) ([]byte, error) {
			select {
			case <-time.After(5 * time.Second):
				return []byte("9.9.9"), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		f.m.detectAll(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	for f.m.Detecting() {
		if time.Since(start) > 3*time.Second {
			t.Fatal("still detecting after 3 seconds")
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Logf("detected in %v", time.Since(start))
	if v, _ := f.m.Get("gemini"); v.System == nil || v.System.Path == "" || v.System.Version != "" {
		t.Fatalf("gemini before its version %+v", v.System)
	}
}

// TestListReadsCacheOnly 锁住检测做完以后 List 只读缓存：不再找命令、不跑命令，整张目录一次平均在 50 毫秒以内。
func TestListReadsCacheOnly(t *testing.T) {
	var lookups, runs atomic.Int32
	f := newFixture(t, func(o *Options) {
		o.LookPath = func(name string) (string, error) {
			lookups.Add(1)
			if name == "codex-acp" {
				return "/usr/bin/codex-acp", nil
			}
			return "", errNotFound
		}
		o.Run = func(context.Context, Cmd) ([]byte, error) {
			runs.Add(1)
			return []byte("1.0.0"), nil
		}
	})
	f.m.Start()
	f.m.detectAll(context.Background())
	// 第一次列出时的后台检测、探测和注册表刷新已经算过了，这里只量读缓存。
	f.m.mu.Lock()
	f.m.kicked = true
	f.m.mu.Unlock()
	before := lookups.Load() + runs.Load()
	const n = 50
	var views []View
	start := time.Now()
	for i := 0; i < n; i++ {
		views, _ = f.m.List()
	}
	avg := time.Since(start) / n
	t.Logf("%d items, %v per List", len(views), avg)
	if len(views) < 10 {
		t.Fatalf("listed %d items", len(views))
	}
	if lookups.Load()+runs.Load() != before {
		t.Fatal("List looked up or ran commands")
	}
	if avg > 50*time.Millisecond {
		t.Fatalf("List took %v", avg)
	}
}

// BenchmarkDetectOnThisMachine 量本机上一遍全量检测的找命令部分：真的 PATH（macOS 和 Linux 上含登录 shell）和常见安装目录，
// 不跑任何命令、不碰网络。只在 -bench 时运行，用来核对「全量检测小于 3 秒」。
func BenchmarkDetectOnThisMachine(b *testing.B) {
	s := &memSettings{}
	m := New(Options{
		Root: b.TempDir(), RuntimeDir: b.TempDir(), Version: "test",
		HTTP: &http.Client{Transport: offline{}},
		Run:  func(context.Context, Cmd) ([]byte, error) { return nil, nil },
		Dial: func(context.Context, acp.Options) (*acp.Client, error) {
			return nil, ErrNotInstalled
		},
		Vault: &memVault{}, Settings: s.get, SaveSettings: s.save,
	})
	b.Cleanup(m.Close)
	m.mu.Lock()
	entries := slices.Clone(m.entries)
	m.mu.Unlock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.paths.mu.Lock()
		m.paths.dirs = nil
		m.paths.mu.Unlock()
		for _, e := range entries {
			m.detectFast(e)
		}
	}
	b.ReportMetric(float64(len(entries)), "entries")
}

// TestPrepareDetectsOneBeforeFirstList 锁住第一次列出之前 Prepare 只检测这一项：之后 Ready 按记下的状态判断、Launch 用找到的命令，
// 全量检测仍算没做。
func TestPrepareDetectsOneBeforeFirstList(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.LookPath = func(name string) (string, error) {
			if name == "codex-acp" {
				return "/usr/bin/codex-acp", nil
			}
			return "", errNotFound
		}
	})
	f.m.mu.Lock()
	f.m.recs["codex"] = &record{State: StateOnline, CheckedAt: f.clock.Now()}
	f.m.mu.Unlock()
	if f.m.Ready("codex") {
		t.Fatal("ready before any detection")
	}
	f.m.Prepare("codex")
	if !f.m.Ready("codex") {
		v, _ := f.m.Get("codex")
		t.Fatalf("not ready after Prepare: %s %q %+v", v.State, v.Detail, v.Install)
	}
	if l, err := f.m.Launch("codex"); err != nil || l.Command != "/usr/bin/codex-acp" {
		t.Fatalf("launch %+v %v", l, err)
	}
	if !f.m.Detecting() {
		t.Fatal("Prepare counted as the first full scan")
	}
	f.m.Prepare("missing")
	f.m.Prepare(YoyoID)
}

// TestDetectSystemAndCLIOnly 锁住注入的 LookPath 决定系统安装，只有 CLI 时视图标 cliOnly 且仍是未安装。
func TestDetectSystemAndCLIOnly(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.LookPath = func(name string) (string, error) {
			switch name {
			case "gemini":
				return "/usr/local/bin/gemini", nil
			case "claude":
				return "/usr/local/bin/claude", nil
			}
			return "", errNotFound
		}
	})
	f.m.detectAll(context.Background())
	gemini, _ := f.m.Get("gemini")
	if gemini.Install.Kind != "system" || gemini.System == nil || gemini.System.Path != "/usr/local/bin/gemini" || gemini.State != StateProbing {
		t.Fatalf("gemini %+v", gemini)
	}
	claude, _ := f.m.Get("claude")
	if claude.Install.Kind != "none" || !claude.CLIOnly || claude.State != StateNotInstalled {
		t.Fatalf("claude %+v", claude)
	}
	codex, _ := f.m.Get("codex")
	if codex.State != StateNotInstalled || codex.CLIOnly {
		t.Fatalf("codex %+v", codex)
	}
	yoyo, _ := f.m.Get(YoyoID)
	if yoyo.State != StateOnline || !yoyo.Builtin {
		t.Fatalf("yoyo %+v", yoyo)
	}
}

// TestManagedWinsOverSystem 锁住托管和系统都在时默认用托管，mode 改成 system 后启动系统版。
func TestManagedWinsOverSystem(t *testing.T) {
	f := newFixture(t, func(o *Options) {
		o.LookPath = func(name string) (string, error) {
			if name == "gemini" {
				return "/usr/local/bin/gemini", nil
			}
			return "", errNotFound
		}
	})
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
	v, _ := f.m.Get("gemini")
	if v.Install.Kind != "managed" || v.System == nil {
		t.Fatalf("view %+v", v.Install)
	}
	l, err := f.m.Launch("gemini")
	if err != nil || l.Command != node || len(l.Args) != 2 || l.Args[0] != entry {
		t.Fatalf("managed launch %+v %v", l, err)
	}
	if path := getEnv(l.Env, "PATH"); filepath.Dir(node) != path[:len(filepath.Dir(node))] {
		t.Fatalf("node dir not first on PATH: %s", path)
	}
	mode := "system"
	if _, err := f.m.Configure("gemini", ConfigPatch{Mode: &mode}); err != nil {
		t.Fatal(err)
	}
	l, err = f.m.Launch("gemini")
	if err != nil || l.Command != "/usr/local/bin/gemini" || len(l.Args) == 0 || l.Args[0] != "--acp" {
		t.Fatalf("system launch %+v %v", l, err)
	}
}

// TestMissingEntryMeansNotInstalled 锁住 install.json 还在但入口文件没了时当作没装。
func TestMissingEntryMeansNotInstalled(t *testing.T) {
	f := newFixture(t, nil)
	_ = f.m.writeInstall("gemini", &InstallRecord{Kind: "npm", Command: "/nope/node", Entry: "/nope/gemini.js", Runtime: "node"})
	if rec, _ := f.m.readInstall("gemini"); rec != nil {
		t.Fatalf("stale install %+v", rec)
	}
}

// TestParseShim 锁住 npm 生成的 .cmd 垫片能解析出 JS 入口；解释器不是 node 时不改写。
func TestParseShim(t *testing.T) {
	shim := "@ECHO off\r\nGOTO start\r\n:find_dp0\r\nSET dp0=%~dp0\r\nEXIT /b\r\n:start\r\nSETLOCAL\r\nCALL :find_dp0\r\n\r\nIF EXIST \"%dp0%\\node.exe\" (\r\n  SET \"_prog=%dp0%\\node.exe\"\r\n) ELSE (\r\n  SET \"_prog=node\"\r\n  SET PATHEXT=%PATHEXT:;.JS;=;%\r\n)\r\n\r\nendLocal & goto #_undefined_# 2>NUL || title %COMSPEC% & \"%_prog%\"  \"%dp0%\\node_modules\\@google\\gemini-cli\\bundle\\gemini.js\" %*\r\n"
	entry, ok := parseShim(shim, filepath.FromSlash("/npm"))
	want := filepath.Join(filepath.FromSlash("/npm"), "node_modules", "@google", "gemini-cli", "bundle", "gemini.js")
	if !ok || entry != want {
		t.Fatalf("entry %q ok %v", entry, ok)
	}
	sh := "SET \"_prog=%dp0%\\sh.exe\"\r\n\"%_prog%\"  \"%dp0%\\node_modules\\x\\run.sh\" %*\r\n"
	if _, ok := parseShim(sh, "/npm"); ok {
		t.Fatal("non-node shim parsed")
	}
	if !nodeShebang([]byte("#!/usr/bin/env node\nconsole.log(1)")) || nodeShebang([]byte("#!/bin/sh\nexec x")) || !nodeShebang([]byte("'use strict'")) {
		t.Fatal("shebang detection")
	}
}

// TestPathFinderExtensions 锁住 Windows 上没写扩展名时按 PATHEXT 找到 .cmd，Unix 上要有执行位。
func TestPathFinderExtensions(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "gemini.cmd"), []byte("@echo off"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "gemini"), []byte("#!/bin/sh"), 0o644)
	win := &PathFinder{goos: "windows", now: newFixture(t, nil).clock.Now, env: func(name string) string {
		if name == "PATH" {
			return dir
		}
		return ""
	}}
	if got, err := win.LookPath("gemini"); err != nil || filepath.Base(got) != "gemini.cmd" {
		t.Fatalf("windows lookup %q %v", got, err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	unix := &PathFinder{goos: "linux", now: win.now, env: win.env}
	if _, err := unix.LookPath("gemini"); err == nil {
		t.Fatal("found a file without the exec bit")
	}
	_ = os.Chmod(filepath.Join(dir, "gemini"), 0o755)
	if got, err := unix.LookPath("gemini"); err != nil || filepath.Base(got) != "gemini" {
		t.Fatalf("unix lookup %q %v", got, err)
	}
}
