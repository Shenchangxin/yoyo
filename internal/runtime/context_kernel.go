package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/zeebo/blake3"
)

const (
	skillBodyCapRunes      = 5_000
	skillBootstrapCapRunes = 12_000
	skillTotalCapRunes     = 25_000
)

// ContextKernel owns pin/hot/dynamic assembly, the prefix contract, and
// checkpoint. Run consumes Prompt / Checkpoint; it does not reshape history.
type ContextKernel struct {
	req      *RunRequest
	identity string
	pins     string
	prefix   string
	// seriesPrefix is the system text actually placed at the head. A later
	// prompt change is appended after history so this head stays byte-stable.
	seriesPrefix string
	bill         contextBill
	spill        *Spill
	notes        string
	lastHash     string
	prevHot      string
	dyn          string
}

func newContextKernel(req *RunRequest) *ContextKernel {
	spill := spillOf(*req)
	notes := capRunes(ReadNotes(spill), 2000)
	suggested := ""
	if req.Tools != nil {
		suggested = FormatSkillHints(req.Tools.SkillHints)
	}
	bill := buildBill(billInput{
		Loop:      req.Loop,
		Fragments: req.Fragments,
		Playbook:  req.Playbook,
		Skills:    req.Skills,
		Workspace: req.Workspace,
		Tools:     req.Tools,
		Soft:      req.SoftHorizon,
		Page:      req.PageContext,
		Profile:   req.ProfileMemory,
		Plan:      planTextOf(req.Tools),
		Notes:     notes,
		Suggested: suggested,
	})
	WriteRulesIndex(req.Workspace)
	return &ContextKernel{
		req:      req,
		identity: bill.systemWire,
		pins:     bill.pinsWire,
		prefix:   bill.prefix(),
		bill:     bill,
		spill:    spill,
		notes:    notes,
	}
}

const promptUpdateLead = "Prompt update. The earlier system message stays only so the cached prefix can be reused. Follow this version.\n提示更新。前面的系统消息留在原处是为了保住缓存前缀。以这一版为准。\n"

func (k *ContextKernel) cachePrefix() string {
	if k == nil {
		return ""
	}
	if k.seriesPrefix != "" {
		return k.seriesPrefix
	}
	return k.prefix
}

func (k *ContextKernel) seed() ([]Message, error) {
	identity, pins := k.identity, k.pins
	k.seriesPrefix = k.prefix
	update := ""
	admitted := false
	latest := ""
	if head, text, ok := seriesPrompt(k.req.Trace, k.req.SessionID); ok {
		admitted = true
		latest = text
		identity, pins = head.identity, head.pins
		k.seriesPrefix = identity + pins
		if k.prefix != latest || (k.prefix != k.seriesPrefix && !transcriptHasPrompt(stripSystem(k.req.History), k.prefix)) {
			update = promptUpdateLead + k.prefix
		}
	}
	if !admitted || k.prefix != latest {
		emit(*k.req, trace.TypeSystem, "runtime", map[string]any{
			"kind":     "prompt",
			"text":     k.prefix,
			"identity": identity,
			"pins":     pins,
		})
	}
	messages := []Message{{Role: RoleSystem, Content: identity}}
	if strings.TrimSpace(pins) != "" {
		messages = append(messages, Message{Role: RoleDeveloper, Content: pins})
	}
	if len(k.req.History) > 0 {
		messages = append(messages, stripSystem(k.req.History)...)
	}
	if update != "" && !transcriptHasPrompt(messages, k.prefix) {
		emit(*k.req, trace.TypeInject, "prompt-update", map[string]any{"text": update, "name": "prompt-update"})
		messages = append(messages, Message{Role: RoleUser, Content: update})
	}
	user := NormalizeUserText(k.req.User)
	if user == "" && len(k.req.UserParts) > 0 {
		user = "(image attached)"
	}
	if user != "" && !k.req.UserCommitted {
		payload := map[string]any{"text": user}
		if len(k.req.UserParts) > 0 {
			payload["parts"] = k.req.UserParts
		}
		if rid := strings.TrimSpace(k.req.RequestID); rid != "" {
			payload["request_id"] = rid
		}
		emit(*k.req, trace.TypeUser, "user", payload)
	}
	inject := strings.TrimSpace(k.req.Inject)
	if inject != "" {
		emit(*k.req, trace.TypeInject, "mention", map[string]any{"text": inject})
	}
	if k.req.Tools != nil {
		for _, name := range k.req.Tools.Autoloaded {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			emit(*k.req, trace.TypeInject, "skill", map[string]any{"name": name, "auto": true})
		}
	}
	if user != "" {
		messages = append(messages, Message{Role: RoleUser, Content: AttachMention(user, inject), Parts: k.req.UserParts})
	} else if inject != "" {
		messages = append(messages, Message{Role: RoleUser, Content: AttachMention("", inject)})
	} else if len(stripSystem(k.req.History)) == 0 {
		return nil, fmt.Errorf("nothing to continue")
	}
	return messages, nil
}

