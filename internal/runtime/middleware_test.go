package runtime

import (
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
