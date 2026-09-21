package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/zeebo/blake3"
)

const (
	skillBodyCapRunes  = 5_000
	skillTotalCapRunes = 25_000
)

// ContextKernel owns pin/hot/dynamic assembly, the prefix contract, and
// checkpoint. Run consumes Prompt / Checkpoint; it does not reshape history.
type ContextKernel struct {
	req      *RunRequest
	identity string
	pins     string
	prefix   string
	spill    *Spill
	notes    string
	lastHash string
	prevHot  string
	dyn      string
}

func newContextKernel(req *RunRequest) *ContextKernel {
	loop := req.Loop
	rules, src := LoadWorkspaceRules(req.Workspace, loop.RulesTokens)
	if loop.StackAgentsMD {
		if stacked := LoadStackedAgents(req.Workspace, 32*1024); stacked != "" {
			rules = stacked
			src = "AGENTS.md"
		}
	}
	identity := AssembleIdentity(loop, req.Fragments)
	pins := AssemblePins(loop, req.Playbook, req.Skills, rules, src)
	if strings.TrimSpace(req.ProfileMemory) != "" {
		pins += "\n## Profile memory\n" + req.ProfileMemory
	}
	if mem := ReadWorkspaceMemory(req.Workspace); mem != "" {
		pins += "\n## Project memory\n" + mem
	}
	WriteRulesIndex(req.Workspace)
	return &ContextKernel{
		req:      req,
		identity: identity,
		pins:     pins,
		prefix:   identity + pins,
		spill:    spillOf(*req),
		notes:    capRunes(ReadNotes(spillOf(*req)), 2000),
	}
}

func (k *ContextKernel) seed() ([]Message, error) {
	messages := []Message{{Role: RoleSystem, Content: k.identity}}
	if strings.TrimSpace(k.pins) != "" {
		messages = append(messages, Message{Role: RoleDeveloper, Content: k.pins})
	}
	if len(k.req.History) > 0 {
		messages = append(messages, stripSystem(k.req.History)...)
	}
	if strings.TrimSpace(k.req.Inject) != "" {
		emit(*k.req, trace.TypeInject, "mention", map[string]any{"text": k.req.Inject})
		messages = append(messages, Message{Role: RoleUser, Content: mentionUserPrefix + "\n" + k.req.Inject})
	}
	if user := strings.TrimSpace(k.req.User); user != "" {
		emit(*k.req, trace.TypeUser, "user", map[string]any{"text": k.req.User})
		messages = append(messages, Message{Role: RoleUser, Content: k.req.User})
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

func (k *ContextKernel) chatMode() bool {
	return k.req != nil && k.req.SoftHorizon
}

func (k *ContextKernel) refreshDynamic(messages []Message) []Message {
	ckpt := k.req.Checkpoint
	if ckpt == "" {
		ckpt = lastCheckpoint(messages)
	}
	loaded := capLoadedSkills(loadedFrom(k.req.Tools))
	k.dyn = AssembleDynamic(planTextOf(k.req.Tools), loaded, k.notes, ckpt, AssembleToday(k.req.Tools))
	k.dyn = withChatVoice(*k.req, k.dyn, k.notes)
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
		out = Legalize(copyMessages(messages))
		report.Budget = effectiveBudget(opts)
	} else {
		out, report = Shape(messages, opts)
		out = Legalize(out)
	}
	fillLedger(&report, k.prefix, k.dyn, tools, k.req.ModelWindow)
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

func (k *ContextKernel) shouldCheckpoint(report ShapeReport) bool {
	if !k.chatMode() || report.Budget <= 0 {
		return false
	}
	return k.usedTokens(report) >= report.Budget
}

func (k *ContextKernel) checkpoint(messages []Message, trigger, focus string) []Message {
	hist := forceCheckpoint(k.req.Trace, k.req.SessionID, messages, k.req.Loop, k.spill, k.req.Client, k.req.Model, k.req.ModelWindow, "runtime", k.req.Fragments, CompactOpts{Trigger: trigger, Focus: focus, Req: k.req})
	k.notes = capRunes(ReadNotes(k.spill), 2000)
	k.lastHash = ""
	k.prevHot = ""
	if k.req.Tools != nil {
		k.req.Tools.CommitToolUnlocks()
	}
	return k.reseed(hist)
}

func capLoadedSkills(bodies []string) []string {
	if len(bodies) == 0 {
		return nil
	}
	used := 0
	var newest []string
	for i := len(bodies) - 1; i >= 0; i-- {
		body := capRunes(bodies[i], skillBodyCapRunes)
		n := utf8.RuneCountInString(body)
		if used+n > skillTotalCapRunes {
			continue
		}
		used += n
		newest = append(newest, body)
	}
	out := make([]string, 0, len(newest))
	for i := len(newest) - 1; i >= 0; i-- {
		out = append(out, newest[i])
	}
	return out
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
		"kind":            "shape",
		"note":            report.Note,
		"tokens":          report.Tokens,
		"budget":          report.Budget,
		"window":          report.Window,
		"prefix_tokens":   report.PrefixTokens,
		"dynamic_tokens":  report.DynamicTokens,
		"schema_tokens":   report.SchemaTokens,
		"layers":          report.Layers,
		"elided":          report.Elided,
		"cache_stable":    report.CacheStable,
		"prefix_hash":     report.PrefixHash,
		"dynamic_at":      report.DynamicAt,
		"cached_tokens":   report.CachedTokens,
		"cache_reported":  report.CacheReported,
		"trigger":         report.Trigger,
		"hydrated":        report.Hydrated,
	}
	if report.ProviderPrompt > 0 {
		p["provider_prompt"] = report.ProviderPrompt
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}
