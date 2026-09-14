package eval

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	rt "github.com/Shenchangxin/yoyo/internal/runtime"
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
	if rep.Metrics.HeldInPass != 1 || rep.Metrics.HeldOutPass != 1 {
		t.Fatalf("%+v", rep)
	}
}
