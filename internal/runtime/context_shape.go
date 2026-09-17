package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// ShapeReport is the observable context-engineering ledger.
type ShapeReport struct {
	Note           string   `json:"note"`
	Tokens         int      `json:"tokens"`
	Budget         int      `json:"budget"`
	Window         int      `json:"window,omitempty"`
	PrefixTokens   int      `json:"prefix_tokens,omitempty"`
	DynamicTokens  int      `json:"dynamic_tokens,omitempty"`
	SchemaTokens   int      `json:"schema_tokens,omitempty"`
	ProviderPrompt int      `json:"provider_prompt,omitempty"`
	Layers         []string `json:"layers,omitempty"`
	Elided         int      `json:"elided,omitempty"`
}

type ShapeOpts struct {
	Loop          artifact.LoopPreset
	Spill         *Spill
	ModelWindow   int
	OutputReserve int
	Overhead      int
	Aggressive    bool
}

// Compact is the eval-stable entry point: deterministic shapers only.
func Compact(msgs []Message, loop artifact.LoopPreset) ([]Message, string) {
	out, rep := Shape(msgs, ShapeOpts{Loop: loop})
	return out, rep.Note
}

// Shape is a read-time projection. The live transcript is not mutated.
// Order is cheapest-first and lossless where possible: legalize-prep →
// budget (spill+preview) → microcompact stubs → forceFit stubs → snip old
// turns into a spill pointer. User/assistant text is never rewritten; tool
// bodies are stubbed with recall ids. LLM summarization is not part of Shape.
func Shape(msgs []Message, opts ShapeOpts) ([]Message, ShapeReport) {
	rep := ShapeReport{}
	if len(msgs) == 0 {
		return msgs, rep
	}
	loop := opts.Loop
	keep := loop.CompactionKeep
	if keep <= 0 {
		keep = 24
	}
	if opts.Aggressive && keep > 8 {
		keep = 8
	}
	budget := effectiveBudget(opts)
	per := loop.ToolResultBudget
	if per <= 0 {
		per = 8_000
	}
	if opts.Aggressive && per > 1_500 {
		per = 1_500
	}
	microKeep := loop.MicroKeep
	if microKeep <= 0 {
		microKeep = 4
	}
	if opts.Aggressive {
		microKeep = 1
	}
	rep.Budget = budget
	out := copyMessages(msgs)

	if n := applyBudget(out, per, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "budget")
		rep.Elided += n
	}
	if n := microcompact(out, microKeep, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "microcompact")
		rep.Elided += n
	}
	if messagesTokens(out)+opts.Overhead > budget {
		if n := forceFit(out, budget-opts.Overhead, opts.Spill); n > 0 {
			rep.Layers = append(rep.Layers, "force")
			rep.Elided += n
		}
	}
	if messagesTokens(out)+opts.Overhead > budget {
		if snipped, n := snipWindow(out, keep, opts.Spill); snipped != nil {
			out = snipped
			rep.Layers = append(rep.Layers, "snip")
			rep.Elided += n
		}
	}
	rep.Tokens = messagesTokens(out)
	rep.Note = strings.Join(rep.Layers, "+")
	return out, rep
}

func copyMessages(msgs []Message) []Message {
	out := make([]Message, len(msgs))
	for i, m := range msgs {
		out[i] = m
		if len(m.ToolCalls) > 0 {
			out[i].ToolCalls = append([]ToolCall(nil), m.ToolCalls...)
		}
	}
	return out
}

func applyBudget(msgs []Message, per int, spill *Spill) int {
	n := 0
	for i := range msgs {
		if msgs[i].Role != RoleTool {
			continue
		}
		if alreadyStubbed(msgs[i].Content) {
			continue
		}
		capped, trunc := capText(msgs[i].Content, per)
		if !trunc {
			continue
		}
		id := msgs[i].ToolCallID
		if spill != nil {
			id = spill.Put(id, msgs[i].Content)
		}
		msgs[i].Content = stubTool(id, msgs[i].Name, len(msgs[i].Content)) + "\n" + capped
		n++
	}
	return n
}

func snipWindow(msgs []Message, keep int, spill *Spill) ([]Message, int) {
	head := pinnedPrefix(msgs)
	if len(msgs) <= keep+head {
		return nil, 0
	}
	cut := pairingBoundary(msgs, len(msgs)-keep)
	if cut <= head {
		return nil, 0
	}
	dropped := msgs[head:cut]
	id := "snip"
	if spill != nil {
		raw, _ := json.Marshal(dropped)
		id = spill.Put("", string(raw))
	}
	marker := Message{
		Role: RoleUser,
		Content: fmt.Sprintf(
			"[elided %d earlier messages id=%s — call recall_context with this id; original transcript is intact on disk]",
			len(dropped), id,
		),
	}
	out := make([]Message, 0, head+1+len(msgs)-cut)
	out = append(out, msgs[:head]...)
	out = append(out, marker)
	out = append(out, msgs[cut:]...)
	return out, len(dropped)
}

func pinnedPrefix(msgs []Message) int {
	n := 0
	for n < len(msgs) && (msgs[n].Role == RoleSystem || msgs[n].Role == RoleDeveloper || msgs[n].Role == RoleMemory) {
		n++
	}
	if n == 0 {
		return 1
	}
	return n
}

func microcompact(msgs []Message, keepLast int, spill *Spill) int {
	ids := toolIndexes(msgs)
	if len(ids) <= keepLast {
		return 0
	}
	n := 0
	cutoff := ids[len(ids)-keepLast]
	start := pinnedPrefix(msgs)
	for i := start; i < cutoff; i++ {
		if msgs[i].Role != RoleTool || alreadyStubbed(msgs[i].Content) {
			continue
		}
		if len(msgs[i].Content) < 400 {
			continue
		}
		id := msgs[i].ToolCallID
		if spill != nil {
			id = spill.Put(id, msgs[i].Content)
		}
		msgs[i].Content = stubTool(id, msgs[i].Name, len(msgs[i].Content))
		n++
	}
	return n
}

func forceFit(msgs []Message, budget int, spill *Spill) int {
	if budget < 0 {
		budget = 0
	}
	n := 0
	start := pinnedPrefix(msgs)
	for i := start; i < len(msgs) && messagesTokens(msgs) > budget; i++ {
		if msgs[i].Role != RoleTool || alreadyStubbed(msgs[i].Content) {
			continue
		}
		id := msgs[i].ToolCallID
		if spill != nil {
			id = spill.Put(id, msgs[i].Content)
		}
		msgs[i].Content = stubTool(id, msgs[i].Name, len(msgs[i].Content))
		n++
	}
	return n
}

func toolIndexes(msgs []Message) []int {
	var ids []int
	for i, m := range msgs {
		if m.Role == RoleTool {
			ids = append(ids, i)
		}
	}
	return ids
}

func alreadyStubbed(s string) bool {
	return strings.HasPrefix(s, "[elided ") || strings.HasPrefix(s, "[collapsed ")
}

func stubTool(id, name string, bytes int) string {
	if id == "" {
		id = "unknown"
	}
	if name == "" {
		name = "tool"
	}
	return fmt.Sprintf("[elided tool_result id=%s name=%s bytes=%d — call recall_context with this id]", id, name, bytes)
}
