package desktop

import "testing"

func TestCompanionPulseKind(t *testing.T) {
	if companionPulseKind("user", "user") != "loading" {
		t.Fatal("user turn should light the pet immediately")
	}
	if companionPulseKind("user", "steer") != "" {
		t.Fatal("steer must not spawn a new loading pulse")
	}
	if companionPulseKind("tool_call", "runtime") != "loading" {
		t.Fatal("tools stay on the loading channel")
	}
	if companionPulseKind("turn_end", "runtime") != "done" {
		t.Fatal("turn_end clears loading")
	}
	if companionPulseKind("assistant", "runtime") != "" {
		t.Fatal("token deltas must not spam pulses")
	}
	if companionPulseKind("reasoning", "runtime") != "" {
		t.Fatal("reasoning deltas must not spam pulses")
	}
	if companionPulseKind("system", "update") != "update" {
		t.Fatal("signed OTA ready pulse")
	}
}

func TestCompanionPulseTitleClipsUserText(t *testing.T) {
	title := companionPulseTitle("loading", "user", "Working", "please look at the weekly report and also the inbox", "", "")
	if title == "Working" || len([]rune(title)) > 43 {
		t.Fatalf("got %q", title)
	}
	if companionPulseTitle("loading", "tool_call", "Working", "", "web_search", "") != "web_search" {
		t.Fatal("tool name")
	}
}

func TestCompanionPulseTitleShowsSkillName(t *testing.T) {
	if companionPulseTitle("loading", "tool_call", "Working", "", "load_skill", `{"name":"brainstorming"}`) != "brainstorming" {
		t.Fatal("load_skill must title the pet with the skill id")
	}
	if companionPulseTitle("loading", "tool_call", "Working", "", "load_skill", "") != "load_skill" {
		t.Fatal("unknown skill still discloses the tool")
	}
	if companionPulseTitle("loading", "tool_call", "Working", "", "read_skill_file", `{"skill":"brainstorming","path":"references/guide.md"}`) != "brainstorming · references/guide.md" {
		t.Fatal("read_skill_file")
	}
}

func TestAnsweredOrAskSkipsRepeats(t *testing.T) {
	if !answeredOrAsk("approval", map[string]any{"id": "a1", "decision": "once"}) {
		t.Fatal("a decision event answers an approval; it must not notify again")
	}
	if !answeredOrAsk("approval", map[string]any{"id": "a2", "action": "ask_user", "command": "which file?"}) {
		t.Fatal("the approval mirror of an ask_user question must not double the toast")
	}
	if answeredOrAsk("approval", map[string]any{"id": "a3", "action": "shell", "command": "npm test"}) {
		t.Fatal("a pending shell approval must notify")
	}
	if answeredOrAsk("ask_user", map[string]any{"question": "which file?"}) {
		t.Fatal("the ask_user item is the announcement itself")
	}
}

func TestQuietOutcomeKeepsStopAndPlanOpenOutOfToasts(t *testing.T) {
	if !quietOutcome("error", map[string]any{"kind": "canceled", "error": "Stopped"}) {
		t.Fatal("a Stop is not news")
	}
	if quietOutcome("error", map[string]any{"kind": "provider"}) {
		t.Fatal("a real provider error must toast")
	}
	if !quietOutcome("turn_end", map[string]any{"stop": "plan_open"}) {
		t.Fatal("a plan_open turn is reported by its plan card")
	}
	if quietOutcome("turn_end", map[string]any{"stop": "end_turn"}) {
		t.Fatal("a finished turn must toast")
	}
}

func TestToastLinesStayReadable(t *testing.T) {
	if got := approvalLine(map[string]any{"action": "shell", "command": "npm   test\n --watch"}); got != "shell: npm test --watch" {
		t.Fatalf("got %q", got)
	}
	if got := approvalLine(map[string]any{"action": "write_file", "path": "a.txt"}); got != "write_file: a.txt" {
		t.Fatalf("got %q", got)
	}
	if got := approvalLine(map[string]any{"action": "shell"}); got != "shell" {
		t.Fatalf("got %q", got)
	}
	if got := lineJoin("", " 修复登录 ", "shell"); got != "修复登录 · shell" {
		t.Fatalf("got %q", got)
	}
	if got := oneLine("", 10); got != "" {
		t.Fatalf("empty summary must stay empty, got %q", got)
	}
}
