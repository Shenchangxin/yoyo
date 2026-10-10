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
	read := []ToolCall{{Name: "read_file", Arguments: `{"path":"qa/verify.mjs"}`}}
	for w.idle < checkFailIdleStop {
		if w.stuck() {
			t.Fatalf("stuck before idle cap idle=%d", w.idle)
		}
		w.note(read, nil)
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
