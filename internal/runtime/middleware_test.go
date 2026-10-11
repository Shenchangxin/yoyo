package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestMiddlewareNudgeAfterConsecutiveErrors(t *testing.T) {
	loop := artifact.LoopPreset{MaxRecentToolErrors: 2, ToolErrorInstruction: "change strategy"}
	if got := middlewareNudge(loop, []Message{{Content: "ERROR: a"}}); got != "" {
		t.Fatal(got)
	}
	if got := middlewareNudge(loop, []Message{{Content: "ERROR: a"}, {Content: "ERROR: b"}}); got != "change strategy" {
		t.Fatal(got)
	}
}

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

func TestIdleActionNudgeOnRepeatedBrowserOpen(t *testing.T) {
	hits := map[string]int{}
	args := `{"url":"game-adaptations/xiyou-tianming-guiling/build/app/index.html"}`
	recordIdleActions(hits, []ToolCall{
		{Name: "browser_open", Arguments: args},
		{Name: "browser_open", Arguments: args},
		{Name: "browser_open", Arguments: args},
	})
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
	})
	if hitMax(hits) != 0 {
		t.Fatalf("click must reset idle opens: %v", hits)
	}
}

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
