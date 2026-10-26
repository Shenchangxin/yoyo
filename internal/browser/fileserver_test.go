// 本文件锁住本地文件服务能提供 HTML 和相对资源，同一文档不因片段不同而重新加载，以及路径不能逃出目录。
// 由 go test 调用。

package browser

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServeFileURLServesWorkspaceHTML 锁住 serveFileURL 返回的地址能取回写入的 HTML。
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

// TestServeFileURLServesRelativeAsset 锁住同目录的 game.js 也能通过文件服务取到。
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

// TestSameDocumentURLIgnoresFragment 锁住片段不同仍是同一文档，路径不同或一边为空则不是。
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

// TestShouldReloadLocal 锁住 Chrome 不在或想要的地址为空时要导航，同一文档不导航，不同文件要导航。
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

// TestShowingLocalFalseWithoutChrome 锁住没有 CDP 时不会声称正在显示那个本地文件。
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

// TestWithinDirRejectsEscape 锁住逃出根目录的路径被拒绝，同一目录里的文件可以通过。
func TestWithinDirRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if withinDir(root, filepath.Join(root, "..", "other")) {
		t.Fatal("escaped")
	}
	if !withinDir(root, filepath.Join(root, "game.html")) {
		t.Fatal("same dir")
	}
}
