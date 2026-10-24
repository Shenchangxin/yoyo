package runtime

import (
	"fmt"
	"strings"
)

const rewriteStallHits = 3
const errorRepeatHits = 3
const waitLoopHits = 2
const idleActionHits = 3
const rewriteStallStopHits = 6
const errorRepeatStopHits = 6
const waitLoopStopHits = 4
const idleActionStopHits = 8
const checkFailNudgeHits = 2
const checkFailStopHits = 5
const checkFailIdleStop = 18
const checkFailBackstopHits = 8

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

const idleActionPrefixEN = "You already opened this page"
const idleActionPrefixZH = "你已经反复打开同一页面"
const idleActionNudgeEN = "You already opened this page (%s, %d times). browser_open reloads and wipes clicks. Call browser_snapshot then browser_act (ref=) for the remaining path — they do not reload. Do not browser_open the same HTML unless the file changed."
const idleActionNudgeZH = "你已经反复打开同一页面（%s，%d 次）。browser_open 会整页重载并清掉点击。剩余路径先 browser_snapshot，再用 browser_act（ref=）——它们不会重载。除非文件改过，不要再 browser_open 同一份 HTML。"

const clickMicroHits = 8
const clickMicroPrefixEN = "You have been clicking one selector per turn"
const clickMicroPrefixZH = "你已经连续微步点击"
const clickMicroNudgeEN = "You have been clicking one selector per turn (%d times). Pack the remaining path into one browser_act (click/fill/wait/assert/scroll/press). Snapshot then ref=. Do not open a new model round for each known click."
const clickMicroNudgeZH = "你已经连续微步点击（%d 轮）。把剩余路径放进一次 browser_act（click/fill/wait/assert/scroll/press）。先 snapshot 再点 ref=。不要为已知点击再开一轮模型。"

const checkFailPrefixEN = "The same check still FAILs"
const checkFailPrefixZH = "检查项仍以同样方式失败"
const checkFailNudgeEN = "The same check still FAILs (%s, %d identical runs). The output matches the previous run byte for byte, so the last edit did not affect this command. Do not run it again and do not rewrite the verifier. Switch to browser_verify on the page, or read how this script loads the file you changed."
const checkFailNudgeZH = "检查项仍以同样方式失败（%s，已连续 %d 次，输出和上一次逐字相同）。刚才的修改没有进到这次运行。不要再执行同一条命令，也不要改验证器。改用 browser_verify 打开页面取证；若要看脚本，先读它如何加载你改过的文件。"

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

// recordProgress updates the rewrite-stall counters for one round and reports
// whether the round advanced the work. Advancing means a plan step changed or
// a write actually changed a file. A failed or no-op write counts against its
// path, because it is an attempt that changed nothing, and that is what a
// stall is. Counting attempts instead (as this used to) ended turns that were
// making real edits to one large file.
//
// hits counts every write attempt per path since the last plan step. It drives
// the rewrite nudge, which also applies to productive rewrites: rewriting one
// file over and over without updating the plan is worth a nudge. fails counts
// only attempts that changed nothing, and only those can end the turn.
func recordProgress(hits, fails map[string]int, calls []ToolCall, results []Message, tools *WorkspaceTools, lastPlan *string) bool {
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
		clearCounts(hits)
		clearCounts(fails)
		return true
	}
	changed := false
	for i, tc := range calls {
		if !isWriteCall(tc.Name) {
			continue
		}
		ok := i < len(results) && writeChanged(results[i].Content)
		for _, p := range extractWritePaths(tc) {
			if hits != nil {
				hits[p]++
			}
			if ok {
				changed = true
				if fails != nil {
					delete(fails, p)
				}
				continue
			}
			if fails != nil {
				fails[p]++
			}
		}
	}
	return changed
}

func clearCounts(m map[string]int) {
	for k := range m {
		delete(m, k)
	}
}

