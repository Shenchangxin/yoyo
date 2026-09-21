package eval

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	rt "github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestHarborTaskPass(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "local-smoke",
			TaskDir:    root,
			HeldIn:     []string{"write-hello"},
			HeldOut:    []string{"write-answer"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rt.DefaultLoop().MaxTurns != 32 || rt.DefaultLoop().MaxToolMessages != 40 {
		t.Fatalf("harbor loop overlay leaked: %+v", rt.DefaultLoop())
	}
	if rep.Metrics.HeldInPass != 1 || rep.Metrics.HeldOutPass != 1 {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Results) != 2 {
		t.Fatalf("results=%d", len(rep.Results))
	}
	if rep.Results[0].Kind != "held_in" || rep.Results[1].Kind != "held_out" {
		t.Fatalf("kind %+v", rep.Results)
	}
}

func TestOptionalWriteReadmeNotInDefaultSeed(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "opt-in-readme",
			TaskDir:    root,
			HeldIn:     []string{"write-readme"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInPass != 1 {
		t.Fatalf("%+v", rep)
	}
}

func TestSafetyNoEscapeNotInDefaultSeed(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "safety-only",
			TaskDir:    root,
			Safety:     []string{"no-escape"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.SafetyFail != 0 {
		t.Fatalf("safety fail=%+v", rep)
	}
	if rep.Metrics.HeldInTotal != 0 {
		t.Fatalf("safety-only must not count as held-in: %+v", rep)
	}
	if len(rep.Results) != 1 || !rep.Results[0].Pass {
		t.Fatalf("%+v", rep)
	}
}

func TestLogsNoReadSafety(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "logs-no-read",
			TaskDir:    root,
			Safety:     []string{"logs-no-read"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.SafetyFail != 0 {
		t.Fatalf("safety fail=%+v", rep)
	}
	if len(rep.Results) != 1 || !rep.Results[0].Pass {
		t.Fatalf("%+v", rep)
	}
}

func TestSealedSafetyIncludesLogsNoRead(t *testing.T) {
	var s artifact.EvalSuite
	ApplySealed(&s)
	found := false
	for _, id := range s.Safety {
		if id == "logs-no-read" {
			found = true
		}
	}
	if !found {
		t.Fatalf("sealed safety missing logs-no-read: %v", s.Safety)
	}
	if len(SealedHeldOutIDs()) == 0 {
		t.Fatal("held-out ids empty")
	}
}

func TestRepeatsAndTBSubset(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "tb-subset",
			TaskDir:    root,
			HeldIn:     []string{"mkdir-note"},
			HeldOut:    []string{"copy-seed"},
			Repeats:    2,
			TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInPass != 1 || rep.Metrics.HeldOutPass != 1 {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Results) != 2 {
		t.Fatalf("results=%d", len(rep.Results))
	}
	for _, r := range rep.Results {
		if r.Repeats != 2 || r.RepeatsPass != 2 {
			t.Fatalf("repeats %+v", r)
		}
		if r.Isolate != "copy" {
			t.Fatalf("isolate %q", r.Isolate)
		}
		if len(r.Attempts) != 2 {
			t.Fatalf("attempts %+v", r)
		}
	}
}

func TestIsolateWorktree(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	gitRoot := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = gitRoot
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=yoyo", "GIT_AUTHOR_EMAIL=yoyo@local", "GIT_COMMITTER_NAME=yoyo", "GIT_COMMITTER_EMAIL=yoyo@local")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v: %s", args, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "yoyo@local")
	run("config", "user.name", "yoyo")
	run("commit", "--allow-empty", "-m", "seed")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "wt",
			TaskDir:    root,
			HeldIn:     []string{"write-hello"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client:      rt.HeuristicSolver{},
		Loop:        rt.DefaultLoop(),
		Model:       "fixture",
		IsolateRoot: gitRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInPass != 1 {
		t.Fatalf("%+v", rep)
	}
	if rep.Results[0].Isolate != "git-worktree" {
		t.Fatalf("isolate %q", rep.Results[0].Isolate)
	}
}

func TestHarborShapeOnlyEvenIfAllowLLMCompact(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	loop := rt.DefaultLoop()
	loop.AllowLLMCompact = true
	st := trace.NewStore(t.TempDir())
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "compact-resume",
			TaskDir:    root,
			HeldIn:     []string{"write-hello"},
			HeldOut:    []string{"write-answer"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client:    rt.HeuristicSolver{},
		Loop:      loop,
		Model:     "fixture",
		Trace:     st,
		SessionID: "h",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInPass != 1 || rep.Metrics.HeldOutPass != 1 {
		t.Fatalf("%+v", rep)
	}
	seen := map[string]bool{}
	for _, r := range rep.Results {
		for _, a := range r.Attempts {
			if a.SessionID == "" {
				t.Fatalf("missing attempt session %+v", r)
			}
			if seen[a.SessionID] {
				t.Fatalf("duplicate session %s", a.SessionID)
			}
			seen[a.SessionID] = true
			evs, err := st.Read(a.SessionID)
			if err != nil {
				t.Fatal(err)
			}
			for _, ev := range evs {
				kind, _ := ev.Payload["kind"].(string)
				if kind == "checkpoint" {
					t.Fatal("harbor persisted a chat checkpoint")
				}
			}
		}
	}
	if _, err := st.Read("h"); err == nil {
		evs, _ := st.Read("h")
		if len(evs) > 0 {
			t.Fatal("legacy session id should not receive task traces")
		}
	}
}

func TestHarborPromptSensitiveUnchanged(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "sensitive",
			TaskDir:    root,
			HeldIn:     []string{"write-hello"},
			HeldOut:    []string{"write-answer"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client: rt.PromptSensitiveSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInPass != 0 || rep.Metrics.HeldOutPass != 0 {
		t.Fatalf("PromptSensitiveSolver must stay unarmed without placeholder pin: %+v", rep)
	}
}

func TestPrepareWorkOmitsTests(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "instruction.md"), []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "tests", "secret.txt"), []byte("leak"), 0o644); err != nil {
		t.Fatal(err)
	}
	work, cleanup, _, err := prepareWork(src, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(work, "tests")); err == nil {
		t.Fatal("grader leaked into agent worktree")
	}
	if _, err := os.Stat(filepath.Join(work, "instruction.md")); err != nil {
		t.Fatal(err)
	}
}

func TestBehaviorProbesPassHeuristic(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "behavior",
			TaskDir:    root,
			HeldIn:     BehaviorIDs,
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{},
		Loop:   rt.DefaultLoop(),
		Model:  "fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Metrics.HeldInPass != len(BehaviorIDs) {
		t.Fatalf("%+v", rep)
	}
}

func TestHarborSpendFields(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	e := NewEngine(root)
	meter := &rt.Meter{USDPerMTok: 1}
	rep, err := e.Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID: "spend", TaskDir: root, HeldIn: []string{"write-hello"}, Repeats: 1, TimeoutSec: 30,
		},
		Client: rt.HeuristicSolver{}, Loop: rt.DefaultLoop(), Model: "fixture", Meter: meter,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.WallMs <= 0 || len(rep.Results) != 1 || rep.Results[0].WallMs <= 0 {
		t.Fatalf("wall %+v", rep)
	}
}
