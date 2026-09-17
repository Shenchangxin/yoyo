package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestMergeProposalsKeepsOrthogonalL1(t *testing.T) {
	cas := artifact.NewStore(t.TempDir())
	loopH, err := cas.Put(artifact.KindLoopPreset, "d", artifact.LoopPreset{ID: "d", MaxTurns: 8, Bootstrap: "old"})
	if err != nil {
		t.Fatal(err)
	}
	pbH, err := cas.Put(artifact.KindPlaybook, "main", artifact.Playbook{
		ID: "main", Bullets: []artifact.PlaybookBullet{{ID: "b1", Text: "base", Helpful: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	base := artifact.HarnessSnapshot{ID: "p", LoopPreset: loopH, Playbook: pbH}
	parent, err := cas.PutSnapshot(base)
	if err != nil {
		t.Fatal(err)
	}
	base, err = cas.GetSnapshot(parent)
	if err != nil {
		t.Fatal(err)
	}
	props := []Proposal{
		{ID: "a", InstructionSlot: "bootstrap", InstructionText: "write a placeholder file first"},
		{ID: "b", PlaybookBullet: &artifact.PlaybookBullet{ID: "b2", Text: "verify early", Helpful: 1}},
	}
	last, _, err := ApplyProposal(cas, base, parent, props[1])
	if err != nil {
		t.Fatal(err)
	}
	lastLoop, _, err := artifact.Decode[artifact.LoopPreset](cas, last.LoopPreset)
	if err != nil {
		t.Fatal(err)
	}
	if lastLoop.Bootstrap != "old" {
		t.Fatal("last-write playbook snapshot should not carry the instruction edit")
	}
	merged, _, err := MergeProposals(cas, base, parent, props)
	if err != nil {
		t.Fatal(err)
	}
	loop, _, err := artifact.Decode[artifact.LoopPreset](cas, merged.LoopPreset)
	if err != nil {
		t.Fatal(err)
	}
	if loop.Bootstrap != "write a placeholder file first" {
		t.Fatalf("bootstrap %q", loop.Bootstrap)
	}
	if loop.MaxTurns != 8 {
		t.Fatalf("topology mutated: %+v", loop)
	}
	pb, _, err := artifact.Decode[artifact.Playbook](cas, merged.Playbook)
	if err != nil {
		t.Fatal(err)
	}
	if len(pb.Bullets) != 2 {
		t.Fatalf("lost orthogonal bullet: %+v", pb.Bullets)
	}
}
