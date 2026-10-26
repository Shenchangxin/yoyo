// 本文件用假的 CDP WebSocket 锁住往返、空闲连接仍算活着、慢回复不拆连接，以及页面脚本异常会变成错误。
// 由 go test 调用。

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

// fakeCDP 用一条 WebSocket 回答 CDP。Quit 直接关掉，Slow 晚一点回，表达式里含 throw 时返回 exceptionDetails。每次回复前先发一条无关事件。
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
				// Runtime.evaluate 的载荷是 {result, exceptionDetails}，放在响应的 result 对象里，和 CDP 一样。
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

// dialFake 连上假的 WebSocket 并返回 newCDPConn。
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

// TestCDPCallRoundTripSkipsEvents 锁住调用能跳过中间的事件，拿到 Runtime.evaluate 的结果。
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

// TestIdleConnectionStaysAlive 锁住空闲 600 毫秒后连接仍算活着，之后的调用还能成功。
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

// TestSlowReplyDoesNotCloseConnection 锁住慢回复不会把连接拆掉。
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

// TestPeerCloseFailsWaitersAsGone 锁住对端关掉时等待者得到 isCDPGone 能认出的错误，之后的调用也算断开。
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

// TestCloseIsIdempotent 锁住 close 可以调用两次，空连接的 close 和 cdpAlive 也不会误报活着。
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

// TestEvalOnConnSurfacesPageException 锁住页面异常变成错误，普通表达式仍返回 ok。
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

// TestProbeJSEscapesSelectorAndCarriesHelpers 锁住选择器按 JSON 转义，并带上页面里的辅助函数。focusOnly 决定走聚焦还是按压分支。
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

// TestPageHelpersAreFunctionsOnly 锁住 pageHelpers 顶层没有 const 或 let，避免重复执行时重复声明。
func TestPageHelpersAreFunctionsOnly(t *testing.T) {
	// 重复执行的 Runtime.evaluate 里，顶层 const 或 let 第二次会报 already declared。辅助代码必须留在函数里。
	for _, line := range strings.Split(pageHelpers, "\n") {
		if strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") {
			t.Fatalf("top-level declaration would break repeated evaluation: %q", line)
		}
	}
}

// TestKeyForCoversGameKeys 锁住数字、小写字母、换行和汉字各自的 key、code 和 text。
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

// TestFormatActionsListsSelectorsAndDisabled 锁住列表里有选择器和 disabled 标记。
func TestFormatActionsListsSelectorsAndDisabled(t *testing.T) {
	got := formatActions([]Action{
		{Selector: "#opt-0", Tag: "button", Text: "1 走近"},
		{Selector: "text=Save", Tag: "button", Text: "Save", Disabled: true},
	})
	if !strings.Contains(got, "#opt-0  <button>") || !strings.Contains(got, "(disabled)") {
		t.Fatalf("unexpected action list:\n%s", got)
	}
}
