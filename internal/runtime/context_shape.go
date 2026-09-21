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
	CachedTokens   int      `json:"cached_tokens,omitempty"`
	CacheReported  bool     `json:"cache_reported,omitempty"`
	CacheStable    bool     `json:"cache_stable,omitempty"`
	PrefixHash     string   `json:"prefix_hash,omitempty"`
	DynamicAt      string   `json:"dynamic_at,omitempty"`
	Trigger        string   `json:"trigger,omitempty"`
	Hydrated       int      `json:"hydrated,omitempty"`
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
// budget (spill+preview) → stub heavy write args → microcompact stubs →
// if still over budget, stub remaining write args more tightly then
// forceFit results, then snip old turns into a spill pointer.
// User/assistant text is never rewritten; tool bodies are stubbed with
// recall ids. LLM summarization is not part of Shape.
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
	if n := stubHeavyCalls(out, per, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "calls")
		rep.Elided += n
	}
	if n := microcompact(out, microKeep, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "microcompact")
		rep.Elided += n
	}
	if messagesTokens(out)+opts.Overhead > budget {
		callLimit := forceCallRunes
		if opts.Aggressive {
			callLimit = aggressiveCallRunes
		}
		if n := stubHeavyCalls(out, callLimit, opts.Spill); n > 0 {
			rep.Layers = appendLayer(rep.Layers, "calls")
			rep.Elided += n
		}
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
	var saved, rest []Message
	for _, m := range msgs[head:cut] {
		if keepThroughSnip(m) {
			saved = append(saved, m)
		} else {
			rest = append(rest, m)
		}
	}
	if len(saved) > 8 {
		saved = append(saved[:2], saved[len(saved)-6:]...)
	}
	if len(rest) == 0 {
		return nil, 0
	}
	id := "snip"
	if spill != nil {
		raw, _ := json.Marshal(rest)
		id = spill.Put("", string(raw))
	}
	marker := Message{
		Role: RoleUser,
		Content: fmt.Sprintf(
			"[elided %d earlier messages id=%s — call recall_context with this id; original transcript is intact on disk]",
			len(rest), id,
		),
	}
	if paths := writePathsFromMessages(rest); len(paths) > 0 {
		marker.Content += "\nRecent writes still on disk (read_file, do not rewrite from memory): " + strings.Join(paths, ", ")
	}
	out := make([]Message, 0, head+len(saved)+1+len(msgs)-cut)
	out = append(out, msgs[:head]...)
	out = append(out, saved...)
	out = append(out, marker)
	out = append(out, msgs[cut:]...)
	return out, len(rest)
}

func keepThroughSnip(m Message) bool {
	if m.Role != RoleUser {
		return false
	}
	c := strings.TrimSpace(m.Content)
	if c == "" || alreadyStubbed(c) || isControlUser(c) {
		return false
	}
	return true
}

func pinnedPrefix(msgs []Message) int {
	n := 0
	for n < len(msgs) && (msgs[n].Role == RoleSystem || msgs[n].Role == RoleDeveloper) {
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
	hot := map[int]bool{}
	if keepLast >= 16 {
		for idx := range lastPathToolIndexes(msgs, ids, 12) {
			hot[idx] = true
		}
	}
	n := 0
	cutoff := ids[len(ids)-keepLast]
	start := pinnedPrefix(msgs)
	for i := start; i < cutoff; i++ {
		if msgs[i].Role != RoleTool || alreadyStubbed(msgs[i].Content) {
			continue
		}
		if hot[i] {
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

func lastPathToolIndexes(msgs []Message, ids []int, capPaths int) map[int]bool {
	keep := map[int]bool{}
	if capPaths <= 0 {
		return keep
	}
	seen := map[string]bool{}
	for i := len(ids) - 1; i >= 0; i-- {
		idx := ids[i]
		p := toolResultPath(msgs, idx)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		keep[idx] = true
		if len(seen) >= capPaths {
			break
		}
	}
	return keep
}

func toolResultPath(msgs []Message, toolIdx int) string {
	if toolIdx < 0 || toolIdx >= len(msgs) {
		return ""
	}
	id := msgs[toolIdx].ToolCallID
	name := msgs[toolIdx].Name
	for i := toolIdx - 1; i >= 0; i-- {
		if msgs[i].Role != RoleAssistant {
			continue
		}
		for _, tc := range msgs[i].ToolCalls {
			if (id != "" && tc.ID == id) || (id == "" && tc.Name == name) {
				if paths := extractJSONPaths(tc.Arguments); len(paths) > 0 {
					return paths[0]
				}
			}
		}
		if id != "" {
			break
		}
	}
	return ""
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

const heavyCallRunes = 800
const forceCallRunes = 240
const aggressiveCallRunes = 80

func appendLayer(layers []string, name string) []string {
	for _, l := range layers {
		if l == name {
			return layers
		}
	}
	return append(layers, name)
}

func heavyCallName(name string) bool {
	switch name {
	case "write_file", "create_file", "apply_patch", "str_replace", "edit_file",
		"office_create", "office_edit", "cite_sources":
		return true
	default:
		return false
	}
}

// stubHeavyCalls is a Shape projection: write_file/apply_patch bodies live on
// disk (I1 via read_file). Keeping every full write in the prompt makes the
// model go blind on results and rewrite the tree from memory.
func stubHeavyCalls(msgs []Message, per int, spill *Spill) int {
	limit := heavyCallRunes
	if per > 0 && per < limit {
		limit = per
	}
	n := 0
	for i := range msgs {
		if msgs[i].Role != RoleAssistant {
			continue
		}
		for j := range msgs[i].ToolCalls {
			tc := &msgs[i].ToolCalls[j]
			if !heavyCallName(tc.Name) || alreadyStubbed(tc.Arguments) {
				continue
			}
			compact := compactCallArgs(tc.Name, tc.Arguments, limit)
			if compact == tc.Arguments || len(compact) >= len(tc.Arguments) {
				continue
			}
			if spill != nil && tc.ID != "" {
				_ = spill.Put(tc.ID+":args", tc.Arguments)
			}
			tc.Arguments = compact
			n++
		}
	}
	return n
}

func compactCallArgs(name, args string, limit int) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return args
	}
	path, _ := m["path"].(string)
	changed := false
	keys := []string{"content", "patch", "body"}
	if name == "str_replace" || name == "edit_file" || name == "office_edit" {
		keys = append(keys, "old_str", "new_str")
	}
	for _, key := range keys {
		s, ok := m[key].(string)
		if !ok {
			continue
		}
		if len([]rune(s)) <= limit {
			continue
		}
		head := callArgHead(s, 120)
		label := path
		if label == "" {
			label = name
		}
		m[key] = fmt.Sprintf("[elided %s %d chars path=%s — on disk, read_file that path; do not rewrite from memory]\n%s", key, len(s), label, head)
		changed = true
	}
	if !changed {
		return args
	}
	b, err := json.Marshal(m)
	if err != nil {
		return args
	}
	return string(b)
}

func callArgHead(s string, keep int) string {
	if keep <= 0 {
		keep = 120
	}
	r := []rune(s)
	if len(r) <= keep {
		return s
	}
	return string(r[:keep])
}

func writePathsFromMessages(msgs []Message) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range msgs {
		if m.Role != RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if tc.Name != "write_file" && tc.Name != "str_replace" && tc.Name != "apply_patch" {
				continue
			}
			for _, p := range extractJSONPaths(tc.Arguments) {
				if seen[p] {
					continue
				}
				seen[p] = true
				out = append(out, p)
				if len(out) >= 16 {
					return out
				}
			}
		}
	}
	return out
}
