package runtime

import "github.com/Shenchangxin/yoyo/internal/artifact"

const (
	ChatMaxTurns        = 64
	ChatMaxToolMessages = 160
	ChatMicroKeep       = 32
	ChatCompactionKeep  = 96
)

// ApplyChatHorizon is a floor for callers that still use a hard loop.
// Desktop/CLI chat sets RunRequest.SoftHorizon, so MaxTurns / MaxToolMessages
// are not a kill switch. MicroKeep / CompactionKeep are raised so snip cannot
// drop the operator goal and the last read of each hot path (I1, I2).
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
