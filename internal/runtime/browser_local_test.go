package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkspacePreviewRelAcceptsRelativeAndFileURL(t *testing.T) {
	dir := t.TempDir()
	rel := "games/pelican-bicycle.html"
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	got, err := tools.workspacePreviewRel(rel)
	if err != nil || got != rel {
		t.Fatalf("rel: %q %v", got, err)
	}
	got, err = tools.workspacePreviewRel("workspace://" + rel)
	if err != nil || got != rel {
		t.Fatalf("workspace: %q %v", got, err)
	}
	if runtime.GOOS == "windows" {
		u := "file:///" + strings.ReplaceAll(abs, `\`, "/")
		got, err = tools.workspacePreviewRel(u)
		if err != nil || got != rel {
			t.Fatalf("file url %s: %q %v", u, got, err)
		}
	}
	if _, err := tools.workspacePreviewRel("https://example.com"); err == nil {
		t.Fatal("remote url must not be a workspace preview")
	}
	if _, err := tools.workspacePreviewRel("data:text/html,hi"); err == nil {
		t.Fatal("data url must not be a workspace preview")
	}
	if _, err := tools.workspacePreviewRel("missing.html"); err == nil {
		t.Fatal("missing file must error")
	}
}

func TestBrowserOpenWorkspaceDoesNotNeedIsolatedChrome(t *testing.T) {
	dir := t.TempDir()
	rel := "index.html"
	if err := os.WriteFile(filepath.Join(dir, rel), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("browser_open", `{"url":"index.html"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "workstation Browser pane") {
		t.Fatalf("%s", res.Content)
	}
	if !strings.Contains(res.Content, "index.html") {
		t.Fatalf("%s", res.Content)
	}
}

func TestBrowserOpenRejectsDataURL(t *testing.T) {
	tools := &WorkspaceTools{Workspace: t.TempDir()}
	res := tools.Call("browser_open", `{"url":"data:text/html,<h1>x</h1>"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "data:") {
		t.Fatalf("%+v", res)
	}
}

func TestParseBrowserOpenStripsLane(t *testing.T) {
	target, lane := parseBrowserOpen("https://example.com lane=attached")
	if lane != "attached" || strings.Contains(target, "lane=") {
		t.Fatalf("%q %q", target, lane)
	}
}