func (k *ContextKernel) reseed(hist []Message) []Message {
	messages := []Message{{Role: RoleSystem, Content: k.identity}}
	if strings.TrimSpace(k.pins) != "" {
		messages = append(messages, Message{Role: RoleDeveloper, Content: k.pins})
	}
	return append(messages, stripSystem(hist)...)
}

type promptHead struct {
	identity string
	pins     string
}

// seriesPrompt is the latest admitted prompt after the last checkpoint.
// identity and pins are the series head that stays at the front of the request.
func seriesPrompt(st *trace.Store, sessionID string) (promptHead, string, bool) {
	if st == nil || strings.TrimSpace(sessionID) == "" {
		return promptHead{}, "", false
	}
	evs, err := st.Read(sessionID)
	if err != nil || len(evs) == 0 {
		return promptHead{}, "", false
	}
	cut := 0
	for i, ev := range evs {
		if ev.Type != trace.TypeCompact || ev.Payload == nil {
			continue
		}
		if kind, _ := ev.Payload["kind"].(string); kind == "checkpoint" {
			cut = i + 1
		}
	}
	var head promptHead
	latest := ""
	found := false
	for _, ev := range evs[cut:] {
		if ev.Type != trace.TypeSystem || ev.Source != "runtime" || ev.Payload == nil {
			continue
		}
		if kind, _ := ev.Payload["kind"].(string); kind != "prompt" {
			continue
		}
		id, _ := ev.Payload["identity"].(string)
		pins, _ := ev.Payload["pins"].(string)
		text, _ := ev.Payload["text"].(string)
		head = promptHead{identity: id, pins: pins}
		latest = text
		found = true
	}
	return head, latest, found
}

func transcriptHasPrompt(msgs []Message, prefix string) bool {
	want := promptUpdateLead + prefix
	for _, m := range msgs {
		if m.Role == RoleUser && m.Content == want {
			return true
		}
	}
	return false
}

func (k *ContextKernel) chatMode() bool {
	return k.req != nil && k.req.SoftHorizon
}

func (k *ContextKernel) refreshDynamic(messages []Message) []Message {
	ckpt := k.req.Checkpoint
	if ckpt == "" {
		ckpt = lastCheckpoint(messages)
	}
	suggested := ""
	if k.req.Tools != nil {
		suggested = FormatSkillHints(k.req.Tools.SkillHints)
	}
	page := ""
	if k.req != nil {
		page = k.req.PageContext
	}
	k.bill.Bodies = skillBodiesText(k.req.Tools)
	k.bill.Today = AssembleToday(k.req.Tools)
	k.bill.Notes = notesSection(planTextOf(k.req.Tools), k.notes, ckpt, suggested, pathRulesOf(k.req.Tools), page, k.chatMode())
	k.dyn = k.bill.dynamic()
	if k.req != nil {
		lang := languageDynPin(OperatorVoice(k.req.User, k.req.History))
		if lang != "" {
			k.dyn = lang + k.dyn
			k.bill.Notes = lang + k.bill.Notes
		}
	}
	return setDynamic(messages, k.dyn)
}

func (k *ContextKernel) prompt(messages []Message, aggressive bool) (out []Message, tools []ToolJSON, report ShapeReport) {
	messages = k.refreshDynamic(messages)
	tools = AllToolJSON(k.req.Tools)
	overhead := toolsJSONTokens(tools)
	opts := ShapeOpts{
		Loop: k.req.Loop, Spill: k.spill, ModelWindow: k.req.ModelWindow, Overhead: overhead, Aggressive: aggressive,
	}
	if k.chatMode() && !aggressive {
		// SoftHorizon skips snip/forceFit so the cache-stable prefix stays
		// intact, but write_file bodies must still be stubbed. Otherwise a
		// long desktop turn ships every full file back to the local model
		// on every round (session 477c0d8b0b8ac64e: 139 writes, 1M window
		// so checkpoint never fired).
		out, report = projectChat(messages, opts)
	} else {
		out, report = Shape(messages, opts)
		out = Legalize(out)
	}
	fillLedger(&report, k.prefix, k.dyn, tools, k.req.ModelWindow)
	k.bill.write(&report)
	report.ChatTokens = historyTokens(out)
	report.Tokens = messagesTokens(out) + overhead
	hot := hotPrefixBytes(out)
	report.PrefixHash = hotPrefixHash(out)
	report.CacheStable = k.prevHot == "" || strings.HasPrefix(hot, k.prevHot)
	k.prevHot = hot
	k.lastHash = report.PrefixHash
	return out, tools, report
}

