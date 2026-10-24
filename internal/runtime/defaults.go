package runtime

import "github.com/Shenchangxin/yoyo/internal/artifact"

const (
	ChatMaxTurns        = 64
	ChatMaxToolMessages = 160
	ChatMicroKeep       = 32
	ChatCompactionKeep  = 96
	// ChatRulesTokens is the always-on rules cap in runes. Nested
	// AGENTS.md / CLAUDE.md stay out of this prefix and load when a
	// matching path is read. 1.6k runes is about 1.2k tokens.
	ChatRulesTokens   = 1_600
	ChatKeepTokens    = 16_000
	ChatContextBuffer = 20_000
	ChatTaskMaxTurns  = 40
	ChatTaskMaxTools  = 60
	// ChatSoftGuardTurns is the last-resort desktop ceiling. SoftHorizon
	// ignores Harbor MaxTurns while work is progressing; identical-failure
	// and unchanged-observation stalls stop earlier. A nine-step pipeline
	// that keeps changing the observation must be allowed to finish.
	ChatSoftGuardTurns = 192
	// ChatStallWindow is how many tool rounds may pass without a plan step or a
	// changed file before a run is ended as stalled. Reaching ChatSoftGuardTurns
	// inside the window checkpoints and continues the same turn instead.
	ChatStallWindow = 24
	// shapeLogEvery samples the per-round projection ledger into the trajectory.
	shapeLogEvery = 16
)

// softGuard decides what happens when a desktop run reaches ChatSoftGuardTurns.
// reset means the counter restarts because work moved inside ChatStallWindow.
// stop means that window passed with no plan step and no changed file.
func softGuard(turn, sinceProgress int) (reset, stop bool) {
	if ChatSoftGuardTurns <= 0 || turn < ChatSoftGuardTurns {
		return false, false
	}
	if sinceProgress < ChatStallWindow {
		return true, false
	}
	return false, true
}

// ApplyChatHorizon is a floor for callers that still use a hard loop.
// Desktop/CLI chat sets RunRequest.SoftHorizon, so MaxTurns / MaxToolMessages
// are not a kill switch. MicroKeep / CompactionKeep are raised so snip cannot
// drop the operator goal and the last read of each hot path (I1, I2).
// ChatSoftGuardTurns is the nuclear cap if observation never stalls.
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
	// Chat rules are a ceiling. A saved harness that still carries the old
	// 8k/32k floor must not put that file back into the prefix.
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
	// Nested AGENTS.md is discovered on read, not stacked into the prefix.
	loop.StackAgentsMD = false
	// Desktop checkpoints ask the model for a structured summary. Harbor and
	// subagents set this back to false so eval stays deterministic.
	loop.AllowLLMCompact = true
	return loop
}

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

func DefaultPolicy() artifact.PolicyPack {
	return artifact.PolicyPack{
		ID:              "default",
		Mode:            "auto",
		DefaultAllow:    []string{"read_workspace", "write_workspace"},
		RequireApproval: []string{"shell", "high_risk", "network", "send_as_you", "computer_use", "write_connector", "browser", "schedule", "memory_write"},
	}
}

func DefaultSkillMD() string {
	return `---
name: verify-artifact
description: Confirm required output files exist and are non-empty before finishing a task.
---

Before you stop, list the files the task asked for, read them, and fix missing artifacts.
`
}
