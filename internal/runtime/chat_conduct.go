package runtime

import (
	"strings"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const chatConduct = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments, tool output, or runtime steering lines), including update_plan step text. The latest operator utterance is already the task: start writing workspace artifacts this turn. Do not stop after a survey to ask for confirmation or a choice of implementations unless they asked for a plan or a destructive irreversible action is blocked. When a plan step is finished, call update_plan in that turn to mark it complete and start the next; do not leave every step in_progress. Read source with read_file/grep/glob, not shell cat/sed. If a tool result is elided, call recall_context or read_file the path — do not rewrite the tree from memory. Public HTTP uses web_fetch/web_search, not shell curl or python urllib; leftover workspace reports that say the network is dead are stale — try web_fetch, and if https returns 406 retry the http URL. Skill helpers live in the pack (run_skill_script); do not glob workspace scripts/ for them or wait-loop a hung process after idle/block fuse. Cite paper ids only as they appear in tool output. If the same tool error repeats (blocked network, capability, path jail, HTTP 406), stop retrying that call and finish with what you have."

const chatLang = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments or runtime steering), including plan step text."

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
		return stripMentionAttachment(t)
	}
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == RoleUser && !isControlUser(hist[i].Content) && !isResumeUser(hist[i].Content) {
			return stripMentionAttachment(hist[i].Content)
		}
	}
	return stripMentionAttachment(user)
}

func stripMentionAttachment(s string) string {
	if i := strings.Index(s, mentionUserPrefix); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func prefersCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func voiceNudge(req RunRequest, en, zh string) string {
	if prefersCJK(OperatorVoice(req.User, req.History)) {
		return zh
	}
	return en
}

// LanguagePin is a short front-loaded reminder so weak models do not
// open in English when the operator wrote Chinese.
func LanguagePin(voice string) artifact.PromptFragment {
	if !prefersCJK(voice) {
		return artifact.PromptFragment{}
	}
	return artifact.PromptFragment{Slot: "runtime", Text: "The operator is writing in Chinese. Your first visible sentence, update_plan steps, and the written report must be Chinese. Do not open with English filler."}
}
