package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// ShapeReport is the observable context-engineering ledger (Cursor's
// context-ring equivalent). Layers are cheapest-first, Claude Code order.
type ShapeReport struct {
	Note   string   `json:"note"`
	Tokens int      `json:"tokens"`
	Budget int      `json:"budget"`
	Layers []string `json:"layers,omitempty"`
	Elided int      `json:"elided,omitempty"`
}

type ShapeOpts struct {
	Loop  artifact.LoopPreset
	Spill *Spill
}

// Compact is the eval-stable entry point: deterministic shapers only.
func Compact(msgs []Message, loop artifact.LoopPreset) ([]Message, string) {
	out, rep := Shape(msgs, ShapeOpts{Loop: loop})
	return out, rep.Note
}

// Shape is a read-time projection. The live transcript is not mutated.
// Order: budget → snip → microcompact → collapse. LLM summarization is
// opt-in (AllowLLMCompact) because ACE forbids collapsing the playbook
// and Harbor evals must stay reproducible.
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
	budget := loop.CompactionTokens
	if budget <= 0 {
		budget = 24_000
	}
	per := loop.ToolResultBudget
	if per <= 0 {
		per = 8_000
	}
	microKeep := loop.MicroKeep
	if microKeep <= 0 {
		microKeep = 4
	}
	rep.Budget = budget
	out := copyMessages(msgs)

	if n := applyBudget(out, per, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "budget")
		rep.Elided += n
	}
	if snipped := snipWindow(out, keep); snipped != nil {
		out = snipped
		rep.Layers = append(rep.Layers, "snip")
	}
	if n := microcompact(out, microKeep, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "microcompact")
		rep.Elided += n
	}
	collapsed, n := collapseOldTools(out, microKeep)
	if n > 0 {
		out = collapsed
		rep.Layers = append(rep.Layers, "collapse")
		rep.Elided += n
	}
	if messagesTokens(out) > budget {
		if n := forceFit(out, budget, opts.Spill); n > 0 {
			rep.Layers = append(rep.Layers, "force")
			rep.Elided += n
		}
	}
	rep.Tokens = messagesTokens(out)
	rep.Note = strings.Join(rep.Layers, "+")
	return out, rep
}

func copyMessages(msgs []Message) []Message {
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out
}

func applyBudget(msgs []Message, per int, spill *Spill) int {
	n := 0
	for i := range msgs {
		if msgs[i].Role != RoleTool {
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

func snipWindow(msgs []Message, keep int) []Message {
	if len(msgs) <= keep+1 {
		return nil
	}
	head := []Message{msgs[0]}
	return append(head, msgs[len(msgs)-keep:]...)
}

func microcompact(msgs []Message, keepLast int, spill *Spill) int {
	ids := toolIndexes(msgs)
	if len(ids) <= keepLast {
		return 0
	}
	n := 0
	cutoff := ids[len(ids)-keepLast]
	for i := 1; i < cutoff; i++ {
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

func collapseOldTools(msgs []Message, keepLast int) ([]Message, int) {
	ids := toolIndexes(msgs)
	if len(ids) <= keepLast+2 {
		return msgs, 0
	}
	cutoff := ids[len(ids)-keepLast]
	n := 0
	out := make([]Message, 0, len(msgs))
	i := 0
	for i < len(msgs) {
		if i >= cutoff || msgs[i].Role != RoleTool {
			out = append(out, msgs[i])
			i++
			continue
		}
		j := i
		var names []string
		for j < cutoff && msgs[j].Role == RoleTool {
			names = append(names, msgs[j].Name)
			j++
		}
		if j-i < 3 {
			out = append(out, msgs[i:j]...)
			i = j
			continue
		}
		out = append(out, Message{
			Role:       RoleTool,
			ToolCallID: msgs[i].ToolCallID,
			Name:       "collapse",
			Content:    fmt.Sprintf("[collapsed %d tool results: %s]", j-i, strings.Join(names, ", ")),
		})
		n += j - i - 1
		i = j
	}
	return out, n
}

func forceFit(msgs []Message, budget int, spill *Spill) int {
	n := 0
	for i := 1; i < len(msgs) && messagesTokens(msgs) > budget; i++ {
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
