// SoftHorizon 上的停滞计数和催促：整文件重写、重复错误、空转等待、反复打开同一页、微步点击，以及检查输出是否变化。
// Run 在每批工具之后调用这里。硬循环（SoftHorizon 为假）下，这些催促函数直接返回空串。停止次数由 Run 用 hitMax 比较。

package runtime

import (
	"fmt"
	"strings"
)

// 催促阈值和停止阈值。催促只追加用户消息。停止由 Run 在 SoftHorizon 下比较 hitMax。
const rewriteStallHits = 3      // 同一路径的写入次数达到 3 就催促改用 str_replace
const errorRepeatHits = 3       // 同一错误签名达到 3 就催促换做法
const waitLoopHits = 2          // wait 达到 2 次就催促停止空转
const idleActionHits = 3        // 同一页面 browser_open 达到 3 次就催促
const rewriteStallStopHits = 6  // 同一路径无进展写入达到 6 次，Run 以 StopStuck 结束
const errorRepeatStopHits = 6   // 同一错误签名达到 6 次，Run 以 StopStuck 结束
const waitLoopStopHits = 4      // wait 达到 4 次，Run 以 StopStuck 结束
const idleActionStopHits = 8    // 同一页面打开达到 8 次，Run 以 StopStuck 结束
const checkFailNudgeHits = 2    // 同一份失败观察达到 2 次就催促并开始拒绝重跑
const checkFailStopHits = 5     // 改验证器且观察不变达到 5 次，checkWatch.stuck 返回 true
const checkFailIdleStop = 18    // 失败之后空转等待，或被拒绝的重跑，达到 18 次，stuck 返回 true
const checkFailBackstopHits = 8 // 已经警告后，用别的命令重印同一份报告达到 8 次，stuck 返回 true

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

// clickMicroHits 是 8。连续微步点击达到 8 次才催促收成一次 browser_act。
const clickMicroHits = 8
const clickMicroPrefixEN = "You have been clicking one selector per turn"
const clickMicroPrefixZH = "你已经连续微步点击"
const clickMicroNudgeEN = "You have been clicking one selector per turn (%d times). Pack the remaining path into one browser_act (click/fill/wait/assert/scroll/press). Snapshot then ref=. Do not open a new model round for each known click."
const clickMicroNudgeZH = "你已经连续微步点击（%d 轮）。把剩余路径放进一次 browser_act（click/fill/wait/assert/scroll/press）。先 snapshot 再点 ref=。不要为已知点击再开一轮模型。"

const checkFailPrefixEN = "The same check still FAILs"
const checkFailPrefixZH = "检查项仍以同样方式失败"
const checkFailNudgeEN = "The same check still FAILs (%s, %d identical runs). The output matches the previous run byte for byte, so the last edit did not affect this command. Do not run it again and do not rewrite the verifier. Switch to browser_verify on the page, or read how this script loads the file you changed."
const checkFailNudgeZH = "检查项仍以同样方式失败（%s，已连续 %d 次，输出和上一次逐字相同）。刚才的修改没有进到这次运行。不要再执行同一条命令，也不要改验证器。改用 browser_verify 打开页面取证；若要看脚本，先读它如何加载你改过的文件。"

// 拒绝文案是工具结果，不是新的检查输出。必须写明这次没有执行，避免模型把同一句话当成观测又没变。
const checkRepeatRefusalEN = "Not executed. Do not run that command again: %s would rerun the check that still FAILs (%s, %d identical runs). This call produced no new output. Edit the product the check cites, then run it once. Do not rewrite the verifier. Use browser_verify for evidence."
const checkRepeatRefusalZH = "没有执行。不要再执行同一条命令：%s 会重跑仍失败的检查（%s，已连续 %d 次）。这次没有产生新的输出。先改检查指向的产品文件，再运行一次。不要改验证器。取证用 browser_verify。"

const instrumentPrefixEN = "The failing check cites the product"
const instrumentPrefixZH = "失败检查指向产品文件"
const instrumentNudgeEN = "The failing check cites the product, not the verifier. Do not rewrite qa/verify or treat a node -e probe as a fix — edit the cited product (usually html/js/css under build/) so the same check's observation changes."
const instrumentNudgeZH = "失败检查指向产品文件，不是验证器。不要再改 qa/verify 或用 node -e 探针当修复——去改检查所引用的产品（通常是 build/ 下的 html/js/css），让同一次检查的观测结果发生变化。"

