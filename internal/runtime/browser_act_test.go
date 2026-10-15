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
