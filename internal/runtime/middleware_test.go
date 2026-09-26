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
