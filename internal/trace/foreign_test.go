package trace

import (
	"testing"
)

func TestForeignSinceSkipsOwnLinesAndReadsTheOtherProcess(t *testing.T) {
	dir := t.TempDir()
	mine := NewStore(dir)
	other := NewStore(dir)
	if err := mine.Append(Event{Type: TypeUser, SessionID: "main", Payload: map[string]any{"text": "mine-1"}}); err != nil {
		t.Fatal(err)
	}
	from := int64(0)
	evs, from, err := mine.ForeignSince("main", from)
	if err != nil || len(evs) != 0 {
		t.Fatalf("own line came back: %#v %v", evs, err)
	}
	if err := other.AppendDurable(Event{Type: TypeAssistant, Source: "desk", SessionID: "main", Payload: map[string]any{"text": "note"}}); err != nil {
		t.Fatal(err)
	}
	if err := mine.Append(Event{Type: TypeUser, SessionID: "main", Payload: map[string]any{"text": "mine-2"}}); err != nil {
		t.Fatal(err)
	}
	evs, from, err = mine.ForeignSince("main", from)
	if err != nil || len(evs) != 1 || evs[0].Payload["text"] != "note" || evs[0].Source != "desk" {
		t.Fatalf("foreign = %#v %v", evs, err)
	}
	if from != mine.Size("main") {
		t.Fatalf("cursor %d != size %d", from, mine.Size("main"))
	}
	if evs, _, _ := mine.ForeignSince("main", from); len(evs) != 0 {
		t.Fatalf("read twice: %#v", evs)
	}
	all, err := mine.Read("main")
	if err != nil || len(all) != 3 {
		t.Fatalf("read = %d %v", len(all), err)
	}
}

func TestForeignSinceAfterRewriteMovesToTheEnd(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	other := NewStore(dir)
	for i := 0; i < 3; i++ {
		_ = other.Append(Event{Type: TypeAssistant, SessionID: "s", Payload: map[string]any{"i": i}})
	}
	evs, from, _ := s.ForeignSince("s", 0)
	if len(evs) != 3 {
		t.Fatalf("first read = %d", len(evs))
	}
	if err := other.Replace("s", []Event{{Type: TypeAssistant, SessionID: "s"}}); err != nil {
		t.Fatal(err)
	}
	evs, next, _ := s.ForeignSince("s", from)
	if len(evs) != 0 || next != s.Size("s") {
		t.Fatalf("after rewrite = %d events, cursor %d size %d", len(evs), next, s.Size("s"))
	}
}
