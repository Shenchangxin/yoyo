package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const defaultIngestBudget = 8_000

func ingestBudget(loop artifact.LoopPreset) int {
	if loop.ToolResultBudget > 0 {
		return loop.ToolResultBudget
	}
	return defaultIngestBudget
}

// ingestToolResult spills full bytes first, then returns a live/jsonl preview.
// budget is the rune cap for ordinary tools (default ToolResultBudget).
// noStub is set for recall_context so a 16k recall is not stubbed again.
func ingestToolResult(spill *Spill, id, name, content string, budget int, noStub bool) (preview, spillID string) {
	spillID = id
	if strings.TrimSpace(content) == "" {
		return emptyToolAnchor(name), spillID
	}
	if spill != nil {
		spillID = spill.Put(id, content)
	}
	if noStub || alreadyStubbed(content) {
		return content, spillID
	}
	if budget <= 0 {
		budget = defaultIngestBudget
	}
	capped, trunc := capText(content, budget)
	if !trunc {
		return content, spillID
	}
	return stubTool(spillID, name, len(content)) + "\n" + capped, spillID
}