// persistWorkingMemory 把去掉系统消息后的笔记写进 spill，并写发现索引。没有 spill 时原样返回 notes。Next 取计划里第一条未完成步骤。
// req 用来取 spill、Tools、Workspace 和 SessionID。
// messages 去掉系统消息后交给 NotesFromMessages。
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

// firstOpenPlanStep 返回第一条未完成清单在方括号之后的文字。没有未完成步骤时返回空串。
// plan 按行查找，跳过已完成的清单行，返回方括号后的文字。
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

// recordProgress 更新这一轮的重写计数，并报告工作有没有前进。前进是指计划文本变了，或一次写入真的改了文件。失败或没有变化的写入仍记在该路径上，因为停滞就是这些没有改动的尝试。按尝试次数直接结束回合，会把还在改同一个大文件的回合停掉。
// hits 计的是自上次计划步骤以来每个路径的全部写入尝试，催促也覆盖仍在产生改动的反复重写。fails 只计没有改动的尝试，Run 只用这份计数结束回合。lastPlan 非 nil 且计划文本变了时，两份计数都清空并返回 true，同一批里的写入不再入账。
// calls 里的 update_plan 用来看计划是否变化，写入调用用来计数。
// results 与 calls 按下标对齐，交给 writeChanged。
// tools 传给 planTextOf。
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

// clearCounts 删掉 map 里的全部键。m 为 nil 时 range 什么都不做。
func clearCounts(m map[string]int) {
	for k := range m {
		delete(m, k)
	}
}

// isWriteCall 对 write_file、str_replace、apply_patch、office_edit 返回 true。
// name 等于这些名字之一时返回 true。
func isWriteCall(name string) bool {
	switch name {
	case "write_file", "str_replace", "apply_patch", "office_edit":
		return true
	default:
		return false
	}
}

// writeChanged 判断写入结果是不是真的改了文件。去掉空白后为空、以前缀 ERROR 开头，或以前缀 unchanged 加空格开头，都返回 false。
// content 去掉空白后按上述规则判断。
func writeChanged(content string) bool {
	c := strings.TrimSpace(content)
	if c == "" || strings.HasPrefix(c, "ERROR") {
		return false
	}
	return !strings.HasPrefix(c, "unchanged ")
}

// rewriteStallNudge 在 SoftHorizon 下，对达到 rewriteStallHits 的路径里计数最高的那条发催促。SoftHorizon 为假或没有这样的路径时返回空串。
// req 的 SoftHorizon 为假时返回空串，否则传给 voiceNudge。
// hits 里计数不小于 rewriteStallHits 的路径中，取计数最高的一条。
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

// recordToolErrors 按 errorSignature 给本批结果计数。hits 为 nil，或签名为空时跳过。Run 在整批都是查阅时不调用它。
// results 逐条用 Content 交给 errorSignature。
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

// errorSignature 把一条工具结果收成重复计数用的签名。检查项 FAIL 收成 check fail: 加标识列表。status 或 outcome 为 fail 时用固定签名。
// 浏览器点不中按种类归并，不按每次不同的选择器原文。其余 ERROR: 正文截到 80 字节。不是 ERROR:、也不属于 truncated json、context stub、message too big 的正文返回空串。
// content 去掉空白后先收检查 FAIL，再看 status、outcome、ERROR: 和失败正文。
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
	// 浏览器没命中时原文里带着选择器，每次猜测都不同，同一类失败凑不齐阈值。会话 3b9f1bbb660e4236 猜了六次选择器也没有停。这里按种类归并。
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

// failureBody 只把已经是小写的 truncated json、context stub、message too big 当成失败正文。没有 ERROR: 前缀时，errorSignature 只认这三类。
// lower 含这三类片段之一时返回 true。
func failureBody(lower string) bool {
	return strings.Contains(lower, "truncated json") ||
		strings.Contains(lower, "context stub") ||
		strings.Contains(lower, "message too big")
}

