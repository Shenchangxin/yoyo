package runtime

import (
	"fmt"
	"strings"
)

const rewriteStallHits = 3
const errorRepeatHits = 3
const waitLoopHits = 2
const rewriteStallStopHits = 6
const errorRepeatStopHits = 6
const waitLoopStopHits = 4
const checkFailNudgeHits = 2
const checkFailStopHits = 5
const checkFailIdleStop = 18

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

const checkFailPrefixEN = "The same check still FAILs"
const checkFailPrefixZH = "检查项仍以同样方式失败"
const checkFailNudgeEN = "The same check still FAILs (%s, %d hits, %d idle rounds). Progress is an observation that changed. Edit the product, then re-run that same check — do not rewrite the verifier."
const checkFailNudgeZH = "检查项仍以同样方式失败（%s，已 %d 次、观测未变 %d 轮）。进度 = 观测变了。不要再改验证器；改产品文件后重跑同一条检查。"

const instrumentPrefixEN = "The failing check cites the product"
const instrumentPrefixZH = "失败检查指向产品文件"
const instrumentNudgeEN = "The failing check cites the product, not the verifier. Do not rewrite qa/verify or treat a node -e probe as a fix — edit the cited product (usually html/js/css under build/) so the same check's observation changes."
const instrumentNudgeZH = "失败检查指向产品文件，不是验证器。不要再改 qa/verify 或用 node -e 探针当修复——去改检查所引用的产品（通常是 build/ 下的 html/js/css），让同一次检查的观测结果发生变化。"

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
		if tc.Name != "write_file" && tc.Name != "str_replace" && tc.Name != "apply_patch" {
			continue
		}
		for _, p := range extractWritePaths(tc) {
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
	if ids := failCheckIDs(s); len(ids) > 0 {
		return "check fail:" + strings.Join(ids, ",")
	}
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, `"status": "fail"`), strings.Contains(lower, `"status":"fail"`):
		return "status fail"
	case strings.Contains(lower, `"outcome": "fail"`), strings.Contains(lower, `"outcome":"fail"`):
		return "outcome fail"
	}
	if strings.HasPrefix(s, "ERROR:") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "ERROR:"))
		lower = strings.ToLower(s)
	} else if !failureBody(lower) {
		return ""
	}
	switch {
	case strings.Contains(lower, "escapes workspace"):
		return "escapes workspace"
	case strings.Contains(lower, "empty path"), strings.Contains(lower, "empty patch"), strings.Contains(lower, "invalid json arguments"), strings.Contains(lower, "truncated json"):
		return "write args json"
	case strings.Contains(lower, "context stub"):
		return "context stub"
	case strings.Contains(lower, "capability:"):
		return "capability"
	case strings.Contains(lower, "406"):
		return "http 406"
	case strings.Contains(lower, "host is not allowed"):
		return "host blocked"
	case strings.Contains(lower, "message too big"):
		return "payload too big"
	default:
		if !strings.HasPrefix(strings.TrimSpace(content), "ERROR:") {
			return ""
		}
		if len(s) > 80 {
			s = s[:80]
		}
		return s
	}
}

func failureBody(lower string) bool {
	return strings.Contains(lower, "truncated json") ||
		strings.Contains(lower, "context stub") ||
		strings.Contains(lower, "message too big")
}

