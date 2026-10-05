package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestProjectPageDropsOldestTurnsUnderBudget(t *testing.T) {
	var evs []trace.Event
	for i := 0; i < 6; i++ {
		evs = append(evs,
			trace.Event{Seq: int64(i*2 + 1), Type: trace.TypeUser, Payload: map[string]any{"text": "u"}},
			trace.Event{Seq: int64(i*2 + 2), Type: trace.TypeAssistant, Payload: map[string]any{"text": strings.Repeat("x", 800), "id": "r"}},
		)
	}
	page := ProjectPage(trace.Page{Events: evs, HeadSeq: 1, TailSeq: 12, Older: false}, 99, 2_000)
	if !page.Older {
		t.Fatal("expected older after byte cap")
	}
	users := 0
	for _, ev := range page.Events {
		if ev.Type == trace.TypeUser {
			users++
		}
	}
	if users >= 6 || users < 1 {
		t.Fatalf("users %d n=%d", users, len(page.Events))
	}
	if page.HubSeq != 99 {
		t.Fatalf("hub %d", page.HubSeq)
	}
}

func TestProjectPageKeepRetainsTargetUnderBudget(t *testing.T) {
	var evs []trace.Event
	for i := 0; i < 8; i++ {
		evs = append(evs,
			trace.Event{Seq: int64(i*2 + 1), Type: trace.TypeUser, Payload: map[string]any{"text": "u", "id": "u"}},
			trace.Event{Seq: int64(i*2 + 2), Type: trace.TypeAssistant, Payload: map[string]any{"text": strings.Repeat("x", 900), "id": "r"}},
		)
	}
	raw := trace.Page{Events: evs, HeadSeq: 1, TailSeq: 16, Older: false}
	keep := int64(1)
	page := ProjectPageKeep(raw, 99, 2_000, keep)
	found := false
	for _, ev := range page.Events {
		if ev.Seq == keep {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("keep seq %d dropped n=%d", keep, len(page.Events))
	}
	dropped := ProjectPage(raw, 99, 2_000)
	for _, ev := range dropped.Events {
		if ev.Seq == keep {
			t.Fatal("default ProjectPage should drop the oldest user")
		}
	}
}
