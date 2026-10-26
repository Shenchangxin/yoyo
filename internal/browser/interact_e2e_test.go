// 本文件用真实隔离 Chrome 点按钮、发按键，并确认被挡住的按钮点不到。页面 HTML 是数据。
// 没设置 YOYO_BROWSER_E2E=1，或找不到 Chrome（可设 YOYO_CHROME）时跳过。由 go test 调用。

package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// e2ePage 是端到端页面，字符串是数据。里面有稳定 id 的按钮、只靠文字的选项、文档上的 keydown，以及被遮罩挡住的按钮。
const e2ePage = `<!doctype html><html><head><meta charset="utf-8"></head><body>
<div id="ui">
  <button id="opt-0">First option</button>
  <button class="opt" id="opt-1">Second option</button>
</div>
<button id="under" style="position:absolute;left:300px;top:300px;width:120px;height:40px">Under</button>
<div id="cover" style="position:absolute;left:280px;top:280px;width:200px;height:120px"></div>
<p id="msg">waiting</p>
<script>
document.getElementById('opt-0').onclick = () => { document.title = 'picked:0'; };
document.getElementById('opt-1').onclick = () => {
  document.getElementById('msg').textContent = 'picked one';
  document.title = 'picked:1';
};
document.addEventListener('keydown', (e) => { document.title = 'key:' + e.key; });
</script>
</body></html>`

// TestBrowserInteractionE2E 驱动真实隔离 Chrome。没设置 YOYO_BROWSER_E2E=1 或找不到 Chrome 时跳过。
func TestBrowserInteractionE2E(t *testing.T) {
	if os.Getenv("YOYO_BROWSER_E2E") == "" {
		t.Skip("set YOYO_BROWSER_E2E=1 to drive a real isolated Chrome")
	}
	if findChrome() == "" {
		t.Skip("no Chrome or Chromium found (set YOYO_CHROME)")
	}
	// 尽力清理临时目录。Windows 在结束进程后还会锁一会儿 Chrome 的日志，这不应当让测试失败。
	dir, err := os.MkdirTemp("", "yoyo-e2e-page-*")
	if err != nil {
		t.Fatal(err)
	}
	profileDir, err := os.MkdirTemp("", "yoyo-e2e-profile-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		_ = os.RemoveAll(profileDir)
	})
	page := filepath.Join(dir, "index.html")
	if err := os.WriteFile(page, []byte(e2ePage), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := Open(profileDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)

	snap, err := h.OpenLocal(page)
	if err != nil {
		t.Fatalf("OpenLocal: %v", err)
	}
	if !strings.Contains(snap.Actions, "#opt-0") || !strings.Contains(snap.Actions, "ref=") {
		t.Fatalf("actions must offer a ref and the real selector:\n%s", snap.Actions)
	}
	h.mu.Lock()
	port := h.port
	h.mu.Unlock()

	// 空闲时没有调用方在读，连接也应当保持活着。
	time.Sleep(700 * time.Millisecond)
	if !h.chromeAlive() {
		t.Fatal("idle live Chrome reported dead")
	}

	if _, err := h.Click("ref=btn_0"); err != nil {
		t.Fatalf("click ref=btn_0: %v", err)
	}
	if title, _ := h.evalOnConn("document.title"); title != "picked:0" {
		t.Fatalf("click #opt-0 did not reach the page: title=%q", title)
	}

	obs, err := h.Click("text=Second option")
	if err != nil {
		t.Fatalf("click by text=: %v", err)
	}
	if !strings.Contains(obs, "page text changed") {
		t.Fatalf("a click that changes the page must say so: %s", obs)
	}

	if _, err := h.Type("body", "7"); err != nil {
		t.Fatalf("type: %v", err)
	}
	if title, _ := h.evalOnConn("document.title"); title != "key:7" {
		t.Fatalf("keydown listener did not receive the key: title=%q", title)
	}

	if _, err := h.Click("#under"); err == nil || !strings.Contains(err.Error(), "covered") {
		t.Fatalf("a click on an overlaid element must fail as covered, got %v", err)
	}

	_, err = h.Click("#nope")
	if err == nil || !strings.Contains(err.Error(), "matched no element") || !strings.Contains(err.Error(), "#opt-0") {
		t.Fatalf("a miss must list the real actions, got %v", err)
	}

	h.mu.Lock()
	after := h.port
	h.mu.Unlock()
	if after != port {
		t.Fatalf("interaction relaunched Chrome: port %d -> %d", port, after)
	}
	if _, live := h.Refresh(); !live {
		t.Fatal("Refresh must report the live page")
	}
}
