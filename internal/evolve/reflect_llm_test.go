package evolve

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func TestReflectLLMFallsBackWhenUnusable(t *testing.T) {
	pb := artifact.Playbook{ID: "main", Bullets: []artifact.PlaybookBullet{{ID: "b1", Text: "prefer verifiers"}}}
	got, err := ReflectLLM(context.Background(), runtime.HeuristicSolver{}, "fixture", nil, map[string]string{"t": "missing"}, pb)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Insights) == 0 && got.Mechanism == "" && len(got.BulletTags) == 0 {
		fallback := Reflect(nil, map[string]string{"t": "missing"}, pb)
		if len(fallback.Insights) == 0 && fallback.Mechanism == "" {
			t.Fatalf("empty reflection: %+v", got)
		}
	}
}

func TestSanitizeHeldOutStripsTransferIDs(t *testing.T) {
	p := Proposal{PredictedFixes: []string{"si-01-alpha", "so-01-uniform"}, AtRisk: []string{"st-01-ember"}}
	held := map[string]bool{"so-01-uniform": true, "st-01-ember": true}
	got := sanitizeHeldOut(p, held)
	if len(got.PredictedFixes) != 1 || got.PredictedFixes[0] != "si-01-alpha" {
		t.Fatalf("%+v", got)
	}
	if len(got.AtRisk) != 0 {
		t.Fatalf("at-risk leaked: %+v", got.AtRisk)
	}
}

func TestParentWeightDownweightsTransferFail(t *testing.T) {
	ok := Node{ID: "ok", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true}
	bad := ok
	bad.ID = "bad"
	bad.TransferFail = true
	if ParentWeight(bad, 0) >= ParentWeight(ok, 0) {
		t.Fatal("transfer fail must downweight")
	}
}
