// SoftHorizon 才发重复错误、空闲打开、wait 循环和点击微步轻推。失败的动作不重置空闲计数。

package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestMiddlewareNudgeAfterConsecutiveErrors 确认连续错误达到阈值后注入轻推。
func TestMiddlewareNudgeAfterConsecutiveErrors(t *testing.T) {
	loop := artifact.LoopPreset{MaxRecentToolErrors: 2, ToolErrorInstruction: "change strategy"}
	if got := middlewareNudge(loop, []Message{{Content: "ERROR: a"}}); got != "" {
		t.Fatal(got)
	}
	if got := middlewareNudge(loop, []Message{{Content: "ERROR: a"}, {Content: "ERROR: b"}}); got != "change strategy" {
		t.Fatal(got)
	}
}

// TestErrorRepeatNudgeOnSoftHorizon 确认 SoftHorizon 对重复错误发轻推。
func TestErrorRepeatNudgeOnSoftHorizon(t *testing.T) {
	hits := map[string]int{}
	results := []Message{
		{Content: `ERROR: capability: path "https://export.arxiv.org/api/query" escapes workspace`},
		{Content: `ERROR: capability: path "https://arxiv.org/list" escapes workspace`},
		{Content: `ERROR: capability: path "https://html.duckduckgo.com/html" escapes workspace`},
	}
	recordToolErrors(hits, results)
	req := RunRequest{User: "帮我查论文", SoftHorizon: true}
	got := errorRepeatNudge(req, hits)
	if !strings.Contains(got, errorRepeatPrefixZH) || !strings.Contains(got, "escapes workspace") {
		t.Fatalf("%q", got)
	}
	harbor := errorRepeatNudge(RunRequest{User: "帮我查论文"}, hits)
	if harbor != "" {
		t.Fatalf("harbor %q", harbor)
	}
}

// TestIdleActionNudgeOnRepeatedBrowserOpen 确认重复 browser_open 触发空闲轻推。
func TestIdleActionNudgeOnRepeatedBrowserOpen(t *testing.T) {
	hits := map[string]int{}
	args := `{"url":"game-adaptations/xiyou-tianming-guiling/build/app/index.html"}`
	recordIdleActions(hits, []ToolCall{
		{Name: "browser_open", Arguments: args},
		{Name: "browser_open", Arguments: args},
		{Name: "browser_open", Arguments: args},
	}, nil)
	req := RunRequest{User: "把小说做成游戏", SoftHorizon: true}
	got := idleActionNudge(req, hits)
	if !strings.Contains(got, idleActionPrefixZH) || !strings.Contains(got, "index.html") {
		t.Fatalf("%q hits=%v", got, hits)
	}
	if idleActionNudge(RunRequest{User: "把小说做成游戏"}, hits) != "" {
		t.Fatal("harbor")
	}
	hits = map[string]int{}
	recordIdleActions(hits, []ToolCall{
		{Name: "browser_open", Arguments: args},
		{Name: "browser_click", Arguments: `{"selector":"#boot"}`},
	}, []Message{{Content: ""}, {Content: `clicked #boot → <button> "Start" at (10,20); page text changed`}})
	if hitMax(hits) != 0 {
		t.Fatalf("a click that worked must reset idle opens: %v", hits)
	}
}

// TestIdleActionFailedProgressDoesNotReset 确认失败的 str_replace 或 click 不重置空闲计数。
func TestIdleActionFailedProgressDoesNotReset(t *testing.T) {
	args := `{"url":"game-adaptations/xiyou-tianming-guiling/build/app/index.html"}`
	hits := map[string]int{}
	for i := 0; i < 2; i++ {
		recordIdleActions(hits, []ToolCall{{Name: "browser_open", Arguments: args}}, nil)
	}
	recordIdleActions(hits, []ToolCall{
		{Name: "str_replace", Arguments: `{"path":"a.html"}`},
		{Name: "browser_click", Arguments: `{"selector":"#gone"}`},
	}, []Message{
		{Content: "ERROR: str_replace: old_str not found in a.html"},
		{Content: "ERROR: browser: #gone matched no element"},
	})
	if hitMax(hits) != 2 {
		t.Fatalf("failed progress must not reset the idle count: %v", hits)
	}
}

// TestIdleActionUnchangedWriteDoesNotReset 确认结果含 unchanged 的写入不重置空闲计数。
func TestIdleActionUnchangedWriteDoesNotReset(t *testing.T) {
	args := `{"url":"game-adaptations/xiyou-tianming-guiling/build/app/index.html"}`
	hits := map[string]int{}
	recordIdleActions(hits, []ToolCall{{Name: "browser_open", Arguments: args}}, nil)
	recordIdleActions(hits, []ToolCall{{Name: "write_file", Arguments: `{"path":"a.html"}`}},
		[]Message{{Content: "unchanged a.html"}})
	if hitMax(hits) != 1 {
		t.Fatalf("an unchanged write must not reset the idle count: %v", hits)
	}
}

