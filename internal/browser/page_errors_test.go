package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestPageErrorFromEventException(t *testing.T) {
	params := json.RawMessage(`{"timestamp":1,"exceptionDetails":{"exceptionId":3,"text":"Uncaught","lineNumber":1028,"columnNumber":4,"url":"http://127.0.0.1:9/app/index.html?v=2","exception":{"type":"object","subtype":"error","className":"TypeError","description":"TypeError: Cannot read properties of undefined (reading 'h')\n    at hit (index.html:1029:9)\n    at onTap (index.html:1031:3)"}}}`)
	got := pageErrorFromEvent("Runtime.exceptionThrown", params)
	want := "TypeError: Cannot read properties of undefined (reading 'h') (index.html:1029)"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPageErrorFromEventUncaughtWithoutException(t *testing.T) {
	// A syntax error can arrive without an exception object; the text is then
	// the only description there is.
	params := json.RawMessage(`{"exceptionDetails":{"text":"SyntaxError: Unexpected token '}'","lineNumber":0,"url":""}}`)
	if got := pageErrorFromEvent("Runtime.exceptionThrown", params); got != "SyntaxError: Unexpected token '}'" {
		t.Fatalf("got %q", got)
	}
}

func TestPageErrorFromEventConsoleError(t *testing.T) {
	params := json.RawMessage(`{"type":"error","args":[{"type":"string","value":"save failed:"},{"type":"object","subtype":"error","description":"QuotaExceededError: full"}],"executionContextId":1}`)
	if got := pageErrorFromEvent("Runtime.consoleAPICalled", params); got != "console.error: save failed: QuotaExceededError: full" {
		t.Fatalf("got %q", got)
	}
}

func TestPageErrorFromEventIgnoresOrdinaryEvents(t *testing.T) {
	cases := []struct {
		method string
		params string
	}{
		{"Page.loadEventFired", `{"timestamp":1}`},
		{"Runtime.consoleAPICalled", `{"type":"log","args":[{"type":"string","value":"hello"}]}`},
		{"Runtime.consoleAPICalled", `{"type":"warning","args":[{"type":"string","value":"careful"}]}`},
		{"Network.requestWillBeSent", `{"requestId":"1"}`},
	}
	for _, c := range cases {
		if got := pageErrorFromEvent(c.method, json.RawMessage(c.params)); got != "" {
			t.Fatalf("%s: expected no report, got %q", c.method, got)
		}
	}
}

func TestPageErrorsDedupeAndDrain(t *testing.T) {
	var p pageErrors
	ev := json.RawMessage(`{"exceptionDetails":{"text":"Uncaught","url":"http://h/index.html","lineNumber":9,"exception":{"description":"TypeError: x is null"}}}`)
	for i := 0; i < 3; i++ {
		p.add("Runtime.exceptionThrown", ev)
	}
	got := p.take()
	if !strings.Contains(got, "TypeError: x is null (index.html:10) (x3)") {
		t.Fatalf("repeats must collapse into one counted line:\n%s", got)
	}
	if again := p.take(); again != "" {
		t.Fatalf("take must clear the log, got %q", again)
	}
}

func TestPageErrorsCapKeepsMemoryBounded(t *testing.T) {
	var p pageErrors
	for i := 0; i < pageErrorCap+15; i++ {
		p.add("Runtime.consoleAPICalled", json.RawMessage(fmt.Sprintf(`{"type":"error","args":[{"type":"string","value":"e%d"}]}`, i)))
	}
	report := p.take()
	lines := strings.Split(report, "\n")
	// The first line is the heading; then one line per kept error, then one
	// line counting the 15 that did not fit.
	if n := len(lines) - 2; n != pageErrorCap {
		t.Fatalf("kept %d errors, want %d:\n%s", n, pageErrorCap, report)
	}
	if !strings.Contains(report, "and 15 more distinct errors not shown") {
		t.Fatalf("overflow must be counted, not silent:\n%s", report)
	}
}

func TestShortLocationNamesFileAndLine(t *testing.T) {
	if got := shortLocation("http://127.0.0.1:5/a/b/game.html?x=1#y", 4); got != " (game.html:5)" {
		t.Fatalf("got %q", got)
	}
	if got := shortLocation("", 4); got != "" {
		t.Fatalf("no url, no location: got %q", got)
	}
}

// pageEventServer accepts one websocket, emits a page exception before it
// answers anything, then echoes every request with an empty result.
func pageEventServer(t *testing.T) (string, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		go func() {
			ctx := context.Background()
			_ = wsjson.Write(ctx, c, map[string]any{
				"method": "Runtime.exceptionThrown",
				"params": map[string]any{"exceptionDetails": map[string]any{
					"text": "Uncaught",
					"url":  "http://127.0.0.1/index.html",
					"exception": map[string]any{
						"description": "TypeError: boom",
					},
				}},
			})
			for {
				var req struct {
					ID int `json:"id"`
				}
				if err := wsjson.Read(ctx, c, &req); err != nil {
					return
				}
				_ = wsjson.Write(ctx, c, map[string]any{"id": req.ID, "result": map[string]any{}})
			}
		}()
	}))
	return "ws" + strings.TrimPrefix(srv.URL, "http"), srv.Close
}

// A page exception is read off the socket by the same reader that matches
// replies. Because the server sends it before the reply, it is in the log by
// the time the call returns, with no sleep.
func TestCDPConnCollectsPageExceptionsFromSocket(t *testing.T) {
	url, stop := pageEventServer(t)
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := newCDPConn(ws)
	defer c.close()
	if _, err := c.call("Runtime.enable", nil); err != nil {
		t.Fatal(err)
	}
	got := c.errs.take()
	if !strings.Contains(got, "TypeError: boom (index.html:1)") {
		t.Fatalf("exception event was not collected:\n%q", got)
	}
}

func TestHostTakePageErrorsWithoutChromeIsEmpty(t *testing.T) {
	var h *Host
	if got := h.takePageErrors(); got != "" {
		t.Fatalf("nil host: got %q", got)
	}
	h = &Host{}
	if got := h.pageErrorsNote(); got != "" {
		t.Fatalf("no chrome: got %q", got)
	}
}

func TestCoveredWhyNullHitTestIsNotCoveredByNothing(t *testing.T) {
	got := coveredWhy("#shelf", "nothing")
	if strings.Contains(got, "covered by nothing") {
		t.Fatalf("null hit test must not read as an overlay: %q", got)
	}
	if !strings.Contains(got, "no hit-test element at its centre") {
		t.Fatalf("message must say what happened: %q", got)
	}
	if got := coveredWhy("#a", "div.modal"); got != "#a is covered by div.modal, so a real click would not reach it" {
		t.Fatalf("real cover must keep its wording: %q", got)
	}
}
