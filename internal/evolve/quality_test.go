// 锁住准入会拒绝 held 编号、仓库概览、照抄说明、多种载荷和 tool_use。宣言未命中时 hit 为 0。同一表面只留得分更高的一条。

package evolve

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/eval"
)

// TestAdmitQualityGates 要求 held 编号、概览、照抄说明、两种载荷和 tool_use 都被拒绝。
func TestAdmitQualityGates(t *testing.T) {
	held := map[string]bool{"so-01-uniform": true, "hi-01-index": true}
	pb := artifact.Playbook{ID: "main"}
	if err := AdmitQuality(Proposal{
		Surface:  "prompt_fragment",
		Fragment: &artifact.PromptFragment{ID: "x", Text: "do not mention so-01-uniform"},
	}, pb, 200, held, nil); err == nil {
		t.Fatal("held-out id")
	}
	if err := AdmitQuality(Proposal{
		Surface:        "playbook",
		PlaybookBullet: &artifact.PlaybookBullet{ID: "ov", Text: "Repository overview of the workspace tree"},
	}, pb, 2000, held, nil); err == nil {
		t.Fatal("overview")
	}
	inst := strings.Repeat("write a unique held-in instruction sentence now ", 3)
	if err := AdmitQuality(Proposal{
		Surface:  "prompt_fragment",
		Fragment: &artifact.PromptFragment{ID: "c", Text: strings.ToLower(inst)},
	}, pb, 2000, held, []string{inst}); err == nil {
		t.Fatal("copied instruction")
	}
	if err := AdmitQuality(Proposal{
		Surface:         "instruction",
		InstructionText: "verify",
		MiddlewareText:  "also middleware",
	}, pb, 2000, held, nil); err == nil {
		t.Fatal("two surfaces")
	}
	if err := AdmitQuality(Proposal{Surface: "tool_use"}, pb, 2000, held, nil); err == nil {
		t.Fatal("tool_use")
	}
}

// TestManifestoHitZero 要求预测要修的题没有通过时命中为 0、未命中为 1。
func TestManifestoHitZero(t *testing.T) {
	p := Proposal{PredictedFixes: []string{"write-hello"}}
	rep := eval.RunReport{Results: []eval.TaskResult{{ID: "write-hello", Pass: false}}}
	hit, miss := scoreManifesto(p, rep, nil)
	if hit != 0 || miss != 1 {
		t.Fatalf("hit=%d miss=%d", hit, miss)
	}
}

// TestExclusiveSurfacesKeepsBest 要求同一 playbook 表面只留通过数更高的提议，instruction 另留一条。
func TestExclusiveSurfacesKeepsBest(t *testing.T) {
	a := Trial{Proposal: Proposal{ID: "a", Surface: "playbook"}, Metrics: eval.Metrics{HeldInPass: 1, HeldOutPass: 0}}
	b := Trial{Proposal: Proposal{ID: "b", Surface: "playbook"}, Metrics: eval.Metrics{HeldInPass: 2, HeldOutPass: 1}}
	c := Trial{Proposal: Proposal{ID: "c", Surface: "instruction"}, Metrics: eval.Metrics{HeldInPass: 1}}
	got := exclusiveSurfaces([]Trial{a, b, c})
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "c" {
		t.Fatalf("%+v", got)
	}
}
