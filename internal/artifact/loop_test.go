package artifact

import "testing"

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