// hitMax 返回 map 里的最大计数。hits 为 nil 或为空时返回 0。
func hitMax(hits map[string]int) int {
	n := 0
	for _, c := range hits {
		if c > n {
			n = c
		}
	}
	return n
}

// errorRepeatNudge 在 SoftHorizon 下，对达到 errorRepeatHits 且不是 check fail: 前缀的最高签名发催促。检查失败交给 observationNudge。
// req 的 SoftHorizon 为假时返回空串，否则传给 voiceNudge。
// hits 里计数不小于 errorRepeatHits 的签名中，取计数最高且不是 check fail: 前缀的一条。
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

// recordWaits 给本批里的每个 wait 在键 wait 上加一。hits 为 nil 时什么都不做。
// calls 里 Name 为 wait 的每一条给 wait 加一。
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

// waitLoopNudge 在 SoftHorizon 且 wait 的计数不小于 waitLoopHits 时返回空转催促。hits 为 nil 时返回空串。
// req 的 SoftHorizon 为假时返回空串，否则传给 voiceNudge。
func waitLoopNudge(req RunRequest, hits map[string]int) string {
	if !req.SoftHorizon || hits == nil {
		return ""
	}
	if hits["wait"] < waitLoopHits {
		return ""
	}
	return voiceNudge(req, waitLoopNudgeEN, waitLoopNudgeZH)
}

// idleProgressCall 判断这一次调用有没有把任务往前推。点击或按键必须没有 ERROR 前缀，写入必须 writeChanged。失败的 str_replace 不算前进，不能清掉空转计数；以前任意写入都会清，失败编辑就能一直转。
// tc 的 Name 决定是看浏览器结果还是写入结果。
// result 去掉空白后以 ERROR 开头则浏览器调用不算前进；写入调用交给 writeChanged。
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

// recordIdleActions 在本批有一次前进时清空计数。否则给每个 browser_open 的地址加一。hits 为 nil 时什么都不做。
// calls 逐条与 results 对齐，交给 idleProgressCall，没有前进时再按 idleActionKey 计数。
// results 按下标取 Content，下标超出则当成空串。
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

// idleActionKey 对 browser_open 返回 browser_open: 加小写地址。地址先取 url，空则取 path。不是 browser_open 或地址为空时返回空串。
// tc 不是 browser_open 时返回空串，否则从 Arguments 取 url，空则取 path。
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

// idleActionNudge 在 SoftHorizon 下，对达到 idleActionHits 的地址里计数最高的那条发催促。hits 为 nil 时返回空串。
// req 的 SoftHorizon 为假时返回空串，否则传给 voiceNudge。
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

// clickMicroRound 在本批非空、至少有一次点击类调用，且其余只是截图或快照时返回 true。出现任何别的工具就返回 false。空批返回 false。
// calls 长度为 0 时返回 false；含点击类名字且其余只是截图或快照时返回 true。
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

// recordClickMicro 在微步点击轮把计数加一。本批含 browser_act 或 browser_verify 时把计数清零。只有截图或快照时保持原计数。出现其他工具也清零。n 为 nil 或调用为空时什么都不做。
// calls 长度为 0 时返回；是微步点击轮则把计数加一，含 browser_act 或 browser_verify 时清零。
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

// clickMicroNudge 在 SoftHorizon 且 n 不小于 clickMicroHits 时返回催促，否则返回空串。
// req 的 SoftHorizon 为假时返回空串，否则传给 voiceNudge。
func clickMicroNudge(req RunRequest, n int) string {
	if !req.SoftHorizon || n < clickMicroHits {
		return ""
	}
	return fmt.Sprintf(voiceNudge(req, clickMicroNudgeEN, clickMicroNudgeZH), n)
}

// planFirstOverlay 在方法论模式返回空串。否则在计划仍开着时附上 plan-first 技能正文。
func planFirstOverlay(t *WorkspaceTools) string {
	if t != nil && t.Methodology {
		return ""
	}
	return skillOverlay(t, "plan-first", t != nil && PlanOpen(planTextOf(t)))
}

// verifyArtifactOverlay 在 VerifyHint 为真时附上 verify-artifact 技能正文。
func verifyArtifactOverlay(t *WorkspaceTools) string {
	return skillOverlay(t, "verify-artifact", t != nil && t.VerifyHint)
}

