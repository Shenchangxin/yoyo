// 锁住父节点抽样偏向子节点更少的胜者，并跳过还没评过的 online-ace-stage。挖掘仍能看到非 held 的失败题。

package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

// TestSelectParentPrefersUnderexploredWinners 要求子节点更少的已接受胜者 b 权重高于子节点更多的 a，零通过又未接受的 c 权重最低。
// SelectParentSeeded 按这份权重抽样，所以权重的高低就是抽中的偏向。
func TestSelectParentPrefersUnderexploredWinners(t *testing.T) {
	nodes := []Node{
		{ID: "a", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true},
		{ID: "b", Parent: "a", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true},
		{ID: "c", Parent: "a", Metrics: eval.Metrics{HeldInPass: 0, HeldOutPass: 0}},
	}
	if ParentWeight(nodes[1], 0) <= ParentWeight(nodes[0], 2) {
		t.Fatal("underexplored winner should weigh more")
	}
	if ParentWeight(nodes[2], 0) >= ParentWeight(nodes[1], 0) {
		t.Fatal("unaccepted zero-pass node should weigh least")
	}
}

// TestSelectParentSkipsUnevaluatedStaging 要求没有评测总数的 online-ace-stage 不被选成父节点。
func TestSelectParentSkipsUnevaluatedStaging(t *testing.T) {
	nodes := []Node{
		{ID: "prod", Metrics: eval.Metrics{HeldInPass: 1, HeldOutPass: 1, HeldInTotal: 1, HeldOutTotal: 1}, Accepted: true},
		{ID: "stage", Parent: "prod", Note: "online-ace-stage"},
	}
	got := SelectParentSeeded(nodes, "prod", 1)
	if got != "prod" {
		t.Fatalf("got %s", got)
	}
}

// TestHeldOutFilteredFromEvidence 要求传入的失败题仍能形成一簇。本测试没有把该题放进 held。
func TestHeldOutFilteredFromEvidence(t *testing.T) {
	b := Mine(nil, map[string]string{"write-hello": "missing"})
	if len(b.Clusters) != 1 || b.Clusters[0].TaskIDs[0] != "write-hello" {
		t.Fatalf("%+v", b)
	}
}
