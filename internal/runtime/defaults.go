// 聊天视界常量，以及把一份 LoopPreset 调成聊天档的 ApplyChatHorizon。
// DefaultLoop 是评测用的硬循环，不经过 ApplyChatHorizon。桌面和命令行在自己的入口调用它。

package runtime

import "github.com/Shenchangxin/yoyo/internal/artifact"

const (
	// ChatMaxTurns 是 64。ApplyChatHorizon 只在 MaxTurns 小于它时才抬高。
	ChatMaxTurns = 64
	// ChatMaxToolMessages 是 160。ApplyChatHorizon 只在 MaxToolMessages 小于它时才抬高。
	ChatMaxToolMessages = 160
	// ChatMicroKeep 是 32。ApplyChatHorizon 只在 MicroKeep 小于它时才抬高。
	ChatMicroKeep = 32
	// ChatCompactionKeep 是 96。ApplyChatHorizon 只在 CompactionKeep 小于它时才抬高。
	ChatCompactionKeep = 96
	// ChatRulesTokens 是 1600 个 rune。ApplyChatHorizon 把 RulesTokens 直接赋成这个值。嵌套的 AGENTS.md 和 CLAUDE.md 不进这条前缀，读到对应路径时再加载。1600 个 rune 大约是 1200 token。
	ChatRulesTokens = 1_600
	// ChatKeepTokens 是 16000。ApplyChatHorizon 只在 KeepTokens 小于它时才抬高。
	ChatKeepTokens = 16_000
	// ChatContextBuffer 是 20000。ApplyChatHorizon 只在 ContextBuffer 小于等于 0 时写成它。
	ChatContextBuffer = 20_000
	// ChatTaskMaxTurns 是 40。ApplyChatHorizon 只在 TaskMaxTurns 小于它时才抬高。
	ChatTaskMaxTurns = 40
	// ChatTaskMaxTools 是 60。ApplyChatHorizon 只在 TaskMaxToolMessages 小于它时才抬高。
	ChatTaskMaxTools = 60
	// ChatSoftGuardTurns 是 192。softGuard 只在 turn 不小于它时才返回复位或停止。
	ChatSoftGuardTurns = 192
	// ChatStallWindow 是 24。softGuard 用 sinceProgress 和它比较：小于则复位，不小于则停止。
	ChatStallWindow = 24
	// shapeLogEvery 是 16。回合序号能被它整除时，才把当轮投影账本写进轨迹。
	shapeLogEvery = 16
)

// softGuard 在 turn 到达 ChatSoftGuardTurns 时返回复位或停止。ChatSoftGuardTurns 小于等于 0，或 turn 小于 192，reset 和 stop 都是 false。
// turn 不小于 192 且 sinceProgress 小于 ChatStallWindow（24）时，reset 为 true、stop 为 false。turn 不小于 192 且 sinceProgress 不小于 24 时，stop 为 true、reset 为 false。
func softGuard(turn, sinceProgress int) (reset, stop bool) {
	if ChatSoftGuardTurns <= 0 || turn < ChatSoftGuardTurns {
		return false, false
	}
	if sinceProgress < ChatStallWindow {
		return true, false
	}
	return false, true
}

// ApplyChatHorizon 调整一份循环预设。MaxTurns、MaxToolMessages、MicroKeep、CompactionKeep、KeepTokens、TaskMaxTurns、TaskMaxToolMessages 只在小于对应常量时抬高，已经更大的值保持不变。ContextBuffer 只在小于等于 0 时写成 ChatContextBuffer。
// RulesTokens 直接赋成 ChatRulesTokens（1600），不论原来更大还是更小；保存下来的 8k 或 32k 会被盖掉。StackAgentsMD 写成 false，嵌套 AGENTS.md 改由读取时发现。AllowLLMCompact 写成 true。评测入口和子代理会在别处把它改回 false。
func ApplyChatHorizon(loop artifact.LoopPreset) artifact.LoopPreset {
	if loop.MaxTurns < ChatMaxTurns {
		loop.MaxTurns = ChatMaxTurns
	}
	if loop.MaxToolMessages < ChatMaxToolMessages {
		loop.MaxToolMessages = ChatMaxToolMessages
	}
	if loop.MicroKeep < ChatMicroKeep {
		loop.MicroKeep = ChatMicroKeep
	}
	if loop.CompactionKeep < ChatCompactionKeep {
		loop.CompactionKeep = ChatCompactionKeep
	}
	// 直接覆盖。保存的 8k 或 32k 不能再进始终在场的规则前缀。
	loop.RulesTokens = ChatRulesTokens
	if loop.KeepTokens < ChatKeepTokens {
		loop.KeepTokens = ChatKeepTokens
	}
	if loop.ContextBuffer <= 0 {
		loop.ContextBuffer = ChatContextBuffer
	}
	if loop.TaskMaxTurns < ChatTaskMaxTurns {
		loop.TaskMaxTurns = ChatTaskMaxTurns
	}
	if loop.TaskMaxToolMessages < ChatTaskMaxTools {
		loop.TaskMaxToolMessages = ChatTaskMaxTools
	}
	// 嵌套 AGENTS.md 在读取时发现，不叠进前缀。
	loop.StackAgentsMD = false
	// 桌面检查点向模型要结构化摘要。
	loop.AllowLLMCompact = true
	return loop
}

// DefaultLoop 返回评测用的硬循环。MaxTurns 是 32，MaxToolMessages 是 40，MaxParallel 是 4，CompactionKeep 是 24，CompactionTokens 是 24000，ToolResultBudget 是 8000，MicroKeep 是 4，PlaybookTokens 是 2000，RulesTokens 是 1500。AllowLLMCompact 保持零值 false。
func DefaultLoop() artifact.LoopPreset {
	return artifact.LoopPreset{
		ID:               "default",
		MaxTurns:         32,
		MaxToolMessages:  40,
		MaxParallel:      4,
		CompactionKeep:   24,
		CompactionTokens: 24_000,
		ToolResultBudget: 8_000,
		MicroKeep:        4,
		PlaybookTokens:   2_000,
		RulesTokens:      1_500,
		Bootstrap:        "Start by inspecting the workspace and identifying the required output artifact. Create an initial version as early as possible.",
		Execution:        "Prefer concrete repo changes over generic advice. Keep edits tightly scoped to the task.",
		Verification:     "Before concluding, verify the result with the most targeted command, file read, or test you can run.",
		FailureRecovery:  "If a tool call fails, inspect the error and adapt; do not blindly retry the same action.",
	}
}

// DefaultPolicy 返回自动模式策略。工作区读写默认放行，shell、高风险、网络、代发、计算机使用、连接器写入、浏览器、日程和记忆写入需要审批。
func DefaultPolicy() artifact.PolicyPack {
	return artifact.PolicyPack{
		ID:              "default",
		Mode:            "auto",
		DefaultAllow:    []string{"read_workspace", "write_workspace"},
		RequireApproval: []string{"shell", "high_risk", "network", "send_as_you", "computer_use", "write_connector", "browser", "schedule", "memory_write"},
	}
}

// DefaultSkillMD 返回名为 verify-artifact 的技能正文，要求结束前核对任务点名的文件存在且非空。
func DefaultSkillMD() string {
	return `---
name: verify-artifact
description: Confirm required output files exist and are non-empty before finishing a task.
---

Before you stop, list the files the task asked for, read them, and fix missing artifacts.
`
}