// failCheckIDs 从 FAIL 行和 `"name": "FAIL"` 字段收集检查标识，小写、去重。fail、pass、not_run 和空标识丢掉。
// content 按行拆开，从 FAIL 前缀和 JSON FAIL 字段收集检查标识。
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

// checkIdent 判断是不是检查名，例如 render 或 coreLoop。像 FAIL 明细: 这种小节标题不是：会话 9ec78b5c78561d94 每次运行都印这行，观察会被当成没变。空串或长度超过 32 返回 false。首字符必须是字母，其后可以是字母、数字、下划线或连字符。
// id 为空或长度超过 32 时返回 false，否则按上述字符规则检查。
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

// jsonFailKey 从一行里读出 `"render": "FAIL"` 这种字段。status 是汇总，不是检查项，所以 render 修好但 outcome 仍失败时签名会变。键不是合法检查名，或值不是 FAIL，就返回 false。
// line 去掉空白和末尾逗号后必须以引号开头。
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

// checkWatch 记住最近一份失败观察。identical 为真表示这份摘要和上一份逐字相同。allowRerun 为真表示产品文件改过，同一条脚本可以再跑一次。
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

// note 记下这一批的失败观察。接收者为 nil 时什么都不做。
// 通过的检查会清掉失败闩：官方 browser_verify 为 PASS，或被拦住的脚本这次真的跑完且不再打印 FAIL。
// 失败闩还在时，只有空转等待和被拒绝的重跑才累加 idle。列目录、删除验证器、改计划都不是「同一份观察又出现了」。
// calls 先交给 inspectsOnly，再用来判断是否在写产品或验证器，以及抽出脚本。
// results 逐条用 Content 交给 failObservation，取第一份非空摘要。
func (w *checkWatch) note(calls []ToolCall, results []Message) {
	if w == nil {
		return
	}
	// 读 qa/verification.json 或任何 FAIL 转储不是在重跑检查。把这些读取算进次数，会在一次失败的 Watch 之后停住评测回合：JSON 里仍写着 outcome FAIL，产品还没改，SoftHorizon 就报同一条检查一直失败。
	if inspectsOnly(calls) {
		return
	}
	digest, sig := "", ""
	for i, m := range results {
		if d := failObservation(m.Content); d != "" {
			digest = d
			sig = "check fail:" + strings.Join(failCheckIDs(m.Content), ",")
			break
		}
		if i < len(calls) && calls[i].Name == "browser_verify" {
			if d, s := verifyFailObservation(m.Content); d != "" {
				digest, sig = d, s
				break
			}
		}
	}
	if digest == "" && (officialVerifyPass(calls, results) || scriptPass(w, calls, results)) {
		w.clear()
		return
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
			// 另一份失败报告就是观察变了，计数从 1 重新开始。
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
	if spinsAfterFailure(calls, results, w.refusal) {
		w.idle++
	}
}

// clear 丢掉失败闩。通过的检查说明上一份 FAIL 已经不是当前观察，脚本拦截和空转计数一并失效。
func (w *checkWatch) clear() {
	if w == nil {
		return
	}
	*w = checkWatch{}
}

// stuck 在还没有失败签名时返回 false。同一条检查原样重跑本身不在这里判停。改验证器达到 checkFailStopHits，或失败后的空转与被拒绝的重跑达到 checkFailIdleStop，返回 true。已经警告之后，绕开封锁重印同一份报告达到 checkFailBackstopHits，也返回 true。
func (w *checkWatch) stuck() bool {
	if w == nil || w.sig == "" {
		return false
	}
	// 同一份输出再跑一次检查只是被拒绝，这里不因此返回 true。改写验证器、空转等待、连续撞上拒绝，或用另一条命令重印同一份报告，仍会停。
	if w.harnessHits >= checkFailStopHits || w.idle >= checkFailIdleStop {
		return true
	}
	return w.warned && w.bypassHits >= checkFailBackstopHits
}

// repeatBlock 在相同报告已经警告过之后，返回要拒绝的脚本和拒绝文案。产品文件改过一次后 allowRerun 为真，这次返回空串，让那条脚本再跑一次。未警告、脚本为空或文案为空时也返回空串。
func (w *checkWatch) repeatBlock() (script, refusal string) {
	if w == nil || !w.warned || w.allowRerun || w.script == "" || w.refusal == "" {
		return "", ""
	}
	return w.script, w.refusal
}

// reason 返回观察没有变化时的停止说明。
func (w *checkWatch) reason() string {
	return "the same check kept failing without the observation changing"
}

// observationNudge 在 SoftHorizon 下发检查催促。同一份观察的次数不小于 checkFailNudgeHits 且还不能重跑时，记下 warned，把「没有执行」写进拒绝文案，并返回给模型的催促。否则若这批在写验证器路径，返回改产品文件的催促。其余情况返回空串。
// req 的 SoftHorizon 为假时返回空串，否则传给 voiceNudge。
// calls 传给 writesHarness。
func observationNudge(req RunRequest, w *checkWatch, calls []ToolCall) string {
	if !req.SoftHorizon || w == nil || w.sig == "" {
		return ""
	}
	if w.identical && w.hits >= checkFailNudgeHits && !w.allowRerun {
		w.warned = true
		target := w.script
		if target == "" {
			target = w.sig
		}
		w.refusal = fmt.Sprintf(voiceNudge(req, checkRepeatRefusalEN, checkRepeatRefusalZH), target, w.sig, w.hits)
		return fmt.Sprintf(voiceNudge(req, checkFailNudgeEN, checkFailNudgeZH), w.sig, w.hits)
	}
	if writesHarness(calls) {
		return voiceNudge(req, instrumentNudgeEN, instrumentNudgeZH)
	}
	return ""
}

// failObservation 把 FAIL 行和 JSON 里的 FAIL 字段收成一份摘要，空白折叠成单个空格。没有失败行时返回空串。
// content 按行拆开，保留 FAIL 前缀行，并用原行交给 jsonFailKey。
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

// checkScript 从 shell 参数里抽出 qa 脚本路径。没有脚本但调用了 browser_verify 时返回 browser_verify。都没有时返回空串。
// calls 里 browser_verify 记一笔，shell 的 Arguments 交给 harnessScriptIn。
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

// harnessScriptIn 在文本里找验证器路径下的 .js、.cjs 或 .mjs。找到就返回该路径，否则返回空串。
// text 先交给 normSlash，再按空白和标点拆开查找。
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

// officialVerifyPass 在某次 browser_verify 的自身结果写着 status=PASS 时返回 true。别的工具印出同样的句子不算通过，否则失败脚本可以自己解开拦截。
func officialVerifyPass(calls []ToolCall, results []Message) bool {
	for i, c := range calls {
		if c.Name != "browser_verify" || i >= len(results) {
			continue
		}
		if verifyStatus(results[i].Content) == "pass" {
			return true
		}
	}
	return false
}

// scriptPass 在被拦住的脚本这次真的执行完、没有 FAIL 行、并且打印了 PASS 检查行时返回 true。拒绝文本不是执行结果。
func scriptPass(w *checkWatch, calls []ToolCall, results []Message) bool {
	if w == nil || w.script == "" || w.script == "browser_verify" {
		return false
	}
	if !batchRerunsScript(calls, w.script) || resultsAreRefusal(results, w.refusal) || !batchHasPassLine(results) {
		return false
	}
	for _, m := range results {
		if failObservation(m.Content) != "" {
			return false
		}
	}
	return true
}

// verifyStatus 读 browser_verify 首行里的 status。同时出现时失败优先。没有这行时返回空串。
func verifyStatus(content string) string {
	low := strings.ToLower(content)
	switch {
	case strings.Contains(low, "browser_verify status=fail"):
		return "fail"
	case strings.Contains(low, "browser_verify status=pass"):
		return "pass"
	default:
		return ""
	}
}

// verifyFailObservation 从 browser_verify 的文本里收集失败检查。没有状态行时返回空串。只有总状态失败、没有逐项失败时，摘要是 browser_verify=FAIL。
func verifyFailObservation(content string) (digest, sig string) {
	if verifyStatus(content) == "" {
		return "", ""
	}
	var lines, ids []string
	seen := map[string]struct{}{}
	for _, line := range strings.Split(content, "\n") {
		name, status, ok := splitCheckStatus(strings.TrimSpace(line))
		if !ok || !strings.EqualFold(status, "FAIL") {
			continue
		}
		id := strings.ToLower(name)
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		lines = append(lines, id+"=FAIL")
		ids = append(ids, id)
	}
	if len(lines) == 0 {
		if verifyStatus(content) != "fail" {
			return "", ""
		}
		return "browser_verify=FAIL", "check fail:browser_verify"
	}
	return strings.Join(lines, "\n"), "check fail:" + strings.Join(ids, ",")
}

// splitCheckStatus 从 `render: FAIL` 这种行取出检查名和状态。检查名不合法时返回 false。
func splitCheckStatus(line string) (name, status string, ok bool) {
	i := strings.IndexByte(line, ':')
	if i <= 0 {
		return "", "", false
	}
	name = strings.TrimSpace(line[:i])
	status = strings.Trim(strings.TrimSpace(line[i+1:]), `"'`)
	if cut := strings.IndexAny(status, " \t"); cut > 0 {
		status = status[:cut]
	}
	if !checkIdent(name) || status == "" {
		return "", "", false
	}
	return name, status, true
}

// hasPassLine 在以 PASS 开头、后面跟着合法检查名的行上返回 true。
func hasPassLine(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		s := strings.TrimSpace(line)
		if !strings.HasPrefix(s, "PASS") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(s, "PASS"))
		rest = strings.TrimLeft(rest, " \t:-")
		fields := strings.Fields(rest)
		if len(fields) > 0 && checkIdent(fields[0]) {
			return true
		}
	}
	return false
}

