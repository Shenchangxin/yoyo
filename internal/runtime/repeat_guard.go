package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Repeat reminders fire only when a tracked call's canonical arguments match
// the previous tracked call. Reads and plan updates are transparent: they
// neither count nor reset. The reminder is a separate message; the tool
// result itself is never rewritten, and the chain never stops the turn.
const (
	repeatGentleHits  = 3
	repeatDetailA     = 5
	repeatDetailB     = 8
	repeatArgsPreview = 500
)

const repeatGentleEN = "You are repeating the exact same tool call with identical arguments. Carefully analyze the previous result before calling again: if the task is not complete, try a different approach or different arguments instead of repeating the call."
const repeatGentleZH = "你正在用完全相同的参数重复调用同一个工具。先看上一次的结果再决定：任务还没完成就换做法或换参数，不要再发这一次调用。"

const repeatDetailPrefixEN = "Repeated tool call detected:"
const repeatDetailPrefixZH = "检测到重复的工具调用："

const repeatDetailEN = repeatDetailPrefixEN + "\n- tool: %s\n- consecutive_calls: %d\n- arguments: %s\nThe repeated calls are not making progress. Do not call this tool with these exact arguments again. Inspect the latest result and choose a different action, different arguments, or finish the task if enough evidence has been gathered."
const repeatDetailZH = repeatDetailPrefixZH + "\n- tool: %s\n- consecutive_calls: %d\n- arguments: %s\n这些重复没有带来进展。不要再用这组参数调用该工具。看最新的结果，换一个动作、换参数，或者证据已经够了就结束任务。"

// repeatTracker is one agent's consecutive-call chain. The key is the tool
// name plus canonical arguments, not the result.
type repeatTracker struct {
	key   string
	count int
	name  string
	canon string
}

func newRepeatTracker() *repeatTracker {
	return &repeatTracker{}
}

func (r *repeatTracker) reset() {
	if r == nil {
		return
	}
	*r = repeatTracker{}
}

// observe advances the chain for this batch and returns the reminder for a
// threshold that this batch just hit. Later calls in the same batch replace
// an earlier reminder. An empty return means this batch stays quiet.
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

func repeatTransparent(name string) bool {
	switch name {
	case "read_file", "list_dir", "grep", "glob", "search_files", "read_artifact", "update_plan":
		return true
	default:
		return false
	}
}

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

func previewArgs(canon string, cap int) string {
	if cap < 1 || utf8.RuneCountInString(canon) <= cap {
		return canon
	}
	r := []rune(canon)
	return string(r[:cap]) + fmt.Sprintf("… (+%d more chars)", len(r)-cap)
}
