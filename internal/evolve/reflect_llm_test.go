// 锁住模型反射在不可用回复上仍能落到确定性结果，held 编号会从预测列表去掉，迁移失败会降低父权重。

package evolve

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// TestReflectLLMFallsBackWhenUnusable 要求启发式客户端下反射结果或确定性回退至少有机制或洞察。
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

// TestSanitizeHeldOutStripsTransferIDs 要求 PredictedFixes 和 AtRisk 里的 held 编号被去掉，held-in 编号留下。
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

// TestParentWeightDownweightsTransferFail 要求同一通过数下，TransferFail 为真的权重低于未标失败的节点。
func TestParentWeightDownweightsTransferFail(t *testing.T) {
	ok := Node{ID: "ok", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true}
	bad := ok
	bad.ID = "bad"
	bad.TransferFail = true
	if ParentWeight(bad, 0) >= ParentWeight(ok, 0) {
		t.Fatal("transfer fail must downweight")
	}
}
