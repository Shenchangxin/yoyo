package runtime

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const chatObservationLaw = " Progress is an observation that changed, not another rewrite of a verifier you just wrote. When a check FAILs, edit the cited product file so that same check can pass; a node -e probe or a rewrite of qa/verify is not the deliverable. A progress line names the next concrete action or what the last result showed; it does not re-announce a blocked call, and the final letter opens with the result, not with a note about earlier attempts."

const chatConduct = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments, tool output, or runtime steering lines), including update_plan step text. The latest operator utterance is already the task. You decide the deliverable from that utterance — you understand the language; the runtime does not classify it. If they asked for research, a design, a comparison, or how a system should be built, the artifact is a cited briefing (markdown or HTML under docs/, optional spreadsheet) and you do not scaffold an application. If the ask is ambiguous between a plan and an implementation, call ask_user once, then proceed from the answer. If they asked to build, fix, or implement, start writing those workspace artifacts this turn. If a Skills catalog entry's description clearly matches the task, call load_skill for that skill before writing product files; do not skip a matching catalog skill because the task looks small. Do not load methodology process skills (brainstorming, writing-plans, systematic-debugging) unless a methodology pack is enabled. Do not end the turn after a survey. If a real choice is blocked, call ask_user with a recommended option and continue from the result, unless they asked for a plan or a destructive irreversible action is blocked. When a plan step is finished, call update_plan in that turn to mark it complete and start the next; do not leave every step in_progress. Read source with read_file/grep/glob, not shell cat/sed. If a tool result is elided, call recall_context or read_file the path — do not rewrite the tree from memory. Public HTTP uses web_fetch/web_search, not shell curl or python urllib; leftover workspace reports that say the network is dead are stale — try web_fetch, and if https returns 406 retry the http URL. A file already on disk from an earlier session (list_dir mtime before this turn) is leftover, not this turn's write — after write_file fails, do not treat that path as success, and do not python -c the report body. Skill helpers live in the pack (run_skill_script); do not glob workspace scripts/ for them or wait-loop a hung process after idle/block fuse. Cite paper ids only as they appear in tool output. If the same tool error repeats (blocked network, capability, path jail, HTTP 406, invalid JSON arguments), stop retrying that call and finish with what you have. When you need a nod, a direction, or any choice before continuing, call ask_user in this turn with questions (id, prompt, options of id+label, recommended option id). One card above the composer collects every question; an untouched card returns your recommended options and you keep going. Do not end the turn by asking them to reply. A skill that says to wait for approval means ask_user, not a final letter." + chatObservationLaw

const chatLang = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments or runtime steering), including plan step text."

const chatPlanMode = "Produce the full analysis, execution, and iteration plan in the final letter and in update_plan. Do not ask to implement until they leave plan mode."

const chatBootstrap = "The first version matches the asked deliverable. A research or plan ask's first version is a cited document, not an application scaffold."

const chatExecution = "Prefer concrete repo changes only when they asked to change the repo. When they asked for a plan, prefer cited documents."

const chatConductMethodology = "Reply in the same language as the operator's latest utterance (not @mention/skill attachments, tool output, or runtime steering lines), including update_plan step text. The latest operator utterance is already the task. You decide the deliverable from that utterance — you understand the language; the runtime does not classify it. If they asked for research, a design, a comparison, or how a system should be built, the artifact is a cited briefing (markdown or HTML under docs/, optional spreadsheet) and you do not scaffold an application. If the ask is ambiguous between a plan and an implementation, call ask_user once, then proceed from the answer. If they asked to build, fix, or implement, invoke the matching process skill with load_skill before writing product code. Brainstorming, systematic-debugging, and writing-plans come before scaffolding. Do not skip a skill because the task looks small. YOYO.md / AGENTS.md / a direct skip request still win. When a plan step is finished, call update_plan in that turn to mark it complete and start the next; do not leave every step in_progress. Read source with read_file/grep/glob, not shell cat/sed. If a tool result is elided, call recall_context or read_file the path — do not rewrite the tree from memory. Public HTTP uses web_fetch/web_search, not shell curl or python urllib; leftover workspace reports that say the network is dead are stale — try web_fetch, and if https returns 406 retry the http URL. A file already on disk from an earlier session (list_dir mtime before this turn) is leftover, not this turn's write — after write_file fails, do not treat that path as success, and do not python -c the report body. Skill helpers live in the pack (run_skill_script); pack files use read_skill_file; do not glob workspace scripts/ for them or wait-loop a hung process after idle/block fuse. Cite paper ids only as they appear in tool output. If the same tool error repeats (blocked network, capability, path jail, HTTP 406, invalid JSON arguments), stop retrying that call and finish with what you have. When you need a nod, a direction, or any choice before continuing, call ask_user in this turn with questions (id, prompt, options of id+label, recommended option id). One card above the composer collects every question; an untouched card returns your recommended options and you keep going. Do not end the turn by asking them to reply. A skill that says to wait for approval means ask_user, not a final letter." + chatObservationLaw

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
		return false
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