// batchHasPassLine 在任一结果含 PASS 检查行时返回 true。
func batchHasPassLine(results []Message) bool {
	for _, m := range results {
		if hasPassLine(m.Content) {
			return true
		}
	}
	return false
}

// resultsAreRefusal 在拒绝文本非空、结果非空，且每条结果都是这份拒绝文本时返回 true。
func resultsAreRefusal(results []Message, refusal string) bool {
	refusal = strings.TrimSpace(refusal)
	if refusal == "" || len(results) == 0 {
		return false
	}
	for _, m := range results {
		if strings.TrimSpace(m.Content) != refusal {
			return false
		}
	}
	return true
}

// spinsAfterFailure 在整批都是 wait，或都是被拒绝的 shell / browser_verify 重跑时返回 true。夹着别的工作就不算空转。
func spinsAfterFailure(calls []ToolCall, results []Message, refusal string) bool {
	if len(calls) == 0 {
		return false
	}
	refusal = strings.TrimSpace(refusal)
	spin := false
	for i, c := range calls {
		switch c.Name {
		case "wait":
			spin = true
		case "shell", "browser_verify":
			if refusal == "" || i >= len(results) || strings.TrimSpace(results[i].Content) != refusal {
				return false
			}
			spin = true
		default:
			return false
		}
	}
	return spin
}

