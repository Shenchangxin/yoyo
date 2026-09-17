package evolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

func TestTrialLogOmitsHeldOutBodies(t *testing.T) {
	e := &Engine{TrialRoot: t.TempDir()}
	hash := "deadbeefcafebabe"
	e.logTrial(hash, eval.RunReport{
		Results: []eval.TaskResult{
			{ID: "si-01-alpha", Pass: true},
			{ID: "so-01-uniform", Pass: false, Output: "secret held-out body"},
		},
	}, Proposal{ID: "p1"}, map[string]bool{"so-01-uniform": true})
	dir := filepath.Join(e.TrialRoot, hash)
	if _, err := os.Stat(filepath.Join(dir, "metrics.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "si-01-alpha.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "so-01-uniform.json")); !os.IsNotExist(err) {
		t.Fatal("held-out task body leaked into trial FS")
	}
}
