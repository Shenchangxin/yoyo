package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func evalsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(wd, "..", "..", "evals")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestSeedCheckoutRollback(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h1 := a.ActiveHash()
	if h1 == "" {
		t.Fatal("no active")
	}
	loopH, err := a.CAS.Put(artifact.KindLoopPreset, "alt", runtime.DefaultLoop())
	if err != nil {
		t.Fatal(err)
	}
	snap, err := a.LoadSnapshot(h1)
	if err != nil {
		t.Fatal(err)
	}
	snap.Parent = h1
	snap.LoopPreset = loopH
	snap.Note = "child"
	h2, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Checkout(h2); err != nil {
		t.Fatal(err)
	}
	if a.ActiveHash() != h2 {
		t.Fatal(a.ActiveHash())
	}
	if err := a.Rollback(); err != nil {
		t.Fatal(err)
	}
	if a.ActiveHash() != h1 {
		t.Fatalf("got %s want %s", a.ActiveHash(), h1)
	}
}

func TestAgentLoopTrajectory(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	out, err := a.Send(context.Background(), sess.ID, "Write hello.txt containing hello", runtime.HeuristicSolver{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	b, err := os.ReadFile(filepath.Join(ws, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("%q", b)
	}
	evs, err := a.Trajectory(sess.ID)
	if err != nil || len(evs) == 0 {
		t.Fatalf("trajectory %v %d", err, len(evs))
	}
	var sawTool bool
	for _, ev := range evs {
		if ev.Type == trace.TypeToolCall {
			sawTool = true
		}
		if ev.HarnessSnapshot == "" {
			t.Fatal("event missing harness hash")
		}
	}
	if !sawTool {
		t.Fatal("no tool call in trajectory")
	}
}

func TestEvalAndSelfHarnessPromote(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ctx := context.Background()
	solver := runtime.PromptSensitiveSolver{}
	base, err := a.RunEval(ctx, solver)
	if err != nil {
		t.Fatal(err)
	}
	if base.Metrics.HeldInPass != 0 {
		t.Fatalf("baseline should fail without placeholder instruction: %+v", base)
	}
	res, err := a.EvolveOnce(ctx, solver, map[string]string{
		"write-hello":  "missing file",
		"write-answer": "missing file",
	})
	if err != nil {
		t.Fatal(err)
	}
	var accepted bool
	for _, tr := range res.Tried {
		if tr.Accepted {
			accepted = true
		}
	}
	if !accepted && res.Promoted == "" {
		t.Fatalf("expected a promotion, got %+v", res)
	}
	after, err := a.RunEval(ctx, solver)
	if err != nil {
		t.Fatal(err)
	}
	if after.Metrics.HeldInPass+after.Metrics.HeldOutPass == 0 {
		t.Fatalf("evolved harness still failing: %+v", after)
	}
}

func TestWASMAdmitDispose(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	bin := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x07, 0x01, 0x60,
		0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01,
		0x03, 0x61, 0x64, 0x64, 0x00, 0x00, 0x0a, 0x09, 0x01, 0x07, 0x00, 0x20,
		0x00, 0x20, 0x01, 0x6a, 0x0b,
	}
	fiber, err := evolve.AdmitWASM(a.Kernel, a.WASM, "math", bin, "add")
	if err != nil {
		t.Fatal(err)
	}
	if a.WASM.Get("math") == nil {
		t.Fatal("not loaded")
	}
	if err := fiber.Dispose(); err != nil {
		t.Fatal(err)
	}
	if a.WASM.Get("math") != nil {
		t.Fatal("wasm leaked after fiber dispose")
	}
	_ = kernel.New()
	_ = wasm.NewHost()
}

func TestCapabilityJail(t *testing.T) {
	dir := t.TempDir()
	outside, _ := os.MkdirTemp("", "outside-*")
	defer os.RemoveAll(outside)
	tools := &runtime.WorkspaceTools{Workspace: dir}
	res := tools.Call("read_file", `{"path":"`+filepath.ToSlash(filepath.Join(outside, "x.txt"))+`"}`)
	if res.Err == nil {
		t.Fatal("expected jail")
	}
}
