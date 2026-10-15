package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseVerifyPlanStripsAgentStatus(t *testing.T) {
	plan, err := parseVerifyPlan(map[string]any{
		"open":   "build/app/index.html",
		"status": "PASS",
		"checks": map[string]any{"launch": "PASS"},
		"steps": []any{
			map[string]any{"op": "screenshot", "path": "qa/evidence/r.png", "check": "render"},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "" || plan.Checks != nil {
		t.Fatalf("agent verdict leaked: %+v", plan)
	}
	if plan.Open != "build/app/index.html" || len(plan.Steps) != 1 {
		t.Fatalf("%+v", plan)
	}
}

func TestDeriveVerificationPassAndFail(t *testing.T) {
	plan := VerifyPlan{Open: "build/app/index.html", Terminal: "designed-outcome", Restart: "initial-state"}
	ok := []browserStepResult{
		{Op: "open", Check: "launch", OK: true, Detail: "opened"},
		{Op: "screenshot", Check: "render", OK: true, Detail: "shot", Visual: "qa/evidence/render.png"},
		{Op: "click", Check: "input", OK: true, Detail: "clicked #opt"},
		{Op: "click", Check: "coreLoop", OK: true, Detail: "clicked #hot"},
		{Op: "click", Check: "outcome", OK: true, Detail: "clicked #summon"},
		{Op: "click", Check: "restart", OK: true, Detail: "clicked #restart"},
	}
	_, verify := deriveVerification("game-x", plan, ok, true, true)
	if verify["status"] != "PASS" {
		t.Fatalf("%v", verify)
	}
	checks, _ := verify["checks"].(map[string]string)
	for _, name := range qaCheckNames {
		if checks[name] != "PASS" {
			t.Fatalf("%s=%s", name, checks[name])
		}
	}

	fail := append([]browserStepResult{}, ok...)
	fail[2].OK = false
	fail[2].Detail = "covered"
	_, verify = deriveVerification("game-x", plan, fail, true, true)
	if verify["status"] != "FAIL" {
		t.Fatalf("want FAIL got %v", verify["status"])
	}
	checks, _ = verify["checks"].(map[string]string)
	if checks["input"] != "FAIL" {
		t.Fatalf("input=%s", checks["input"])
	}
}

func TestDeriveVerificationNoChromeIsNotRun(t *testing.T) {
	plan := VerifyPlan{Open: "build/app/index.html", Terminal: "designed-outcome", Restart: "initial-state"}
	_, verify := deriveVerification("", plan, nil, false, true)
	if verify["status"] != "NOT_RUN" {
		t.Fatalf("%v", verify)
	}
}

func TestWriteQAFilesAtomic(t *testing.T) {
	dir := t.TempDir()
	runDoc := map[string]any{"schemaVersion": 1, "runId": "main-path"}
	verifyDoc := map[string]any{"schemaVersion": 3, "status": "FAIL"}
	if err := writeQAFiles(dir, runDoc, verifyDoc); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "qa", "verification.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "FAIL" {
		t.Fatalf("%s", raw)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "qa", "evidence", "run.json")); err != nil {
		t.Fatal(err)
	}
}

func TestInferVerifyRoot(t *testing.T) {
	got := inferVerifyRoot("game-adaptations/xiyou/build/app/index.html")
	if !strings.HasSuffix(strings.ReplaceAll(got, "\\", "/"), "game-adaptations/xiyou") {
		t.Fatalf("%q", got)
	}
}

func TestParseVerifyPlanDropsCSSRoot(t *testing.T) {
	plan, err := parseVerifyPlan(map[string]any{
		"open":    "game-adaptations/tianming-guiling/build/app/index.html",
		"root":    "#stage",
		"restart": "#insightBtn",
		"steps": []any{
			map[string]any{"op": "wait", "ms": 200, "check": "launch"},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Root != "" {
		t.Fatalf("css root leaked: %q", plan.Root)
	}
	if plan.Restart != "initial-state" {
		t.Fatalf("css restart marker: %q", plan.Restart)
	}
}

func TestNormalizeVerifyRootIgnoresSelector(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	got := normalizeVerifyRoot(tools, "#stage", "game-adaptations/tianming-guiling/build/app/index.html")
	if got != "game-adaptations/tianming-guiling" {
		t.Fatalf("%q", got)
	}
}

func TestFormatVerifyResultNamesWrittenPaths(t *testing.T) {
	got := formatVerifyResult("FAIL", map[string]any{
		"checks": map[string]string{"launch": "PASS", "render": "PASS", "input": "PASS", "coreLoop": "PASS", "outcome": "PASS", "restart": "PASS"},
	}, []browserStepResult{{Index: 31, Op: "click", OK: false, Detail: "covered"}}, "game-x/qa/evidence/run.json", "game-x/qa/verification.json")
	if !strings.Contains(got, "game-x/qa/verification.json") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "six checks PASS but a later step failed") {
		t.Fatal(got)
	}
}

func TestLooksLikeCSSSelector(t *testing.T) {
	if !looksLikeCSSSelector("#stage") || !looksLikeCSSSelector(".hud") || !looksLikeCSSSelector("[data-id]") {
		t.Fatal("selectors")
	}
	if looksLikeCSSSelector("game-adaptations/tianming-guiling") || looksLikeCSSSelector("qa") {
		t.Fatal("paths")
	}
}

func TestBrowserVerifyRequiresPlan(t *testing.T) {
	tools := &WorkspaceTools{Workspace: t.TempDir()}
	res := tools.Call("browser_verify", `{}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "need path") {
		t.Fatalf("%+v", res)
	}
}

func TestBrowserVerifyCSSRootWritesUnderProject(t *testing.T) {
	dir := t.TempDir()
	html := "game-adaptations/tianming-guiling/build/app/index.html"
	if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(filepath.Dir(html))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(html)), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := (&WorkspaceTools{Workspace: dir}).Call("browser_verify", `{"open":"`+html+`","root":"#stage","steps":[{"op":"wait","ms":1,"check":"launch"}]}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	want := filepath.Join(dir, "game-adaptations", "tianming-guiling", "qa", "verification.json")
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "#stage")); !os.IsNotExist(err) {
		t.Fatal("must not create #stage")
	}
	if !strings.Contains(res.Content, `ignored root "#stage"`) || !strings.Contains(res.Content, "game-adaptations/tianming-guiling/qa/verification.json") {
		t.Fatal(res.Content)
	}
}
