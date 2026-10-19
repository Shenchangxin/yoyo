package runtime

import (
	"strings"
	"testing"
)

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

func TestParseBrowserStepsJSONString(t *testing.T) {
	steps, err := parseBrowserSteps(`[{"op":"click","selector":"text=Start"}]`)
	if err != nil || len(steps) != 1 || steps[0].Selector != "text=Start" {
		t.Fatalf("%+v %v", steps, err)
	}
}

func TestParseBrowserStepsCap(t *testing.T) {
	raw := make([]any, maxBrowserActSteps+1)
	for i := range raw {
		raw[i] = map[string]any{"op": "click", "selector": "#x"}
	}
	if _, err := parseBrowserSteps(raw); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseBrowserStepRejectsUnknown(t *testing.T) {
	if _, err := parseBrowserStep(map[string]any{"op": "cdp"}); err == nil {
		t.Fatal("expected unknown op")
	}
}

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

func TestFormatBrowserActStops(t *testing.T) {
	got := formatBrowserAct([]browserStepResult{
		{Index: 1, Op: "click", OK: true, Detail: "clicked #a"},
		{Index: 2, Op: "click", OK: false, Detail: "#b is covered by #deck", Err: "covered"},
	}, true)
	if !strings.Contains(got, "step 1 click ok") || !strings.Contains(got, "FAIL") || !strings.Contains(got, "stopped on first failure") {
		t.Fatal(got)
	}
}

func TestBrowserActRequiresIsolatedChrome(t *testing.T) {
	tools := &WorkspaceTools{Workspace: t.TempDir()}
	res := tools.Call("browser_act", `{"steps":[{"op":"click","selector":"#a"}]}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "no isolated browser") {
		t.Fatalf("%+v", res)
	}
}
