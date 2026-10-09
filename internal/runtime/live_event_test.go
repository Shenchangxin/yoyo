package runtime

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestCompactCallArgsOmitsFileBytes(t *testing.T) {
	body := strings.Repeat("const GROUND = H - 78;\n", 400)
	args := `{"path":"game.html","new_str":` + mustJSONString(body) + `,"old_str":"x"}`
	got := compactCallArgs("str_replace", args, 800)
	if strings.Contains(got, "const GROUND") {
		t.Fatalf("leaked file bytes: %s", got[:min(240, len(got))])
	}
	if !strings.Contains(got, "(omitted") || !strings.Contains(got, "Do not paste this placeholder") {
		t.Fatalf("%s", got)
	}
	if !looksLikeContextStub(got) {
		t.Fatal("compact stub must be rejected if the model pastes it back")
	}
}

func TestSlimLiveEventStubsWriteArgs(t *testing.T) {
	body := strings.Repeat("x", 4000)
	args := `{"path":"a.ts","content":"` + body + `"}`
	orig := map[string]any{"name": "write_file", "arguments": args, "id": "w1"}
	ev := trace.Event{Type: trace.TypeToolCall, Payload: orig}
	live := slimLiveEvent(ev)
	got, _ := live.Payload["arguments"].(string)
	if got == args || strings.Contains(got, body) {
		t.Fatalf("live still has full body: %s", got[:min(180, len(got))])
	}
	if orig["arguments"] != args {
		t.Fatal("persist payload must keep full write")
	}
	if live.Payload["bytes"] != len(args) {
		t.Fatalf("bytes %v", live.Payload["bytes"])
	}
}

func TestSlimLiveEventStubsTruncatedWritePath(t *testing.T) {
	args := `{"path": "pelican-bike.html", "content": "<!DOCTYPE html>\n<html`
	ev := trace.Event{Type: trace.TypeToolCall, Payload: map[string]any{"name": "write_file", "arguments": args, "id": "w1"}}
	live := slimLiveEvent(ev)
	got, _ := live.Payload["arguments"].(string)
	if !strings.Contains(got, "pelican-bike.html") {
		t.Fatalf("path stub missing: %s", got)
	}
	if strings.Contains(got, "<!DOCTYPE") {
		t.Fatalf("truncated body leaked: %s", got)
	}
	if live.Payload["path"] != "pelican-bike.html" {
		t.Fatalf("path field %v", live.Payload["path"])
	}
	if ev.Payload["arguments"] != args {
		t.Fatal("persist payload must keep truncated write")
	}
}

func TestSlimLiveEventLeavesShellAlone(t *testing.T) {
	args := `{"cmd":"echo hi"}`
	ev := trace.Event{Type: trace.TypeToolCall, Payload: map[string]any{"name": "shell", "arguments": args}}
	live := slimLiveEvent(ev)
	if live.Payload["arguments"] != args {
		t.Fatalf("%v", live.Payload["arguments"])
	}
}

func TestSlimTrajectoryStubsWriteArgs(t *testing.T) {
	body := strings.Repeat("z", 4000)
	args := `{"path":"b.ts","content":"` + body + `"}`
	evs := []trace.Event{{
		Type:    trace.TypeToolCall,
		Payload: map[string]any{"name": "write_file", "arguments": args, "id": "w2"},
	}}
	got := SlimTrajectory(evs)
	if evs[0].Payload["arguments"] != args {
		t.Fatal("store event must keep full write")
	}
	out, _ := got[0].Payload["arguments"].(string)
	if out == args || strings.Contains(out, body) {
		t.Fatalf("ui trajectory still has full body")
	}
}

func TestSlimLiveEventCapsToolResult(t *testing.T) {
	body := strings.Repeat("log\n", 2000)
	orig := map[string]any{"name": "shell", "content": body, "id": "c1"}
	ev := trace.Event{Type: trace.TypeToolResult, Payload: orig}
	live := slimLiveEvent(ev)
	got, _ := live.Payload["content"].(string)
	if len(got) > uiResultBytes+8 || !strings.Contains(got, "…") {
		t.Fatalf("live result not capped: %d", len(got))
	}
	if orig["content"] != body {
		t.Fatal("persist payload must keep full result")
	}
}

