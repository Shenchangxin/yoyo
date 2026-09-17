package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestSendUnknownSessionDoesNotMintID(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	_, err = a.Send(context.Background(), "missing-session", "hello", runtime.HeuristicSolver{}, nil)
	if err == nil {
		t.Fatal("expected unknown session")
	}
}

func TestForkPinsHarness(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	src, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dst, err := a.ForkSession(src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dst.HarnessPolicy != session.Pin {
		t.Fatalf("policy %s", dst.HarnessPolicy)
	}
	if dst.Harness != src.Harness && dst.Harness != a.ActiveHash() {
		t.Fatalf("harness %s", dst.Harness)
	}
}

func TestMaterialsDecodeFailsVisible(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	snap, err := a.LoadSnapshot(a.ActiveHash())
	if err != nil {
		t.Fatal(err)
	}
	snap.LoopPreset = "not-a-hash"
	hash, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, err := a.Materials(hash); err == nil {
		t.Fatal("expected materials decode error")
	}
}

func TestHubSpillOverflow(t *testing.T) {
	dir := t.TempDir()
	h := NewHub()
	h.Overflow = dir
	ch := make(chan trace.Event)
	h.mu.Lock()
	h.subs["s"] = map[chan trace.Event]struct{}{ch: {}}
	h.mu.Unlock()
	h.Publish(trace.Event{SessionID: "s", Type: trace.TypeUser, Payload: map[string]any{"text": "x"}})
	if _, err := os.Stat(filepath.Join(dir, "s.jsonl")); err != nil {
		t.Fatal(err)
	}
	<-ch
}

func TestSeedAdvertisesHostTools(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	snap, err := a.LoadSnapshot(a.ActiveHash())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Tools) == 0 {
		t.Fatal("seed tools empty")
	}
	_ = artifact.KindToolSpec
}
