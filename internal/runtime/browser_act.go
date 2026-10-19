package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

const (
	maxBrowserActSteps   = 40
	maxBrowserWaitMS     = 5000
	maxBrowserCondWaitMS = 15000
)

// BrowserStep is one host-owned action on the isolated Chrome page.
// The model names selectors; it never talks to CDP.
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

type browserStepResult struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Check  string `json:"check,omitempty"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Visual string `json:"visual,omitempty"`
	Err    string `json:"error,omitempty"`
}

func parseBrowserSteps(v any) ([]BrowserStep, error) {
	return parseBrowserStepsN(v, maxBrowserActSteps)
}

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
		// path may be empty; browserScreenshot fills a default
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

func firstNonEmptyStep(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

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

func (t *WorkspaceTools) checkBrowserSeq(action string) error {
	if t == nil || t.Browser == nil {
		return fmt.Errorf("no isolated browser")
	}
	if err := t.check(capability.Browser, action, "", ""); err != nil {
		return err
	}
	return t.Browser.EnsurePreview()
}

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

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return truncRunes(s, 240)
}

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

func isQACheck(name string) bool {
	switch name {
	case "launch", "render", "input", "coreLoop", "outcome", "restart":
		return true
	default:
		return false
	}
}