func isWriteCall(name string) bool {
	switch name {
	case "write_file", "str_replace", "apply_patch", "office_edit":
		return true
	default:
		return false
	}
}

// writeChanged reports whether a write result describes a file that changed:
// not an error and not the "unchanged" receipt.
func writeChanged(content string) bool {
	c := strings.TrimSpace(content)
	if c == "" || strings.HasPrefix(c, "ERROR") {
		return false
	}
	return !strings.HasPrefix(c, "unchanged ")
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
	case strings.Contains(lower, "chrome debug port did not come up"):
		return "chrome debug port"
	case strings.Contains(lower, "message too big"):
		return "payload too big"
	// Browser misses name the selector, so the raw text differs on every
	// attempt and a repeat of the same failure never reached the threshold
	// (session 3b9f1bbb660e4236: six selector guesses, no stop). Group by kind.
	case strings.Contains(lower, "matched no element"):
		return "browser: no match"
	case strings.Contains(lower, "is covered by"):
		return "browser: covered"
	case strings.Contains(lower, "is not visible"):
		return "browser: hidden"
	case strings.Contains(lower, "is disabled"):
		return "browser: disabled"
	case strings.Contains(lower, "not a valid css selector"):
		return "browser: bad selector"
	case strings.Contains(lower, "page script error"):
		return "browser: page script error"
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

// idleProgressCall reports whether one call moved the task forward. A click or
// keypress must have worked, and a write must have changed a file. A failed
// str_replace moved nothing, so it must not clear the idle count; before this,
// any write call did, and an agent could loop on failing edits forever.
func idleProgressCall(tc ToolCall, result string) bool {
	switch tc.Name {
	case "browser_click", "browser_type", "browser_fill", "browser_act", "browser_verify",
		"browser_scroll", "browser_hover", "browser_select", "browser_press":
		return !strings.HasPrefix(strings.TrimSpace(result), "ERROR")
	case "write_file", "str_replace", "apply_patch":
		return writeChanged(result)
	default:
		return false
	}
}

func recordIdleActions(hits map[string]int, calls []ToolCall, results []Message) {
	if hits == nil {
		return
	}
	progressed := false
	for i, tc := range calls {
		res := ""
		if i < len(results) {
			res = results[i].Content
		}
		if idleProgressCall(tc, res) {
			progressed = true
		}
	}
	if progressed {
		for k := range hits {
			delete(hits, k)
		}
		return
	}
	for _, tc := range calls {
		if key := idleActionKey(tc); key != "" {
			hits[key]++
		}
	}
}

func idleActionKey(tc ToolCall) string {
	if tc.Name != "browser_open" {
		return ""
	}
	m := parseToolArgs(tc.Arguments)
	raw := str(m["url"])
	if raw == "" {
		raw = str(m["path"])
	}
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	return "browser_open:" + raw
}

func idleActionNudge(req RunRequest, hits map[string]int) string {
	if !req.SoftHorizon || hits == nil {
		return ""
	}
	key, n := "", 0
	for k, c := range hits {
		if c >= idleActionHits && c > n {
			key, n = k, c
		}
	}
	if key == "" {
		return ""
	}
	return fmt.Sprintf(voiceNudge(req, idleActionNudgeEN, idleActionNudgeZH), key, n)
}

func clickMicroRound(calls []ToolCall) bool {
	if len(calls) == 0 {
		return false
	}
	hasClick := false
	for _, tc := range calls {
		switch tc.Name {
		case "browser_click", "browser_type", "browser_fill", "browser_hover", "browser_select", "browser_press", "browser_scroll":
			hasClick = true
		case "browser_screenshot", "browser_snapshot":
		default:
			return false
		}
	}
	return hasClick
}

func recordClickMicro(n *int, calls []ToolCall) {
	if n == nil || len(calls) == 0 {
		return
	}
	if clickMicroRound(calls) {
		*n++
		return
	}
	onlyObserve := true
	for _, tc := range calls {
		switch tc.Name {
		case "browser_screenshot", "browser_snapshot":
		case "browser_act", "browser_verify":
			*n = 0
			return
		default:
			onlyObserve = false
		}
	}
	if !onlyObserve {
		*n = 0
	}
}

func clickMicroNudge(req RunRequest, n int) string {
	if !req.SoftHorizon || n < clickMicroHits {
		return ""
	}
	return fmt.Sprintf(voiceNudge(req, clickMicroNudgeEN, clickMicroNudgeZH), n)
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
		if strings.HasPrefix(s, "FAIL") {
			rest := strings.TrimSpace(strings.TrimPrefix(s, "FAIL"))
			rest = strings.TrimLeft(rest, " \t:")
			fields := strings.Fields(rest)
			if len(fields) > 0 && checkIdent(fields[0]) {
				add(fields[0])
			}
			continue
		}
		if key, ok := jsonFailKey(s); ok {
			add(key)
		}
	}
	return ids
}

