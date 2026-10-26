// 锁住拓扑比较：只改文案、中间件和 ID 时仍然相等，改 MaxTurns 则不相等。runtime 槽写进 Execution；middleware 在次数未设时写成 2。

package artifact

import "testing"

// TestLoopTopologyEqualIgnoresInstruction 确认文案、中间件和 ID 不算拓扑，MaxTurns 算。
func TestLoopTopologyEqualIgnoresInstruction(t *testing.T) {
	a := LoopPreset{ID: "a", MaxTurns: 12, CompactionKeep: 8, Bootstrap: "old", Verification: "v"}
	b := a
	b.ID = "b"
	b.Bootstrap = "write a placeholder file first"
	b.Verification = "re-read outputs"
	b.ToolErrorInstruction = "stop retrying"
	b.MaxRecentToolErrors = 2
	b.TaskInstruction = "stay in workspace"
	if !LoopTopologyEqual(a, b) {
		t.Fatal("instruction/middleware must not count as L3 topology")
	}
	b.MaxTurns = 99
	if LoopTopologyEqual(a, b) {
		t.Fatal("max_turns is topology")
	}
}

// TestSetInstructionSlotRuntime 确认 runtime 写入 Execution，middleware 会把未设的次数写成 2。
func TestSetInstructionSlotRuntime(t *testing.T) {
	p := LoopPreset{}.SetInstructionSlot("runtime", "implement then verify")
	if p.Execution != "implement then verify" {
		t.Fatalf("%+v", p)
	}
	p = p.SetInstructionSlot("middleware", "change strategy after two errors")
	if p.ToolErrorInstruction == "" || p.MaxRecentToolErrors != 2 {
		t.Fatalf("%+v", p)
	}
}
