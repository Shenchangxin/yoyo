// 在隔离的 Chrome 页面上按步骤执行点击、输入和断言。
// 模型只给选择器，不直接说话给 CDP。不启动浏览器进程。browser_verify 复用这里的步骤解析和执行。

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

const (
	// maxBrowserActSteps 是 browser_act 一次最多接受的步数。
	maxBrowserActSteps = 40
	// maxBrowserWaitMS 是普通等待的上限，毫秒。
	maxBrowserWaitMS = 5000
	// maxBrowserCondWaitMS 是带选择器、文本或 URL 条件的等待上限，毫秒。
	maxBrowserCondWaitMS = 15000
)

// BrowserStep 是隔离 Chrome 页面上的一步宿主动作。模型给出选择器，不直接对 CDP 说话。
type BrowserStep struct {
	Op        string `json:"op"`
	Selector  string `json:"selector,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Text      string `json:"text,omitempty"`
	Value     string `json:"value,omitempty"`
	Path      string `json:"path,omitempty"`
	Contains  string `json:"contains,omitempty"`
	URL       string `json:"url,omitempty"`
	Check     string `json:"check,omitempty"`
	Direction string `json:"direction,omitempty"`
	Key       string `json:"key,omitempty"`
	Level     string `json:"level,omitempty"`
	Tab       string `json:"tab,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	MS        int    `json:"ms,omitempty"`
}

