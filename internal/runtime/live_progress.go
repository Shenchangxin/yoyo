// 流式工具参数的进度事件。chat 在参数增量通过 toolProgressGate 时调用 emitToolProgress。
// 不执行工具。JSONL 是否落盘由 emit 决定：progress 为真的 tool_call 不写入轨迹。

package runtime

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// toolProgressGate 按工具调用 id 节流进度事件。
type toolProgressGate struct {
	id string
	n  int
	t  time.Time
}

// allow 在 id 为空时返回 false。同一个 id、本次参数增量长度与上次相差小于 2048、且间隔小于 250 毫秒时返回 false。否则记下 id、长度和时间并返回 true。
func (g *toolProgressGate) allow(id string, n int, now time.Time) bool {
	if id == "" {
		return false
	}
	if id == g.id && n-g.n < 2048 && now.Sub(g.t) < 250*time.Millisecond {
		return false
	}
	g.id = id
	g.n = n
	g.t = now
	return true
}

// emitToolProgress 向实时总线发一条 progress 为真的 tool_call。id 或名字去掉空白后为空则返回。arguments 只保留从参数里抽出的 path，bytes 仍是原始参数的字节长度。
func emitToolProgress(req RunRequest, roundID string, tc ToolCall) {
	if strings.TrimSpace(tc.ID) == "" || strings.TrimSpace(tc.Name) == "" {
		return
	}
	path := salvageJSONStringField(tc.Arguments, "path")
	args := map[string]any{}
	if path != "" {
		args["path"] = path
	}
	raw, _ := json.Marshal(args)
	payload := map[string]any{
		"name":      tc.Name,
		"id":        tc.ID,
		"round":     roundID,
		"progress":  true,
		"bytes":     len(tc.Arguments),
		"arguments": string(raw),
	}
	if path != "" {
		payload["path"] = path
	}
	emit(req, trace.TypeToolCall, "model", payload)
}

// payloadProgress 读取 progress 字段。p 为 nil 或该字段不是 bool 时返回 false。
func payloadProgress(p map[string]any) bool {
	if p == nil {
		return false
	}
	v, _ := p["progress"].(bool)
	return v
}
