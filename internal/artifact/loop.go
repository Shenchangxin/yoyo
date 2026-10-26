// 循环预设的拓扑比较和文案槽赋值。不保存预设。检出前的 L3 判断和演化的采纳会调用它。

package artifact

// LoopTopologyEqual 判断两份循环预设的拓扑是否相同。ID、Bootstrap、Execution、Verification、FailureRecovery、MaxRecentToolErrors、ToolErrorInstruction、TaskInstruction 不比较；其余字段有一处不同就返回 false。
// b 的 MaxTurns、MaxToolMessages、CompactionKeep、CompactionTokens、ToolResultBudget、MicroKeep、PlaybookTokens、RulesTokens、AllowLLMCompact、MaxBudgetUSD、PlanMode、MaxParallel、KeepTokens、ContextBuffer、OutputReserve、TaskMaxTurns、TaskMaxToolMessages、StackAgentsMD 分别和 a 的同名字段比较。
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
		a.PlanMode == b.PlanMode &&
		a.MaxParallel == b.MaxParallel &&
		a.KeepTokens == b.KeepTokens &&
		a.ContextBuffer == b.ContextBuffer &&
		a.OutputReserve == b.OutputReserve &&
		a.TaskMaxTurns == b.TaskMaxTurns &&
		a.TaskMaxToolMessages == b.TaskMaxToolMessages &&
		a.StackAgentsMD == b.StackAgentsMD
}

// SetInstructionSlot 写一个 L1 文案槽并返回副本。认 bootstrap、execution、runtime、verification、failure_recovery、task、middleware、tool_error；不认识的槽位原样返回。runtime 写入 Execution。middleware 和 tool_error 写入 ToolErrorInstruction，若 MaxRecentToolErrors 还小于等于 0 就写成 2。
// slot 为 bootstrap 时写入 Bootstrap，execution 或 runtime 写入 Execution，verification 写入 Verification，failure_recovery 写入 FailureRecovery，task 写入 TaskInstruction，middleware 或 tool_error 写入 ToolErrorInstruction，其他值不改字段。
// text 赋给 slot 选中的那个字符串字段。
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
