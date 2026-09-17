package runtime

import (
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MeasureOpts is the same bill of materials the live loop uses to Shape a turn.
type MeasureOpts struct {
	Loop        artifact.LoopPreset
	Fragments   []artifact.PromptFragment
	Playbook    artifact.Playbook
	Skills      []artifact.Skill
	Workspace   string
	History     []Message
	Tools       *WorkspaceTools
	Spill       *Spill
	ModelWindow int
	PlanText    string
	Notes       string
}

// MeasureContext is a read-only projection of what the next model call would
// send: prefix + dynamic tail + tool schemas + shaped history. It does not
// depend on the in-RAM last ShapeReport, so usage survives process restart.
func MeasureContext(opts MeasureOpts) ShapeReport {
	loop := opts.Loop
	rules, src := LoadWorkspaceRules(opts.Workspace, loop.RulesTokens)
	prefix := AssemblePrefix(loop, opts.Fragments, opts.Playbook, opts.Skills, rules, src)
	notes := opts.Notes
	if notes == "" && opts.Spill != nil {
		notes = ReadNotes(opts.Spill)
	}
	plan := opts.PlanText
	if plan == "" {
		plan = planTextOf(opts.Tools)
	}
	dyn := AssembleDynamic(plan, loadedFrom(opts.Tools), notes, "")
	msgs := []Message{{Role: RoleSystem, Content: prefix + dyn}}
	msgs = append(msgs, stripSystem(opts.History)...)
	toolsJSON := AllToolJSON(opts.Tools)
	overhead := toolsJSONTokens(toolsJSON)
	compacted, report := Shape(msgs, ShapeOpts{
		Loop: loop, Spill: opts.Spill, ModelWindow: opts.ModelWindow, Overhead: overhead,
	})
	compacted = Legalize(compacted)
	fillLedger(&report, prefix, dyn, toolsJSON, opts.ModelWindow)
	report.Tokens = messagesTokens(compacted) + overhead
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
			Note:           payloadStr(p, "note"),
			Tokens:         tokens,
			Budget:         payloadInt(p, "budget"),
			Window:         payloadInt(p, "window"),
			PrefixTokens:   prefix,
			DynamicTokens:  payloadInt(p, "dynamic_tokens"),
			SchemaTokens:   payloadInt(p, "schema_tokens"),
			ProviderPrompt: payloadInt(p, "provider_prompt"),
			Elided:         payloadInt(p, "elided"),
			Layers:         payloadStrings(p, "layers"),
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
