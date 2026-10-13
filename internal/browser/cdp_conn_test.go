package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// fakeCDP answers CDP requests over one websocket. "Quit" closes the socket
// without replying, "Slow" replies late, and Runtime.evaluate with an
// expression containing "throw" returns exceptionDetails. It also emits an
// unsolicited event before every reply so the reader must skip events.
func fakeCDP(t *testing.T) (string, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		go func() {
			ctx := context.Background()
			for {
				var req struct {
					ID     int    `json:"id"`
					Method string `json:"method"`
					Params struct {
						Expression string `json:"expression"`
					} `json:"params"`
				}
				if err := wsjson.Read(ctx, c, &req); err != nil {
					return
				}
				switch req.Method {
				case "Quit":
					_ = c.CloseNow()
					return
				case "Slow":
					time.Sleep(300 * time.Millisecond)
				}
				_ = wsjson.Write(ctx, c, map[string]any{"method": "Page.loadEventFired", "params": map[string]any{}})
				// Runtime.evaluate's payload is {result, exceptionDetails} and
				// sits inside the response's "result" object, as in CDP.
				var payload map[string]any
				switch {
				case req.Method == "Runtime.evaluate" && strings.Contains(req.Params.Expression, "throw"):
					payload = map[string]any{
						"result": map[string]any{"type": "object"},
						"exceptionDetails": map[string]any{
							"text":      "Uncaught",
							"exception": map[string]any{"description": "TypeError: boom"},
						},
					}
				case req.Method == "Runtime.evaluate":
					payload = map[string]any{"result": map[string]any{"type": "string", "value": "ok"}}
				default:
					payload = map[string]any{}
				}
				_ = wsjson.Write(ctx, c, map[string]any{"id": req.ID, "result": payload})
			}
		}()
	}))
	return "ws" + strings.TrimPrefix(srv.URL, "http"), srv.Close
}

func dialFake(t *testing.T, wsURL string) *cdpConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	return newCDPConn(ws)
}

func TestCDPCallRoundTripSkipsEvents(t *testing.T) {
	url, stop := fakeCDP(t)
	defer stop()
	c := dialFake(t, url)
	defer c.close()
	raw, err := c.call("Runtime.evaluate", map[string]any{"expression": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"value":"ok"`) {
		t.Fatalf("unexpected reply %s", raw)
	}
}

// Regression for session 3b9f1bbb660e4236: liveness was a Ping that nothing
// read, so an idle healthy connection was reported dead, Chrome was relaunched
// onto about:blank, and every click reported "selector not found".
func TestIdleConnectionStaysAlive(t *testing.T) {
	url, stop := fakeCDP(t)
	defer stop()
	c := dialFake(t, url)
	defer c.close()
	time.Sleep(600 * time.Millisecond)
	if !cdpAlive(c) {
		t.Fatal("idle healthy connection reported dead")
	}
	if _, err := c.call("Page.navigate", map[string]any{"url": "about:blank"}); err != nil {
		t.Fatalf("call after idle: %v", err)
	}
}

func TestSlowReplyDoesNotCloseConnection(t *testing.T) {
	url, stop := fakeCDP(t)
	defer stop()
	c := dialFake(t, url)
	defer c.close()
	if _, err := c.call("Slow", nil); err != nil {
		t.Fatal(err)
	}
	if !cdpAlive(c) {
		t.Fatal("a slow reply tore the connection down")
	}
}

func TestPeerCloseFailsWaitersAsGone(t *testing.T) {
	url, stop := fakeCDP(t)
	defer stop()
	c := dialFake(t, url)
	defer c.close()
	_, err := c.call("Quit", nil)
	if err == nil {
		t.Fatal("expected an error when the peer closes")
	}
	if !isCDPGone(err) {
		t.Fatalf("closed socket must read as gone so callCDP reconnects: %v", err)
	}
	if cdpAlive(c) {
		t.Fatal("closed connection still reported alive")
	}
	if _, err := c.call("Runtime.evaluate", nil); !isCDPGone(err) {
		t.Fatalf("call on a dead connection: %v", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	url, stop := fakeCDP(t)
	defer stop()
	c := dialFake(t, url)
	c.close()
	c.close()
	if cdpAlive(c) {
		t.Fatal("closed connection reported alive")
	}
	var nilConn *cdpConn
	nilConn.close()
	if cdpAlive(nilConn) {
		t.Fatal("nil connection reported alive")
	}
}

func TestEvalOnConnSurfacesPageException(t *testing.T) {
	url, stop := fakeCDP(t)
	defer stop()
	h := &Host{cdp: dialFake(t, url)}
	defer h.cdp.close()
	if _, err := h.evalOnConn("throw new Error('boom')"); err == nil || !strings.Contains(err.Error(), "TypeError: boom") {
		t.Fatalf("page exception must be an error, got %v", err)
	}
	out, err := h.evalOnConn("1+1")
	if err != nil || out != "ok" {
		t.Fatalf("plain evaluation: %q %v", out, err)
	}
}

func TestProbeJSEscapesSelectorAndCarriesHelpers(t *testing.T) {
	js := probeJS(`#a"b`, false)
	if !strings.Contains(js, `const sel = "#a\"b";`) {
		t.Fatalf("selector not JSON-escaped: %s", js)
	}
	if !strings.Contains(js, "function __yoyoVisible") || !strings.Contains(js, "__yoyoResolve(sel)") {
		t.Fatal("probe must resolve through the shared page helpers")
	}
	if !strings.Contains(probeJS("x", true), "if (true) {") {
		t.Fatal("focus-only probe must take the focus branch")
	}
	if !strings.Contains(probeJS("x", false), "if (false) {") {
		t.Fatal("pointer probe must take the press branch")
	}
}

func TestPageHelpersAreFunctionsOnly(t *testing.T) {
	// Top-level const/let in a repeated Runtime.evaluate would throw
	// "already declared" on the second call. Helpers must stay in functions.
	for _, line := range strings.Split(pageHelpers, "\n") {
		if strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") {
			t.Fatalf("top-level declaration would break repeated evaluation: %q", line)
		}
	}
}

func TestKeyForCoversGameKeys(t *testing.T) {
	d := keyFor('2')
	if d.key != "2" || d.code != "Digit2" || d.text != "2" {
		t.Fatalf("digit: %+v", d)
	}
	l := keyFor('a')
	if l.key != "a" || l.code != "KeyA" || l.vk != 65 {
		t.Fatalf("lower letter: %+v", l)
	}
	e := keyFor('\n')
	if e.key != "Enter" || e.vk != 13 {
		t.Fatalf("enter: %+v", e)
	}
	c := keyFor('开')
	if c.text != "开" || c.code != "" {
		t.Fatalf("cjk char keeps its text: %+v", c)
	}
}

func TestFormatActionsListsSelectorsAndDisabled(t *testing.T) {
	got := formatActions([]Action{
		{Selector: "#opt-0", Tag: "button", Text: "1 走近"},
		{Selector: "text=Save", Tag: "button", Text: "Save", Disabled: true},
	})
	if !strings.Contains(got, "#opt-0  <button>") || !strings.Contains(got, "(disabled)") {
		t.Fatalf("unexpected action list:\n%s", got)
	}
}
