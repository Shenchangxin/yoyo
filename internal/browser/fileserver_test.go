package browser

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
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

func TestWithinDirRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if withinDir(root, filepath.Join(root, "..", "other")) {
		t.Fatal("escaped")
	}
	if !withinDir(root, filepath.Join(root, "game.html")) {
		t.Fatal("same dir")
	}
}
