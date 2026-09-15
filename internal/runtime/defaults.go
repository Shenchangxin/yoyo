package runtime

import "github.com/Shenchangxin/yoyo/internal/artifact"

func DefaultLoop() artifact.LoopPreset {
	return artifact.LoopPreset{
		ID:               "default",
		MaxTurns:         32,
		MaxToolMessages:  40,
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
		DefaultAllow:    []string{string(artifact.KindPromptFragment)},
		RequireApproval: []string{"high_risk", "network"},
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
