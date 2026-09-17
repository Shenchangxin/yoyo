package app_test

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
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
	if err := a.CheckoutOpts(h2, app.CheckoutOpts{ConfirmL3: true}); err != nil {
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
	out2, err := a.Send(context.Background(), sess.ID, "confirm the file", runtime.HeuristicSolver{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = out2
	evs2, err := a.Trajectory(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var users int
	for _, ev := range evs2 {
		if ev.Type == trace.TypeUser {
			users++
		}
	}
	if users < 2 {
		t.Fatalf("expected continued session, user events=%d", users)
	}
	active := a.ActiveHash()
	if staged := a.Refs.GetOrEmpty(artifact.RefStaging); staged != "" && staged == active {
		t.Fatal("online ACE staging must not move refs/active")
	}
}

func TestContextUsageSurvivesReopen(t *testing.T) {
	home := t.TempDir()
	a, err := app.Open(home, evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	fresh := a.ContextUsage(sess.ID)
	if fresh.Tokens <= 0 || fresh.PrefixTokens <= 0 || fresh.SchemaTokens <= 0 || fresh.Window <= 0 {
		t.Fatalf("idle usage %+v", fresh)
	}
	if _, err := a.Send(context.Background(), sess.ID, "Write hello.txt containing hello", runtime.HeuristicSolver{}, nil); err != nil {
		t.Fatal(err)
	}
	used := a.ContextUsage(sess.ID)
	if used.Tokens <= fresh.Tokens {
		t.Fatalf("turn should grow usage: idle=%d after=%d", fresh.Tokens, used.Tokens)
	}
	a.Close()
	a2, err := app.Open(home, evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a2.Close()
	again := a2.ContextUsage(sess.ID)
	if again.Tokens <= 0 || again.PrefixTokens <= 0 || again.Window <= 0 {
		t.Fatalf("after reopen %+v", again)
	}
	if again.Tokens < used.Tokens/2 {
		t.Fatalf("reopen lost context: before=%d after=%d", used.Tokens, again.Tokens)
	}
}

func TestSessionTraceIncludesSpillArtifact(t *testing.T) {
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
	body := strings.Repeat("artifact-bytes-", 200)
	sp := runtime.BindSpill(a.Home.Root, ws, sess.ID)
	if sp == nil {
		t.Fatal("spill")
	}
	if id := sp.Put("c1", body); id != "c1" {
		t.Fatalf("put %q", id)
	}
	_ = a.Traces.Append(trace.Event{Type: trace.TypeUser, SessionID: sess.ID, Payload: map[string]any{"text": "dump", "id": "u1"}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeToolCall, SessionID: sess.ID, Payload: map[string]any{"id": "c1", "name": "shell", "arguments": `{"cmd":"cat"}`}})
	_ = a.Traces.Append(trace.Event{Type: trace.TypeToolResult, SessionID: sess.ID, Payload: map[string]any{
		"id": "c1", "name": "shell", "content": body[:80], "spill_id": "c1", "bytes": len(body), "elapsed_ms": 7,
	}})
	tr, err := a.SessionTrace(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Stats.ToolCalls != 1 || tr.Stats.Users != 1 {
		t.Fatalf("stats %+v", tr.Stats)
	}
	var sawSpill bool
	for _, art := range tr.Artifacts {
		if art.Kind == "spill" && art.ID == "c1" && art.Bytes == len(body) {
			sawSpill = true
		}
	}
	if !sawSpill {
		t.Fatalf("artifacts %+v", tr.Artifacts)
	}
	blob, err := a.SpillBlob(sess.ID, "c1")
	if err != nil || blob.Text != body || blob.Truncated {
		t.Fatalf("blob err=%v len=%d trunc=%v", err, len(blob.Text), blob.Truncated)
	}
}

func TestCheckoutRequiresL3(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h1 := a.ActiveHash()
	loop := runtime.DefaultLoop()
	loop.MaxTurns = 99
	loopH, err := a.CAS.Put(artifact.KindLoopPreset, "alt", loop)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := a.LoadSnapshot(h1)
	if err != nil {
		t.Fatal(err)
	}
	snap.Parent = h1
	snap.LoopPreset = loopH
	h2, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Checkout(h2)
	if _, ok := app.AsL3(err); !ok {
		t.Fatalf("want L3, got %v", err)
	}
	if a.ActiveHash() != h1 {
		t.Fatal("checkout without L3 moved active")
	}
	if err := a.CheckoutOpts(h2, app.CheckoutOpts{ConfirmL3: true}); err != nil {
		t.Fatal(err)
	}
	if a.ActiveHash() != h2 {
		t.Fatal(a.ActiveHash())
	}
	if err := a.Rollback(); err != nil {
		t.Fatal(err)
	}
	if a.ActiveHash() != h1 {
		t.Fatalf("rollback should bypass L3, got %s", a.ActiveHash())
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
	active := a.ActiveHash()
	res, err := a.EvolveWith(ctx, solver, map[string]string{
		"write-hello":  "missing file",
		"write-answer": "missing file",
	}, app.EvolveRun{K: 3, PromoteRepeats: 1})
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
	if res.ActiveMoved {
		t.Fatal("default evolve must not move refs/active")
	}
	if a.ActiveHash() != active {
		t.Fatal("evolve moved active")
	}
	canary := a.Refs.GetOrEmpty(artifact.RefCanary)
	if canary == "" {
		canary = res.Promoted
	}
	if canary == "" {
		t.Fatal("missing canary")
	}
	if err := a.Checkout(canary); err != nil {
		t.Fatal(err)
	}
	after, err := a.RunEval(ctx, solver)
	if err != nil {
		t.Fatal(err)
	}
	if after.Metrics.HeldInPass+after.Metrics.HeldOutPass == 0 {
		t.Fatalf("evolved harness still failing: %+v", after)
	}
	for _, c := range res.Evidence.Clusters {
		for _, id := range c.TaskIDs {
			if id == "write-answer" {
				t.Fatal("held-out task leaked into proposer evidence")
			}
		}
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

func TestForkRenameAndPlaybookThumb(t *testing.T) {
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
	if _, err := a.Send(context.Background(), sess.ID, "Write hello.txt containing hello", runtime.HeuristicSolver{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.RenameSession(sess.ID, "pinned"); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetSession(sess.ID)
	if err != nil || got.Title != "pinned" {
		t.Fatalf("%+v %v", got, err)
	}
	fork, err := a.ForkSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fork.ID == sess.ID {
		t.Fatal("fork reused id")
	}
	evs, err := a.Trajectory(fork.ID)
	if err != nil || len(evs) == 0 {
		t.Fatalf("fork trajectory %v %d", err, len(evs))
	}
	pb, err := a.Playbook()
	if err != nil || len(pb.Bullets) == 0 {
		t.Fatalf("%+v %v", pb, err)
	}
	before := pb.Bullets[0].Helpful
	active := a.ActiveHash()
	next, err := a.RatePlaybook(pb.Bullets[0].ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if next.Bullets[0].Helpful != before+1 {
		t.Fatalf("helpful %d -> %d", before, next.Bullets[0].Helpful)
	}
	if a.ActiveHash() != active {
		t.Fatal("thumbs must not move refs/active")
	}
	st := a.Refs.GetOrEmpty(artifact.RefStaging)
	if st == "" || st == active {
		t.Fatal("thumbs must write refs/staging")
	}
}

func TestBestOfModelsPrefersWorkingSolver(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	rep, err := a.BestOfModels(context.Background(), []string{"fail", "ok"}, map[string]runtime.Client{
		"fail": runtime.FailingSolver{},
		"ok":   runtime.HeuristicSolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Kind != "models" || rep.BestModel != "ok" {
		t.Fatalf("%+v", rep)
	}
	if rep.Best.Metrics.HeldInPass+rep.Best.Metrics.HeldOutPass == 0 {
		t.Fatalf("winner should pass tasks: %+v", rep.Best)
	}
}

func TestCheckUpdateNoURL(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	out := a.CheckUpdate()
	if out["staged"] != false {
		t.Fatalf("%+v", out)
	}
	if _, ok := out["staging"]; !ok {
		t.Fatal("missing staging")
	}
}

func TestSaveConfigWritesKeymap(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.Config.Keymap = map[string]string{"palette": "Mod+P"}
	if err := a.SaveConfig(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(a.Home.Root, "keymap.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Mod+P") {
		t.Fatalf("%s", b)
	}
}

func TestC5ForkRecallHitsCopiedSpill(t *testing.T) {
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
	sp := runtime.BindSpill(a.Home.Root, ws, sess.ID)
	sp.Put("fat", "secret-full-bytes")
	fork, err := a.ForkSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{
		Workspace: fork.Workspace,
		Spill:     runtime.BindSpill(a.Home.Root, fork.Workspace, fork.ID),
	}
	res := tools.Call("recall_context", `{"id":"fat"}`)
	if res.Err != nil || !strings.Contains(res.Content, "secret-full-bytes") {
		t.Fatalf("%+v", res)
	}
}

func TestC10DeleteSessionRemovesContext(t *testing.T) {
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
	if _, err := a.Send(context.Background(), sess.ID, "Write hello.txt containing hello", runtime.HeuristicSolver{}, nil); err != nil {
		t.Fatal(err)
	}
	sp := runtime.BindSpill(a.Home.Root, ws, sess.ID)
	sp.Put("notes", "## Objective\nclean me")
	sp.Put("x", "bytes")
	runtime.WriteDiscoverIndex(ws, sess.ID, sp)
	id := sess.ID
	if err := a.DeleteSession(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.Home.SessionSpill(id)); !os.IsNotExist(err) {
		t.Fatal("home spill remains")
	}
	if _, err := os.Stat(filepath.Join(ws, ".yoyo", "context", id)); !os.IsNotExist(err) {
		t.Fatal("workspace context remains")
	}
	if _, err := os.Stat(filepath.Join(a.Home.Sessions(), id+".meta.json")); !os.IsNotExist(err) {
		t.Fatal("meta remains")
	}
	if _, err := os.Stat(filepath.Join(a.Home.Sessions(), id+".jsonl")); !os.IsNotExist(err) {
		t.Fatal("jsonl remains")
	}
}

func TestCheckoutInstructionSkipsL3(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h1 := a.ActiveHash()
	loop := runtime.DefaultLoop()
	loop.Bootstrap = "write a placeholder file first"
	loopH, err := a.CAS.Put(artifact.KindLoopPreset, "copy", loop)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := a.LoadSnapshot(h1)
	if err != nil {
		t.Fatal(err)
	}
	snap.Parent = h1
	snap.LoopPreset = loopH
	h2, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Checkout(h2); err != nil {
		t.Fatalf("instruction-only checkout should skip L3: %v", err)
	}
	if a.ActiveHash() != h2 {
		t.Fatal(a.ActiveHash())
	}
}

func TestSeedIncludesSafety(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	_, _, _, _, suite, _, err := a.Materials(a.ActiveHash())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range suite.Safety {
		if id == "no-escape" {
			found = true
		}
	}
	if !found {
		t.Fatalf("seed missing safety: %+v", suite)
	}
}

func TestUnsignedWASMRejectedOnCheckout(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h1 := a.ActiveHash()
	ph, err := a.CAS.Put(artifact.KindWASMPlugin, "math", artifact.WASMPlugin{ID: "math", Export: "add"})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := a.LoadSnapshot(h1)
	if err != nil {
		t.Fatal(err)
	}
	snap.Parent = h1
	snap.WASMPlugins = []string{ph}
	h2, err := a.CAS.PutSnapshot(snap)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Checkout(h2); err == nil {
		t.Fatal("unsigned wasm checked out")
	}
	if a.ActiveHash() != h1 {
		t.Fatal("unsigned wasm moved active")
	}
}

func TestSignedWASMStagesNotActive(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	a.SetWASMPublicKey(pub)
	a.Config.AutoAllow = true
	bin := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x07, 0x01, 0x60,
		0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01,
		0x03, 0x61, 0x64, 0x64, 0x00, 0x00, 0x0a, 0x09, 0x01, 0x07, 0x00, 0x20,
		0x00, 0x20, 0x01, 0x6a, 0x0b,
	}
	sig := wasm.Sign(priv, bin)
	active := a.ActiveHash()
	if err := a.StageSignedWASM("math", bin, "add", sig); err != nil {
		t.Fatal(err)
	}
	if a.ActiveHash() != active {
		t.Fatal("load wasm moved active")
	}
	st := a.Refs.GetOrEmpty(artifact.RefStaging)
	if st == "" {
		t.Fatal("expected staging snapshot")
	}
	if err := a.Checkout(st); err != nil {
		t.Fatal(err)
	}
}

func TestSealedSuiteCounts(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	rep, err := a.RunEvalMut(context.Background(), runtime.HeuristicSolver{}, func(suite *artifact.EvalSuite) {
		eval.ApplySealed(suite)
		suite.Repeats = 1
		suite.HeldIn = suite.HeldIn[:1]
		suite.HeldOut = suite.HeldOut[:1]
		suite.Transfer = nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInTotal != 1 || rep.Metrics.HeldOutTotal != 1 {
		t.Fatalf("sealed materialize %+v", rep)
	}
	if rep.Metrics.HeldInPass != 1 || rep.Metrics.HeldOutPass != 1 {
		t.Fatalf("heuristic should pass sealed writes: %+v", rep)
	}
}

func TestEvolvePromoteFlagMovesActive(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ctx := context.Background()
	solver := runtime.PromptSensitiveSolver{}
	active := a.ActiveHash()
	res, err := a.EvolveWith(ctx, solver, map[string]string{"write-hello": "missing file"}, app.EvolveRun{
		K: 3, PromoteActive: true, PromoteRepeats: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Promoted == "" {
		t.Fatalf("expected a promotion, got %+v", res)
	}
	if !res.ActiveMoved {
		t.Fatal("explicit --promote must move refs/active")
	}
	if a.ActiveHash() == active {
		t.Fatal("active unchanged despite PromoteActive")
	}
}
