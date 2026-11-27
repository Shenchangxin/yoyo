// 聊天叠层的行为片段和语言钉。Harbor 评测不接收这些片段。
// 不预解析用户句子来决定交付物。桌面会话组提示、runtime 循环和上下文核都调用这里。

package runtime

import (
	"strings"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const chatObservationLaw = " Progress is an observation that changed, not another rewrite of a verifier you just wrote. When a check FAILs, edit the cited product file. The final letter opens with the result."

const chatConductCore = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments, tool output, or runtime steering lines), including update_plan step text. The latest operator utterance is the task. You decide the deliverable from that utterance. If they asked for research, a design, a comparison, or how a system should be built, the artifact is a cited briefing under docs/, not an application scaffold. If the ask is ambiguous between a plan and an implementation, call ask_user once, then proceed from the answer. Do not end the turn after a survey. When a plan step is finished, call update_plan in that turn. Read source with read_file/grep/glob, not shell cat/sed. If a tool result is elided, call recall_context or read_file the path — do not rewrite the tree from memory. A file already on disk from an earlier session is leftover, not this turn's write. Public HTTP uses web_fetch/web_search. If the same tool error repeats, load_skill yoyo-conduct and stop retrying that call. When you need a choice, call ask_user and continue; do not end the turn by asking them to reply. A skill that says to wait for approval means ask_user, not a final letter."

const chatWriteNow = " If they asked to build, fix, or implement, start writing those workspace artifacts this turn. If a Skills catalog entry's description matches, load_skill for that skill before writing product files. Do not load methodology process skills unless a methodology pack is enabled."

const chatConduct = chatConductCore + chatWriteNow + chatObservationLaw

const chatLang = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments or runtime steering), including plan step text."

const chatPlanMode = "Produce the full analysis, execution, and iteration plan in the final letter and in update_plan. Do not ask to implement until they leave plan mode."

const chatBootstrap = "The first version matches the asked deliverable. A research or plan ask's first version is a cited document, not an application scaffold."

const chatExecution = "Prefer concrete repo changes only when they asked to change the repo. When they asked for a plan, prefer cited documents."

const chatMethodologyExtra = " A methodology pack is enabled. load_skill its bootstrap skill before writing product code. Do not skip a skill the pack names. Read pack files with read_skill_file; run pack scripts with run_skill_script. Brainstorming, systematic-debugging, and writing-plans come before scaffolding when that pack says so. YOYO.md / AGENTS.md / a direct skip request still win."

const chatConductMethodology = chatConductCore + chatMethodologyExtra + chatObservationLaw

const languagePinZH = "The operator is writing in Chinese. Every visible assistant sentence this turn — progress narration, status lines, update_plan steps, and the final letter — must be Chinese. English is limited to code, shell, identifiers, and proper nouns. Do not open with I'll / Let me / I will."

// ConductOpts 选择计划模式，以及是否启用方法论包的行为片段。
type ConductOpts struct {
	Plan        bool
	Methodology bool
}

// ChatConduct 在计划模式只返回语言和计划模式两句。否则返回引导、执行和运行时三槽。Methodology 为真时运行时槽换成方法论版本。
// opts 的 Plan 为真时只返回语言和计划模式一句，否则用 Methodology 选择运行时槽文本。
func ChatConduct(opts ConductOpts) []artifact.PromptFragment {
	if opts.Plan {
		return []artifact.PromptFragment{{Slot: "runtime", Text: chatLang + " " + chatPlanMode}}
	}
	runtimeText := chatConduct
	if opts.Methodology {
		runtimeText = chatConductMethodology
	}
	return []artifact.PromptFragment{
		{Slot: "bootstrap", Text: chatBootstrap},
		{Slot: "execution", Text: chatExecution},
		{Slot: "runtime", Text: runtimeText},
	}
}

// OperatorVoice 是最近一条真正的操作者发言，跳过转向和轻推。空的当前输入会回头找历史。
// user 去掉空白后非空，且不是控制或续跑发言时，去掉附件尾巴后返回。
// hist 在 user 不合用时从末尾往前找 Role 为用户、且不是控制或续跑的消息。
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

// stripMentionAttachment 丢掉 mentionUserPrefix 及其后的附件尾巴。
func stripMentionAttachment(s string) string {
	if i := strings.Index(s, mentionUserPrefix); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// prefersCJK 在文本里出现汉字时返回真。
func prefersCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// LanguagePin 是放在前面的短提醒，避免操作者写中文时弱模型用英文开场。不含汉字则返回空片段。
// voice 交给 prefersCJK，不含汉字时返回空片段。
func LanguagePin(voice string) artifact.PromptFragment {
	if !prefersCJK(voice) {
		return artifact.PromptFragment{}
	}
	return artifact.PromptFragment{Slot: "voice", Text: languagePinZH}
}
