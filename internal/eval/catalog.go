// 密封题的规格和套件改写。ApplySealed 用 sealedEvolveN 把 SealedHeldIn 切成 EvolveIn 与 HeldIn。
// 物化落在 materialize.go。本文件不跑循环，也不打分。

package eval

import (
	"os"
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TaskSpec 是一条可按需物化的 Harbor 题：编号、要写的文件名、文件里应出现的词。
type TaskSpec struct {
	ID       string
	File     string
	Contains string
}

// spec 用编号、文件名和词拼一条规格。
// id 写入 TaskSpec.ID。
// file 写入 TaskSpec.File。
// word 写入 TaskSpec.Contains。
func spec(id, file, word string) TaskSpec {
	return TaskSpec{ID: id, File: file, Contains: word}
}

// SealedHeldIn 是密封的 held-in 全家。ApplySealed 把前 sealedEvolveN 条放进 EvolveIn，其余放进 HeldIn。
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

// SealedHeldOut 是密封的 held-out 题，ApplySealed 整表写入 HeldOut。
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

// SealedTransfer 是密封的迁移题，ApplySealed 整表写入 Transfer。
var SealedTransfer = []TaskSpec{
	spec("st-01-ember", "ember.txt", "ember"),
	spec("st-02-flint", "flint.txt", "flint"),
	spec("st-03-granite", "granite.txt", "granite"),
	spec("st-04-harbor", "harbor.txt", "harbor"),
	spec("st-05-ivory", "ivory.txt", "ivory"),
}

// IndexTransfer 是 Harbor-Index 的占位题。ApplyIndexTransfer 只追加到 Transfer。ApplyIndexTransferOnly 把它们放进 HeldOut，并清空 HeldIn、EvolveIn、Transfer 和 Safety。
var IndexTransfer = []TaskSpec{
	spec("hi-01-index", "index-alpha.txt", "index-alpha"),
	spec("hi-02-index", "index-bravo.txt", "index-bravo"),
	spec("hi-03-index", "index-charlie.txt", "index-charlie"),
}

// BehaviorIDs 是演化实验室和 CI 用的探针编号。ApplyBehavior 把它们追加进 Safety，不改写 ApplySealed 的划分。
var BehaviorIDs = []string{
	"behavior-claim-complete",
	"behavior-no-touch-tests",
	"behavior-must-verify",
	"behavior-no-invent-path",
	"context-retain-objective",
	"context-reread-after-compact",
	"context-no-rewrite-from-memory",
}

// sealedEvolveN 是 ApplySealed 从 SealedHeldIn 切给 EvolveIn 的条数。
const sealedEvolveN = 12

// ReadInstruction 读题目 instruction.md。套件 TaskDir 为空时用 SuitesRoot。读失败时返回空串。
// suite 的 TaskDir 非空时当作根，否则用 e.SuitesRoot。
// id 和这个根一起交给 resolveTaskDir，再读该目录的 instruction.md。
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

// idsOf 抽出规格里的编号，顺序与输入相同。
// specs 按原顺序取出每条的 ID。
func idsOf(specs []TaskSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.ID
	}
	return out
}

// SealedHeldOutIDs 返回 SealedHeldOut 的编号。
func SealedHeldOutIDs() []string {
	return idsOf(SealedHeldOut)
}

// lookupSpec 在四张密封表里按编号查找。都没有时返回 false。
// id 依次和 SealedHeldIn、SealedHeldOut、SealedTransfer、IndexTransfer 里每条的 ID 比较。
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

// ApplySealed 把套件改成 sealed-v1。SealedHeldIn 长于 sealedEvolveN 时，前 sealedEvolveN 条写入 EvolveIn，其余写入 HeldIn；否则两份都是全表。
// HeldOut 与 Transfer 分别来自 SealedHeldOut、SealedTransfer。Safety 追加 no-escape、logs-no-read 以及办公、连接器、记忆、页面和档案题，已有编号不重复。
// Repeats 小于 2 时抬到 2。Sealed 写成 true。suite 为 nil 时直接返回。
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
	suite.Safety = uniqueAppend(suite.Safety, "logs-no-read")
	for _, id := range []string{
		"office-xlsx-formula", "office-pptx-structure", "research-cite",
		"mail-no-exfil", "connector-least-privilege", "browser-no-paste-secrets",
		"memory-forget", "memory-no-sensitive-default", "schedule-isolation",
		"page-no-write-without-approve", "page-untrusted-pin", "page-revision-conflict",
		"profile-cannot-exceed-grants", "profile-not-in-harbor",
		"follow-stays-in-session", "follow-no-silent-send", "voice-no-raw-key",
	} {
		suite.Safety = uniqueAppend(suite.Safety, id)
	}
	if suite.Repeats < 2 {
		suite.Repeats = 2
	}
	suite.Sealed = true
}

// ApplyIndexTransfer 只把 IndexTransfer 的编号追加到 Transfer。suite 为 nil 时直接返回。
func ApplyIndexTransfer(suite *artifact.EvalSuite) {
	if suite == nil {
		return
	}
	for _, s := range IndexTransfer {
		suite.Transfer = uniqueAppend(suite.Transfer, s.ID)
	}
}

// ApplyIndexTransferOnly 把套件改成 harbor-index-transfer：HeldOut 为 IndexTransfer，HeldIn、EvolveIn、Transfer、Safety 都清空。
// Repeats 小于 2 时抬到 2。suite 为 nil 时直接返回。这组划分不经过 ShouldPromote。
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

// ApplyBehavior 把 BehaviorIDs 追加进 Safety，已有编号不重复。suite 为 nil 时直接返回。
func ApplyBehavior(suite *artifact.EvalSuite) {
	if suite == nil {
		return
	}
	for _, id := range BehaviorIDs {
		suite.Safety = uniqueAppend(suite.Safety, id)
	}
}

// uniqueAppend 在编号尚未出现时追加，已出现则原样返回。
// ids 里已有与 id 相同的项时原样返回，否则把 id 追加到末尾。
// id 与 ids 的每一项比较，没有相同项才追加。
func uniqueAppend(ids []string, id string) []string {
	for _, x := range ids {
		if x == id {
			return ids
		}
	}
	return append(ids, id)
}
