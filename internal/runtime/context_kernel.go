// 一轮请求的前缀、动态尾和检查点。Run 取 prompt 的投影去调用模型，轨迹仍由 emit 追加。
// 已准入的系统头留在原处，前缀变更追加在历史后面。动态块接在历史末尾，不写进系统头。

package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/zeebo/blake3"
)

const (
	// skillBodyCapRunes 是单份非引导技能正文的 rune 上限。
	skillBodyCapRunes = 5_000
	// skillBootstrapCapRunes 是带引导标记的技能正文 rune 上限。
	skillBootstrapCapRunes = 12_000
	// skillTotalCapRunes 是全部技能正文加在一起的 rune 上限。放不下的整份跳过。
	skillTotalCapRunes = 25_000
)

// ContextKernel 持有这一轮的身份、pins、动态尾和检查点。Run 消费 prompt 和 checkpoint 的结果，不在这里改写已经落盘的轨迹。
type ContextKernel struct {
	req      *RunRequest
	identity string
	pins     string
	prefix   string
	// seriesPrefix 是真正放在请求头部的系统文本，等于已准入的 identity 加 pins。前缀变更追加在历史后面，这段头部字节保持不动。检查点之后改回当前 prefix。
	seriesPrefix string
	bill         contextBill
	spill        *Spill
	notes        string
	lastHash     string
	// prevHot 是上一轮热前缀的字节。本轮仍以它为开头时 CacheStable 为真。
	prevHot string
	dyn     string
}

// newContextKernel 按当前材料建账本。笔记从 spill 读出并截到 2000 个 rune。同时写下规则索引。seriesPrefix 留到 seed 再填。
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

// promptUpdateLead 是追加在历史后面的前缀更新开头。旧的系统消息留在头部，模型以这一版为准。isControlUser 认得这个前缀。
const promptUpdateLead = "Prompt update. The earlier system message stays only so the cached prefix can be reused. Follow this version.\n提示更新。前面的系统消息留在原处是为了保住缓存前缀。以这一版为准。\n"

// cachePrefix 返回缓存键用的头部。seriesPrefix 非空时用它，否则用当前 prefix。追加在历史后的更新不算进这段头部。
func (k *ContextKernel) cachePrefix() string {
	if k == nil {
		return ""
	}
	if k.seriesPrefix != "" {
		return k.seriesPrefix
	}
	return k.prefix
}

// seed 组装这一轮的消息。用户原文在返回前写入轨迹。
// 最后一次检查点之后若有准入的 prompt（还没有检查点就看整段轨迹），头部用那次事件里的 identity 和 pins，seriesPrefix 等于这两段相接，头部字节保持不动。
// 当前 prefix 和该事件的 text 不同，或 prefix 已经不是这段头部且历史里还没有对应更新时，把 promptUpdateLead 加当前 prefix，作为用户消息追加在历史后面。
// 同一轮要写入的 prompt 事件仍带头部的 identity 和 pins，text 才是当前 prefix。历史里已经有同样的更新就不再追加。
// 用户文本经 NormalizeUserText。没有用户、没有注入、去掉系统消息后历史也是空的，返回错误。
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
		// 热前缀钉在已准入事件的 identity 和 pins 上，头部字节不动。当前 prefix 和那次 text 不同时，把新前缀当成用户消息追加，不改写系统消息。
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
	// UserCommitted 为真表示 sendLocked 已经把用户行写入轨迹，这里不再发第二条。
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
		// 没有新用户文本、没有注入、去掉系统消息后历史也是空的，这一轮无从继续。
		return nil, fmt.Errorf("nothing to continue")
	}
	return messages, nil
}

// reseed 用当前 identity 和 pins 接上去掉系统消息的历史。检查点之后调用，头部换成当前 prefix，不再沿用检查点前的 seriesPrefix。
func (k *ContextKernel) reseed(hist []Message) []Message {
	messages := []Message{{Role: RoleSystem, Content: k.identity}}
	if strings.TrimSpace(k.pins) != "" {
		messages = append(messages, Message{Role: RoleDeveloper, Content: k.pins})
	}
	return append(messages, stripSystem(hist)...)
}

// promptHead 是某次准入 prompt 事件里放在头部的 identity 和 pins。
type promptHead struct {
	identity string
	pins     string
}

// seriesPrompt 取最后一次检查点之后、最后一条 kind 为 prompt 的系统事件。没有则返回 false。
// text 是当时的前缀全文。identity 和 pins 是当时放在头部的那一份，可以和 text 不同。
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

// transcriptHasPrompt 报告历史里是否已有内容恰好等于 promptUpdateLead 加 prefix 的用户消息。
func transcriptHasPrompt(msgs []Message, prefix string) bool {
	want := promptUpdateLead + prefix
	for _, m := range msgs {
		if m.Role == RoleUser && m.Content == want {
			return true
		}
	}
	return false
}

// chatMode 在请求打开 SoftHorizon 时为真。
func (k *ContextKernel) chatMode() bool {
	return k.req != nil && k.req.SoftHorizon
}