// browserStepResult 是一步的执行结果。Visual 在截图成功时是图片路径。
type browserStepResult struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Check  string `json:"check,omitempty"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Visual string `json:"visual,omitempty"`
	Err    string `json:"error,omitempty"`
}

// parseBrowserSteps 按 maxBrowserActSteps 解析步骤数组。
func parseBrowserSteps(v any) ([]BrowserStep, error) {
	return parseBrowserStepsN(v, maxBrowserActSteps)
}

// parseBrowserStepsN 把数组或 JSON 字符串解析成步骤。空数组失败。超过 max 失败。max 小于等于 0 时用 maxBrowserActSteps。
func parseBrowserStepsN(v any, max int) ([]BrowserStep, error) {
	raw, err := coerceJSONArray(v)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("steps must be a non-empty array")
	}
	if max <= 0 {
		max = maxBrowserActSteps
	}
	if len(raw) > max {
		return nil, fmt.Errorf("%d steps exceeds the %d-step cap; split the path or drop redundant clicks", len(raw), max)
	}
	out := make([]BrowserStep, 0, len(raw))
	for i, item := range raw {
		st, err := parseBrowserStep(item)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		out = append(out, st)
	}
	return out, nil
}

// parseBrowserStep 规范化一步。ref 优先于 selector，并写成 ref=。wait 未给毫秒时，有条件用 10000，否则用 200。毫秒超过对应上限则截断。
func parseBrowserStep(v any) (BrowserStep, error) {
	var st BrowserStep
	switch t := v.(type) {
	case BrowserStep:
		st = t
	case map[string]any:
		b, err := json.Marshal(t)
		if err != nil {
			return st, err
		}
		if err := json.Unmarshal(b, &st); err != nil {
			return st, err
		}
		if st.MS == 0 {
			st.MS = intArg(t["wait_ms"])
		}
		if st.Contains == "" {
			st.Contains = str(t["text_contains"])
		}
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return st, fmt.Errorf("step must be an object")
		}
		if err := json.Unmarshal(b, &st); err != nil {
			return st, fmt.Errorf("step must be an object")
		}
	}
	st.Op = normalizeBrowserOp(st.Op)
	st.Selector = stepHandle(st.Selector, st.Ref)
	st.Path = strings.TrimSpace(st.Path)
	st.Text = st.Text
	st.Value = strings.TrimSpace(st.Value)
	st.Contains = strings.TrimSpace(st.Contains)
	st.URL = strings.TrimSpace(st.URL)
	st.Check = strings.TrimSpace(st.Check)
	st.Direction = strings.TrimSpace(st.Direction)
	st.Key = strings.TrimSpace(st.Key)
	st.Level = strings.TrimSpace(st.Level)
	st.Tab = strings.TrimSpace(st.Tab)
	if st.Op == "" {
		return st, fmt.Errorf("missing op")
	}
	switch st.Op {
	case "click", "hover":
		if st.Selector == "" {
			return st, fmt.Errorf("%s needs selector or ref", st.Op)
		}
	case "assert":
		if st.Selector == "" && st.Contains == "" {
			return st, fmt.Errorf("%s needs selector or contains", st.Op)
		}
	case "type", "fill":
		if st.Text == "" && st.Value == "" {
			return st, fmt.Errorf("%s needs text", st.Op)
		}
		if st.Text == "" {
			st.Text = st.Value
		}
	case "select":
		if st.Selector == "" || (st.Value == "" && st.Text == "") {
			return st, fmt.Errorf("select needs selector/ref and value")
		}
		if st.Value == "" {
			st.Value = st.Text
		}
	case "press":
		if st.Key == "" && st.Text == "" {
			return st, fmt.Errorf("press needs key")
		}
		if st.Key == "" {
			st.Key = st.Text
		}
	case "screenshot":
		// 路径可以空着，browserScreenshot 会填默认路径。
	case "open":
		if st.Path == "" && st.Selector == "" && st.URL == "" {
			return st, fmt.Errorf("open needs path")
		}
		if st.Path == "" {
			st.Path = firstNonEmptyStep(st.URL, st.Selector)
		}
	case "wait":
		if st.MS <= 0 {
			if st.Selector != "" || st.Contains != "" || st.URL != "" {
				st.MS = 10000
			} else {
				st.MS = 200
			}
		}
	case "scroll":
		if st.Direction == "" {
			st.Direction = "down"
		}
	case "snapshot", "tab":
	default:
		return st, fmt.Errorf("unknown op %q (click, type, fill, wait, screenshot, assert, open, snapshot, scroll, hover, select, press, tab)", st.Op)
	}
	capMS := maxBrowserWaitMS
	if st.Op == "wait" && (st.Selector != "" || st.Contains != "" || st.URL != "") {
		capMS = maxBrowserCondWaitMS
	}
	if st.MS > capMS {
		st.MS = capMS
	}
	return st, nil
}

// normalizeBrowserOp 把别名收成规范操作名。未知名字原样返回，由调用方拒绝。
func normalizeBrowserOp(op string) string {
	op = strings.ToLower(strings.TrimSpace(op))
	switch op {
	case "click", "tap":
		return "click"
	case "type", "keydown":
		return "type"
	case "fill":
		return "fill"
	case "press":
		return "press"
	case "wait", "wait_ms", "sleep":
		return "wait"
	case "screenshot", "shot":
		return "screenshot"
	case "assert", "assert_text", "visible":
		return "assert"
	case "open", "navigate":
		return "open"
	case "snapshot", "refresh":
		return "snapshot"
	case "scroll":
		return "scroll"
	case "hover":
		return "hover"
	case "select":
		return "select"
	case "tab", "switch_tab":
		return "tab"
	default:
		return op
	}
}

// stepHandle 在 ref 非空时返回 ref= 句柄，已带 ref= 或 ref: 则不重复加。否则返回 selector。
func stepHandle(selector, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		if strings.HasPrefix(strings.ToLower(ref), "ref=") || strings.HasPrefix(strings.ToLower(ref), "ref:") {
			return ref
		}
		return "ref=" + ref
	}
	return strings.TrimSpace(selector)
}

// firstNonEmptyStep 返回第一段非空白字符串。
func firstNonEmptyStep(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// coerceJSONArray 把 []any、[]BrowserStep 或 JSON 数组字符串收成 []any。
func coerceJSONArray(v any) ([]any, error) {
	switch t := v.(type) {
	case nil:
		return nil, fmt.Errorf("missing steps")
	case []any:
		return t, nil
	case []BrowserStep:
		out := make([]any, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil, fmt.Errorf("missing steps")
		}
		var out []any
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return nil, fmt.Errorf("steps is not a JSON array")
		}
		return out, nil
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return nil, fmt.Errorf("steps is not a JSON array")
		}
		var out []any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("steps is not a JSON array")
		}
		return out, nil
	}
}

// browserAct 解析 steps 并在第一次失败时停下。没有隔离浏览器时返回错误。
func (t *WorkspaceTools) browserAct(args map[string]any) ToolResult {
	steps, err := parseBrowserSteps(args["steps"])
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.checkBrowserSeq("browser_act"); err != nil {
		return ToolResult{Err: err}
	}
	results, stopped := t.runBrowserSteps(steps, true)
	return ToolResult{Content: formatBrowserAct(results, stopped)}
}

// checkBrowserSeq 要求已有隔离浏览器，通过 Browser 能力检查，并 EnsurePreview。
func (t *WorkspaceTools) checkBrowserSeq(action string) error {
	if t == nil || t.Browser == nil {
		return fmt.Errorf("no isolated browser")
	}
	if err := t.check(capability.Browser, action, "", ""); err != nil {
		return err
	}
	return t.Browser.EnsurePreview()
}

// runBrowserSteps 按顺序执行。stopOnFail 为真时在第一步失败后返回，stopped 为真。
func (t *WorkspaceTools) runBrowserSteps(steps []BrowserStep, stopOnFail bool) (out []browserStepResult, stopped bool) {
	out = make([]browserStepResult, 0, len(steps))
	for i, st := range steps {
		res := t.runBrowserStep(i+1, st)
		out = append(out, res)
		if !res.OK && stopOnFail {
			return out, true
		}
	}
	return out, false
}

// runBrowserStep 执行一步。无条件的 wait 在本进程 Sleep。带条件的 wait 交给浏览器。截图路径为空时，结果里的 Visual 写成 .yoyo/captures/browser.png。
func (t *WorkspaceTools) runBrowserStep(index int, st BrowserStep) browserStepResult {
	res := browserStepResult{Index: index, Op: st.Op, Check: st.Check}
	if t == nil || t.Browser == nil {
		res.Err = "no isolated browser"
		res.Detail = res.Err
		return res
	}
	switch st.Op {
	case "open":
		got := t.browserOpen(st.Path)
		if got.Err != nil {
			res.Err = got.Err.Error()
			res.Detail = res.Err
			return res
		}
		res.OK = true
		res.Detail = got.Content
		return res
	case "click":
		got := t.browserClick(st.Selector)
		return finishBrowserStep(res, got)
	case "type":
		got := t.browserType(st.Selector, st.Text)
		return finishBrowserStep(res, got)
	case "fill":
		got := t.browserFill(st.Selector, st.Text)
		return finishBrowserStep(res, got)
	case "screenshot":
		got := t.browserScreenshot(st.Path)
		res = finishBrowserStep(res, got)
		if res.OK {
			path := strings.TrimSpace(st.Path)
			if path == "" {
				path = ".yoyo/captures/browser.png"
			}
			res.Visual = strings.ReplaceAll(path, "\\", "/")
		}
		return res
	case "snapshot":
		got := t.browserSnapshot(st.Level)
		return finishBrowserStep(res, got)
	case "wait":
		ms := st.MS
		if ms <= 0 {
			ms = 200
		}
		if st.Selector == "" && st.Contains == "" && st.URL == "" {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			res.OK = true
			res.Detail = fmt.Sprintf("waited %d ms", ms)
			return res
		}
		obs, err := t.Browser.WaitCondition(st.Selector, st.Contains, st.URL, time.Duration(ms)*time.Millisecond)
		if err != nil {
			res.Err = err.Error()
			res.Detail = res.Err
			return res
		}
		res.OK = true
		res.Detail = obs
		return res
	case "scroll":
		got := t.browserScroll(st.Direction, st.Amount, st.Selector)
		return finishBrowserStep(res, got)
	case "hover":
		got := t.browserHover(st.Selector)
		return finishBrowserStep(res, got)
	case "select":
		got := t.browserSelect(st.Selector, st.Value)
		return finishBrowserStep(res, got)
	case "press":
		got := t.browserPress(st.Key)
		return finishBrowserStep(res, got)
	case "tab":
		got := t.browserTabs("switch", firstNonEmptyStep(st.Tab, st.Path, st.Selector), "")
		return finishBrowserStep(res, got)
	case "assert":
		if st.Selector != "" {
			obs, err := t.Browser.WaitVisible(st.Selector)
			if err != nil {
				res.Err = err.Error()
				res.Detail = res.Err
				return res
			}
			res.Detail = obs
		}
		if st.Contains != "" {
			snap, live := t.Browser.Refresh()
			if !live {
				res.Err = "isolated Chrome is not running"
				res.Detail = res.Err
				return res
			}
			if !strings.Contains(snap.Text, st.Contains) {
				res.Err = fmt.Sprintf("page text does not contain %q", truncRunes(st.Contains, 80))
				res.Detail = res.Err
				return res
			}
			if res.Detail != "" {
				res.Detail += "; "
			}
			res.Detail += fmt.Sprintf("page contains %q", truncRunes(st.Contains, 80))
		}
		res.OK = true
		return res
	default:
		res.Err = fmt.Sprintf("unknown op %q", st.Op)
		res.Detail = res.Err
		return res
	}
}

// finishBrowserStep 把工具结果抄进步骤结果。有错误则 OK 保持假。
func finishBrowserStep(res browserStepResult, got ToolResult) browserStepResult {
	if got.Err != nil {
		res.Err = got.Err.Error()
		res.Detail = res.Err
		return res
	}
	res.OK = true
	res.Detail = got.Content
	return res
}

// formatBrowserAct 逐行列出步骤，并在中途停下时写明停在第一次失败。正文提醒不要自起静态服务或连接 CDP 9333/9222。
func formatBrowserAct(results []browserStepResult, stopped bool) string {
	var b strings.Builder
	ok := 0
	for _, r := range results {
		if r.OK {
			ok++
		}
		status := "ok"
		if !r.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "step %d %s %s: %s", r.Index, r.Op, status, firstLine(r.Detail))
		if r.Check != "" {
			fmt.Fprintf(&b, " [check=%s]", r.Check)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "completed %d/%d", ok, len(results))
	if stopped && ok < len(results) {
		b.WriteString("; stopped on first failure")
	}
	b.WriteByte('\n')
	b.WriteString("Do not start python -m http.server, spawn Chrome, or connect to CDP 9333/9222.")
	return strings.TrimSpace(b.String())
}

// firstLine 取第一行并截到 240 个符文。
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return truncRunes(s, 240)
}

// truncRunes 按符文截断，超出时加上省略号。n 小于等于 0 返回空串。
func truncRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// isQACheck 判断名字是不是 launch、render、input、coreLoop、outcome 或 restart。
func isQACheck(name string) bool {
	switch name {
	case "launch", "render", "input", "coreLoop", "outcome", "restart":
		return true
	default:
		return false
	}
}
