// 下一轮模型调用的只读计量。按组提示的同一套材料计算，不读进程里上一份 ShapeReport，因此重启后仍能报用量。
// 不改轨迹。有模型窗口时走 projectChat，否则 Shape 后再 Legalize。动态块接在历史后面。

package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MeasureOpts 是计量用的材料，和循环组一轮提示时用的是同一批。
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

// MeasureContext 投影下一轮会发给模型的内容并填写账本。笔记为空且有 spill 时读 spill 里的 notes。
// ModelWindow 大于 0 时用 projectChat，否则 Shape 后再 Legalize。ChatTokens 只数历史。CacheStable 固定写成 true。
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

// ShapeFromEvents 从最后一条带用量的 compact 或 turn_end 恢复账本。tokens 和 prefix_tokens 都是 0 的事件跳过。
// 一条都没有时，window 大于 0 就只返回窗口。事件自己的窗口为 0 时用参数补上。
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

// payloadStrings 读取字符串切片，或 any 切片里的非空字符串。键缺失或类型不符时返回 nil。
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