// refreshDynamic 重算技能正文、今日和笔记。操作者语种需要提醒时，把 languageDynPin 加在动态块前面。
// setDynamic 去掉旧的动态块，再把新块接到历史末尾。
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

// prompt 生成发给模型的投影和账本。先把动态块接到历史末尾。
// SoftHorizon 且非 aggressive 时用 projectChat：截工具结果，压力够了再把旧的只读结果换成回执，然后 Legalize。不做 snip 和 forceFit，写操作参数原样保留。
// 其他情况 Shape 之后再 Legalize。Legalize 只改这份投影，不写 JSONL。
// 热前缀跳过动态块。CacheStable 表示本轮热前缀仍以前一轮热前缀的字节为开头。
func (k *ContextKernel) prompt(messages []Message, aggressive bool) (out []Message, tools []ToolJSON, report ShapeReport) {
	messages = k.refreshDynamic(messages)
	tools = AllToolJSON(k.req.Tools)
	overhead := toolsJSONTokens(tools)
	opts := ShapeOpts{
		Loop: k.req.Loop, Spill: k.spill, ModelWindow: k.req.ModelWindow, Overhead: overhead, Aggressive: aggressive,
	}
	if k.chatMode() && !aggressive {
		// SoftHorizon 且这次不是 aggressive：走 projectChat，不做 snip 和 forceFit，写操作参数原样保留。超长回合靠检查点，不靠改写头部。
		out, report = projectChat(messages, opts)
	} else {
		// 硬循环，或溢出重试把 aggressive 设为真。Shape 之后再 Legalize。projectChat 在返回前已经 Legalize。两处都只改投影，不写 JSONL。
		out, report = Shape(messages, opts)
		out = Legalize(out)
	}
	fillLedger(&report, k.prefix, k.dyn, tools, k.req.ModelWindow)
	k.bill.write(&report)
	report.ChatTokens = historyTokens(out)
	report.Tokens = messagesTokens(out) + overhead
	hot := hotPrefixBytes(out)
	report.PrefixHash = hotPrefixHash(out)
	// 本轮热前缀仍以前一轮的字节开头才算缓存稳定。第一轮没有前一轮，也算稳定。检查点会把 prevHot 清掉，下一轮重新算。
	report.CacheStable = k.prevHot == "" || strings.HasPrefix(hot, k.prevHot)
	k.prevHot = hot
	k.lastHash = report.PrefixHash
	return out, tools, report
}

// usedTokens 在提供商报了提示 token 时用 ProviderPrompt，否则用本地估算的 Tokens。
func (k *ContextKernel) usedTokens(report ShapeReport) int {
	if report.ProviderPrompt > 0 {
		return report.ProviderPrompt
	}
	return report.Tokens
}

// chatCompactAt 是桌面检查点用的预算比例。有模型窗口时 shouldCheckpoint 乘上它，给摘要和后面几轮留余量。
const chatCompactAt = 0.65

// shouldCheckpoint 只在 SoftHorizon 且预算大于 0 时为真。有模型窗口时阈值是预算乘 chatCompactAt，否则用整段预算。
func (k *ContextKernel) shouldCheckpoint(report ShapeReport) bool {
	if !k.chatMode() || report.Budget <= 0 {
		return false
	}
	limit := report.Budget
	// 有模型窗口时，用到预算的 0.65 就检查点，给摘要和后面几轮留余量。窗口是 0 时用整段预算。
	if k.req != nil && k.req.ModelWindow > 0 {
		limit = int(float64(report.Budget) * chatCompactAt)
	}
	return k.usedTokens(report) >= limit
}

// checkpoint 做一次检查点，然后用当前身份重新播种。笔记截到 2000 个 rune，lastHash 和 prevHot 清空，seriesPrefix 改回当前 prefix。工具解锁在这里提交。
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

// capLoadedSkills 截断已加载技能。带引导标记的单份上限是 skillBootstrapCapRunes，其余是 skillBodyCapRunes，总和不超过 skillTotalCapRunes。
// 引导正文按原顺序先占额度。其余从最新往旧收，输出时较新的仍在后面。放不下的整份跳过。
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

// isBootstrapBody 在正文含引导标记标签时返回真。
func isBootstrapBody(s string) bool {
	return strings.Contains(s, "<"+bootstrapMarker+">")
}

// hotPrefixHash 对热前缀做 blake3，取前 16 字节的十六进制。动态块不计入。
func hotPrefixHash(msgs []Message) string {
	h := blake3.New()
	_, _ = h.Write([]byte(hotPrefixBytes(msgs)))
	return hex16(h.Sum(nil))
}

// hotPrefixBytes 拼热前缀。跳过 RoleMemory 和以 dynMarker 开头的内容，因此动态块不在这段字节里。
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

// reportPayload 组装 kind 为 shape 的事件载荷。ProviderPrompt 大于 0 才写入。extra 里的键覆盖同名字段。
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
