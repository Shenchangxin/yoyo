package artifact

// LoopTopologyEqual reports whether two loop presets share the L3 control
// surface: turn/tool budgets, compaction, and plan mode. Instruction copy and
// declarative middleware are L1 and may change without a human gate.
func LoopTopologyEqual(a, b LoopPreset) bool {
	return a.MaxTurns == b.MaxTurns &&
		a.MaxToolMessages == b.MaxToolMessages &&
		a.CompactionKeep == b.CompactionKeep &&
		a.CompactionTokens == b.CompactionTokens &&
		a.ToolResultBudget == b.ToolResultBudget &&
		a.MicroKeep == b.MicroKeep &&
		a.PlaybookTokens == b.PlaybookTokens &&
		a.RulesTokens == b.RulesTokens &&
		a.AllowLLMCompact == b.AllowLLMCompact &&
		a.MaxBudgetUSD == b.MaxBudgetUSD &&
		a.PlanMode == b.PlanMode
}

// SetInstructionSlot writes one L1 instruction field. Unknown slots are ignored.
func (p LoopPreset) SetInstructionSlot(slot, text string) LoopPreset {
	switch slot {
	case "bootstrap":
		p.Bootstrap = text
	case "execution", "runtime":
		p.Execution = text
	case "verification":
		p.Verification = text
	case "failure_recovery":
		p.FailureRecovery = text
	case "task":
		p.TaskInstruction = text
	case "middleware", "tool_error":
		p.ToolErrorInstruction = text
		if p.MaxRecentToolErrors <= 0 {
			p.MaxRecentToolErrors = 2
		}
	}
	return p
}
