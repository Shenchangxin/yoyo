package eval

import (
	"os"
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TaskSpec is a Harbor-layout task that can be materialized on demand.
// Smoke tasks (write-hello, write-answer, …) live on disk under evals/.
// Sealed tasks are derived from this catalog so the 12/8/10 split stays
// the source of truth rather than a pile of copy-pasted fixtures.
type TaskSpec struct {
	ID       string
	File     string
	Contains string
}

func spec(id, file, word string) TaskSpec {
	return TaskSpec{ID: id, File: file, Contains: word}
}

// SealedHeldIn is the full sealed held-in family. ApplySealed splits it into
// EvolveIn (proposer / inner trials) and HeldIn (promote-set).
var SealedHeldIn = []TaskSpec{
	spec("si-01-alpha", "alpha.txt", "alpha"),
	spec("si-02-bravo", "bravo.txt", "bravo"),
	spec("si-03-charlie", "charlie.txt", "charlie"),
	spec("si-04-delta", "delta.txt", "delta"),
	spec("si-05-echo", "echo.txt", "echo"),
	spec("si-06-foxtrot", "foxtrot.txt", "foxtrot"),
	spec("si-07-golf", "golf.txt", "golf"),
	spec("si-08-hotel", "hotel.txt", "hotel"),
	spec("si-09-india", "india.txt", "india"),
	spec("si-10-juliet", "juliet.txt", "juliet"),
	spec("si-11-kilo", "kilo.txt", "kilo"),
	spec("si-12-lima", "lima.txt", "lima"),
	spec("si-13-mike", "mike.txt", "mike"),
	spec("si-14-november", "november.txt", "november"),
	spec("si-15-oscar", "oscar.txt", "oscar"),
	spec("si-16-papa", "papa.txt", "papa"),
	spec("si-17-quebec", "quebec.txt", "quebec"),
	spec("si-18-romeo", "romeo.txt", "romeo"),
	spec("si-19-sierra", "sierra.txt", "sierra"),
	spec("si-20-tango", "tango.txt", "tango"),
}

var SealedHeldOut = []TaskSpec{
	spec("so-01-uniform", "uniform.txt", "uniform"),
	spec("so-02-victor", "victor.txt", "victor"),
	spec("so-03-whiskey", "whiskey.txt", "whiskey"),
	spec("so-04-xray", "xray.txt", "xray"),
	spec("so-05-yankee", "yankee.txt", "yankee"),
	spec("so-06-zulu", "zulu.txt", "zulu"),
	spec("so-07-amber", "amber.txt", "amber"),
	spec("so-08-bronze", "bronze.txt", "bronze"),
	spec("so-09-copper", "copper.txt", "copper"),
	spec("so-10-dune", "dune.txt", "dune"),
}

var SealedTransfer = []TaskSpec{
	spec("st-01-ember", "ember.txt", "ember"),
	spec("st-02-flint", "flint.txt", "flint"),
	spec("st-03-granite", "granite.txt", "granite"),
	spec("st-04-harbor", "harbor.txt", "harbor"),
	spec("st-05-ivory", "ivory.txt", "ivory"),
}

// IndexTransfer is a Harbor-Index stand-in used only as post-canary transfer.
// Real Index adapters replace these files; ids never enter EvolveIn or Propose.
var IndexTransfer = []TaskSpec{
	spec("hi-01-index", "index-alpha.txt", "index-alpha"),
	spec("hi-02-index", "index-bravo.txt", "index-bravo"),
	spec("hi-03-index", "index-charlie.txt", "index-charlie"),
}

// BehaviorIDs are cheap probes for the evolve lab and CI. They are not the
// default 89-task promote gate.
var BehaviorIDs = []string{
	"behavior-claim-complete",
	"behavior-no-touch-tests",
	"behavior-must-verify",
	"behavior-no-invent-path",
}

const sealedEvolveN = 12

func (e *Engine) ReadInstruction(suite artifact.EvalSuite, id string) string {
	root := suite.TaskDir
	if root == "" {
		root = e.SuitesRoot
	}
	b, err := os.ReadFile(filepath.Join(e.resolveTaskDir(root, id), "instruction.md"))
	if err != nil {
		return ""
	}
	return string(b)
}

func idsOf(specs []TaskSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.ID
	}
	return out
}

func lookupSpec(id string) (TaskSpec, bool) {
	for _, s := range SealedHeldIn {
		if s.ID == id {
			return s, true
		}
	}
	for _, s := range SealedHeldOut {
		if s.ID == id {
			return s, true
		}
	}
	for _, s := range SealedTransfer {
		if s.ID == id {
			return s, true
		}
	}
	for _, s := range IndexTransfer {
		if s.ID == id {
			return s, true
		}
	}
	return TaskSpec{}, false
}

// ApplySealed rewrites a suite onto the sealed 20/10 split plus transfer and
// safety. Repeats is at least 2 so promotion is not a one-shot lottery.
func ApplySealed(suite *artifact.EvalSuite) {
	if suite == nil {
		return
	}
	suite.ID = "sealed-v1"
	ids := idsOf(SealedHeldIn)
	if len(ids) > sealedEvolveN {
		suite.EvolveIn = append([]string{}, ids[:sealedEvolveN]...)
		suite.HeldIn = append([]string{}, ids[sealedEvolveN:]...)
	} else {
		suite.EvolveIn = append([]string{}, ids...)
		suite.HeldIn = append([]string{}, ids...)
	}
	suite.HeldOut = idsOf(SealedHeldOut)
	suite.Transfer = idsOf(SealedTransfer)
	suite.Safety = uniqueAppend(suite.Safety, "no-escape")
	for _, id := range []string{
		"office-xlsx-formula", "office-pptx-structure", "research-cite",
		"mail-no-exfil", "connector-least-privilege", "browser-no-paste-secrets",
		"memory-forget", "memory-no-sensitive-default", "schedule-isolation",
	} {
		suite.Safety = uniqueAppend(suite.Safety, id)
	}
	if suite.Repeats < 2 {
		suite.Repeats = 2
	}
	suite.Sealed = true
}

// ApplyIndexTransfer appends Harbor-Index stand-in ids to Transfer only.
func ApplyIndexTransfer(suite *artifact.EvalSuite) {
	if suite == nil {
		return
	}
	for _, s := range IndexTransfer {
		suite.Transfer = uniqueAppend(suite.Transfer, s.ID)
	}
}

// ApplyIndexTransferOnly grades the Index subset as held-out transfer. Not a promote gate.
func ApplyIndexTransferOnly(suite *artifact.EvalSuite) {
	if suite == nil {
		return
	}
	suite.ID = "harbor-index-transfer"
	suite.HeldIn = nil
	suite.EvolveIn = nil
	suite.HeldOut = idsOf(IndexTransfer)
	suite.Transfer = nil
	suite.Safety = nil
	if suite.Repeats < 2 {
		suite.Repeats = 2
	}
}

func ApplyBehavior(suite *artifact.EvalSuite) {
	if suite == nil {
		return
	}
	for _, id := range BehaviorIDs {
		suite.Safety = uniqueAppend(suite.Safety, id)
	}
}

func uniqueAppend(ids []string, id string) []string {
	for _, x := range ids {
		if x == id {
			return ids
		}
	}
	return append(ids, id)
}
