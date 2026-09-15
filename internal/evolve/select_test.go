package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/eval"
)

func TestSelectParentPrefersUnderexploredWinners(t *testing.T) {
	nodes := []Node{
		{ID: "a", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true},
		{ID: "b", Parent: "a", Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 2}, Accepted: true},
		{ID: "c", Parent: "a", Metrics: eval.Metrics{HeldInPass: 0, HeldOutPass: 0}},
	}
	got := SelectParent(nodes, "a")
	if got != "b" {
		t.Fatalf("got %s", got)
	}
}

func TestSelectParentSkipsUnevaluatedStaging(t *testing.T) {
	nodes := []Node{
		{ID: "prod", Metrics: eval.Metrics{HeldInPass: 1, HeldOutPass: 1, HeldInTotal: 1, HeldOutTotal: 1}, Accepted: true},
		{ID: "stage", Parent: "prod", Note: "online-ace-stage"},
	}
	got := SelectParent(nodes, "prod")
	if got != "prod" {
		t.Fatalf("got %s", got)
	}
}

func TestHeldOutFilteredFromEvidence(t *testing.T) {
	b := Mine(nil, map[string]string{"write-hello": "missing"})
	if len(b.Clusters) != 1 || b.Clusters[0].TaskIDs[0] != "write-hello" {
		t.Fatalf("%+v", b)
	}
}
