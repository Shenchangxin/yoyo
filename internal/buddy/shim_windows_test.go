//go:build windows

// shim_windows_test.go 锁住 Windows 上 npm 垫片的改写：能解析的 .cmd 改成 node 加 JS 入口，解析不了的走 cmd.exe。

package buddy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCommandForShim 锁住垫片旁边有 node.exe 时用它；没有时用注入的 node；不是垫片的 .cmd 标成 ViaCmd。
func TestCommandForShim(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "node_modules", "@google", "gemini-cli", "bundle", "gemini.js")
	_ = os.MkdirAll(filepath.Dir(entry), 0o755)
	_ = os.WriteFile(entry, []byte("#!/usr/bin/env node\n"), 0o644)
	shim := filepath.Join(dir, "gemini.cmd")
	body := "@ECHO off\r\nSET dp0=%~dp0\r\nIF EXIST \"%dp0%\\node.exe\" (\r\n  SET \"_prog=%dp0%\\node.exe\"\r\n) ELSE (\r\n  SET \"_prog=node\"\r\n)\r\n\"%_prog%\"  \"%dp0%\\node_modules\\@google\\gemini-cli\\bundle\\gemini.js\" %*\r\n"
	_ = os.WriteFile(shim, []byte(body), 0o644)
	node := func() (string, error) { return `C:\node\node.exe`, nil }

	l := commandFor(shim, []string{"--acp"}, node)
	if l.Command != `C:\node\node.exe` || len(l.Args) != 2 || l.Args[0] != entry || l.Args[1] != "--acp" || l.ViaCmd {
		t.Fatalf("shim launch %+v", l)
	}
	local := filepath.Join(dir, "node.exe")
	_ = os.WriteFile(local, []byte("exe"), 0o755)
	if l := commandFor(shim, nil, node); l.Command != local {
		t.Fatalf("local node not preferred: %+v", l)
	}
	other := filepath.Join(dir, "cursor-agent.cmd")
	_ = os.WriteFile(other, []byte("@echo off\r\npowershell -File \"%~dp0\\index.ps1\" %*\r\n"), 0o644)
	if l := commandFor(other, []string{"acp"}, node); l.Command != other || !l.ViaCmd || l.Args[0] != "acp" {
		t.Fatalf("plain cmd %+v", l)
	}
	exe := filepath.Join(dir, "opencode.exe")
	if l := commandFor(exe, []string{"acp"}, node); l.Command != exe || l.ViaCmd {
		t.Fatalf("exe %+v", l)
	}
}
