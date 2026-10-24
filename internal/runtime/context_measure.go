package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MeasureOpts is the same bill of materials the live loop uses to Shape a turn.
type MeasureOpts struct {
	Loop          artifact.LoopPreset
	Fragments     []artifact.PromptFragment
	Playbook      artifact.Playbook
	Skills        []artifact.Skill
	Workspace     string
	History       []Message
	Tools         *WorkspaceTools
	Spill         *Spill
	ModelWindow   int
	PlanText      string
	Notes         string
	SoftHorizon   bool
	PageContext   string
	ProfileMemory string
}

// MeasureContext is a read-only projection of what the next model call would
// send: prefix + dynamic tail + tool schemas + shaped history. It does not
// depend on the in-RAM last ShapeReport, so usage survives process restart.
func MeasureContext(opts MeasureOpts) ShapeReport {
	loop := opts.Loop
	notes := opts.Notes
	if notes == "" && opts.Spill != nil {
		notes = ReadNotes(opts.Spill)
	}
	plan := opts.PlanText
	if plan == "" {
		plan = planTextOf(opts.Tools)
	}
	suggested := ""
	if opts.Tools != nil {
		suggested = FormatSkillHints(opts.Tools.SkillHints)
	}
	soft := opts.SoftHorizon
	if opts.Tools != nil && opts.Tools.ChatOverlay {
		soft = true
	}
	bill := buildBill(billInput{
		Loop:      loop,
		Fragments: opts.Fragments,
		Playbook:  opts.Playbook,
		Skills:    opts.Skills,
		Workspace: opts.Workspace,
		Tools:     opts.Tools,
		Soft:      soft,
		Page:      opts.PageContext,
		Profile:   opts.ProfileMemory,
		Plan:      plan,
		Notes:     notes,
		Suggested: suggested,
	})
	prefix := bill.prefix()
	dyn := bill.dynamic()
	msgs := []Message{{Role: RoleSystem, Content: bill.systemWire}}
	if strings.TrimSpace(bill.pinsWire) != "" {
		msgs = append(msgs, Message{Role: RoleDeveloper, Content: bill.pinsWire})
	}
	msgs = append(msgs, stripSystem(opts.History)...)
	msgs = setDynamic(msgs, dyn)
	toolsJSON := AllToolJSON(opts.Tools)
	overhead := toolsJSONTokens(toolsJSON)
	var compacted []Message
	var report ShapeReport
	if opts.ModelWindow > 0 {
		compacted, report = projectChat(msgs, ShapeOpts{
			Loop: loop, Spill: opts.Spill, ModelWindow: opts.ModelWindow, Overhead: overhead,
		})
	} else {
		compacted, report = Shape(msgs, ShapeOpts{
			Loop: loop, Spill: opts.Spill, ModelWindow: opts.ModelWindow, Overhead: overhead,
		})
		compacted = Legalize(compacted)
	}
	fillLedger(&report, prefix, dyn, toolsJSON, opts.ModelWindow)
	bill.write(&report)
	report.ChatTokens = historyTokens(compacted)
	report.Tokens = messagesTokens(compacted) + overhead
	report.PrefixHash = hotPrefixHash(compacted)
	report.CacheStable = true
	return report
}

// ShapeFromEvents recovers a ledger from the last compact/turn_end payload
// when materials cannot be assembled (missing harness). Empty shape events
// are ignored — those are never written.
func ShapeFromEvents(evs []trace.Event, window int) ShapeReport {
	var last ShapeReport
	found := false
	for _, ev := range evs {
		if ev.Type != trace.TypeCompact && ev.Type != trace.TypeTurnEnd {
			continue
		}
		p := ev.Payload
		tokens := payloadInt(p, "tokens")
		prefix := payloadInt(p, "prefix_tokens")
		if tokens == 0 && prefix == 0 {
			continue
		}
		found = true
		last = ShapeReport{
			Note:            payloadStr(p, "note"),
			Tokens:          tokens,
			Budget:          payloadInt(p, "budget"),
			Window:          payloadInt(p, "window"),
			PrefixTokens:    prefix,
			DynamicTokens:   payloadInt(p, "dynamic_tokens"),
			SchemaTokens:    payloadInt(p, "schema_tokens"),
			IdentityTokens:  payloadInt(p, "identity_tokens"),
			RulesTokens:     payloadInt(p, "rules_tokens"),
			CatalogTokens:   payloadInt(p, "catalog_tokens"),
			SkillBodyTokens: payloadInt(p, "skill_body_tokens"),
			MemoryTokens:    payloadInt(p, "memory_tokens"),
			TodayTokens:     payloadInt(p, "today_tokens"),
			NotesTokens:     payloadInt(p, "notes_tokens"),
			ChatTokens:      payloadInt(p, "chat_tokens"),
			ProviderPrompt:  payloadInt(p, "provider_prompt"),
			CachedTokens:    payloadInt(p, "cached_tokens"),
			CacheReported:   payloadInt(p, "cached_tokens") > 0 || payloadBool(p, "cache_reported"),
			CacheStable:     payloadBool(p, "cache_stable"),
			PrefixHash:      payloadStr(p, "prefix_hash"),
			DynamicAt:       payloadStr(p, "dynamic_at"),
			Trigger:         payloadStr(p, "trigger"),
			Hydrated:        payloadInt(p, "hydrated"),
			Elided:          payloadInt(p, "elided"),
			Layers:          payloadStrings(p, "layers"),
		}
	}
	if !found {
		if window > 0 {
			return ShapeReport{Window: window}
		}
		return ShapeReport{}
	}
	if last.Window == 0 {
		last.Window = window
	}
	return last
}

func payloadStrings(p map[string]any, key string) []string {
	if p == nil {
		return nil
	}
	switch v := p[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := payloadString(item)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
