package evolve

import (
	"context"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func TestAssignAnonIDsStable(t *testing.T) {
	a := assignAnonIDs([]string{"c1", "c2"}, "task", 2, OverlayModeAgg)
	b := assignAnonIDs([]string{"c2", "c1"}, "task", 2, OverlayModeAgg)
	if len(a) != 2 || a["A"] == "" {
		t.Fatalf("%v", a)
	}
	if a["A"] != b["A"] || a["B"] != b["B"] {
		t.Fatalf("unstable %v %v", a, b)
	}
}

func TestSelectionLeakIsTripwire(t *testing.T) {
	if !selectionLeaked("this inv1 edit helped", "abc", []string{"inv1", "c1"}) {
		t.Fatal("inv1")
	}
	if selectionLeaked("output A is complete", "abc", []string{"c1", "c2"}) {
		t.Fatal("false positive")
	}
}

func TestOverlayParallelProposeIdentity(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	cli := dimClient{}
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent:          parent,
		Task:            "write hello.txt",
		K:               2,
		Client:          cli,
		ParallelPropose: true,
		Roller: scriptedRoller(map[string]bool{
			"base1": true, "base2": true,
			"inv1": true, "inv2": true,
			"dom1": true, "dom2": true,
			"c1": true, "c2": true,
		}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	if res.R != 8 || res.P != 6 {
		t.Fatalf("R=%d P=%d", res.R, res.P)
	}
	assertSeedRefs(t, eng)
}

type dimClient struct{}

func (dimClient) Chat(_ context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	var sys string
	for _, m := range req.Messages {
		if m.Role == runtime.RoleSystem {
			sys += m.Content
		}
	}
	switch {
	case strings.Contains(sys, "INV harness"):
		return runtime.Message{Content: `{"surface":"instruction","slot":"verification","text":"Create required output artifacts early and re-read them before stopping.","expected":"x","audit":"a"}`}, nil
	case strings.Contains(sys, "DOM harness"):
		return runtime.Message{Content: `{"surface":"skill","text":"When an output path is named, write it, then re-read and check it against the request before stopping.","expected":"x","audit":"a"}`}, nil
	case strings.Contains(sys, "compose a successor"):
		return runtime.Message{Content: `{"decision":"keep_base","notes":"no contrast"}`}, nil
	default:
		return runtime.Message{Content: "<why>A is complete</why>\n```choice\nA\n```"}, nil
	}
}

func TestOverlaySeqK2Identity(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		K:      2,
		Mode:   OverlayModeSeq,
		Roller: scriptedRoller(map[string]bool{
			"base1": true, "base2": true,
			"inv1": true, "inv2": true,
			"dom1": true, "dom2": true,
			"c1": true, "c2": true,
		}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != OverlayModeSeq {
		t.Fatalf("mode %s", res.Mode)
	}
	if res.R != 8 || res.P != 6 {
		t.Fatalf("R=%d P=%d", res.R, res.P)
	}
	if res.Adopted {
		t.Fatal("same deliverable must keep base")
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	assertSeedRefs(t, eng)
}

func TestOverlaySelectLeakKeepsBase(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	cli := &scriptClient{replies: []string{
		`{"surface":"instruction","slot":"verification","text":"Create required output artifacts early and re-read them before stopping.","expected":"x","audit":"a"}`,
		`{"surface":"skill","text":"When an output path is named, write it, then re-read and check it against the request before stopping.","expected":"x","audit":"a"}`,
		`{"surface":"instruction","slot":"verification","text":"Create required output artifacts early and re-read them before stopping.","expected":"x","audit":"a"}`,
		`{"surface":"skill","text":"When an output path is named, write it, then re-read and check it against the request before stopping.","expected":"x","audit":"a"}`,
		`{"decision":"keep_base","notes":"no contrast"}`,
		`{"decision":"keep_base","notes":"no contrast"}`,
		"<why>inv1 was better</why>\n```choice\nA\n```",
	}}
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		K:      2,
		Client: cli,
		Roller: scriptedRoller(map[string]bool{
			"base1": true, "base2": true,
			"inv1": true, "inv2": true,
			"dom1": true, "dom2": true,
			"c1": true, "c2": true,
		}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Adopted {
		t.Fatalf("leak must not adopt: %+v", res.Selection)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	blob := overlayEvidenceBlob(res)
	if strings.Contains(blob, parent.Hash) && strings.Contains(res.Selection.Why, parent.Hash) {
		t.Fatal("parent hash in why")
	}
	assertSeedRefs(t, eng)
}
