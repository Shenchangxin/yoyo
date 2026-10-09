package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/browser"
)

func TestResolveTakeoverFileUsesAbsoluteSessionPath(t *testing.T) {
	cfg := t.TempDir()
	sess := t.TempDir()
	page := filepath.Join(sess, "pelican-bike", "index.html")
	if err := os.MkdirAll(filepath.Dir(page), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(page, []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := browser.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	a := &App{Config: Config{Workspace: cfg}, Browser: h}
	got := a.resolveTakeoverFile(page)
	if got != filepath.Clean(page) {
		t.Fatalf("got %q want %q", got, page)
	}
}

func TestResolveTakeoverFileUsesPreviewWhenRelativeMissesConfigWorkspace(t *testing.T) {
	cfg := t.TempDir()
	sess := t.TempDir()
	page := filepath.Join(sess, "pelican-bike", "index.html")
	if err := os.MkdirAll(filepath.Dir(page), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(page, []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := browser.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	h.SetPreview(page)
	a := &App{Config: Config{Workspace: cfg}, Browser: h}
	got := a.resolveTakeoverFile("pelican-bike/index.html")
	if got != filepath.Clean(page) {
		t.Fatalf("got %q want %q", got, page)
	}
}

func TestResolveTakeoverFileRelativeInsideConfigWorkspace(t *testing.T) {
	cfg := t.TempDir()
	page := filepath.Join(cfg, "game.html")
	if err := os.WriteFile(page, []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := browser.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	a := &App{Config: Config{Workspace: cfg}, Browser: h}
	got := a.resolveTakeoverFile("game.html")
	if got != filepath.Clean(page) {
		t.Fatalf("got %q want %q", got, page)
	}
}
