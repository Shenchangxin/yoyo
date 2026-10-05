package trace

import (
	"strings"
	"testing"
)

func TestOutlineTitleStripsMentionsAndCaps(t *testing.T) {
	got := OutlineTitle("@file:src/loop.go  Fix the parser in loop.go\nmore")
	if got != "Fix the parser in loop.go more" {
		t.Fatalf("title %q", got)
	}
	long := OutlineTitle(strings.Repeat("字", 50))
	r := []rune(long)
	if len(r) != outlineTitleRunes+1 || r[len(r)-1] != '…' {
		t.Fatalf("cjk title %q len %d", long, len(r))
	}
}

func TestOutlineTurnsSkipsSteer(t *testing.T) {
	s := NewStore(t.TempDir())
	id := "sess"
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Source: "user", Payload: map[string]any{"text": "first", "id": "u1"}})
	_ = s.Append(Event{Type: TypeAssistant, SessionID: id, Payload: map[string]any{"text": "ok"}})
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Source: "steer", Payload: map[string]any{"text": "do something else", "name": "steer"}})
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Source: "user", Payload: map[string]any{"text": "second prompt", "id": "u2"}})
	out, err := s.OutlineTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("turns %d %+v", len(out), out)
	}
	if out[0].ID != "u1" || out[0].Title != "first" {
		t.Fatalf("first %+v", out[0])
	}
	if out[1].ID != "u2" || out[1].Title != "second prompt" {
		t.Fatalf("second %+v", out[1])
	}
	if out[0].Seq == 0 || out[1].Seq <= out[0].Seq {
		t.Fatalf("seq %+v", out)
	}
}

func TestPageAroundUserPutsTargetInWindow(t *testing.T) {
	s := NewStore(t.TempDir())
	id := "sess"
	var userSeq [10]int64
	for i := 0; i < 10; i++ {
		if err := s.Append(Event{Type: TypeUser, SessionID: id, Payload: map[string]any{"text": string(rune('a' + i)), "id": string(rune('a' + i))}}); err != nil {
			t.Fatal(err)
		}
		_ = s.Append(Event{Type: TypeAssistant, SessionID: id, Payload: map[string]any{"text": "ok"}})
	}
	out, err := s.OutlineTurns(id)
	if err != nil || len(out) != 10 {
		t.Fatalf("outline %v %d", err, len(out))
	}
	for i, trow := range out {
		userSeq[i] = trow.Seq
	}
	page, err := s.PageAroundUser(id, userSeq[2], 4)
	if err != nil {
		t.Fatal(err)
	}
	users := userTexts(page)
	if len(users) != 4 || users[0] != "c" {
		t.Fatalf("early window %v older=%v", users, page.Older)
	}
	if !page.Older {
		t.Fatal("expected older prefix before turn c")
	}
	found := false
	for _, ev := range page.Events {
		if ev.Type == TypeUser && ev.Seq == userSeq[2] {
			found = true
		}
	}
	if !found {
		t.Fatalf("target seq %d missing", userSeq[2])
	}

	tail, err := s.PageAroundUser(id, userSeq[9], 4)
	if err != nil {
		t.Fatal(err)
	}
	tailUsers := userTexts(tail)
	if len(tailUsers) != 4 || tailUsers[len(tailUsers)-1] != "j" {
		t.Fatalf("eof window %v", tailUsers)
	}
	hit := false
	for _, ev := range tail.Events {
		if ev.Type == TypeUser && ev.Seq == userSeq[9] {
			hit = true
		}
	}
	if !hit {
		t.Fatal("last user not in eof window")
	}
}

func TestPageAroundUserSkipsSteer(t *testing.T) {
	s := NewStore(t.TempDir())
	id := "sess"
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Source: "user", Payload: map[string]any{"text": "a", "id": "a"}})
	_ = s.Append(Event{Type: TypeAssistant, SessionID: id, Payload: map[string]any{"text": "ok"}})
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Source: "steer", Payload: map[string]any{"text": "hurry", "name": "steer"}})
	_ = s.Append(Event{Type: TypeUser, SessionID: id, Source: "user", Payload: map[string]any{"text": "b", "id": "b"}})
	out, err := s.OutlineTurns(id)
	if err != nil || len(out) != 2 {
		t.Fatalf("outline %v %d", err, len(out))
	}
	page, err := s.PageAroundUser(id, out[0].Seq, 2)
	if err != nil {
		t.Fatal(err)
	}
	users := operatorTexts(page)
	if len(users) != 2 || users[0] != "a" || users[1] != "b" {
		t.Fatalf("window counted steer as a turn: %v", users)
	}
	steerKept := false
	for _, ev := range page.Events {
		if IsSteerUser(ev) {
			steerKept = true
		}
	}
	if !steerKept {
		t.Fatal("steer inject dropped from the body window")
	}
	page2, err := s.PageTurns(id, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	got := operatorTexts(page2)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("PageTurns counted steer: %v", got)
	}
}

func operatorTexts(page Page) []string {
	var out []string
	for _, ev := range page.Events {
		if ev.Type == TypeUser && !IsSteerUser(ev) {
			out = append(out, payloadString(ev, "text"))
		}
	}
	return out
}

func userTexts(page Page) []string {
	var out []string
	for _, ev := range page.Events {
		if ev.Type == TypeUser {
			out = append(out, payloadString(ev, "text"))
		}
	}
	return out
}
