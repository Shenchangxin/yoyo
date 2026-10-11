package runtime

import (
	"crypto/sha256"
	"encoding/hex"
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

// projectChat is the desktop-chat prompt projection: Legalize, spill
// previews, and receipts for old re-fetchable reads. Write arguments stay
// intact — replacing them with placeholders taught the model to emit
// placeholders. It does not snip or forceFit, so the identity/pins prefix
// stays cache-stable. Live transcript is not mutated.
func projectChat(msgs []Message, opts ShapeOpts) ([]Message, ShapeReport) {
	rep := ShapeReport{Budget: effectiveBudget(opts)}
	if len(msgs) == 0 {
		return msgs, rep
	}
	out := copyMessages(msgs)
	per := opts.Loop.ToolResultBudget
	if per <= 0 {
		per = 8_000
	}
	if n := applyBudget(out, per, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "budget")
		rep.Elided += n
	}
	if n := clearStaleReads(out, chatKeepReads); n > 0 {
		rep.Layers = append(rep.Layers, "reads")
		rep.Elided += n
	}
	rep.Note = strings.Join(rep.Layers, "+")
	return Legalize(out), rep
}

// chatKeepReads is how many of the newest read-only results stay verbatim in
// the desktop prompt. Older ones become receipts (see clearStaleReads).
const chatKeepReads = 12

// refetchableTool names results that can be produced again from the workspace.
// Writes, shell output, and browser state are never cleared: they are not
// reproducible from the workspace alone.
func refetchableTool(name string) bool {
	switch name {
	case "read_file", "grep", "glob", "list_dir", "git_status", "git_diff":
		return true
	default:
		return false
	}
}

// clearStaleReads turns older re-fetchable results into receipts. The assistant
// tool call that produced each result stays in the prompt, so the model keeps
// the record that it read the file. The receipt carries the path, size, and a
// hash of what was shown, so the model can tell whether a later read changed
// anything. This is the cleared-tool-results pattern from Anthropic's context
// editing, applied with the same rule: the call is kept, the bulky observation
// is dropped, and it can be recovered by calling the tool again.
func clearStaleReads(msgs []Message, keep int) int {
	var ids []int
	for i, m := range msgs {
		if m.Role == RoleTool && refetchableTool(m.Name) {
			ids = append(ids, i)
		}
	}
	if len(ids) <= keep {
		return 0
	}
	cut := ids[len(ids)-keep]
	n := 0
	for _, i := range ids {
		if i >= cut {
			break
		}
		if alreadyStubbed(msgs[i].Content) || len(msgs[i].Content) < 400 {
			continue
		}
		msgs[i].Content = readReceipt(msgs, i)
		n++
	}
	return n
}

// readReceipt is the replacement body for a cleared read. It is plain prose
// with real identifiers, so nothing in it can be mistaken for tool arguments.
func readReceipt(msgs []Message, toolIdx int) string {
	m := msgs[toolIdx]
	path := toolResultPath(msgs, toolIdx)
	if path == "" {
		path = "(unknown path)"
	}
	sum := sha256.Sum256([]byte(m.Content))
	return fmt.Sprintf("[cleared %s path=%s bytes=%d sha256=%s — this earlier result left the prompt; call read_file again to see the current content]",
		m.Name, path, len(m.Content), hex.EncodeToString(sum[:6]))
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
	if n := microcompact(out, microKeep, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "microcompact")
		rep.Elided += n
	}
	if messagesTokens(out)+opts.Overhead > budget {
		// Tool-call arguments are never elided here. A placeholder in the
		// arguments teaches the model to emit placeholders (session
		// facb2e21ba0b31a3). Results are stubbed instead, and they carry
		// recall ids.
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
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[elided ") || strings.HasPrefix(s, "[collapsed ") || strings.HasPrefix(s, "(omitted ") || strings.HasPrefix(s, "[cleared ") {
		return true
	}
	return strings.Contains(s, `"_elided"`)
}

func looksLikeContextStub(s string) bool {
	if strings.Contains(s, "[elided ") || strings.Contains(s, "[collapsed ") {
		return true
	}
	if strings.Contains(s, "(omitted ") && strings.Contains(s, "Do not paste this placeholder") {
		return true
	}
	return strings.Contains(s, `"_elided"`)
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

// heavyCallRunes is the UI slimming threshold for write-like arguments (see
// live_event.go). It never applies to the model prompt.
const heavyCallRunes = 800

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

func compactCallArgs(name, args string, limit int) string {
	return compactCallArgsMeta(name, args, limit, "")
}

func compactCallArgsMeta(name, args string, limit int, recall string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return args
	}
	changed := false
	var fields []map[string]any
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
		// Drop the heavy string entirely. An English sentence in `content`
		// is still a valid write_file argument — models copy it as a
		// protocol for large files (session 77eb12a42ce43baa).
		delete(m, key)
		fields = append(fields, map[string]any{"field": key, "runes": len([]rune(s))})
		changed = true
	}
	if !changed {
		return args
	}
	m["_elided"] = fields
	m["_hint"] = "on disk; read_file/str_replace; do not paste _elided as content"
	if recall != "" {
		m["_recall"] = recall
	}
	b, err := json.Marshal(m)
	if err != nil {
		return args
	}
	return string(b)
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