// checkIdent is a check name such as render or coreLoop. A section heading
// like "FAIL 明细:" is not one: session 9ec78b5c78561d94 printed that heading
// on every run, so the watch treated render FAIL and render PASS as the same
// observation.
func checkIdent(id string) bool {
	if id == "" || len(id) > 32 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && (c >= '0' && c <= '9' || c == '_' || c == '-'):
		default:
			return false
		}
	}
	return true
}

// jsonFailKey reads one `"render": "FAIL"` field. "status" is the rollup, not
// a check, so a run that fixes render and still fails outcome changes signature.
func jsonFailKey(line string) (string, bool) {
	line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
	if !strings.HasPrefix(line, `"`) {
		return "", false
	}
	end := strings.Index(line[1:], `"`)
	if end < 1 {
		return "", false
	}
	key := line[1 : 1+end]
	rest := strings.TrimSpace(line[1+end+1:])
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	rest = strings.Trim(strings.TrimSpace(rest[1:]), `"`)
	if !strings.EqualFold(rest, "FAIL") || strings.EqualFold(key, "status") || !checkIdent(key) {
		return "", false
	}
	return key, true
}

type checkWatch struct {
	sig         string
	digest      string
	script      string
	refusal     string
	hits        int
	idle        int
	harnessHits int
	bypassHits  int
	identical   bool
	warned      bool
	allowRerun  bool
}