func TestUITrajectoryCapsSettledAssistant(t *testing.T) {
	body := strings.Repeat("a", 40_000)
	orig := map[string]any{"text": body, "id": "r1"}
	evs := []trace.Event{{Type: trace.TypeAssistant, Payload: orig}}
	got := UITrajectory(evs)
	if len(got) != 1 {
		t.Fatalf("n=%d", len(got))
	}
	out, _ := got[0].Payload["text"].(string)
	if len(out) > uiAssistantBytes+8 || !strings.Contains(out, "…") {
		t.Fatalf("assistant not capped: %d", len(out))
	}
	if orig["text"] != body {
		t.Fatal("store event must keep full assistant")
	}
	delta := slimLiveEvent(trace.Event{
		Type:    trace.TypeAssistant,
		Payload: map[string]any{"text": body, "delta": true, "id": "r1"},
	})
	if delta.Payload["text"] != body {
		t.Fatal("token slices must stay intact for live concat")
	}
}

func TestUITrajectoryKeepsTurnsDropsNoise(t *testing.T) {
	var evs []trace.Event
	evs = append(evs, trace.Event{Type: trace.TypeUser, Source: "user", Payload: map[string]any{"text": "old"}})
	evs = append(evs, trace.Event{Type: trace.TypeAssistant, Payload: map[string]any{"text": "old-a", "id": "r0"}})
	for i := 0; i < 40; i++ {
		id := "c" + strconv.Itoa(i)
		evs = append(evs, trace.Event{Type: trace.TypeToolCall, Payload: map[string]any{"id": id, "name": "read_file", "arguments": `{"path":"x"}`}})
		evs = append(evs, trace.Event{Type: trace.TypeFileChange, Payload: map[string]any{"path": "x"}})
		evs = append(evs, trace.Event{Type: trace.TypeToolResult, Payload: map[string]any{"id": id, "name": "read_file", "content": "ok"}})
	}
	evs = append(evs, trace.Event{Type: trace.TypeUser, Source: "user", Payload: map[string]any{"text": "new"}})
	evs = append(evs, trace.Event{Type: trace.TypeToolCall, Payload: map[string]any{"id": "now", "name": "read_file", "arguments": `{"path":"y"}`}})
	evs = append(evs, trace.Event{Type: trace.TypeToolResult, Payload: map[string]any{"id": "now", "name": "read_file", "content": "y"}})
	got := UITrajectory(evs)
	calls := 0
	sawOld, sawNew, sawFile := false, false, false
	for _, ev := range got {
		if ev.Type == trace.TypeToolCall {
			calls++
		}
		if ev.Type == trace.TypeFileChange {
			sawFile = true
		}
		if ev.Type == trace.TypeUser {
			if ev.Payload["text"] == "old" {
				sawOld = true
			}
			if ev.Payload["text"] == "new" {
				sawNew = true
			}
		}
	}
	if sawFile {
		t.Fatal("file_change leaked into UI trajectory")
	}
	if !sawNew || !sawOld {
		t.Fatalf("wanted both user turns, old=%v new=%v n=%d", sawOld, sawNew, len(got))
	}
	if calls != 41 {
		t.Fatalf("tools dropped from UI trajectory: %d", calls)
	}
}

func TestUITrajectoryDropsSupersededErrors(t *testing.T) {
	stopped := trace.Event{Type: trace.TypeError, Payload: map[string]any{"kind": "canceled", "title": "Stopped", "hint": "This turn was interrupted."}}
	kept := UITrajectory([]trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "go"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "working", "id": "r1"}},
		stopped,
	})
	if n := countType(kept, trace.TypeError); n != 1 {
		t.Fatalf("terminal stop should stay: %d", n)
	}
	resumed := UITrajectory([]trace.Event{
		{Type: trace.TypeUser, Payload: map[string]any{"text": "go"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "working", "id": "r1"}},
		stopped,
		{Type: trace.TypeAssistant, Payload: map[string]any{"text": "continued", "id": "r1"}},
	})
	if n := countType(resumed, trace.TypeError); n != 0 {
		t.Fatalf("resumed turn still has stop card: %d", n)
	}
}

func countType(evs []trace.Event, typ trace.EventType) int {
	n := 0
	for _, ev := range evs {
		if ev.Type == typ {
			n++
		}
	}
	return n
}