// batchRerunsScript 在某个 shell 调用会执行 script 时返回 true。命令先从 JSON 的 command 字段取。
func batchRerunsScript(calls []ToolCall, script string) bool {
	for _, c := range calls {
		if commandRerunsScript(shellCommand(c), script) {
			return true
		}
	}
	return false
}

// shellCommand 返回 shell 调用里的命令文本。不是 shell，或参数里没有 command 时返回空串。
func shellCommand(tc ToolCall) string {
	if tc.Name != "shell" {
		return ""
	}
	if m := parseToolArgs(tc.Arguments); len(m) > 0 {
		if s, ok := m["command"].(string); ok {
			return s
		}
	}
	return ""
}

// commandRerunsScript 在任一命令段会执行 script 时返回 true。删除、移动、阅读和列目录返回 false。
func commandRerunsScript(command, script string) bool {
	if strings.TrimSpace(command) == "" || strings.TrimSpace(script) == "" {
		return false
	}
	for _, seg := range commandSegments(command) {
		if segmentRerunsScript(seg, script) {
			return true
		}
	}
	return false
}

// commandSegments 按换行、分号、&&、||、管道和单独的 & 拆开命令，引号里的分隔符保留。
func commandSegments(command string) []string {
	var segs []string
	var b strings.Builder
	quote := byte(0)
	flush := func() {
		s := strings.TrimSpace(b.String())
		b.Reset()
		if s != "" {
			segs = append(segs, s)
		}
	}
	for i := 0; i < len(command); i++ {
		c := command[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			b.WriteByte(c)
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
			b.WriteByte(c)
		case '\n', '\r', ';':
			flush()
		case '&':
			if i+1 < len(command) && command[i+1] == '&' {
				flush()
				i++
				continue
			}
			flush()
		case '|':
			flush()
			if i+1 < len(command) && command[i+1] == '|' {
				i++
			}
		default:
			b.WriteByte(c)
		}
	}
	flush()
	return segs
}

