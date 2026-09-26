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
}

func TestCompanionPulseTitleClipsUserText(t *testing.T) {
	title := companionPulseTitle("loading", "user", "Working", "please look at the weekly report and also the inbox", "")
	if title == "Working" || len([]rune(title)) > 43 {
		t.Fatalf("got %q", title)
	}
	if companionPulseTitle("loading", "tool_call", "Working", "", "web_search") != "web_search" {
		t.Fatal("tool name")
	}
}
