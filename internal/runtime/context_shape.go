// 发给模型的投影。在副本上截断工具结果、微压缩、强制适配和剪旧回合。桌面聊天走 projectChat，不做 snip 和 forceFit。
// 不改调用方手里的消息，因此不改 JSONL。不调用模型做摘要。循环、计量和检查点会调用。

package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// ShapeReport 是一轮上下文投影的账本，给计量和 compact 事件用。
type ShapeReport struct {
	Note            string `json:"note"`
	Tokens          int    `json:"tokens"`
	Budget          int    `json:"budget"`
	Window          int    `json:"window,omitempty"`
	PrefixTokens    int    `json:"prefix_tokens,omitempty"`
	DynamicTokens   int    `json:"dynamic_tokens,omitempty"`
	SchemaTokens    int    `json:"schema_tokens,omitempty"`
	IdentityTokens  int    `json:"identity_tokens,omitempty"`
	RulesTokens     int    `json:"rules_tokens,omitempty"`
	CatalogTokens   int    `json:"catalog_tokens,omitempty"`
	SkillBodyTokens int    `json:"skill_body_tokens,omitempty"`
	MemoryTokens    int    `json:"memory_tokens,omitempty"`
	TodayTokens     int    `json:"today_tokens,omitempty"`
	NotesTokens     int    `json:"notes_tokens,omitempty"`
	// ChatTokens 是用户、助手和工具的真实历史。新会话可以是 0，所以序列化时保留 0。
	ChatTokens     int  `json:"chat_tokens"`
	ProviderPrompt int  `json:"provider_prompt,omitempty"`
	CachedTokens   int  `json:"cached_tokens,omitempty"`
	CacheReported  bool `json:"cache_reported,omitempty"`
	// CacheStable 表示热前缀仍以前一轮热前缀为开头。MeasureContext 固定写成 true。
	CacheStable bool `json:"cache_stable,omitempty"`
	// PrefixHash 是热前缀的哈希，不含动态块。
	PrefixHash string `json:"prefix_hash,omitempty"`
	// DynamicAt 为空时由 fillLedger 写成 tail，表示动态块在历史后面。
	DynamicAt string   `json:"dynamic_at,omitempty"`
	Trigger   string   `json:"trigger,omitempty"`
	Hydrated  int      `json:"hydrated,omitempty"`
	Layers    []string `json:"layers,omitempty"`
	Elided    int      `json:"elided,omitempty"`
}

// ShapeOpts 控制投影预算。ModelWindow 为 0 时用 CompactionTokens。Aggressive 收紧保留条数和单条工具结果上限。Overhead 是工具 schema 的 token，算进预算比较。
type ShapeOpts struct {
	Loop          artifact.LoopPreset
	Spill         *Spill
	ModelWindow   int
	OutputReserve int
	Overhead      int
	Aggressive    bool
}

// Compact 是评测用的稳定入口，只做确定性整形，不调用模型。返回投影和 Shape 的 Note。
func Compact(msgs []Message, loop artifact.LoopPreset) ([]Message, string) {
	out, rep := Shape(msgs, ShapeOpts{Loop: loop})
	return out, rep.Note
}

// projectChat 是桌面聊天的投影。在副本上按条截断工具结果，压力够了再把旧的可重读结果换成回执，最后 Legalize。
// 不做 snip，也不 forceFit，写操作的参数保持原样。不改调用方的消息，因此不改 JSONL。
func projectChat(msgs []Message, opts ShapeOpts) ([]Message, ShapeReport) {
	rep := ShapeReport{Budget: effectiveBudget(opts)}
	if len(msgs) == 0 {
		return msgs, rep
	}
	out := copyMessages(msgs)
	per := opts.Loop.ToolResultBudget
	if per <= 0 {
		// 单条工具结果没给预算时用 8000。这是投影截断，不是聊天规则前缀的 1600。
		per = 8_000
	}
	if n := applyBudget(out, per, opts.Spill); n > 0 {
		rep.Layers = append(rep.Layers, "budget")
		rep.Elided += n
	}
	if readsUnderPressure(out, opts) {
		if n := clearStaleReads(out, chatKeepReads, chatReadClearBatch); n > 0 {
			rep.Layers = append(rep.Layers, "reads")
			rep.Elided += n
		}
	}
	rep.Note = strings.Join(rep.Layers, "+")
	// 只整理这份副本。调用方手里的消息和 JSONL 保持原样。
	return Legalize(out), rep
}

