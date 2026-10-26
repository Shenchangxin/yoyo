// 锁住父节点抽样偏向子节点更少的胜者，并跳过还没评过的 online-ace-stage。挖掘仍能看到非 held 的失败题。

package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

// TestSelectParentPrefersUnderexploredWinners 要求 BestParent 选子节点更少且已接受的 b，而不是子节点更多的 a。
func TestSelectParentPrefersUnderexploredWinners(t *testing.T) {
	nodes := []Node{
		{ID: "a", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true},
		{ID: "b", Parent: "a", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true},
		{ID: "c", Parent: "a", Metrics: eval.Metrics{HeldInPass: 0, HeldOutPass: 0}},
	}
	got := BestParent(nodes, "a")
	if got != "b" {
		t.Fatalf("got %s", got)
	}
	if ParentWeight(nodes[1], 0) <= ParentWeight(nodes[0], 2) {
		t.Fatal("underexplored winner should weigh more")
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
