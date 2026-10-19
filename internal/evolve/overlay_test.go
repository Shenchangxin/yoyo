package evolve

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/journal"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func seedOverlayParent(t *testing.T) (MaterialSet, *Engine) {
	t.Helper()
	dir := t.TempDir()
	cas := artifact.NewStore(dir + "/cas")
	refs := artifact.NewRefs(dir + "/refs")
	if err := refs.Set(artifact.RefActive, "seed-active"); err != nil {
		t.Fatal(err)
	}
	if err := refs.Set(artifact.RefCanary, "seed-canary"); err != nil {
		t.Fatal(err)
	}
	if err := refs.Set(artifact.RefStaging, "seed-staging"); err != nil {
		t.Fatal(err)
	}
	loopH, err := cas.Put(artifact.KindLoopPreset, "d", artifact.LoopPreset{ID: "d", MaxTurns: 8, PlaybookTokens: 2000, Bootstrap: "boot", Verification: "old"})
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
	hash, err := cas.PutSnapshot(base)
	if err != nil {
		t.Fatal(err)
	}
	base, err = cas.GetSnapshot(hash)
	if err != nil {
		t.Fatal(err)
	}
	j, err := journal.Open(dir + "/journal")
	if err != nil {
		t.Fatal(err)
	}
	eng := &Engine{CAS: cas, Refs: refs, Journal: j}
	loop, _, err := artifact.Decode[artifact.LoopPreset](cas, loopH)
	if err != nil {
		t.Fatal(err)
	}
	pb, _, err := artifact.Decode[artifact.Playbook](cas, pbH)
	if err != nil {
		t.Fatal(err)
	}
	return MaterialSet{Snap: base, Hash: hash, Loop: loop, Playbook: pb}, eng
}

func scriptedRoller(deliver map[string]bool, artifact string) OverlayRoller {
	return func(_ context.Context, label string, _ MaterialSet) (OverlayRollout, error) {
		ok, known := deliver[label]
		if !known {
			ok = true
		}
		out := artifact
		if out == "" {
			out = label
		}
		return OverlayRollout{
			Delivered:   ok,
			Trajectory:  "ACTION write " + label,
			Deliverable: out,
		}, nil
	}
}

type scriptClient struct {
	replies []string
	i       int
}

func (s *scriptClient) Chat(_ context.Context, _ runtime.ChatRequest) (runtime.Message, error) {
	if len(s.replies) == 0 {
		return runtime.Message{}, nil
	}
	if s.i >= len(s.replies) {
		return runtime.Message{Content: s.replies[len(s.replies)-1]}, nil
	}
	c := s.replies[s.i]
	s.i++
	return runtime.Message{Content: c}, nil
}

