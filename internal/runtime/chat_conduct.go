package runtime

import (
	"regexp"
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

const languageRewriteNudgeZH = "上一句可见回复用了英文开场（I'll / Let me / I will）。操作者在说中文。请用中文重写可见句子；已排队的工具调用会照常执行。不要以 I'll / Let me / I will 开头。"

var reEnglishOpener = regexp.MustCompile(`(?i)^(?:okay[,!.]?\s+|sure[,!.]?\s+|alright[,!.]?\s+)?(i['’]ll\b|i will\b|let me\b|i['’]m going\b|i am going\b|here['’]s\b|here is\b)`)

type ConductOpts struct {
	Plan        bool
	Methodology bool
}

// ChatConductFragments is desktop/CLI chat overlay. Harbor eval never
// receives it: eval ≠ personal voice (I6). Language follows the user
// message; there is no per-locale copy of these rules.
//
// Deliverable type (plan vs implement) is the model's job. These fragments
// state the mapping; they do not pre-parse the utterance.
func ChatConductFragments(plan bool) []artifact.PromptFragment {
	return ChatConduct(ConductOpts{Plan: plan})
}

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

func OverlayConduct(frags []artifact.PromptFragment, opts ConductOpts) []artifact.PromptFragment {
	var out []artifact.PromptFragment
	for _, f := range frags {
		if isChatConductText(f.Text) {
			continue
		}
		out = append(out, f)
	}
	return append(out, ChatConduct(opts)...)
}

func isChatConductText(t string) bool {
	t = strings.TrimSpace(t)
	switch t {
	case chatConduct, chatConductMethodology, chatLang + " " + chatPlanMode, chatBootstrap, chatExecution:
		return true
	default:
		// Earlier builds kept a long failure cookbook in this slot.
		return strings.HasPrefix(t, "Reply in the same language as the operator's latest utterance") && len(t) > 900
	}
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
	return artifact.PromptFragment{Slot: "voice", Text: languagePinZH}
}

func languageDynPin(voice string) string {
	if !prefersCJK(voice) {
		return ""
	}
	return "\n## Output language\n" + languagePinZH + "\n"
}

func firstVisibleLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func englishOpener(s string) bool {
	return reEnglishOpener.MatchString(firstVisibleLine(s))
}

// pinVisibleLanguage drops a CJK-operator reply that still opened in English.
// Tool calls are kept; the caller re-asks for Chinese visible sentences.
func pinVisibleLanguage(msg *Message, voice string) string {
	if msg == nil || !prefersCJK(voice) || !englishOpener(msg.Content) {
		return ""
	}
	msg.Content = ""
	return languageRewriteNudgeZH
}