// segmentRerunsScript 判断这一段会不会把 script 当程序执行。解释器的 --check 和 -e 不算执行。
func segmentRerunsScript(segment, script string) bool {
	toks := commandTokens(segment)
	i := 0
	for i < len(toks) && isEnvAssign(toks[i]) {
		i++
	}
	for i < len(toks) && isLauncher(exeBase(toks[i])) {
		i++
		for i < len(toks) && isSwitch(toks[i]) {
			i++
		}
	}
	if i >= len(toks) {
		return false
	}
	if strings.ContainsAny(toks[i], " \t") {
		return commandRerunsScript(toks[i], script)
	}
	if scriptToken(toks[i], script) {
		return true
	}
	if !isRunner(exeBase(toks[i])) {
		return false
	}
	return runnerExecutesScript(toks[i:], script)
}

// commandTokens 按空白拆开，引号里的空白留在同一个词里。引号本身不进入词。
func commandTokens(s string) []string {
	var out []string
	var b strings.Builder
	quote := byte(0)
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
				continue
			}
			b.WriteByte(c)
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case ' ', '\t', '\r', '\n':
			flush()
		default:
			b.WriteByte(c)
		}
	}
	flush()
	return out
}

// runnerExecutesScript 在解释器会运行 script 文件时返回 true。语法检查和行内代码返回 false。
func runnerExecutesScript(toks []string, script string) bool {
	if len(toks) == 0 {
		return false
	}
	base := exeBase(toks[0])
	if denoInspectOnly(base, toks[1:]) {
		return false
	}
	sawScript := false
	for _, arg := range toks[1:] {
		if isInlineFlag(base, arg) || isCheckFlag(base, arg) {
			return false
		}
		if scriptToken(arg, script) {
			sawScript = true
		}
	}
	return sawScript
}

// denoInspectOnly 在 deno 的第一个子命令是 check、lint、fmt、doc 或 info 时返回 true。
func denoInspectOnly(base string, args []string) bool {
	if base != "deno" {
		return false
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") || isEnvAssign(a) {
			continue
		}
		switch a {
		case "check", "lint", "fmt", "doc", "info":
			return true
		default:
			return false
		}
	}
	return false
}

// isInlineFlag 判断这个参数会让解释器执行行内代码，而不是脚本文件。
func isInlineFlag(base, arg string) bool {
	switch base {
	case "node", "nodejs", "bun", "deno":
		return arg == "-e" || arg == "--eval" || arg == "-p" || arg == "--print" || strings.HasPrefix(arg, "--eval=") || strings.HasPrefix(arg, "--print=")
	case "python", "python3", "py":
		return arg == "-c" || (strings.HasPrefix(arg, "-c") && len(arg) > 2 && arg[2] != '-')
	default:
		return false
	}
}

// isCheckFlag 判断这个参数只做语法检查。node 的 -c 是语法检查，python 的 -c 不是。
func isCheckFlag(base, arg string) bool {
	switch arg {
	case "--check", "--syntax-check", "--check-syntax":
		return base == "node" || base == "nodejs" || base == "bun" || base == "deno"
	case "-c":
		return base == "node" || base == "nodejs"
	default:
		return false
	}
}

// scriptToken 在 tok 去掉引号和 ./ 之后等于 script，或以 /script 结尾时返回 true。
func scriptToken(tok, script string) bool {
	tok = cleanPathToken(tok)
	script = cleanPathToken(script)
	if tok == "" || script == "" {
		return false
	}
	return tok == script || strings.HasSuffix(tok, "/"+script)
}