func TestAdmitOverlayAlwaysOnTriggerAndCap(t *testing.T) {
	pb := artifact.Playbook{ID: "main"}
	if err := AdmitOverlay(Proposal{
		Surface:         "instruction",
		InstructionSlot: "verification",
		InstructionText: "When the user names an output path, create it before stopping.",
	}, "INV", pb, 2000, nil, nil); err == nil {
		t.Fatal("trigger must be skill")
	}
	if err := AdmitOverlay(Proposal{
		Surface:         "instruction",
		InstructionSlot: "verification",
		InstructionText: strings.Repeat("keep going ", 400),
	}, "INV", pb, 2000, nil, nil); err == nil {
		t.Fatal("always-on cap")
	}
	if err := AdmitOverlay(Proposal{
		Surface:         "instruction",
		InstructionSlot: "verification",
		InstructionText: "Create required output artifacts early and re-read them before stopping.",
	}, "INV", pb, 2000, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAdmitOverlayDOMPlaybookRejected(t *testing.T) {
	pb := artifact.Playbook{ID: "main"}
	if err := AdmitOverlay(Proposal{
		Surface:        "playbook",
		PlaybookBullet: &artifact.PlaybookBullet{ID: "x", Text: "the fee rose to 250 on Jan 1"},
	}, "DOM", pb, 2000, nil, nil); err == nil {
		t.Fatal("dom playbook")
	}
	if err := AdmitOverlay(Proposal{
		Surface:         "instruction",
		InstructionSlot: "verification",
		InstructionText: "Create required output artifacts early and re-read them before stopping.",
	}, "INV", pb, 2000, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestOverlayKeepBaseDoesNotMoveRefs(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		Roller: scriptedRoller(map[string]bool{"base": true, "inv1": true, "dom1": true, "c1": true}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Adopted || !res.KeptBase {
		t.Fatalf("want keep_base: %+v", res)
	}
	if res.Hash != parent.Hash {
		t.Fatalf("hash %s parent %s", res.Hash, parent.Hash)
	}
	assertSeedRefs(t, eng)
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
}

func TestOverlayAdoptDoesNotMoveRefs(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		Roller: scriptedRoller(map[string]bool{"base": false, "inv1": true, "dom1": false, "c1": true}, ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Adopted || res.KeptBase {
		t.Fatalf("want adopt: %+v", res)
	}
	if res.Hash == "" || res.Hash == parent.Hash {
		t.Fatalf("expected child snapshot, got %s", res.Hash)
	}
	if res.INV.Surface != "instruction" {
		t.Fatalf("inv surface %s", res.INV.Surface)
	}
	succ, err := eng.CAS.GetSnapshot(res.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if succ.Parent != parent.Hash {
		t.Fatalf("parent %s", succ.Parent)
	}
	loop, _, err := artifact.Decode[artifact.LoopPreset](eng.CAS, succ.LoopPreset)
	if err != nil {
		t.Fatal(err)
	}
	if !artifact.LoopTopologyEqual(parent.Loop, loop) {
		t.Fatalf("topology %+v vs %+v", parent.Loop, loop)
	}
	assertSeedRefs(t, eng)
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
}

func TestOverlayLedgerIdentityCachedBase(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	base := OverlayRollout{Delivered: false, Trajectory: "base miss", Deliverable: ""}
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent:    parent,
		Task:      "write hello.txt",
		ReuseBase: &base,
		Roller:    scriptedRoller(map[string]bool{"inv1": true, "dom1": false, "c1": true}, "hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ledger.R() != OverlayR || res.Ledger.P() != OverlayP {
		t.Fatalf("R=%d P=%d", res.Ledger.R(), res.Ledger.P())
	}
	if !res.Rollouts[0].Cached {
		t.Fatal("base should be cached")
	}
	assertSeedRefs(t, eng)
}

func TestOverlayHeldOutNotInEvidence(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent:  parent,
		Task:    "do not solve write-answer; write hello.txt",
		HeldOut: []string{"write-answer"},
		Roller:  scriptedRoller(map[string]bool{"base": true, "inv1": true, "dom1": true, "c1": true}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	blob := overlayEvidenceBlob(res)
	if strings.Contains(blob, "write-answer") {
		t.Fatalf("held-out leaked: %s", blob)
	}
}

func TestOverlayInvalidINVInheritsAndHitsBudget(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	cli := &scriptClient{replies: []string{
		`{"surface":"playbook","text":"remember the fee is 250"}`,
		`{"surface":"playbook","text":"remember the fee is 250"}`,
		`{"surface":"skill","text":"When an output path is named, write it, then re-read and check it against the request before stopping."}`,
		`{"decision":"keep_base","notes":"no contrast"}`,
	}}
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		Client: cli,
		Roller: scriptedRoller(map[string]bool{"base": true, "inv1": true, "dom1": true, "c1": true}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	fell := false
	for _, x := range res.FellBack {
		if x == "inv" {
			fell = true
		}
	}
	if !fell {
		t.Fatalf("expected inv inherit: %+v", res.FellBack)
	}
	if res.SessionAttempts < OverlayP+1 {
		t.Fatalf("session_attempts %d want >= %d", res.SessionAttempts, OverlayP+1)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	if res.Adopted {
		t.Fatal("playbook inv must not adopt")
	}
	assertSeedRefs(t, eng)
}

func TestOverlayK2IdentityAndFrozenRefs(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		K:      2,
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
	if res.Adopted || !res.KeptBase {
		t.Fatalf("want keep_base: %+v", res)
	}
	if res.K != 2 || res.R != 8 || res.P != 6 {
		t.Fatalf("k=%d R=%d P=%d", res.K, res.R, res.P)
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	assertSeedRefs(t, eng)
}

func TestOverlayK2AdoptDoesNotMoveRefs(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		K:      2,
		Roller: scriptedRoller(map[string]bool{
			"base1": false, "base2": false,
			"inv1": true, "inv2": false,
			"dom1": false, "dom2": false,
			"c1": true, "c2": false,
		}, "hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Adopted || res.KeptBase {
		t.Fatalf("want adopt: %+v", res)
	}
	if res.Hash == parent.Hash {
		t.Fatal("hash")
	}
	if res.R != 8 || res.P != 6 {
		t.Fatalf("R=%d P=%d", res.R, res.P)
	}
	assertSeedRefs(t, eng)
}

func TestOverlayAdoptWritesLibrary(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	lib := &Library{Root: t.TempDir(), Domain: "alpha"}
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent:  parent,
		Task:    "write hello.txt",
		Domain:  "alpha",
		TaskID:  "sess-1",
		Library: lib,
		Roller:  scriptedRoller(map[string]bool{"base": false, "inv1": true, "dom1": false, "c1": true}, "hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Adopted {
		t.Fatalf("want adopt: %+v", res)
	}
	if res.LibraryStatus != LibraryAdded || res.Experience == nil {
		t.Fatalf("library %s exp %+v", res.LibraryStatus, res.Experience)
	}
	if !HasExperience(*res.Experience) {
		t.Fatal("incomplete experience")
	}
	if !experienceTransferable(*res.Experience) {
		t.Fatal("verdicts")
	}
	if lib.Context() == "" || !strings.Contains(lib.Context(), res.Experience.Cue) {
		t.Fatalf("index %s", lib.Context())
	}
	if _, err := os.Stat(filepath.Join(lib.entryDir("sess-1"), "harness", "instruction", "verification.md")); err != nil {
		t.Fatal(err)
	}
	assertSeedRefs(t, eng)
}

func TestOverlayEvidenceTreeOnDisk(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	scratch := t.TempDir()
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent:       parent,
		Task:         "write hello.txt",
		Scratch:      scratch,
		KeepEvidence: true,
		Roller:       scriptedRoller(map[string]bool{"base": true, "inv1": true, "dom1": true, "c1": true}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(scratch, "task.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(scratch, "harness", "instruction", "verification.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(scratch, "rollouts", "base.md")); err != nil {
		t.Fatal(err)
	}
	if res.Evidence == nil {
		t.Fatal("evidence")
	}
}

func TestOverlayKeepBaseSkipsLibrary(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	lib := &Library{Root: t.TempDir(), Domain: "alpha"}
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent:  parent,
		Task:    "write hello.txt",
		Library: lib,
		Roller:  scriptedRoller(map[string]bool{"base": true, "inv1": true, "dom1": true, "c1": true}, "same"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.LibraryStatus != LibraryNoEntry {
		t.Fatalf("status %s", res.LibraryStatus)
	}
	if lib.Context() != "" {
		t.Fatalf("index %s", lib.Context())
	}
}

func TestOverlayC1MissKeepsBase(t *testing.T) {
	parent, eng := seedOverlayParent(t)
	res, err := eng.Overlay(context.Background(), OverlayOpts{
		Parent: parent,
		Task:   "write hello.txt",
		Roller: scriptedRoller(map[string]bool{"base": false, "inv1": true, "dom1": false, "c1": false}, "hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Adopted || !res.KeptBase {
		t.Fatalf("c1 miss must keep base: %+v", res)
	}
	if res.Hash != parent.Hash {
		t.Fatal("hash moved")
	}
	if err := res.Ledger.IdentityOK(); err != nil {
		t.Fatal(err)
	}
	assertSeedRefs(t, eng)
}

func assertSeedRefs(t *testing.T, eng *Engine) {
	t.Helper()
	if eng.Refs.GetOrEmpty(artifact.RefActive) != "seed-active" {
		t.Fatalf("active %s", eng.Refs.GetOrEmpty(artifact.RefActive))
	}
	if eng.Refs.GetOrEmpty(artifact.RefCanary) != "seed-canary" {
		t.Fatalf("canary %s", eng.Refs.GetOrEmpty(artifact.RefCanary))
	}
	if eng.Refs.GetOrEmpty(artifact.RefStaging) != "seed-staging" {
		t.Fatalf("staging %s", eng.Refs.GetOrEmpty(artifact.RefStaging))
	}
	listed, err := eng.Refs.List()
	if err != nil {
		t.Fatal(err)
	}
	for name := range listed {
		if strings.HasPrefix(name, "archive/") {
			t.Fatalf("overlay wrote archive %s", name)
		}
	}
}
