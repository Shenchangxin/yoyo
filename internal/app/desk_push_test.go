package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestDeskPushWritesOnce(t *testing.T) {
	a := &App{Traces: trace.NewStore(t.TempDir())}
	note := DeskNote{SessionID: "s", Kind: "brief", Slot: "morning", Day: "2026-11-02", Text: "早上这一份在这里。", Chip: "晨间简报"}
	if !a.DeskPush(note) {
		t.Fatal("first push")
	}
	if !a.DeskPush(note) {
		t.Fatal("second push should report the note is already there")
	}
	evs, err := a.Traces.Read("s")
	if err != nil || len(evs) != 1 {
		t.Fatalf("events %d %v", len(evs), err)
	}
	if evs[0].Source != "desk" || evs[0].Type != trace.TypeAssistant {
		t.Fatalf("event = %#v", evs[0])
	}
}

func TestDeskPushBriefStaysInTheFirstSession(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	first := DeskNote{SessionID: "older", Kind: "brief", Slot: "morning", Day: "2026-11-04", Text: "早上这一份在这里。", Chip: "晨间简报", Brief: `{"cards":[{"id":"a","title":"first"}]}`}
	if !a.DeskPush(first) {
		t.Fatal("first push")
	}
	again := first
	again.SessionID = "newer"
	again.Brief = `{"cards":[{"id":"b","title":"second"}]}`
	if !a.DeskPush(again) {
		t.Fatal("later push should count as delivered")
	}
	evs, err := a.Traces.Read("newer")
	if err != nil || len(evs) != 0 {
		t.Fatalf("newer session events %d %v", len(evs), err)
	}
	list := a.BriefList()
	if len(list) != 1 || list[0].Brief != first.Brief {
		t.Fatalf("ledger = %+v", list)
	}
}

func TestSealBriefsDropsTheLaterCopy(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	early := trace.Event{
		TS: time.Date(2026, 11, 4, 1, 40, 0, 0, time.UTC), Type: trace.TypeAssistant, Source: "desk", SessionID: "old",
		Payload: map[string]any{"id": "brief:morning:2026-11-04", "kind": "brief", "slot": "morning", "day": "2026-11-04", "text": "早上这一份在这里。", "chip": "晨间简报", "brief": `{"cards":[{"id":"a","title":"first"}]}`},
	}
	late := early
	late.TS = time.Date(2026, 11, 4, 9, 29, 0, 0, time.UTC)
	late.SessionID = "main"
	late.Payload = map[string]any{"id": "brief:morning:2026-11-04", "kind": "brief", "slot": "morning", "day": "2026-11-04", "text": "早上这一份在这里。", "chip": "晨间简报", "brief": `{"cards":[{"id":"b","title":"rewritten"}]}`}
	if err := a.Traces.AppendDurable(early); err != nil {
		t.Fatal(err)
	}
	if err := a.Traces.AppendDurable(late); err != nil {
		t.Fatal(err)
	}
	a.sealBriefs()
	kept, err := a.Traces.Read("old")
	if err != nil || len(kept) != 1 {
		t.Fatalf("early copy %d %v", len(kept), err)
	}
	gone, err := a.Traces.Read("main")
	if err != nil || len(gone) != 0 {
		t.Fatalf("later copy %d %v", len(gone), err)
	}
	list := a.BriefList()
	if len(list) != 1 || list[0].Brief != `{"cards":[{"id":"a","title":"first"}]}` {
		t.Fatalf("ledger = %+v", list)
	}
}