// cleanPathToken 归一化斜杠和大小写，并去掉包着的引号和开头的 ./。
func cleanPathToken(p string) string {
	p = strings.Trim(normSlash(p), `"'`)
	for strings.HasPrefix(p, "./") {
		p = strings.TrimPrefix(p, "./")
	}
	return p
}

// isRunner 判断是不是会执行脚本文件的解释器。
func isRunner(base string) bool {
	switch base {
	case "node", "nodejs", "bun", "deno", "tsx", "ts-node", "python", "python3", "py":
		return true
	default:
		return false
	}
}

// isLauncher 判断是不是后面还会跟上真正命令的外壳。
func isLauncher(base string) bool {
	switch base {
	case "cmd", "powershell", "pwsh", "sh", "bash", "env", "npx", "pnpm", "yarn", "call", "start":
		return true
	default:
		return false
	}
}

// isEnvAssign 判断是不是命令前的 KEY=VAL。开关不是赋值。
func isEnvAssign(tok string) bool {
	if strings.HasPrefix(tok, "-") {
		return false
	}
	i := strings.IndexByte(tok, '=')
	return i > 0
}

// isSwitch 判断是不是外壳开关。cmd 的 /c 算，路径不算。
func isSwitch(tok string) bool {
	if strings.HasPrefix(tok, "-") {
		return true
	}
	return len(tok) == 2 && (tok[0] == '/' || tok[0] == '\\')
}

// seedCheckWatch 重放这一轮已有的历史，使继续下来的回合直接拒绝上一份相同报告的脚本，而不是再跑到停止卡片。重放之后把 harnessHits、idle、bypassHits 清零，这三项只计本轮。相同观察已经达到催促次数时补发一次 observationNudge。
// req 在相同观察已达到催促次数时传给 observationNudge。
// w 为 nil 时返回，否则重放后把 harnessHits、idle、bypassHits 清零。
// messages 按助手工具调用和紧随的工具结果交给 note。
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

// extractWritePaths 从 write_file、str_replace、edit_file、create_file 和 apply_patch 的参数里取出路径。其他工具返回 nil。补丁先看 *** Update/Add/Delete File 行。
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

// patchFilePaths 收集补丁里 Update、Add、Delete 文件行上的路径。参数是 JSON 且带 patch 字段时先用该字段。
// args 先交给 parseToolArgs，有非空 patch 字段时改用该字段。
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

// writesHarness 在任一写入路径是验证器路径时返回 true。
// calls 的写入路径里有验证器路径时返回 true。
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

// writesProduct 在任一写入路径是产品文件时返回 true。验证器路径不算产品。
// calls 的写入路径里有产品文件时返回 true。
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

// inspectsOnly 在调用非空且全部是 read_file、list_dir、grep、glob、search_files、read_artifact 时返回 true。空批返回 false。
// calls 长度为 0 时返回 false，否则要求每个 Name 都是这些查阅工具。
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

// isHarnessPath 判断路径是不是验证器。含 /qa/、以 qa/ 开头、落在 __tests__ 或 tests 里，文件名含 verify 或 whitebox，或检查用的 .mjs 文件名含 check、spec、test，都返回 true。
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

// isProductPath 在路径不是验证器，且后缀是 js、mjs、cjs、html、css、ts、tsx、jsx 之一时返回 true。
// p 先交给 isHarnessPath，是验证器则返回 false，否则按这些后缀判断。
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

// normSlash 把反斜杠换成斜杠，转成小写，并去掉开头的 ./。
// p 按上述规则变换后返回。
func normSlash(p string) string {
	return strings.TrimPrefix(strings.ToLower(strings.ReplaceAll(p, "\\", "/")), "./")
}

// skillOverlay 在聊天覆盖层打开、条件为真、技能尚未加载且正文非空时，返回带 Skill 标题的正文。否则返回空串。
// t 为 nil、ChatOverlay 为假或 Skills 为 nil 时返回空串。
// name 为空、已在 Loaded 里，或对应正文去掉空白后为空时返回空串，否则写进标题。
// cond 为假时返回空串。
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
