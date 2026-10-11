package browser

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFileURLServesWorkspaceHTML(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "game.html")
	if err := os.WriteFile(page, []byte("<!doctype html><title>ok</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	u, err := h.serveFileURL(page)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "<!doctype html><title>ok</title>" {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}

func TestServeFileURLServesRelativeAsset(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "index.html")
	js := filepath.Join(dir, "game.js")
	if err := os.WriteFile(page, []byte(`<script src="game.js"></script>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(js, []byte("window.GAME=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	u, err := h.serveFileURL(page)
	if err != nil {
		t.Fatal(err)
	}
	asset := strings.TrimSuffix(u, "/index.html") + "/game.js"
	resp, err := http.Get(asset)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "window.GAME=1" {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}

func TestSameDocumentURLIgnoresFragment(t *testing.T) {
	u := "http://127.0.0.1:9/index.html"
	if !sameDocumentURL(u, u) {
		t.Fatal("identical")
	}
	if !sameDocumentURL(u+"#boot", u) {
		t.Fatal("fragment")
	}
	if sameDocumentURL(u, "http://127.0.0.1:9/other.html") {
		t.Fatal("different path")
	}
	if sameDocumentURL("", u) {
		t.Fatal("empty live")
	}
}

func TestShouldReloadLocal(t *testing.T) {
	u := "http://127.0.0.1:9/index.html"
	if !shouldReloadLocal(false, u, u) {
		t.Fatal("dead chrome must navigate")
	}
	if !shouldReloadLocal(true, u, "") {
		t.Fatal("unknown want url must navigate")
	}
	if shouldReloadLocal(true, u, u) {
		t.Fatal("live matching document must not reload")
	}
	if !shouldReloadLocal(true, u, "http://127.0.0.1:9/other.html") {
		t.Fatal("different file must navigate")
	}
}

func TestShowingLocalFalseWithoutChrome(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "index.html")
	if err := os.WriteFile(page, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	if _, err := h.serveFileURL(page); err != nil {
		t.Fatal(err)
	}
	if h.ShowingLocal(page) {
		t.Fatal("no CDP must not claim the page is showing")
	}
}

func TestWithinDirRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if withinDir(root, filepath.Join(root, "..", "other")) {
		t.Fatal("escaped")
	}
	if !withinDir(root, filepath.Join(root, "game.html")) {
		t.Fatal("same dir")
	}
}