// chatKeepReads 是桌面投影里保留原文的最新只读结果条数。更旧的在有压力时换成回执。
const chatKeepReads = 12

// chatReadClearAt 是开始把旧读换成回执的预算比例。低于它时每条读都保持原文。chatCompactAt 更靠后，压力先到这里时用回执换出空间。
const chatReadClearAt = 0.5

// chatReadClearBatch 是一次清除的条数。只清除 batch 的整数倍，两次清除之间已处理前缀的字节保持不变。
const chatReadClearBatch = 8

// readsUnderPressure 在预算小于等于 0 时返回真。否则历史 token 加 Overhead 达到有效预算的 chatReadClearAt 才返回真。
func readsUnderPressure(msgs []Message, opts ShapeOpts) bool {
	budget := effectiveBudget(opts)
	if budget <= 0 {
		return true
	}
	est := messagesTokens(msgs) + opts.Overhead
	return float64(est) >= float64(budget)*chatReadClearAt
}

// refetchableTool 判断结果能否只靠工作区再生成。写、shell 和浏览器状态不在此列。
func refetchableTool(name string) bool {
	switch name {
	case "read_file", "grep", "glob", "list_dir", "git_status", "git_diff":
		return true
	default:
		return false
	}
}

// clearStaleReads 把较旧的可重读结果换成回执。发出该结果的助手调用留在投影里。回执带路径、read_file 的行号范围、字节数和所示内容的哈希。
// 只清除过期条数里 batch 的整数倍，因此两次清除之间已清除前缀的字节保持不变。短于 400 字节或已经是占位的跳过。
func clearStaleReads(msgs []Message, keep, batch int) int {
	var ids []int
	for i, m := range msgs {
		if m.Role == RoleTool && refetchableTool(m.Name) {
			ids = append(ids, i)
		}
	}
	stale := len(ids) - keep
	if stale <= 0 {
		return 0
	}
	if batch > 1 {
		stale -= stale % batch
	}
	n := 0
	for _, i := range ids[:stale] {
		if alreadyStubbed(msgs[i].Content) || len(msgs[i].Content) < 400 {
			continue
		}
		msgs[i].Content = readReceipt(msgs, i)
		n++
	}
	return n
}

// readReceipt 是被清掉的只读结果的替换正文。用明文和真实标识，避免被当成工具参数。
func readReceipt(msgs []Message, toolIdx int) string {
	m := msgs[toolIdx]
	path := toolResultPath(msgs, toolIdx)
	if path == "" {
		path = "(unknown path)"
	}
	span := ""
	if m.Name == "read_file" {
		if s := numberedSpan(m.Content); s != "" {
			span = " " + s
		}
	}
	sum := sha256.Sum256([]byte(m.Content))
	return fmt.Sprintf("[cleared %s path=%s%s bytes=%d sha256=%s — this earlier result left the prompt; call read_file again to see the current content]",
		m.Name, path, span, len(m.Content), hex.EncodeToString(sum[:6]))
}

// numberedSpan 读取 read_file 行首的行号，返回第一条和最后一条，例如 lines=1-264。没有行号时返回空串。
func numberedSpan(content string) string {
	first, last := 0, 0
	for _, line := range strings.Split(content, "\n") {
		bar := strings.IndexByte(line, '|')
		if bar <= 0 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[:bar]))
		if err != nil || n <= 0 {
			continue
		}
		if first == 0 {
			first = n
		}
		last = n
	}
	if first == 0 {
		return ""
	}
	return fmt.Sprintf("lines=%d-%d", first, last)
}

