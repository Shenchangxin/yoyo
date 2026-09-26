package runtime

import (
	"fmt"
	"strings"
)

const rewriteStallHits = 3
const errorRepeatHits = 3
const waitLoopHits = 2

const rewriteStallPrefixEN = "You have rewritten "
const rewriteStallPrefixZH = "你已经反复整文件覆盖 "

const rewriteStallNudgeEN = "You have rewritten %s %d times without updating the plan. Call read_file on that path, mark the finished plan step complete with update_plan, and use str_replace. Do not rewrite the tree from memory."
const rewriteStallNudgeZH = "你已经反复整文件覆盖 %s %d 次却没有更新计划。先 read_file，用 update_plan 勾掉完成步骤，再用 str_replace。不要凭记忆重写整棵树。"

const errorRepeatPrefixEN = "The same tool error keeps repeating"
const errorRepeatPrefixZH = "同一个工具错误在重复"

const errorRepeatNudgeEN = "The same tool error keeps repeating (%s). Stop retrying that call. Use a different approach or finish with what you already have."
const errorRepeatNudgeZH = "同一个工具错误在重复（%s）。不要再用同样的调用。换一条路，或用已经拿到的结果收束。"

const waitLoopPrefixEN = "You have been waiting in a poll loop"
const waitLoopPrefixZH = "你已经在空转等待"

const waitLoopNudgeEN = "You have been waiting in a poll loop. Stop wait/tasklist. If the shell was idle/block fused, it is already stopped. For public HTTP use web_fetch (retry http if https is 406), or finish with what you have."
const waitLoopNudgeZH = "你已经在空转等待。停止 wait/tasklist。若 shell 已 idle/block fuse，进程已被停掉。公开 HTTP 用 web_fetch（https 返回 406 就改 http），或用已有结果收束。"

func persistWorkingMemory(req RunRequest, messages []Message, notes string) string {
	spill := spillOf(req)
	if spill == nil {
		return notes
	}
	n := NotesFromMessages(stripSystem(messages))
	n.Next = firstOpenPlanStep(planTextOf(req.Tools))
	WriteNotes(spill, n)
	WriteDiscoverIndex(req.Workspace, req.SessionID, spill)
	return n.Markdown()
}

func firstOpenPlanStep(plan string) string {
	for _, line := range strings.Split(plan, "\n") {
		st, ok := planLineStatus(line)
		if !ok || planStatusDone(st) {
			continue
		}
		end := strings.IndexByte(line, ']')
		if end < 0 || end+1 >= len(line) {
			return strings.TrimSpace(line)
		}
		step := strings.TrimSpace(line[end+1:])
		if step != "" {
			return step
		}
	}
	return ""
}

func recordProgress(hits map[string]int, calls []ToolCall, tools *WorkspaceTools, lastPlan *string) {
	if hits == nil {
		return
	}
	advanced := false
	for _, tc := range calls {
		if tc.Name != "update_plan" {
			continue
		}
		cur := planTextOf(tools)
		if lastPlan != nil && cur != *lastPlan {
			*lastPlan = cur
			advanced = true
		}
	}
	if advanced {
		for k := range hits {
			delete(hits, k)
		}
		return
	}
	for _, tc := range calls {
		if tc.Name != "write_file" && tc.Name != "str_replace" {
			continue
		}
		for _, p := range extractJSONPaths(tc.Arguments) {
			hits[p]++
		}
	}
}

func rewriteStallNudge(req RunRequest, hits map[string]int) string {
	if !req.SoftHorizon {
		return ""
	}
	path, n := "", 0
	for p, c := range hits {
		if c >= rewriteStallHits && c > n {
			path, n = p, c
		}
	}
	if path == "" {
		return ""
	}
	return fmt.Sprintf(voiceNudge(req, rewriteStallNudgeEN, rewriteStallNudgeZH), path, n)
}

func recordToolErrors(hits map[string]int, results []Message) {
	if hits == nil {
		return
	}
	for _, m := range results {
		sig := errorSignature(m.Content)
		if sig == "" {
			continue
		}
		hits[sig]++
	}
}

func errorSignature(content string) string {
	s := strings.TrimSpace(content)
	if !strings.HasPrefix(s, "ERROR:") {
		return ""
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "ERROR:"))
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "escapes workspace"):
		return "escapes workspace"
	case strings.Contains(lower, "capability:"):
		return "capability"
	case strings.Contains(lower, "406"):
		return "http 406"
	case strings.Contains(lower, "host is not allowed"):
		return "host blocked"
	default:
		if len(s) > 80 {
			s = s[:80]
		}
		return s
	}
}

func errorRepeatNudge(req RunRequest, hits map[string]int) string {
	if !req.SoftHorizon {
		return ""
	}
	sig, n := "", 0
	for s, c := range hits {
		if c >= errorRepeatHits && c > n {
			sig, n = s, c
		}
	}
	if sig == "" {
		return ""
	}
	return fmt.Sprintf(voiceNudge(req, errorRepeatNudgeEN, errorRepeatNudgeZH), sig)
}

func recordWaits(hits map[string]int, calls []ToolCall) {
	if hits == nil {
		return
	}
	for _, tc := range calls {
		if tc.Name == "wait" {
			hits["wait"]++
		}
	}
}

func waitLoopNudge(req RunRequest, hits map[string]int) string {
	if !req.SoftHorizon || hits == nil {
		return ""
	}
	if hits["wait"] < waitLoopHits {
		return ""
	}
	return voiceNudge(req, waitLoopNudgeEN, waitLoopNudgeZH)
}

func planFirstOverlay(t *WorkspaceTools) string {
	return skillOverlay(t, "plan-first", t != nil && PlanOpen(planTextOf(t)))
}

func verifyArtifactOverlay(t *WorkspaceTools) string {
	return skillOverlay(t, "verify-artifact", t != nil && t.VerifyHint)
}

func skillOverlay(t *WorkspaceTools, name string, cond bool) string {
	if t == nil || !t.ChatOverlay || !cond || name == "" {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, n := range t.Loaded {
		if n == name {
			return ""
		}
	}
	if t.Skills == nil {
		return ""
	}
	body := strings.TrimSpace(t.Skills[name])
	if body == "" {
		return ""
	}
	return "## Skill: " + name + "\n" + body
}
