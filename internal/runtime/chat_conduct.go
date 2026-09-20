package runtime

import (
	"strings"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const chatConductEN = "Reply in the same language as the latest user message, including update_plan step text. The latest user message is already the task: start writing workspace artifacts this turn. Do not stop after a survey to ask for confirmation or a choice of implementations unless they asked for a plan or a destructive irreversible action is blocked. When a plan step is finished, call update_plan in that turn to mark it complete and start the next; do not leave every step in_progress. Read source with read_file/grep/glob, not shell cat/sed. If a tool result is elided, call recall_context or read_file the path — do not rewrite the tree from memory."

const chatConductZH = "用用户最新一条消息的语言回复；中文任务用中文，update_plan 的步骤正文也用中文。用户已经下达任务：本轮直接改仓库、产出文件，不要先写长篇评估再请确认才动手。他们点名的技术栈就用，不要改成选择题。完成计划中的一步就立刻 update_plan 标 complete 并推进下一步，不要让步骤一直停在 in_progress。读代码用 read_file/grep/glob，不要用 shell cat/sed 翻源码。工具结果被 elided 时用 recall_context 或再 read_file，禁止凭记忆整树重写。"

const chatLangEN = "Reply in the same language as the latest user message, including plan step text."

const chatLangZH = "用用户最新一条消息的语言回复；中文任务用中文，计划步骤正文也用中文。"

// ChatConductFragments is desktop/CLI chat overlay. Harbor eval never
// receives it: eval ≠ personal voice (I6).
func ChatConductFragments(voice, locale string, plan bool) []artifact.PromptFragment {
	zh := looksCJK(voice) || isChineseLocale(locale)
	text := chatConductEN
	if plan {
		text = chatLangEN
		if zh {
			text = chatLangZH
		}
	} else if zh {
		text = chatConductZH
	}
	return []artifact.PromptFragment{{Slot: "runtime", Text: text}}
}

// OperatorVoice is the latest real operator utterance, skipping steer/nudge.
func OperatorVoice(user string, hist []Message) string {
	if t := strings.TrimSpace(user); t != "" && !isControlUser(t) {
		return t
	}
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == RoleUser && !isControlUser(hist[i].Content) {
			return hist[i].Content
		}
	}
	return user
}

func looksCJK(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) {
			return true
		}
	}
	return false
}

func isChineseLocale(locale string) bool {
	l := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(locale), "_", "-"))
	return strings.HasPrefix(l, "zh")
}