// Shape 在副本上做只读投影，不改调用方的消息。顺序是按条截断、微压缩，仍超预算再把工具结果强制换成占位，还超就把旧回合剪成 spill 指针。
// 用户和助手的正文不改写，工具参数也不换成占位。不调用模型做摘要。
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
		// 溢出重试把保留的压缩条数收到 8。平时的 CompactionKeep 不走这里。
		keep = 8
	}
	budget := effectiveBudget(opts)
	per := loop.ToolResultBudget
	if per <= 0 {
		per = 8_000
	}
	if opts.Aggressive && per > 1_500 {
		// 同一次重试把单条工具结果收到 1500。写操作参数仍不换成占位。
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
		// 这里不缩短工具参数。参数里的占位会被模型照抄。超预算只给工具结果换上带 recall id 的占位。
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

// copyMessages 浅拷贝消息，并单独复制 ToolCalls，避免投影改到调用方。
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

// applyBudget 把超过 per 个 rune 的工具结果先写入 spill，正文换成占位加头尾预览。已经是占位的跳过。
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

// snipWindow 钉住开头的系统消息，从配对边界切掉中段。能留下的用户消息超过 8 条时只留头 2 条和尾 6 条。其余进 spill，用一条用户消息指向 id。切点不够靠后时返回 nil。
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

// keepThroughSnip 保留非空、不是占位、也不是控制行的用户消息。
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

// pinnedPrefix 返回前导 system 和 developer 的条数。一条都没有时返回 1，避免把第一条历史切掉。
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

// microcompact 把最后 keepLast 条工具结果之前的长结果换成占位。短于 400 字节或已经是占位的跳过。
// keepLast 大于等于 16 时，每个路径最新的一条保留，最多 12 个路径。
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

// lastPathToolIndexes 从新到旧为每个路径保留最新的工具结果，路径数达到 capPaths 就停。capPaths 小于等于 0 时返回空表。
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

// toolResultPath 往前找发出这次结果的助手调用，返回参数里的第一个路径。找不到返回空串。
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

// forceFit 从钉住的前缀之后把工具结果换成占位，直到 token 不超过预算。不改工具参数。
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

// toolIndexes 返回所有 role=tool 的下标。
func toolIndexes(msgs []Message) []int {
	var ids []int
	for i, m := range msgs {
		if m.Role == RoleTool {
			ids = append(ids, i)
		}
	}
	return ids
}

// alreadyStubbed 识别以 elided、collapsed、omitted、cleared 开头，或含有 _elided 的正文。
func alreadyStubbed(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[elided ") || strings.HasPrefix(s, "[collapsed ") || strings.HasPrefix(s, "(omitted ") || strings.HasPrefix(s, "[cleared ") {
		return true
	}
	return strings.Contains(s, `"_elided"`)
}

// looksLikeContextStub 在正文任意位置识别占位。omitted 还要带 Do not paste this placeholder。写文件工具用它拒绝把占位写回磁盘。
func looksLikeContextStub(s string) bool {
	if strings.Contains(s, "[elided ") || strings.Contains(s, "[collapsed ") {
		return true
	}
	if strings.Contains(s, "(omitted ") && strings.Contains(s, "Do not paste this placeholder") {
		return true
	}
	return strings.Contains(s, `"_elided"`)
}

// stubTool 返回带 recall id 的占位。id 或名字为空时用 unknown 和 tool。
func stubTool(id, name string, bytes int) string {
	if id == "" {
		id = "unknown"
	}
	if name == "" {
		name = "tool"
	}
	return fmt.Sprintf("[elided tool_result id=%s name=%s bytes=%d — call recall_context with this id]", id, name, bytes)
}

// heavyCallRunes 是界面缩短写操作参数的阈值。模型投影不用它。压缩摘要在送进摘要模型之前用同一阈值。
const heavyCallRunes = 800

// appendLayer 在同名层尚未出现时追加。
func appendLayer(layers []string, name string) []string {
	for _, l := range layers {
		if l == name {
			return layers
		}
	}
	return append(layers, name)
}

// heavyCallName 判断这些写操作的参数算不算重。Shape 不拿它改发给模型的参数。
func heavyCallName(name string) bool {
	switch name {
	case "write_file", "create_file", "apply_patch", "str_replace", "edit_file",
		"office_create", "office_edit", "cite_sources":
		return true
	default:
		return false
	}
}

// compactCallArgs 按 limit 删掉超长的写操作字段，不附带 recall id。
func compactCallArgs(name, args string, limit int) string {
	return compactCallArgsMeta(name, args, limit, "")
}

// compactCallArgsMeta 删掉超限的 content、patch、body。str_replace、edit_file、office_edit 再加 old_str 和 new_str。解析失败则原样返回。
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
		// 整段删掉。content 里留一句说明仍是合法的 write_file 参数，模型会把它当成大文件的写法照抄。
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

// writePathsFromMessages 从助手的 write_file、str_replace 和 apply_patch 收集路径，最多 16 个。
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
