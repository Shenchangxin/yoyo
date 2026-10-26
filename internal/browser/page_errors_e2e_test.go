// 本文件用真实隔离 Chrome 确认加载期异常、点击处理函数抛出的异常和 console.error 会进入观察结果。页面 HTML 是数据。
// 没设置 YOYO_BROWSER_E2E=1 或找不到 Chrome 时跳过。由 go test 调用。

package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// e2eErrorPage 是会在加载、点击和 console.error 三处抛错的页面，字符串是数据。
const e2eErrorPage = `<!doctype html><html><head><meta charset="utf-8"></head><body>
<button id="boom">Boom</button>
<button id="ok">OK</button>
<button id="log">Log</button>
<p id="msg">idle</p>
<script>
const loadTime = null;
void loadTime.x;
</script>
<script>
document.getElementById('boom').onclick = () => { const o = null; void o.h; };
document.getElementById('ok').onclick = () => { document.getElementById('msg').textContent = 'fine'; };
document.getElementById('log').onclick = () => { console.error('save failed', 42); };
</script>
</body></html>`

// TestBrowserPageErrorsE2E 驱动真实 Chrome。打开后能拿到加载期异常，点击 #boom 和 #log 时观察结果里带上对应错误。
func TestBrowserPageErrorsE2E(t *testing.T) {
	if os.Getenv("YOYO_BROWSER_E2E") == "" {
		t.Skip("set YOYO_BROWSER_E2E=1 to drive a real isolated Chrome")
	}
	if findChrome() == "" {
		t.Skip("no Chrome or Chromium found (set YOYO_CHROME)")
	}
	// 尽力清理。Windows 在结束进程后还会锁一会儿 Chrome 的文件，这不应当让测试失败。
	dir, err := os.MkdirTemp("", "yoyo-e2e-errors-*")
	if err != nil {
		t.Fatal(err)
	}
	profileDir, err := os.MkdirTemp("", "yoyo-e2e-errors-profile-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		_ = os.RemoveAll(profileDir)
	})
	page := filepath.Join(dir, "index.html")
	if err := os.WriteFile(page, []byte(e2eErrorPage), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := Open(profileDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)

	if _, err := h.OpenLocal(page); err != nil {
		t.Fatalf("OpenLocal: %v", err)
	}
	if errs := h.PageErrors(); !strings.Contains(errs, "Cannot read properties of null (reading 'x')") {
		t.Fatalf("a load-time exception must be available to the open result:\n%q", errs)
	}

	obs, err := h.Click("#boom")
	if err != nil {
		t.Fatalf("click #boom: %v", err)
	}
	if !strings.Contains(obs, "page errors raised since the last browser call") ||
		!strings.Contains(obs, "Cannot read properties of null (reading 'h')") {
		t.Fatalf("a click whose handler threw must report the exception:\n%s", obs)
	}

	obs, err = h.Click("#log")
	if err != nil {
		t.Fatalf("click #log: %v", err)
	}
	if !strings.Contains(obs, "console.error: save failed 42") {
		t.Fatalf("console.error must be reported:\n%s", obs)
	}

	obs, err = h.Click("#ok")
	if err != nil {
		t.Fatalf("click #ok: %v", err)
	}
	if strings.Contains(obs, "page errors raised") {
		t.Fatalf("errors are reported once; a clean click must not repeat them:\n%s", obs)
	}
}
