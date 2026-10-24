package runtime

import (
	"strings"
	"testing"
)

const verifyBanner = `ERROR: exit status 1

=== 权威验证（同一次完整运行）===
PASS  launch    启动 — 规则器已加载
FAIL  render    真实渲染 — 渲染输出为空或缺少首屏内容（0 字符）。
PASS  input     输入
PASS  coreLoop  核心循环
PASS  outcome   设计结果
PASS  restart   重开

结果：5 PASS / 1 FAIL`

func TestFailCheckIDsFromVerifyBanner(t *testing.T) {
	ids := failCheckIDs(verifyBanner)
	if len(ids) != 1 || ids[0] != "render" {
		t.Fatalf("%v", ids)
	}
	if got := errorSignature(verifyBanner); got != "check fail:render" {
		t.Fatalf("sig %q", got)
	}
}

func TestErrorSignatureDoesNotGroupByPassBanner(t *testing.T) {
	// The old 80-char ERROR: prefix started at "exit status 1" / PASS launch,
	// so four identical FAIL render shells never grouped.
	sig := errorSignature(verifyBanner)
	if strings.Contains(sig, "PASS") || strings.Contains(sig, "exit status") {
		t.Fatalf("grouped by banner %q", sig)
	}
}

func TestFailCheckIDsIgnoresSectionHeading(t *testing.T) {
	first := "{\n  \"status\": \"FAIL\",\n  \"checks\": {\n    \"render\": \"FAIL\",\n    \"outcome\": \"FAIL\"\n  }\n}\nFAIL 明细:\nrender: missing png\n"
	second := "{\n  \"status\": \"FAIL\",\n  \"checks\": {\n    \"render\": \"PASS\",\n    \"outcome\": \"FAIL\"\n  }\n}\nFAIL 明细:\n"
	if got := strings.Join(failCheckIDs(first), ","); got != "render,outcome" {
		t.Fatalf("first %q", got)
	}
	if got := strings.Join(failCheckIDs(second), ","); got != "outcome" {
		t.Fatalf("second %q", got)
	}
	var w checkWatch
	w.note(nil, []Message{{Content: first}})
	for i := 0; i < 10; i++ {
		w.note([]ToolCall{{Name: "read_file", Arguments: `{"path":"build/app/state.js"}`}}, nil)
	}
	w.note(nil, []Message{{Content: second}})
	if w.sig != "check fail:outcome" || w.hits != 1 || w.idle != 1 {
		t.Fatalf("changed observation must restart the watch: %+v", w)
	}
	if w.stuck() {
		t.Fatal("render passing is progress, not a stuck loop")
	}
}

func TestCheckWatchStopsUnchangedFail(t *testing.T) {
	var w checkWatch
	fail := []Message{{Content: verifyBanner}}
	harness := []ToolCall{{Name: "str_replace", Arguments: `{"path":"qa/verify.mjs"}`}}
	for i := 0; i < checkFailStopHits; i++ {
		w.note(harness, fail)
	}
	if w.hits != checkFailStopHits {
		t.Fatalf("hits %d", w.hits)
	}
	if !w.stuck() {
		t.Fatal("same FAIL must stop")
	}
	if observationNudge(RunRequest{User: "把小说做成游戏", SoftHorizon: true}, &w, harness) == "" {
		t.Fatal("harness edit after FAIL must nudge")
	}
}

func TestCheckWatchIdleStopsWithoutProductEdit(t *testing.T) {
	var w checkWatch
	fail := []Message{{Content: verifyBanner}}
	w.note(nil, fail)
	wait := []ToolCall{{Name: "wait", Arguments: `{"ms":1000}`}}
	for w.idle < checkFailIdleStop {
		if w.stuck() {
			t.Fatalf("stuck before idle cap idle=%d", w.idle)
		}
		w.note(wait, nil)
	}
	if !w.stuck() {
		t.Fatalf("idle %d must stop", w.idle)
	}
}

func TestCheckWatchProductWriteResetsIdleNotHits(t *testing.T) {
	var w checkWatch
	fail := []Message{{Content: verifyBanner}}
	w.note(nil, fail)
	w.note([]ToolCall{{Name: "str_replace", Arguments: `{"path":"build/app/game.js"}`}}, fail)
	if w.idle != 0 {
		t.Fatalf("product write must reset idle, got %d", w.idle)
	}
	if w.hits != 2 {
		t.Fatalf("hits %d", w.hits)
	}
	if w.stuck() {
		t.Fatal("two FAILs after a product edit is not stuck")
	}
}

