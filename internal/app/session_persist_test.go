package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestUserPersistsBeforeModel(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hang := &blockChat{started: make(chan struct{})}
	errc := make(chan error, 1)
	go func() {
		_, e := a.Send(context.Background(), sess.ID, "durable hello", hang, nil)
		errc <- e
	}()
	select {
	case <-hang.started:
	case <-time.After(5 * time.Second):
		t.Fatal("model was never called")
	}
	evs, err := a.Traces.Read(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var users int
	for _, ev := range evs {
		if ev.Type == trace.TypeUser {
			users++
			if ev.Payload["text"] != "durable hello" {
				t.Fatalf("text %v", ev.Payload["text"])
			}
		}
	}
	if users != 1 {
		t.Fatalf("user rows %d", users)
	}
	_ = a.Interrupt(sess.ID)
	select {
	case <-errc:
	case <-time.After(5 * time.Second):
		t.Fatal("send did not return")
	}
}

func TestCrashResumeDoesNotDuplicateUser(t *testing.T) {
	home := t.TempDir()
	a, err := Open(home, filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.commitUser(sess.ID, "resume me", nil, "turn-1"); err != nil {
		t.Fatal(err)
	}
	a.Close()

	b, err := Open(home, filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	evs, err := b.Traces.Read(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if countUsers(evs) != 1 {
		t.Fatalf("persisted users %d", countUsers(evs))
	}
	if _, err := b.Send(context.Background(), sess.ID, "", runtime.HeuristicSolver{}, nil); err != nil {
		t.Fatal(err)
	}
	evs, err = b.Traces.Read(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if countUsers(evs) != 1 {
		t.Fatalf("duplicate user after continue: %d", countUsers(evs))
	}
}

func countUsers(evs []trace.Event) int {
	n := 0
	for _, ev := range evs {
		if ev.Type == trace.TypeUser {
			n++
		}
	}
	return n
}

func TestTrajectoryPageIsCappedWindow(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: sess.ID, Payload: map[string]any{"text": "u"}})
		_ = a.Traces.Append(trace.Event{Type: trace.TypeAssistant, SessionID: sess.ID, Payload: map[string]any{"text": "a"}})
	}
	page, err := a.TrajectoryPage(sess.ID, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Older {
		t.Fatal("expected older")
	}
	users := 0
	for _, ev := range page.Events {
		if ev.Type == trace.TypeUser {
			users++
		}
	}
	if users != 3 {
		t.Fatalf("users %d", users)
	}
}

func TestTrajectoryOutlineAndAround(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: sess.ID, Payload: map[string]any{"text": filepath.Base(t.Name()) + string(rune('a'+i)), "id": string(rune('a' + i))}})
		_ = a.Traces.Append(trace.Event{Type: trace.TypeAssistant, SessionID: sess.ID, Payload: map[string]any{"text": "a"}})
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: sess.ID, Source: "steer", Payload: map[string]any{"text": "nope", "name": "steer"}})
	out, err := a.TrajectoryOutline(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 6 {
		t.Fatalf("outline %d", len(out))
	}
	page, err := a.TrajectoryAround(sess.ID, out[1].Seq, 3)
	if err != nil {
		t.Fatal(err)
	}
	users := 0
	hit := false
	for _, ev := range page.Events {
		if ev.Type == trace.TypeUser {
			users++
			if ev.Seq == out[1].Seq {
				hit = true
			}
		}
	}
	if users != 3 || !hit {
		t.Fatalf("around users=%d hit=%v", users, hit)
	}
}

func TestStopTurnStates(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	idle := a.StopTurn(sess.ID)
	if idle.State != "idle" {
		t.Fatalf("%s", idle.State)
	}
	hang := &blockChat{started: make(chan struct{})}
	go func() { _, _ = a.Send(context.Background(), sess.ID, "x", hang, nil) }()
	select {
	case <-hang.started:
	case <-time.After(5 * time.Second):
		t.Fatal("hang")
	}
	got := a.StopTurn(sess.ID)
	if got.State != "cancelled" && got.State != "already_done" {
		t.Fatalf("state %s", got.State)
	}
}
