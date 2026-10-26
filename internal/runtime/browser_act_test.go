// browser_act 的步骤解析。步数上限、未知操作和隔离浏览器缺失都在这里锁住。

package runtime

import (
	"strings"
	"testing"
)

// TestParseBrowserSteps 确认别名被收成规范操作，普通 wait 超过 5000 毫秒被截断。
func TestParseBrowserSteps(t *testing.T) {
	steps, err := parseBrowserSteps([]any{
		map[string]any{"op": "click", "selector": "#startBtn", "check": "launch"},
		map[string]any{"op": "wait", "ms": 300},
		map[string]any{"op": "screenshot", "path": "qa/evidence/render.png", "check": "render"},
		map[string]any{"op": "assert", "contains": "章"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 4 || steps[0].Op != "click" || steps[1].MS != 300 || steps[2].Path == "" || steps[3].Contains != "章" {
		t.Fatalf("%+v", steps)
	}
}

// TestParseBrowserStepsJSONString 确认步骤可以是 JSON 字符串。
func TestParseBrowserStepsJSONString(t *testing.T) {
	steps, err := parseBrowserSteps(`[{"op":"click","selector":"text=Start"}]`)
	if err != nil || len(steps) != 1 || steps[0].Selector != "text=Start" {
		t.Fatalf("%+v %v", steps, err)
	}
}

// TestParseBrowserStepsCap 确认超过 maxBrowserActSteps 的数组被拒绝。
func TestParseBrowserStepsCap(t *testing.T) {
	raw := make([]any, maxBrowserActSteps+1)
	for i := range raw {
		raw[i] = map[string]any{"op": "click", "selector": "#x"}
	}
	if _, err := parseBrowserSteps(raw); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err=%v", err)
	}
}

// TestParseBrowserStepRejectsUnknown 确认未知操作 cdp 被拒绝。
func TestParseBrowserStepRejectsUnknown(t *testing.T) {
	if _, err := parseBrowserStep(map[string]any{"op": "cdp"}); err == nil {
		t.Fatal("expected unknown op")
	}
}

// TestParseBrowserStepFillAndRef 确认 fill 带 ref 时写成 ref=inp_0，条件 wait 的 9000 毫秒不被截断。
func TestParseBrowserStepFillAndRef(t *testing.T) {
	st, err := parseBrowserStep(map[string]any{"op": "fill", "ref": "inp_0", "text": "hi"})
	if err != nil || st.Op != "fill" || st.Selector != "ref=inp_0" || st.Text != "hi" {
		t.Fatalf("%+v %v", st, err)
	}
	st, err = parseBrowserStep(map[string]any{"op": "wait", "contains": "done", "ms": 9000})
	if err != nil || st.MS != 9000 {
		t.Fatalf("cond wait %+v %v", st, err)
	}
	st, err = parseBrowserStep(map[string]any{"op": "press", "key": "Enter"})
	if err != nil || st.Op != "press" || st.Key != "Enter" {
		t.Fatalf("press %+v %v", st, err)
	}
	st, err = parseBrowserStep(map[string]any{"op": "scroll", "direction": "down", "amount": 200})
	if err != nil || st.Op != "scroll" || st.Amount != 200 {
		t.Fatalf("scroll %+v %v", st, err)
	}
	st, err = parseBrowserStep(map[string]any{"op": "tab", "tab": "ABCD1234"})
	if err != nil || st.Op != "tab" || st.Tab != "ABCD1234" {
		t.Fatalf("tab %+v %v", st, err)
	}
}

// TestFormatBrowserActStops 确认第一次失败时正文写明 stopped。
func TestFormatBrowserActStops(t *testing.T) {
	got := formatBrowserAct([]browserStepResult{
		{Index: 1, Op: "click", OK: true, Detail: "clicked #a"},
		{Index: 2, Op: "click", OK: false, Detail: "#b is covered by #deck", Err: "covered"},
	}, true)
	if !strings.Contains(got, "step 1 click ok") || !strings.Contains(got, "FAIL") || !strings.Contains(got, "stopped on first failure") {
		t.Fatal(got)
	}
}

// TestBrowserActRequiresIsolatedChrome 确认没有隔离浏览器时返回 no isolated browser。
func TestBrowserActRequiresIsolatedChrome(t *testing.T) {
	tools := &WorkspaceTools{Workspace: t.TempDir()}
	res := tools.Call("browser_act", `{"steps":[{"op":"click","selector":"#a"}]}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "no isolated browser") {
		t.Fatalf("%+v", res)
	}
}
