// 锁住 Harbor 写下的轨迹带着题号。evolve 的 Mine 按 TaskID 把工具调用归到题上，缺了题号就只会挖出 tools=0 的 missing_artifact。

package eval

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	rt "github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestHarborTraceCarriesTaskID 确认每次重复的会话里，工具调用都带着这道题的 TaskID，且没有事件带着别的题号。
func TestHarborTraceCarriesTaskID(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "evals")
	st := trace.NewStore(t.TempDir())
	rep, err := NewEngine(root).Run(context.Background(), RunOpts{
		Suite: artifact.EvalSuite{
			ID:         "trace-task-id",
			TaskDir:    root,
			HeldIn:     []string{"write-hello"},
			Repeats:    1,
			TimeoutSec: 30,
		},
		Client:    rt.HeuristicSolver{},
		Loop:      rt.DefaultLoop(),
		Model:     "fixture",
		Trace:     st,
		SessionID: "h",
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	for _, r := range rep.Results {
		for _, a := range r.Attempts {
			evs, err := st.Read(a.SessionID)
			if err != nil {
				t.Fatal(err)
			}
			for _, ev := range evs {
				if ev.TaskID != "" && ev.TaskID != r.ID {
					t.Fatalf("event %s carries task %q, want %q", ev.Type, ev.TaskID, r.ID)
				}
				if ev.Type == trace.TypeToolCall {
					if ev.TaskID != r.ID {
						t.Fatalf("tool call without task id: %+v", ev)
					}
					calls++
				}
			}
		}
	}
	if calls == 0 {
		t.Fatal("harbor trace has no tool calls to mine")
	}
}
