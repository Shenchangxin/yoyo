package trace

import (
	"path/filepath"
	"testing"
)

func TestDefaultPageWindowFitsTheRenderer(t *testing.T) {
	if DefaultPageTurns > 12 {
		t.Fatalf("DefaultPageTurns %d", DefaultPageTurns)
	}
	if DefaultPageBytes > 300_000 {
		t.Fatalf("DefaultPageBytes %d", DefaultPageBytes)
	}
}

func TestIndexPagesByUserTurns(t *testing.T) {
	s := NewStore(t.TempDir())
	id := "sess"
	for i := 0; i < 6; i++ {
		if err := s.Append(Event{Type: TypeUser, SessionID: id, Payload: map[string]any{"text": filepath.Base(t.Name()) + string(rune('a'+i))}}); err != nil {
			t.Fatal(err)
		}
		if err := s.Append(Event{Type: TypeAssistant, SessionID: id, Payload: map[string]any{"text": "ok"}}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.PageTurns(id, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Older {
		t.Fatal("expected older prefix")
	}
	users := 0
	for _, ev := range page.Events {
		if ev.Type == TypeUser {
			users++
		}
	}
	if users != 2 {
		t.Fatalf("users %d events %d", users, len(page.Events))
	}
	older, err := s.PageTurns(id, page.HeadSeq, 2)
	if err != nil {
		t.Fatal(err)
	}
	if older.TailSeq >= page.HeadSeq {
		t.Fatalf("overlap tail=%d head=%d", older.TailSeq, page.HeadSeq)
	}
}

func TestReadFromCheckpointSeeks(t *testing.T) {
	s := NewStore(t.TempDir())
	id := "ckpt"
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Payload: map[string]any{"text": "old"}})
	_ = s.Append(Event{Type: TypeAssistant, SessionID: id, Payload: map[string]any{"text": "a"}})
	_ = s.Append(Event{Type: TypeCompact, SessionID: id, Payload: map[string]any{"kind": "checkpoint", "summary": "sum", "note": "checkpoint"}})
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Payload: map[string]any{"text": "new"}})
	evs, err := s.ReadFromCheckpoint(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("want checkpoint+user, got %d", len(evs))
	}
	if evs[0].Type != TypeCompact || evs[1].Type != TypeUser {
		t.Fatalf("types %s %s", evs[0].Type, evs[1].Type)
	}
}

func TestDurableUserSurvivesReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sess")
	s := NewStore(dir)
	if err := s.AppendDurable(Event{Type: TypeUser, SessionID: "a", Payload: map[string]any{"text": "keep me"}}); err != nil {
		t.Fatal(err)
	}
	s2 := NewStore(dir)
	evs, err := s2.Read("a")
	if err != nil || len(evs) != 1 {
		t.Fatalf("%v %d", err, len(evs))
	}
	if evs[0].Payload["text"] != "keep me" {
		t.Fatalf("%v", evs[0].Payload)
	}
}
