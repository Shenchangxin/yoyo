package eval

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestSealedCatalogSplit(t *testing.T) {
	if len(SealedHeldIn) < 20 {
		t.Fatalf("held-in %d", len(SealedHeldIn))
	}
	if len(SealedHeldOut) < 10 {
		t.Fatalf("held-out %d", len(SealedHeldOut))
	}
	if len(SealedTransfer) < 5 {
		t.Fatalf("transfer %d", len(SealedTransfer))
	}
	seen := map[string]string{}
	mark := func(kind string, specs []TaskSpec) {
		t.Helper()
		for _, s := range specs {
			if s.ID == "" || s.File == "" || s.Contains == "" {
				t.Fatalf("incomplete spec %+v", s)
			}
			if prev, ok := seen[s.ID]; ok {
				t.Fatalf("id %s in %s and %s", s.ID, prev, kind)
			}
			seen[s.ID] = kind
		}
	}
	mark("held_in", SealedHeldIn)
	mark("held_out", SealedHeldOut)
	mark("transfer", SealedTransfer)
}

func TestApplySealedRepeatsAndSafety(t *testing.T) {
	s := artifact.EvalSuite{ID: "local-smoke", Repeats: 1}
	ApplySealed(&s)
	if s.ID != "sealed-v1" {
		t.Fatal(s.ID)
	}
	if len(s.HeldIn) != 20 || len(s.HeldOut) != 10 || len(s.Transfer) != 5 {
		t.Fatalf("%+v", s)
	}
	if s.Repeats < 2 {
		t.Fatalf("repeats %d", s.Repeats)
	}
	found := false
	for _, id := range s.Safety {
		if id == "no-escape" {
			found = true
		}
	}
	if !found {
		t.Fatalf("safety %+v", s.Safety)
	}
}

func TestMaterializeSealedTask(t *testing.T) {
	e := NewEngine(t.TempDir())
	spec := SealedHeldIn[0]
	dir := e.resolveTaskDir(e.SuitesRoot, spec.ID)
	task, err := LoadHarborTask(dir)
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != spec.ID {
		t.Fatalf("id %s", task.ID)
	}
}
