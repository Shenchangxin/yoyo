package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// e2eErrorPage raises three kinds of page error: one while the page loads, one
// from a click handler, and one through console.error. Before page_errors.go,
// none of them reached the agent; a click on #boom looked like a click that
// did nothing.
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

// TestBrowserPageErrorsE2E drives a real Chrome. It is gated like the other
// browser e2e test because it launches Chrome.
func TestBrowserPageErrorsE2E(t *testing.T) {
	if os.Getenv("YOYO_BROWSER_E2E") == "" {
		t.Skip("set YOYO_BROWSER_E2E=1 to drive a real isolated Chrome")
	}
	if findChrome() == "" {
		t.Skip("no Chrome or Chromium found (set YOYO_CHROME)")
	}
	// Best-effort cleanup: Windows keeps Chrome's files locked for a moment
	// after the kill, and that must not fail the test.
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
