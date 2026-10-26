// 同一工具、同一组规范化参数的连续调用提醒。读文件和计划更新既不计数也不清零。
// 提醒是另插的一条用户消息。工具结果本身不改写，这条链也不结束回合。Run 在每批工具之后调用 observe。

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// 短提醒在计数第一次到达 3 时发出，长提醒在 5 和 8 时发出。8 之后同一条链不再出提醒。参数预览最多 500 个 rune。
const (
	repeatGentleHits  = 3
	repeatDetailA     = 5
	repeatDetailB     = 8
	repeatArgsPreview = 500
)

// 短提醒和长提醒各有英文和中文。voiceNudge 按操作者语言挑选，本文件不改这两段文案。
const repeatGentleEN = "You are repeating the exact same tool call with identical arguments. Carefully analyze the previous result before calling again: if the task is not complete, try a different approach or different arguments instead of repeating the call."
const repeatGentleZH = "你正在用完全相同的参数重复调用同一个工具。先看上一次的结果再决定：任务还没完成就换做法或换参数，不要再发这一次调用。"

const repeatDetailPrefixEN = "Repeated tool call detected:"
const repeatDetailPrefixZH = "检测到重复的工具调用："

const repeatDetailEN = repeatDetailPrefixEN + "\n- tool: %s\n- consecutive_calls: %d\n- arguments: %s\nThe repeated calls are not making progress. Do not call this tool with these exact arguments again. Inspect the latest result and choose a different action, different arguments, or finish the task if enough evidence has been gathered."
const repeatDetailZH = repeatDetailPrefixZH + "\n- tool: %s\n- consecutive_calls: %d\n- arguments: %s\n这些重复没有带来进展。不要再用这组参数调用该工具。看最新的结果，换一个动作、换参数，或者证据已经够了就结束任务。"

// repeatTracker 是一个代理的连续调用链。键是工具名加规范化参数，不是结果。
type repeatTracker struct {
	key   string
	count int
	name  string
	canon string
}

// newRepeatTracker 返回一条空链。
func newRepeatTracker() *repeatTracker {
	return &repeatTracker{}
}

// reset 把链清成零值。接收者为 nil 时什么都不做。用户插话时 Run 调用它。
func (r *repeatTracker) reset() {
	if r == nil {
		return
	}
	*r = repeatTracker{}
}

// observe 按本批调用推进链条，返回本批刚好跨过的那条提醒。同一批里后面的调用会换掉前面的提醒。接收者为 nil，或本批没有跨过 3、5、8，就返回空串。
func (r *repeatTracker) observe(req RunRequest, calls []ToolCall) string {
	if r == nil {
		return ""
	}
	reminder := ""
	for _, tc := range calls {
		if repeatTransparent(tc.Name) {
			continue
		}
		canon := canonicalArgs(tc.Arguments)
		key := tc.Name + "\x00" + canon
		if key == r.key {
			r.count++
		} else {
			r.key = key
			r.count = 1
			r.name = tc.Name
			r.canon = canon
		}
		switch r.count {
		case repeatGentleHits:
			reminder = voiceNudge(req, repeatGentleEN, repeatGentleZH)
		case repeatDetailA, repeatDetailB:
			preview := previewArgs(r.canon, repeatArgsPreview)
			reminder = fmt.Sprintf(voiceNudge(req, repeatDetailEN, repeatDetailZH), r.name, r.count, preview)
		}
	}
	return reminder
}

// repeatTransparent 对 read_file、list_dir、grep、glob、search_files、read_artifact、update_plan 返回 true。这些调用跳过，计数和当前键都不变。
func repeatTransparent(name string) bool {
	switch name {
	case "read_file", "list_dir", "grep", "glob", "search_files", "read_artifact", "update_plan":
		return true
	default:
		return false
	}
}

// canonicalArgs 得到比较用的参数。去掉空白后为空则返回 {}。合法 JSON 重新编码，键顺序不影响结果；解析失败时返回去掉空白后的原文。
func canonicalArgs(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return string(b)
}

// previewArgs 在 cap 小于 1 或 rune 数不超过 cap 时返回全文。否则保留前 cap 个 rune，并标出剩下的 rune 数。
func previewArgs(canon string, cap int) string {
	if cap < 1 || utf8.RuneCountInString(canon) <= cap {
		return canon
	}
	r := []rune(canon)
	return string(r[:cap]) + fmt.Sprintf("… (+%d more chars)", len(r)-cap)
}
