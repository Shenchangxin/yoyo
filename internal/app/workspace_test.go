// 锁住工作区必须是绝对目录、打开时会补上代理目录和独立的视频目录，以及预览不放出工作区外的文件。

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWorkspaceReady 锁住空串、.、./ 不是工作区，文件和缺失路径也不是，绝对目录才是。
func TestWorkspaceReady(t *testing.T) {
	if WorkspaceReady("") || WorkspaceReady(".") || WorkspaceReady("./") {
		t.Fatal("placeholders must not be ready")
	}
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !WorkspaceReady(abs) {
		t.Fatal(abs)
	}
	if WorkspaceReady(filepath.Join(abs, "missing")) {
		t.Fatal("missing dir")
	}
	if err := os.WriteFile(filepath.Join(abs, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if WorkspaceReady(filepath.Join(abs, "f.txt")) {
		t.Fatal("file is not a workspace")
	}
}

// TestDefaultWorkspaceOnOpen 锁住 Open 后 Config.Workspace 等于 Home.Workspace 且可用。
func TestDefaultWorkspaceOnOpen(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	evals, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := Open(t.TempDir(), evals)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if !a.WorkspaceReady() {
		t.Fatalf("default workspace not ready: %q", a.Config.Workspace)
	}
	if a.Config.Workspace != a.Home.Workspace() {
		t.Fatalf("got %q want %q", a.Config.Workspace, a.Home.Workspace())
	}
}

// TestDefaultVideoWorkspaceOnOpen 锁住视频工作区等于 Home.VideoWorkspace，并且和代理工作区不是同一目录。
func TestDefaultVideoWorkspaceOnOpen(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	evals, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := Open(t.TempDir(), evals)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if !WorkspaceReady(a.VideoWorkspace()) {
		t.Fatalf("video workspace not ready: %q", a.VideoWorkspace())
	}
	if a.Config.VideoWorkspace != a.Home.VideoWorkspace() {
		t.Fatalf("got %q want %q", a.Config.VideoWorkspace, a.Home.VideoWorkspace())
	}
	if a.VideoWorkspace() == a.Workspace() {
		t.Fatalf("video workspace must not share the agent folder: %q", a.Workspace())
	}
}

// TestDefaultWorkspaceReplacesPlaceholder 锁住配置里的 workspace 为 . 时，Open 会换成 Home.Workspace。
func TestDefaultWorkspaceReplacesPlaceholder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	evals, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("workspace: \".\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := Open(root, evals)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if !a.WorkspaceReady() {
		t.Fatalf("placeholder workspace not replaced: %q", a.Config.Workspace)
	}
	if a.Config.Workspace != a.Home.Workspace() {
		t.Fatalf("got %q want %q", a.Config.Workspace, a.Home.Workspace())
	}
}

// TestPreviewWorkspaceFile 锁住 html 带回文本，go 的 lang 为 go，越出工作区的路径失败，png 和 pdf 只标 binary 且 pdf 不带 text 字段。
func TestPreviewWorkspaceFile(t *testing.T) {
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	html := "<!doctype html><html><body>hi</body></html>"
	if err := os.WriteFile(filepath.Join(abs, "index.html"), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(abs, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{Config: Config{Workspace: abs}}
	got, err := a.PreviewWorkspaceFile(abs, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	if got["html"] != true || got["text"] != html {
		t.Fatalf("html %+v", got)
	}
	goFile, err := a.PreviewWorkspaceFile(abs, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	if goFile["lang"] != "go" || goFile["html"] != false {
		t.Fatalf("go %+v", goFile)
	}
	if !strings.Contains(goFile["text"].(string), "package main") {
		t.Fatalf("go text %+v", goFile)
	}
	outside := filepath.Join(abs, "..", "outside.txt")
	if _, err := a.PreviewWorkspaceFile(abs, outside); err == nil {
		t.Fatal("escaped workspace")
	}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(abs, "shot.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := a.PreviewWorkspaceFile(abs, "shot.png")
	if err != nil {
		t.Fatal(err)
	}
	if img["kind"] != "image" || img["binary"] != true {
		t.Fatalf("image %+v", img)
	}
	blob, err := a.ReadWorkspaceBlob(abs, "shot.png")
	if err != nil {
		t.Fatal(err)
	}
	if blob["kind"] != "image" || blob["base64"] == "" {
		t.Fatalf("blob %+v", blob)
	}
	if err := os.WriteFile(filepath.Join(abs, "notes.pdf"), []byte("%PDF-1.4 junk stream"), 0o644); err != nil {
		t.Fatal(err)
	}
	pdf, err := a.PreviewWorkspaceFile(abs, "notes.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if pdf["kind"] != "pdf" || pdf["binary"] != true {
		t.Fatalf("pdf %+v", pdf)
	}
	if _, ok := pdf["text"]; ok {
		t.Fatalf("pdf must not leak extracted text %+v", pdf)
	}
}