// TestWaitLoopNudgeOnSoftHorizon 确认 SoftHorizon 对 wait 循环发轻推。
func TestWaitLoopNudgeOnSoftHorizon(t *testing.T) {
	hits := map[string]int{}
	recordWaits(hits, []ToolCall{{Name: "wait"}, {Name: "wait"}})
	req := RunRequest{User: "帮我查论文", SoftHorizon: true}
	got := waitLoopNudge(req, hits)
	if !strings.Contains(got, waitLoopPrefixZH) {
		t.Fatalf("%q", got)
	}
	if waitLoopNudge(RunRequest{User: "帮我查论文"}, hits) != "" {
		t.Fatal("harbor")
	}
	if waitLoopNudge(req, map[string]int{"wait": 1}) != "" {
		t.Fatal("one wait is not a loop")
	}
}

// TestErrorRepeatNudgeOnQAFail 确认重复的检查失败会发轻推。
func TestErrorRepeatNudgeOnQAFail(t *testing.T) {
	hits := map[string]int{}
	results := []Message{
		{Content: `{"status":"FAIL","outcome":"FAIL","checks":["play.html"]}`},
		{Content: `{"status": "FAIL", "outcome": "FAIL"}`},
		{Content: `{"status":"FAIL"}`},
	}
	recordToolErrors(hits, results)
	req := RunRequest{User: "把小说做成游戏", SoftHorizon: true}
	got := errorRepeatNudge(req, hits)
	if !strings.Contains(got, errorRepeatPrefixZH) || !strings.Contains(got, "status fail") {
		t.Fatalf("%q hits=%v", got, hits)
	}
}

// TestErrorSignatureContextStubAndTruncatedJSON 确认 context stub、截断 JSON、空路径和空补丁收成 write args json，websocket message too big 收成 payload too big。
func TestErrorSignatureContextStubAndTruncatedJSON(t *testing.T) {
	if got := errorSignature("ERROR: refusing to write a context stub; read_file that path"); got != "context stub" {
		t.Fatalf("stub %q", got)
	}
	if got := errorSignature("ERROR: truncated JSON arguments; refusing a partial write"); got != "write args json" {
		t.Fatalf("truncated %q", got)
	}
	if got := errorSignature("ERROR: websocket: message too big"); got != "payload too big" {
		t.Fatalf("ws %q", got)
	}
	if got := errorSignature("ERROR: browser: chrome debug port did not come up: open C:\\Users\\scx\\.yoyo\\browser\\profile-headed\\DevToolsActivePort"); got != "chrome debug port" {
		t.Fatalf("chrome %q", got)
	}
}

// TestErrorRepeatNudgeOnEmptyPath 确认空路径的重复错误会发轻推。
func TestErrorRepeatNudgeOnEmptyPath(t *testing.T) {
	hits := map[string]int{}
	results := []Message{
		{Content: "ERROR: empty path"},
		{Content: "ERROR: empty path"},
		{Content: "ERROR: empty patch"},
	}
	recordToolErrors(hits, results)
	req := RunRequest{User: "帮我查论文", SoftHorizon: true}
	got := errorRepeatNudge(req, hits)
	if !strings.Contains(got, errorRepeatPrefixZH) || !strings.Contains(got, "write args json") {
		t.Fatalf("%q hits=%v", got, hits)
	}
}

// TestClickMicroNudgeAfterConsecutiveClicks 确认连续点击触发微步轻推，成功点击会重置计数。
func TestClickMicroNudgeAfterConsecutiveClicks(t *testing.T) {
	n := 0
	click := []ToolCall{{Name: "browser_click", Arguments: `{"selector":"#a"}`}}
	for i := 0; i < clickMicroHits; i++ {
		recordClickMicro(&n, click)
	}
	req := RunRequest{User: "把小说做成游戏", SoftHorizon: true}
	got := clickMicroNudge(req, n)
	if !strings.Contains(got, clickMicroPrefixZH) || !strings.Contains(got, "browser_act") {
		t.Fatalf("%q n=%d", got, n)
	}
	recordClickMicro(&n, []ToolCall{{Name: "browser_act", Arguments: `{"steps":[]}`}})
	if n != 0 {
		t.Fatalf("browser_act must reset micro count, n=%d", n)
	}
	if clickMicroNudge(RunRequest{User: "把小说做成游戏"}, clickMicroHits) != "" {
		t.Fatal("harbor")
	}
}

// TestIdleProgressCallIncludesNewBrowserOps 确认 browser_act 重置微步计数。
func TestIdleProgressCallIncludesNewBrowserOps(t *testing.T) {
	for _, name := range []string{"browser_fill", "browser_scroll", "browser_press", "browser_select", "browser_hover"} {
		if !idleProgressCall(ToolCall{Name: name}, "ok") {
			t.Fatalf("%s should count as progress", name)
		}
		if idleProgressCall(ToolCall{Name: name}, "ERROR: miss") {
			t.Fatalf("%s ERROR must not count as progress", name)
		}
	}
}

// TestClickMicroRoundCountsFillAndScroll 确认 fill、scroll、press、select、hover 在结果不是 ERROR 时算进度。
func TestClickMicroRoundCountsFillAndScroll(t *testing.T) {
	n := 0
	recordClickMicro(&n, []ToolCall{{Name: "browser_fill", Arguments: `{"ref":"inp_0","text":"a"}`}})
	recordClickMicro(&n, []ToolCall{{Name: "browser_scroll", Arguments: `{"direction":"down"}`}})
	if n != 2 {
		t.Fatalf("micro n=%d", n)
	}
}
