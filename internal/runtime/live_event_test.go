package runtime

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

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
	if len(got) > 4_100 || !strings.Contains(got, "…") {
		t.Fatalf("live result not capped: %d", len(got))
	}
	if orig["content"] != body {
		t.Fatal("persist payload must keep full result")
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
