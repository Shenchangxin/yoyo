// 工具结果写入轨迹前的预览。全文先放进 spill，返回的才是截过的预览。循环在派发工具后调用。
// 不负责 recall_context 的分页。

package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// defaultIngestBudget 是工具结果预览的默认 rune 上限。Loop 的 ToolResultBudget 大于 0 时用那个值。
const defaultIngestBudget = 8_000

// ingestBudget 取 ToolResultBudget，否则返回 defaultIngestBudget。
func ingestBudget(loop artifact.LoopPreset) int {
	if loop.ToolResultBudget > 0 {
		return loop.ToolResultBudget
	}
	return defaultIngestBudget
}

// ingestToolResult 先把全文写入 spill，再返回预览。budget 是 rune 上限，小于等于 0 时用 defaultIngestBudget。截断保留头尾。
// 正文为空时返回 emptyToolAnchor。noStub 为真，或正文已经是占位时，预览保持原文。
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