func hitMax(hits map[string]int) int {
	n := 0
	for _, c := range hits {
		if c > n {
			n = c
		}
	}
	return n
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
	if sig == "" || strings.HasPrefix(sig, "check fail:") {
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
	if t != nil && t.Methodology {
		return ""
	}
	return skillOverlay(t, "plan-first", t != nil && PlanOpen(planTextOf(t)))
}

func verifyArtifactOverlay(t *WorkspaceTools) string {
	return skillOverlay(t, "verify-artifact", t != nil && t.VerifyHint)
}

func failCheckIDs(content string) []string {
	var ids []string
	seen := map[string]struct{}{}
	add := func(id string) {
		id = strings.ToLower(strings.Trim(id, ":,;"))
		if id == "" || id == "fail" || id == "pass" || id == "not_run" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, line := range strings.Split(content, "\n") {
		s := strings.TrimSpace(line)
		if !strings.HasPrefix(s, "FAIL") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(s, "FAIL"))
		rest = strings.TrimLeft(rest, " \t:")
		if rest == "" {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 || len(fields[0]) > 32 {
			continue
		}
		add(fields[0])
	}
	return ids
}

type checkWatch struct {
	sig  string
	hits int
	idle int
}

func (w *checkWatch) note(calls []ToolCall, results []Message) {
	if w == nil {
		return
	}
	sig := ""
	for _, m := range results {
		if ids := failCheckIDs(m.Content); len(ids) > 0 {
			sig = "check fail:" + strings.Join(ids, ",")
			break
		}
	}
	product := writesProduct(calls)
	if sig != "" {
		if sig == w.sig {
			w.hits++
		} else {
			w.sig, w.hits = sig, 1
		}
		if product {
			w.idle = 0
		} else {
			w.idle++
		}
		return
	}
	if w.sig == "" {
		return
	}
	if product {
		w.idle = 0
		return
	}
	w.idle++
}

func (w *checkWatch) stuck() bool {
	if w == nil || w.sig == "" {
		return false
	}
	return w.hits >= checkFailStopHits || w.idle >= checkFailIdleStop
}

func (w *checkWatch) reason() string {
	return "the same check kept failing without the observation changing"
}

func observationNudge(req RunRequest, w *checkWatch, calls []ToolCall) string {
	if !req.SoftHorizon || w == nil || w.sig == "" {
		return ""
	}
	if writesHarness(calls) {
		return voiceNudge(req, instrumentNudgeEN, instrumentNudgeZH)
	}
	if w.hits >= checkFailNudgeHits {
		return fmt.Sprintf(voiceNudge(req, checkFailNudgeEN, checkFailNudgeZH), w.sig, w.hits, w.idle)
	}
	return ""
}

func extractWritePaths(tc ToolCall) []string {
	switch tc.Name {
	case "write_file", "str_replace", "edit_file", "create_file":
		return extractJSONPaths(tc.Arguments)
	case "apply_patch":
		if paths := patchFilePaths(tc.Arguments); len(paths) > 0 {
			return paths
		}
		return extractJSONPaths(tc.Arguments)
	default:
		return nil
	}
}

func patchFilePaths(args string) []string {
	text := args
	if m := parseToolArgs(args); len(m) > 0 {
		if p, ok := m["patch"].(string); ok && p != "" {
			text = p
		}
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		s := strings.TrimSpace(line)
		for _, pfx := range []string{"*** Update File:", "*** Add File:", "*** Delete File:"} {
			if rest, ok := strings.CutPrefix(s, pfx); ok {
				if p := strings.TrimSpace(rest); p != "" {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

func writesHarness(calls []ToolCall) bool {
	for _, tc := range calls {
		for _, p := range extractWritePaths(tc) {
			if isHarnessPath(p) {
				return true
			}
		}
	}
	return false
}

func writesProduct(calls []ToolCall) bool {
	for _, tc := range calls {
		for _, p := range extractWritePaths(tc) {
			if isProductPath(p) {
				return true
			}
		}
	}
	return false
}

func isHarnessPath(p string) bool {
	p = normSlash(p)
	base := p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		base = p[i+1:]
	}
	if strings.Contains(p, "/qa/") || strings.HasPrefix(p, "qa/") ||
		strings.Contains(p, "/__tests__/") || strings.Contains(p, "/tests/") {
		return true
	}
	if strings.Contains(base, "verify") || strings.Contains(base, "whitebox") {
		return true
	}
	if strings.HasSuffix(base, ".mjs") && (strings.Contains(base, "check") || strings.Contains(base, "spec") || strings.Contains(base, "test")) {
		return true
	}
	return false
}

func isProductPath(p string) bool {
	if isHarnessPath(p) {
		return false
	}
	p = normSlash(p)
	for _, ext := range []string{".js", ".mjs", ".cjs", ".html", ".css", ".ts", ".tsx", ".jsx"} {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return false
}

func normSlash(p string) string {
	return strings.TrimPrefix(strings.ToLower(strings.ReplaceAll(p, "\\", "/")), "./")
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