func TestCheckWatchIgnoresPlanUpdate(t *testing.T) {
	var w checkWatch
	fail := []Message{{Content: verifyBanner}}
	w.note(nil, fail)
	w.note([]ToolCall{{Name: "update_plan", Arguments: `{"steps":[]}`}}, fail)
	if w.hits != 2 {
		t.Fatalf("plan update must not clear observation hits, got %d", w.hits)
	}
}

func TestObservationNudgeHarborSilent(t *testing.T) {
	var w checkWatch
	w.note(nil, []Message{{Content: verifyBanner}})
	w.note(nil, []Message{{Content: verifyBanner}})
	got := observationNudge(RunRequest{User: "把小说做成游戏"}, &w, nil)
	if got != "" {
		t.Fatalf("harbor %q", got)
	}
}

func TestInstrumentNudgeOnHarnessEdit(t *testing.T) {
	var w checkWatch
	fail := []Message{{Content: verifyBanner}}
	w.note(nil, fail)
	req := RunRequest{User: "把小说做成游戏", SoftHorizon: true}
	got := observationNudge(req, &w, []ToolCall{{
		Name: "apply_patch", Arguments: `{"patch":"*** Update File: qa/verify.mjs\n"}`,
	}})
	if !strings.Contains(got, instrumentPrefixZH) {
		t.Fatalf("%q", got)
	}
	if !isControlUser(got) {
		t.Fatal("instrument nudge must be control")
	}
	check := observationNudge(req, &w, nil)
	if check != "" {
		t.Fatalf("one FAIL is not a check-fail nudge %q", check)
	}
	w.note(nil, fail)
	check = observationNudge(req, &w, nil)
	if !strings.Contains(check, checkFailPrefixZH) || !strings.Contains(check, "check fail:render") {
		t.Fatalf("%q", check)
	}
	if !isControlUser(check) {
		t.Fatal("check-fail nudge must be control")
	}
}

func TestIdenticalFailSwitchesInsteadOfStopping(t *testing.T) {
	script := `{"command":"node --check pelican-bike/game.js; node pelican-bike/qa/headless-verify.cjs"}`
	calls := []ToolCall{{Name: "shell", Arguments: script}}
	same := "FAIL  jump-arc  起跳后离地=false\nFAIL  jump-land  存活期间重新落地=false\n"
	changed := "FAIL  jump-arc  起跳后离地=true\nFAIL  jump-land  存活期间重新落地=false\n"
	var w checkWatch
	req := RunRequest{User: "继续修跳跃", SoftHorizon: true}
	w.note(calls, []Message{{Content: same}})
	w.note(calls, []Message{{Content: same}})
	nudge := observationNudge(req, &w, nil)
	if !strings.Contains(nudge, "不要再执行同一条命令") || strings.Contains(nudge, "重跑同一条检查") {
		t.Fatalf("nudge %q", nudge)
	}
	got, refusal := w.repeatBlock()
	if got != "pelican-bike/qa/headless-verify.cjs" || refusal == "" {
		t.Fatalf("block %q %q", got, refusal)
	}
	for i := 0; i < 6; i++ {
		w.note([]ToolCall{{Name: "str_replace", Arguments: `{"path":"pelican-bike/game.js"}`}}, []Message{{Content: same}})
	}
	if w.stuck() {
		t.Fatal("editing the product and seeing the same FAIL must not stop the turn")
	}
	w.note([]ToolCall{{Name: "str_replace", Arguments: `{"path":"pelican-bike/game.js"}`}}, nil)
	if s, _ := w.repeatBlock(); s != "" {
		t.Fatalf("product edit must allow one rerun, still blocking %q", s)
	}
	w.note(calls, []Message{{Content: changed}})
	if w.hits != 1 || w.identical || w.warned {
		t.Fatalf("changed detail must reset: %+v", w)
	}
}

