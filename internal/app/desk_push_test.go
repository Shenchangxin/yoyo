package app

import (
	"testing"

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
