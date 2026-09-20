package runtime

import (
	"fmt"
	"strings"
)

const rewriteStallHits = 3

const rewriteStallPrefixEN = "You have rewritten "
const rewriteStallPrefixZH = "你已经反复整文件覆盖 "

const rewriteStallNudgeEN = "You have rewritten %s %d times without updating the plan. Call read_file on that path, mark the finished plan step complete with update_plan, and use str_replace. Do not rewrite the tree from memory."
const rewriteStallNudgeZH = "你已经反复整文件覆盖 %s %d 次，且计划没有推进。用 read_file 读回该路径，立刻 update_plan 把完成的步骤标 complete，再用 str_replace 小改。禁止凭记忆整树重写。"

func persistWorkingMemory(req RunRequest, messages []Message, notes string) string {
	spill := spillOf(req)
	if spill == nil {
		return notes
	}
	n := NotesFromMessages(stripSystem(messages))
	n.Next = firstOpenPlanStep(planTextOf(req.Tools))
	WriteNotes(spill, n)
	WriteDiscoverIndex(req.Workspace, req.SessionID, spill)
	out := n.Markdown()
	if mem := ReadWorkspaceMemory(req.Workspace); mem != "" {
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += mem
	}
	return capRunes(out, 2000)
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
	if looksCJK(req.User) || looksCJK(planTextOf(req.Tools)) {
		return fmt.Sprintf(rewriteStallNudgeZH, path, n)
	}
	return fmt.Sprintf(rewriteStallNudgeEN, path, n)
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