func TestSeedCheckWatchBlocksContinuedScript(t *testing.T) {
	calls := []ToolCall{{Name: "shell", Arguments: `{"command":"node pelican-bike/qa/headless-verify.cjs"}`}}
	fail := "FAIL  jump-arc  起跳后离地=false\n"
	hist := []Message{
		{Role: RoleAssistant, ToolCalls: calls},
		{Role: RoleTool, Name: "shell", Content: fail},
		{Role: RoleAssistant, ToolCalls: calls},
		{Role: RoleTool, Name: "shell", Content: fail},
	}
	var w checkWatch
	req := RunRequest{User: "继续", SoftHorizon: true}
	seedCheckWatch(req, &w, hist)
	script, refusal := w.repeatBlock()
	if script != "pelican-bike/qa/headless-verify.cjs" || refusal == "" {
		t.Fatalf("seed block %q %q %+v", script, refusal, w)
	}
	if w.stuck() {
		t.Fatal("a continued turn must not open on the stop card")
	}
	tools := &WorkspaceTools{}
	tools.setRepeatBlock(script, refusal)
	res := tools.shell(`node --check pelican-bike/game.js; node pelican-bike/qa/headless-verify.cjs`, 5)
	if res.Err != nil || !strings.Contains(res.Content, "不要再执行同一条命令") {
		t.Fatalf("%+v", res)
	}
	if _, ok := tools.repeatBlocked("node --check pelican-bike/game.js"); ok {
		t.Fatal("a different command must stay runnable")
	}
	tools.setRepeatBlock("browser_verify", "switch")
	if _, ok := tools.repeatBlocked("node pelican-bike/qa/headless-verify.cjs"); ok {
		t.Fatal("shell must stay runnable when the repeated check was browser_verify")
	}
	vres := tools.browserVerify(nil)
	if vres.Err != nil || !strings.Contains(vres.Content, "switch") {
		t.Fatalf("verify %+v", vres)
	}
}

func TestBypassReprintStillStops(t *testing.T) {
	var w checkWatch
	req := RunRequest{User: "继续", SoftHorizon: true}
	fail := []Message{{Content: verifyBanner}}
	reprint := []ToolCall{{Name: "shell", Arguments: `{"command":"node -e \"print\""}`}}
	w.note(reprint, fail)
	w.note(reprint, fail)
	if observationNudge(req, &w, nil) == "" || w.stuck() {
		t.Fatal("two identical reprints warn and do not stop")
	}
	for w.bypassHits < checkFailBackstopHits {
		w.note(reprint, fail)
	}
	if !w.stuck() {
		t.Fatalf("bypass %d warned %v", w.bypassHits, w.warned)
	}
}

func TestIsHarnessAndProductPath(t *testing.T) {
	if !isHarnessPath(`C:\game\qa\verify.mjs`) || !isHarnessPath("qa/core-loop-check.mjs") {
		t.Fatal("qa scripts are harness")
	}
	if isProductPath("qa/verify.mjs") {
		t.Fatal("verifier is not product")
	}
	if !isProductPath("build/app/game.js") || !isProductPath("build/app/index.html") {
		t.Fatal("game surface is product")
	}
}

func TestCheckWatchIgnoresReadFileOfVerificationJSON(t *testing.T) {
	var w checkWatch
	fail := []Message{{Content: "{\n  \"outcome\": \"FAIL\",\n  \"render\": \"PASS\"\n}"}}
	w.note([]ToolCall{{Name: "browser_verify", Arguments: `{"id":"world-x"}`}}, fail)
	dump := []ToolCall{{Name: "read_file", Arguments: `{"path":"qa/verification.json"}`}}
	for i := 0; i < checkFailIdleStop+2; i++ {
		w.note(dump, fail)
	}
	if w.hits != 1 {
		t.Fatalf("reading verification.json must not re-count the check, hits=%d", w.hits)
	}
	if w.idle != 1 {
		t.Fatalf("inspect-only rounds must not tick idle, idle=%d", w.idle)
	}
	if w.stuck() {
		t.Fatal("inspect-only rounds must not trip the same-check stop")
	}
}

func TestErrorRepeatNudgeLeavesCheckFailToObservation(t *testing.T) {
	hits := map[string]int{}
	recordToolErrors(hits, []Message{{Content: verifyBanner}, {Content: verifyBanner}, {Content: verifyBanner}})
	if hits["check fail:render"] != 3 {
		t.Fatalf("%v", hits)
	}
	got := errorRepeatNudge(RunRequest{User: "把小说做成游戏", SoftHorizon: true}, hits)
	if got != "" {
		t.Fatalf("check-fail is owned by observationNudge, got %q", got)
	}
}
