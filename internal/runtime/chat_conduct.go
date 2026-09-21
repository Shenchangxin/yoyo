package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const chatConduct = "Reply in the same language as the latest user message, including update_plan step text. The latest user message is already the task: start writing workspace artifacts this turn. Do not stop after a survey to ask for confirmation or a choice of implementations unless they asked for a plan or a destructive irreversible action is blocked. When a plan step is finished, call update_plan in that turn to mark it complete and start the next; do not leave every step in_progress. Read source with read_file/grep/glob, not shell cat/sed. If a tool result is elided, call recall_context or read_file the path — do not rewrite the tree from memory."

const chatLang = "Reply in the same language as the latest user message, including plan step text."

// ChatConductFragments is desktop/CLI chat overlay. Harbor eval never
// receives it: eval ≠ personal voice (I6). Language follows the user
// message; there is no per-locale copy of these rules.
func ChatConductFragments(plan bool) []artifact.PromptFragment {
	text := chatConduct
	if plan {
		text = chatLang
	}
	return []artifact.PromptFragment{{Slot: "runtime", Text: text}}
}

// OperatorVoice is the latest real operator utterance, skipping steer/nudge.
func OperatorVoice(user string, hist []Message) string {
	if t := strings.TrimSpace(user); t != "" && !isControlUser(t) && !isResumeUser(t) {
		return t
	}
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == RoleUser && !isControlUser(hist[i].Content) && !isResumeUser(hist[i].Content) {
			return hist[i].Content
		}
	}
	return user
}
