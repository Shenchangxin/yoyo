// 锁住 Harbor 跑题：多数通过、安全题不计入 held-in、重复两次都要通过、隔离标记，以及工作区不含 tests/。
// 调用方把 AllowLLMCompact 设为 true 时，轨迹里也不出现聊天检查点。

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

// TestHarborTaskPass 要求 write-hello 与 write-answer 都通过，种类分别是 held_in 和 held_out，且 DefaultLoop 没有被这次运行改掉。
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

// TestOptionalWriteReadmeNotInDefaultSeed 要求单独把 write-readme 放进 HeldIn 时启发式求解能通过。
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

// TestSafetyNoEscapeNotInDefaultSeed 要求只跑 no-escape 时 SafetyFail 为 0，且 HeldInTotal 仍为 0。
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

// TestLogsNoReadSafety 要求 logs-no-read 作为安全题通过，且 SafetyFail 为 0。
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

// TestSealedSafetyIncludesLogsNoRead 要求 ApplySealed 的 Safety 含 logs-no-read，且 SealedHeldOutIDs 非空。
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

// TestRepeatsAndTBSubset 要求 Repeats 为 2 时两题都两次通过，隔离标记为 copy，且每题有两次尝试。
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

// TestIsolateWorktree 要求 IsolateRoot 指向带空提交的 git 仓库时，隔离标记为 git-worktree 且 held-in 通过。
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

// TestHarborShapeOnlyEvenIfAllowLLMCompact 要求调用方把 AllowLLMCompact 设为 true 时，任务轨迹里仍没有 kind 为 checkpoint 的事件，且各次重复的会话编号互不相同。
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

// TestHarborPromptSensitiveUnchanged 要求 PromptSensitiveSolver 在没有占位钉选时 held-in 与 held-out 的通过数都是 0。
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

// TestPrepareWorkOmitsTests 要求 prepareWork 的工作区里没有 tests/，但有 instruction.md。
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

// TestBehaviorProbesPassHeuristic 要求 BehaviorIDs 全部作为 HeldIn 被启发式求解通过。
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

// TestHarborSpendFields 要求报告和单题的 WallMs 都大于 0。
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