func (w *checkWatch) note(calls []ToolCall, results []Message) {
	if w == nil {
		return
	}
	// Reading qa/verification.json (or any FAIL dump) is not re-running the
	// check. Counting those reads as hits stopped harbor turns after one
	// failed Watch: the JSON still said "outcome":"FAIL", so SoftHorizon
	// fired "the same check kept failing" before the product could change.
	if inspectsOnly(calls) {
		return
	}
	digest, sig := "", ""
	for _, m := range results {
		if d := failObservation(m.Content); d != "" {
			digest = d
			sig = "check fail:" + strings.Join(failCheckIDs(m.Content), ",")
			break
		}
	}
	product := writesProduct(calls)
	harness := writesHarness(calls)
	if digest != "" {
		if digest == w.digest {
			w.hits++
			w.identical = true
			w.allowRerun = false
			if harness {
				w.harnessHits++
			} else if !product {
				w.bypassHits++
			}
		} else {
			// A different failing report is a changed observation.
			w.digest, w.sig, w.hits = digest, sig, 1
			w.identical = false
			w.warned = false
			w.refusal = ""
			w.allowRerun = false
			w.idle = 0
			w.harnessHits = 0
			w.bypassHits = 0
			if harness {
				w.harnessHits = 1
			}
		}
		if s := checkScript(calls); s != "" {
			w.script = s
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
		w.allowRerun = true
		w.idle = 0
		return
	}
	w.idle++
}

func (w *checkWatch) stuck() bool {
	if w == nil || w.sig == "" {
		return false
	}
	// Re-running one check with the same output is blocked, not fatal.
	// Rewriting the verifier, idling without an edit, or bypassing the
	// block with another command that reprints the same report still stops.
	if w.harnessHits >= checkFailStopHits || w.idle >= checkFailIdleStop {
		return true
	}
	return w.warned && w.bypassHits >= checkFailBackstopHits
}

// repeatBlock refuses the next run of the same script after an identical
// report. A product edit clears it for one more run.
func (w *checkWatch) repeatBlock() (script, refusal string) {
	if w == nil || !w.warned || w.allowRerun || w.script == "" || w.refusal == "" {
		return "", ""
	}
	return w.script, w.refusal
}

func (w *checkWatch) reason() string {
	return "the same check kept failing without the observation changing"
}

func observationNudge(req RunRequest, w *checkWatch, calls []ToolCall) string {
	if !req.SoftHorizon || w == nil || w.sig == "" {
		return ""
	}
	if w.identical && w.hits >= checkFailNudgeHits && !w.allowRerun {
		w.warned = true
		w.refusal = fmt.Sprintf(voiceNudge(req, checkFailNudgeEN, checkFailNudgeZH), w.sig, w.hits)
		return w.refusal
	}
	if writesHarness(calls) {
		return voiceNudge(req, instrumentNudgeEN, instrumentNudgeZH)
	}
	return ""
}

func failObservation(content string) string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		s := strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
		if strings.HasPrefix(s, "FAIL") {
			lines = append(lines, s)
			continue
		}
		if key, ok := jsonFailKey(line); ok {
			lines = append(lines, key+"=FAIL")
		}
	}
	return strings.Join(lines, "\n")
}

func checkScript(calls []ToolCall) string {
	script := ""
	verify := false
	for _, c := range calls {
		if c.Name == "browser_verify" {
			verify = true
		}
		if c.Name != "shell" {
			continue
		}
		if s := harnessScriptIn(c.Arguments); s != "" {
			script = s
		}
	}
	if script != "" {
		return script
	}
	if verify {
		return "browser_verify"
	}
	return ""
}

func harnessScriptIn(text string) string {
	text = normSlash(text)
	for _, f := range strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '"', '\'', '`', ',', ';', ':', '{', '}', '(', ')', '[', ']', '&', '|':
			return true
		default:
			return false
		}
	}) {
		if !isHarnessPath(f) {
			continue
		}
		if strings.HasSuffix(f, ".js") || strings.HasSuffix(f, ".cjs") || strings.HasSuffix(f, ".mjs") {
			return f
		}
	}
	return ""
}

// seedCheckWatch replays this turn's history so a continued turn blocks the
// script whose last report was already identical, instead of running it until
// the stop card. Harness and idle counters stay on this turn only.
func seedCheckWatch(req RunRequest, w *checkWatch, messages []Message) {
	if w == nil {
		return
	}
	for i := 0; i < len(messages); {
		if messages[i].Role != RoleAssistant || len(messages[i].ToolCalls) == 0 {
			i++
			continue
		}
		calls := messages[i].ToolCalls
		var results []Message
		j := i + 1
		for j < len(messages) && messages[j].Role == RoleTool {
			results = append(results, messages[j])
			j++
		}
		w.note(calls, results)
		i = j
	}
	w.harnessHits = 0
	w.idle = 0
	w.bypassHits = 0
	if w.identical && w.hits >= checkFailNudgeHits && !w.allowRerun {
		_ = observationNudge(req, w, nil)
	}
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

func inspectsOnly(calls []ToolCall) bool {
	if len(calls) == 0 {
		return false
	}
	for _, tc := range calls {
		switch tc.Name {
		case "read_file", "list_dir", "grep", "glob", "search_files", "read_artifact":
		default:
			return false
		}
	}
	return true
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