func (k *ContextKernel) usedTokens(report ShapeReport) int {
	if report.ProviderPrompt > 0 {
		return report.ProviderPrompt
	}
	return report.Tokens
}

// chatCompactAt is where a desktop turn is checkpointed, as a fraction of the
// effective window. Compacting near the full window leaves the model reasoning
// over a degraded context for the rest of the turn. Around 65% leaves room for
// the summary and for the next few rounds. Claude Code's auto-compact sits in
// the same place (window minus a fixed buffer).
const chatCompactAt = 0.65

func (k *ContextKernel) shouldCheckpoint(report ShapeReport) bool {
	if !k.chatMode() || report.Budget <= 0 {
		return false
	}
	limit := report.Budget
	if k.req != nil && k.req.ModelWindow > 0 {
		limit = int(float64(report.Budget) * chatCompactAt)
	}
	return k.usedTokens(report) >= limit
}

func (k *ContextKernel) checkpoint(messages []Message, trigger, focus string) []Message {
	hist := forceCheckpoint(k.req.Trace, k.req.SessionID, messages, k.req.Loop, k.spill, k.req.Client, k.req.Model, k.req.ModelWindow, "runtime", k.req.Fragments, CompactOpts{Trigger: trigger, Focus: focus, Req: k.req})
	k.notes = capRunes(ReadNotes(k.spill), 2000)
	k.lastHash = ""
	k.prevHot = ""
	k.seriesPrefix = k.prefix
	if k.req.Tools != nil {
		k.req.Tools.CommitToolUnlocks()
	}
	return k.reseed(hist)
}

func capLoadedSkills(bodies []string) []string {
	if len(bodies) == 0 {
		return nil
	}
	var pinned []string
	var rest []string
	for _, body := range bodies {
		if isBootstrapBody(body) {
			pinned = append(pinned, capRunes(body, skillBootstrapCapRunes))
		} else {
			rest = append(rest, body)
		}
	}
	used := 0
	var keepPinned []string
	for _, body := range pinned {
		n := utf8.RuneCountInString(body)
		if used+n > skillTotalCapRunes {
			continue
		}
		used += n
		keepPinned = append(keepPinned, body)
	}
	var newest []string
	for i := len(rest) - 1; i >= 0; i-- {
		body := capRunes(rest[i], skillBodyCapRunes)
		n := utf8.RuneCountInString(body)
		if used+n > skillTotalCapRunes {
			continue
		}
		used += n
		newest = append(newest, body)
	}
	out := append([]string{}, keepPinned...)
	for i := len(newest) - 1; i >= 0; i-- {
		out = append(out, newest[i])
	}
	return out
}

func isBootstrapBody(s string) bool {
	return strings.Contains(s, "<"+bootstrapMarker+">")
}

func hotPrefixHash(msgs []Message) string {
	h := blake3.New()
	_, _ = h.Write([]byte(hotPrefixBytes(msgs)))
	return hex16(h.Sum(nil))
}

func hotPrefixBytes(msgs []Message) string {
	var b strings.Builder
	for _, m := range msgs {
		if m.Role == RoleMemory || strings.HasPrefix(m.Content, dynMarker) {
			continue
		}
		b.WriteString(string(m.Role))
		b.WriteByte(0)
		b.WriteString(m.Content)
		b.WriteByte(0)
		for _, tc := range m.ToolCalls {
			b.WriteString(tc.ID)
			b.WriteByte(0)
			b.WriteString(tc.Name)
			b.WriteByte(0)
			b.WriteString(tc.Arguments)
			b.WriteByte(0)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func reportPayload(report ShapeReport, extra map[string]any) map[string]any {
	p := map[string]any{
		"kind":              "shape",
		"note":              report.Note,
		"tokens":            report.Tokens,
		"budget":            report.Budget,
		"window":            report.Window,
		"prefix_tokens":     report.PrefixTokens,
		"dynamic_tokens":    report.DynamicTokens,
		"schema_tokens":     report.SchemaTokens,
		"identity_tokens":   report.IdentityTokens,
		"rules_tokens":      report.RulesTokens,
		"catalog_tokens":    report.CatalogTokens,
		"skill_body_tokens": report.SkillBodyTokens,
		"memory_tokens":     report.MemoryTokens,
		"today_tokens":      report.TodayTokens,
		"notes_tokens":      report.NotesTokens,
		"chat_tokens":       report.ChatTokens,
		"layers":            report.Layers,
		"elided":            report.Elided,
		"cache_stable":      report.CacheStable,
		"prefix_hash":       report.PrefixHash,
		"dynamic_at":        report.DynamicAt,
		"cached_tokens":     report.CachedTokens,
		"cache_reported":    report.CacheReported,
		"trigger":           report.Trigger,
		"hydrated":          report.Hydrated,
	}
	if report.ProviderPrompt > 0 {
		p["provider_prompt"] = report.ProviderPrompt
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}
