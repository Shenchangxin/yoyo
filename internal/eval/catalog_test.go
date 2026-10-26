// 锁住密封目录的切分：EvolveIn、HeldIn、HeldOut、Transfer 的编号互不重复，Index 编号只进 Transfer。Repeats 至少为 2。

package eval

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestSealedCatalogSplit 要求四张表的规格字段齐全，且编号不跨表重复。
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
	mark("index", IndexTransfer)
}

// TestIndexTransferDisjoint 要求 ApplyIndexTransfer 之后，Index 编号不在 EvolveIn、HeldIn、HeldOut 里，并且出现在 Transfer 里。
func TestIndexTransferDisjoint(t *testing.T) {
	s := artifact.EvalSuite{}
	ApplySealed(&s)
	ApplyIndexTransfer(&s)
	seen := map[string]bool{}
	for _, id := range append(append(append([]string{}, s.EvolveIn...), s.HeldIn...), s.HeldOut...) {
		seen[id] = true
	}
	for _, spec := range IndexTransfer {
		if seen[spec.ID] {
			t.Fatalf("index id %s leaked into evolve/promote sets", spec.ID)
		}
		found := false
		for _, id := range s.Transfer {
			if id == spec.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing transfer %s", spec.ID)
		}
	}
}

// TestApplySealedRepeatsAndSafety 要求套件编号变成 sealed-v1，EvolveIn、HeldIn、HeldOut、Transfer 按密封切分，Repeats 至少为 2，且安全表含指定题号。
func TestApplySealedRepeatsAndSafety(t *testing.T) {
	s := artifact.EvalSuite{ID: "local-smoke", Repeats: 1}
	ApplySealed(&s)
	if s.ID != "sealed-v1" {
		t.Fatal(s.ID)
	}
	if len(s.HeldIn) != 8 || len(s.EvolveIn) != 12 || len(s.HeldOut) != 10 || len(s.Transfer) != 5 {
		t.Fatalf("%+v", s)
	}
	if s.Repeats < 2 {
		t.Fatalf("repeats %d", s.Repeats)
	}
	found := map[string]bool{}
	for _, id := range s.Safety {
		found[id] = true
	}
	for _, id := range []string{"no-escape", "logs-no-read", "office-xlsx-formula", "research-cite", "mail-no-exfil", "browser-no-paste-secrets", "memory-forget", "schedule-isolation", "page-no-write-without-approve", "profile-cannot-exceed-grants"} {
		if !found[id] {
			t.Fatalf("missing safety %s in %+v", id, s.Safety)
		}
	}
}

// TestMaterializeSealedTask 要求第一道密封 held-in 能物化并被 LoadHarborTask 读回同一题号。
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
