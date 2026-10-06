package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestEmitLiveIncludesSkillLabel(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	var live []trace.Event
	req := RunRequest{
		SessionID: "s",
		Trace:     st,
		OnEvent:   func(ev trace.Event) { live = append(live, ev) },
	}
	emit(req, trace.TypeToolCall, "agent", map[string]any{
		"name": "load_skill", "arguments": `{"name":"brainstorming"}`, "id": "s1", "skill": "brainstorming",
	})
	if len(live) != 1 {
		t.Fatalf("live=%d", len(live))
	}
	if live[0].Payload["name"] != "load_skill" || live[0].Payload["skill"] != "brainstorming" {
		t.Fatalf("%v", live[0].Payload)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) == 0 || evs[0].Payload["skill"] != "brainstorming" {
		t.Fatal("jsonl must keep the skill label")
	}
}

func TestUITrajectoryKeepsLoadSkill(t *testing.T) {
	got := UITrajectory([]trace.Event{
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "s1", "name": "load_skill", "arguments": `{"name":"brainstorming"}`, "skill": "brainstorming"}},
		{Type: trace.TypeToolResult, Payload: map[string]any{"id": "s1", "name": "load_skill", "content": "ok"}},
	})
	if len(got) != 2 {
		t.Fatalf("n=%d", len(got))
	}
	if got[0].Payload["skill"] != "brainstorming" {
		t.Fatalf("%v", got[0].Payload)
	}
}
