package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestSendAppliesChatHorizon(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	steps := make([]runtime.Message, 0, 34)
	for i := 0; i < 33; i++ {
		steps = append(steps, runtime.Message{
			Role: runtime.RoleAssistant,
			ToolCalls: []runtime.ToolCall{{
				ID:        fmt.Sprintf("c%d", i),
				Name:      "list_dir",
				Arguments: `{"path":"."}`,
			}},
		})
	}
	steps = append(steps, runtime.Message{Role: runtime.RoleAssistant, Content: "done"})
	out, err := a.Send(context.Background(), sess.ID, "list files", &runtime.ScriptedClient{Steps: steps}, nil)
	if err != nil {
		t.Fatalf("chat overlay should keep the loop alive past 32 turns: %v", err)
	}
	if out != "done" {
		t.Fatalf("out %q", out)
	}
}

func TestChatTurnContextHasNoDeadline(t *testing.T) {
	ctx, cancel := chatTurnContext()
	defer cancel()
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("async chat turns must not expire on a wall clock")
	}
}

type blockChat struct {
	started chan struct{}
}

func (b *blockChat) Chat(ctx context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	if b.started != nil {
		select {
		case <-b.started:
		default:
			close(b.started)
		}
	}
	<-ctx.Done()
	return runtime.Message{}, ctx.Err()
}

func TestInterruptCancelsInFlightSend(t *testing.T) {
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
		_, e := a.Send(context.Background(), sess.ID, "hello", hang, nil)
		errc <- e
	}()
	select {
	case <-hang.started:
	case <-time.After(5 * time.Second):
		t.Fatal("model was never called")
	}
	if err := a.Interrupt(sess.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-errc:
		if e == nil {
			t.Fatal("expected cancel")
		}
		if runtime.ReasonOf(e) != runtime.StopCancelled {
			t.Fatalf("reason %s err=%v", runtime.ReasonOf(e), e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("send did not return after interrupt")
	}
}
