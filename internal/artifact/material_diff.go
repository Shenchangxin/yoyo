// 两份快照解开之后的内容差异。不改指针，也不做只比哈希的逐行对照。会话详情和血统图调用它。

package artifact

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	// OpAdded 表示条目出现在新的一侧，值为 added。
	OpAdded = "added"
	// OpRemoved 表示条目只留在旧的一侧，值为 removed。
	OpRemoved = "removed"
	// OpChanged 表示两侧都有但内容不同，值为 changed。
	OpChanged = "changed"
)

// MaterialChange 是两份快照之间一条给人看的内容差异。Surface 与 SnapshotDiffFields 的字段名对齐。
// L3 为真表示这条差异落在循环拓扑或策略上。零值不是拓扑变化。
type MaterialChange struct {
	Surface string `json:"surface"`
	Op      string `json:"op"`
	ID      string `json:"id,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Detail  string `json:"detail,omitempty"`
	L3      bool   `json:"l3,omitempty"`
}

// MaterialDiff 解开两边的对象，把哈希差异写成可读变更。哈希相同的面不会出现。
// 模型指纹和笔记直接比快照上的字符串。其余面解不开时，退回整对象替换。
func MaterialDiff(cas *Store, a, b HarnessSnapshot) []MaterialChange {
	var out []MaterialChange
	if a.ModelFingerprint != b.ModelFingerprint {
		out = append(out, MaterialChange{Surface: "model", Op: OpChanged, From: a.ModelFingerprint, To: b.ModelFingerprint, Detail: clip(a.ModelFingerprint, 48) + " → " + clip(b.ModelFingerprint, 48)})
	}
	if a.Note != b.Note {
		out = append(out, MaterialChange{Surface: "note", Op: noteOp(a.Note, b.Note), From: a.Note, To: b.Note, Detail: clip(b.Note, 160)})
	}
	out = append(out, diffPlaybook(cas, a.Playbook, b.Playbook)...)
	out = append(out, diffLoop(cas, a.LoopPreset, b.LoopPreset)...)
	out = append(out, diffPolicy(cas, a.PolicyPack, b.PolicyPack)...)
	out = append(out, diffEval(cas, a.EvalSuite, b.EvalSuite)...)
	out = append(out, diffPrompts(cas, a.PromptFragments, b.PromptFragments)...)
	out = append(out, diffSkills(cas, a.Skills, b.Skills)...)
	out = append(out, diffNamedList("tools", a.Tools, b.Tools)...)
	out = append(out, diffNamedList("eval_tools", a.EvalTools, b.EvalTools)...)
	out = append(out, diffWASM(cas, a.WASMPlugins, b.WASMPlugins)...)
	return out
}

// MaterialDiffText 把变更逐行拼成文本。没有变更时返回空串。L3 的行末尾带 [L3]。
func MaterialDiffText(changes []MaterialChange) string {
	if len(changes) == 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range changes {
		line := c.Surface + " " + c.Op
		if c.ID != "" {
			line += " " + c.ID
		}
		if c.Detail != "" {
			line += " " + c.Detail
		} else if c.From != "" || c.To != "" {
			line += " " + c.From + " -> " + c.To
		}
		if c.L3 {
			line += " [L3]"
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// noteOp 在旧笔记为空时记 added，新笔记为空时记 removed，否则记 changed。
func noteOp(from, to string) string {
	if from == "" {
		return OpAdded
	}
	if to == "" {
		return OpRemoved
	}
	return OpChanged
}

// diffPlaybook 按条目 ID 对照；ID 为空则用去空白并转小写的正文。两边都解不开，或解开后条目没有差异，就记一条不带 L3 的整本替换。
func diffPlaybook(cas *Store, ah, bh string) []MaterialChange {
	if ah == bh {
		return nil
	}
	left, lok := load[Playbook](cas, ah)
	right, rok := load[Playbook](cas, bh)
	if !lok || !rok {
		return []MaterialChange{{Surface: "playbook", Op: OpChanged, From: ah, To: bh, Detail: "playbook object replaced"}}
	}
	type bullet struct {
		id, text string
		helpful  int
		harmful  int
	}
	index := func(p Playbook) map[string]bullet {
		out := map[string]bullet{}
		for _, b := range p.Bullets {
			key := b.ID
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(b.Text))
			}
			out[key] = bullet{id: b.ID, text: b.Text, helpful: b.Helpful, harmful: b.Harmful}
		}
		return out
	}
	a := index(left)
	b := index(right)
	var keys []string
	seen := map[string]bool{}
	for k := range a {
		keys = append(keys, k)
		seen[k] = true
	}
	for k := range b {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []MaterialChange
	for _, k := range keys {
		av, aok := a[k]
		bv, bok := b[k]
		id := av.id
		if id == "" {
			id = bv.id
		}
		if aok && !bok {
			out = append(out, MaterialChange{Surface: "playbook", Op: OpRemoved, ID: id, From: av.text, Detail: clip(av.text, 160)})
			continue
		}
		if !aok && bok {
			out = append(out, MaterialChange{Surface: "playbook", Op: OpAdded, ID: id, To: bv.text, Detail: clip(bv.text, 160)})
			continue
		}
		if av.text != bv.text || av.helpful != bv.helpful || av.harmful != bv.harmful {
			detail := clip(bv.text, 120)
			if av.helpful != bv.helpful || av.harmful != bv.harmful {
				detail = fmt.Sprintf("%+d/−%d → %+d/−%d  %s", av.helpful, av.harmful, bv.helpful, bv.harmful, clip(bv.text, 100))
			}
			out = append(out, MaterialChange{Surface: "playbook", Op: OpChanged, ID: id, From: av.text, To: bv.text, Detail: detail})
		}
	}
	if len(out) == 0 {
		return []MaterialChange{{Surface: "playbook", Op: OpChanged, From: ah, To: bh, Detail: "playbook object replaced"}}
	}
	return out
}

// diffLoop 按字段列出变化。拓扑字段只在两份预设拓扑不同时标 L3，文案槽不标。有一边解不开时，记一条带 L3 的整份替换；字段都相同但仍是两个哈希时也记整份替换，这时 L3 仍看拓扑是否相同。
func diffLoop(cas *Store, ah, bh string) []MaterialChange {
	if ah == bh {
		return nil
	}
	left, lok := load[LoopPreset](cas, ah)
	right, rok := load[LoopPreset](cas, bh)
	if !lok || !rok {
		return []MaterialChange{{Surface: "loop", Op: OpChanged, From: ah, To: bh, Detail: "loop preset replaced", L3: true}}
	}
	l3 := !LoopTopologyEqual(left, right)
	type field struct {
		name, from, to string
		topo           bool
	}
	fields := []field{
		{"max_turns", itoa(left.MaxTurns), itoa(right.MaxTurns), true},
		{"max_tool_messages", itoa(left.MaxToolMessages), itoa(right.MaxToolMessages), true},
		{"compaction_keep", itoa(left.CompactionKeep), itoa(right.CompactionKeep), true},
		{"compaction_tokens", itoa(left.CompactionTokens), itoa(right.CompactionTokens), true},
		{"tool_result_budget", itoa(left.ToolResultBudget), itoa(right.ToolResultBudget), true},
		{"micro_keep", itoa(left.MicroKeep), itoa(right.MicroKeep), true},
		{"playbook_tokens", itoa(left.PlaybookTokens), itoa(right.PlaybookTokens), true},
		{"rules_tokens", itoa(left.RulesTokens), itoa(right.RulesTokens), true},
		{"allow_llm_compact", boolish(left.AllowLLMCompact), boolish(right.AllowLLMCompact), true},
		{"max_budget_usd", fmt.Sprintf("%g", left.MaxBudgetUSD), fmt.Sprintf("%g", right.MaxBudgetUSD), true},
		{"plan_mode", boolish(left.PlanMode), boolish(right.PlanMode), true},
		{"max_parallel", itoa(left.MaxParallel), itoa(right.MaxParallel), true},
		{"keep_tokens", itoa(left.KeepTokens), itoa(right.KeepTokens), true},
		{"context_buffer", itoa(left.ContextBuffer), itoa(right.ContextBuffer), true},
		{"output_reserve", itoa(left.OutputReserve), itoa(right.OutputReserve), true},
		{"task_max_turns", itoa(left.TaskMaxTurns), itoa(right.TaskMaxTurns), true},
		{"task_max_tool_messages", itoa(left.TaskMaxToolMessages), itoa(right.TaskMaxToolMessages), true},
		{"stack_agents_md", boolish(left.StackAgentsMD), boolish(right.StackAgentsMD), true},
		{"bootstrap", left.Bootstrap, right.Bootstrap, false},
		{"execution", left.Execution, right.Execution, false},
		{"verification", left.Verification, right.Verification, false},
		{"failure_recovery", left.FailureRecovery, right.FailureRecovery, false},
		{"max_recent_tool_errors", itoa(left.MaxRecentToolErrors), itoa(right.MaxRecentToolErrors), false},
		{"tool_error_instruction", left.ToolErrorInstruction, right.ToolErrorInstruction, false},
		{"task_instruction", left.TaskInstruction, right.TaskInstruction, false},
	}
	var out []MaterialChange
	for _, f := range fields {
		if f.from == f.to {
			continue
		}
		out = append(out, MaterialChange{
			Surface: "loop",
			Op:      OpChanged,
			ID:      f.name,
			From:    f.from,
			To:      f.to,
			Detail:  f.name + " " + clip(f.from, 80) + " → " + clip(f.to, 80),
			L3:      f.topo && l3,
		})
	}
	if len(out) == 0 {
		return []MaterialChange{{Surface: "loop", Op: OpChanged, From: ah, To: bh, Detail: "loop preset replaced", L3: l3}}
	}
	return out
}

// diffPolicy 比较 mode，以及 default_allow、require_approval、network_allow。Notes 不单独比较。解不开，或这些字段没有差异时，仍记一条带 L3 的整包替换。
func diffPolicy(cas *Store, ah, bh string) []MaterialChange {
	if ah == bh {
		return nil
	}
	left, lok := load[PolicyPack](cas, ah)
	right, rok := load[PolicyPack](cas, bh)
	if !lok || !rok {
		return []MaterialChange{{Surface: "policy", Op: OpChanged, From: ah, To: bh, Detail: "policy pack replaced", L3: true}}
	}
	var out []MaterialChange
	if left.Mode != right.Mode {
		out = append(out, MaterialChange{Surface: "policy", Op: OpChanged, ID: "mode", From: left.Mode, To: right.Mode, Detail: "mode " + left.Mode + " → " + right.Mode, L3: true})
	}
	out = append(out, listChanges("policy", "default_allow", left.DefaultAllow, right.DefaultAllow, true)...)
	out = append(out, listChanges("policy", "require_approval", left.RequireApproval, right.RequireApproval, true)...)
	out = append(out, listChanges("policy", "network_allow", left.NetworkAllow, right.NetworkAllow, true)...)
	if len(out) == 0 {
		return []MaterialChange{{Surface: "policy", Op: OpChanged, From: ah, To: bh, Detail: "policy pack replaced", L3: true}}
	}
	return out
}

// diffEval 比较编号、任务目录、重复次数、超时和是否密封，以及 held_in、held_out、safety、transfer。EvolveIn 不参与比较。解不开或没有字段差异时记整套替换，不标 L3。
func diffEval(cas *Store, ah, bh string) []MaterialChange {
	if ah == bh {
		return nil
	}
	left, lok := load[EvalSuite](cas, ah)
	right, rok := load[EvalSuite](cas, bh)
	if !lok || !rok {
		return []MaterialChange{{Surface: "eval", Op: OpChanged, From: ah, To: bh, Detail: "eval suite replaced"}}
	}
	var out []MaterialChange
	if left.ID != right.ID {
		out = append(out, MaterialChange{Surface: "eval", Op: OpChanged, ID: "id", From: left.ID, To: right.ID, Detail: "id " + left.ID + " → " + right.ID})
	}
	if left.TaskDir != right.TaskDir {
		out = append(out, MaterialChange{Surface: "eval", Op: OpChanged, ID: "task_dir", From: left.TaskDir, To: right.TaskDir, Detail: "task_dir " + clip(left.TaskDir, 80) + " → " + clip(right.TaskDir, 80)})
	}
	if left.Repeats != right.Repeats {
		out = append(out, MaterialChange{Surface: "eval", Op: OpChanged, ID: "repeats", From: itoa(left.Repeats), To: itoa(right.Repeats), Detail: "repeats " + itoa(left.Repeats) + " → " + itoa(right.Repeats)})
	}
	if left.TimeoutSec != right.TimeoutSec {
		out = append(out, MaterialChange{Surface: "eval", Op: OpChanged, ID: "timeout_sec", From: itoa(left.TimeoutSec), To: itoa(right.TimeoutSec), Detail: "timeout_sec " + itoa(left.TimeoutSec) + " → " + itoa(right.TimeoutSec)})
	}
	if left.Sealed != right.Sealed {
		out = append(out, MaterialChange{Surface: "eval", Op: OpChanged, ID: "sealed", From: boolish(left.Sealed), To: boolish(right.Sealed), Detail: "sealed " + boolish(left.Sealed) + " → " + boolish(right.Sealed)})
	}
	out = append(out, listChanges("eval", "held_in", left.HeldIn, right.HeldIn, false)...)
	out = append(out, listChanges("eval", "held_out", left.HeldOut, right.HeldOut, false)...)
	out = append(out, listChanges("eval", "safety", left.Safety, right.Safety, false)...)
	out = append(out, listChanges("eval", "transfer", left.Transfer, right.Transfer, false)...)
	if len(out) == 0 {
		return []MaterialChange{{Surface: "eval", Op: OpChanged, From: ah, To: bh, Detail: "eval suite replaced"}}
	}
	return out
}

// diffPrompts 用 ID，否则用槽位，再否则用哈希当键。同一键只留该侧最后一条。键对上但哈希不同算 changed。顺序变化本身不留行；对出空结果时退回哈希增减。
func diffPrompts(cas *Store, a, b []string) []MaterialChange {
	if join(a) == join(b) {
		return nil
	}
	type frag struct {
		hash, id, slot, text string
	}
	loadAll := func(hs []string) []frag {
		var out []frag
		for _, h := range hs {
			f := frag{hash: h}
			if v, ok := load[PromptFragment](cas, h); ok {
				f.id, f.slot, f.text = v.ID, v.Slot, v.Text
			}
			out = append(out, f)
		}
		return out
	}
	left := loadAll(a)
	right := loadAll(b)
	byKey := func(xs []frag) map[string]frag {
		out := map[string]frag{}
		for _, x := range xs {
			k := x.id
			if k == "" {
				k = x.slot
			}
			if k == "" {
				k = x.hash
			}
			out[k] = x
		}
		return out
	}
	lm, rm := byKey(left), byKey(right)
	var keys []string
	seen := map[string]bool{}
	for k := range lm {
		keys = append(keys, k)
		seen[k] = true
	}
	for k := range rm {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []MaterialChange
	for _, k := range keys {
		av, aok := lm[k]
		bv, bok := rm[k]
		id := av.id
		if id == "" {
			id = bv.id
		}
		if id == "" {
			id = k
		}
		if aok && !bok {
			out = append(out, MaterialChange{Surface: "prompts", Op: OpRemoved, ID: id, From: av.text, Detail: clip(promptLabel(av.slot, av.text), 160)})
			continue
		}
		if !aok && bok {
			out = append(out, MaterialChange{Surface: "prompts", Op: OpAdded, ID: id, To: bv.text, Detail: clip(promptLabel(bv.slot, bv.text), 160)})
			continue
		}
		if av.hash != bv.hash {
			out = append(out, MaterialChange{Surface: "prompts", Op: OpChanged, ID: id, From: av.text, To: bv.text, Detail: clip(promptLabel(bv.slot, bv.text), 160)})
		}
	}
	if len(out) == 0 {
		return hashListFallback("prompts", a, b)
	}
	return out
}

// diffSkills 用技能名当键，同名只留最后一条。同名哈希不同算 changed。顺序变化本身不留行；对出空结果时退回哈希增减。
func diffSkills(cas *Store, a, b []string) []MaterialChange {
	if join(a) == join(b) {
		return nil
	}
	type sk struct {
		hash, name, desc string
	}
	loadAll := func(hs []string) map[string]sk {
		out := map[string]sk{}
		for _, h := range hs {
			item := sk{hash: h, name: h}
			if v, ok := load[Skill](cas, h); ok {
				item.name, item.desc = v.Name, v.Description
			}
			key := item.name
			if key == "" {
				key = h
			}
			out[key] = item
		}
		return out
	}
	lm, rm := loadAll(a), loadAll(b)
	var keys []string
	seen := map[string]bool{}
	for k := range lm {
		keys = append(keys, k)
		seen[k] = true
	}
	for k := range rm {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []MaterialChange
	for _, k := range keys {
		av, aok := lm[k]
		bv, bok := rm[k]
		if aok && !bok {
			out = append(out, MaterialChange{Surface: "skills", Op: OpRemoved, ID: av.name, From: av.desc, Detail: clip(av.name+": "+av.desc, 160)})
			continue
		}
		if !aok && bok {
			out = append(out, MaterialChange{Surface: "skills", Op: OpAdded, ID: bv.name, To: bv.desc, Detail: clip(bv.name+": "+bv.desc, 160)})
			continue
		}
		if av.hash != bv.hash {
			out = append(out, MaterialChange{Surface: "skills", Op: OpChanged, ID: bv.name, From: av.desc, To: bv.desc, Detail: clip(bv.name+": "+bv.desc, 160)})
		}
	}
	if len(out) == 0 {
		return hashListFallback("skills", a, b)
	}
	return out
}

// diffWASM 把哈希解成插件名再做增减。解不开时用哈希前 12 位。同名换了模块不会记成 changed。
func diffWASM(cas *Store, a, b []string) []MaterialChange {
	if join(a) == join(b) {
		return nil
	}
	names := func(hs []string) []string {
		var out []string
		for _, h := range hs {
			if v, ok := load[WASMPlugin](cas, h); ok && v.Name != "" {
				out = append(out, v.Name)
				continue
			}
			out = append(out, short(h, 12))
		}
		return out
	}
	return diffNamedList("wasm", names(a), names(b))
}

// diffNamedList 按字符串集合做增减，不标 L3。
func diffNamedList(surface string, a, b []string) []MaterialChange {
	return listChanges(surface, "", a, b, false)
}

// listChanges 忽略空串和顺序，只报告新增和删除。l3 原样抄到每一行。
func listChanges(surface, id string, a, b []string, l3 bool) []MaterialChange {
	added, removed := sliceDelta(a, b)
	var out []MaterialChange
	for _, v := range added {
		out = append(out, MaterialChange{Surface: surface, Op: OpAdded, ID: id, To: v, Detail: clip(label(id, v), 160), L3: l3})
	}
	for _, v := range removed {
		out = append(out, MaterialChange{Surface: surface, Op: OpRemoved, ID: id, From: v, Detail: clip(label(id, v), 160), L3: l3})
	}
	return out
}

// hashListFallback 在条目对不上时按哈希报增减，明细只留前 12 位。
func hashListFallback(surface string, a, b []string) []MaterialChange {
	added, removed := sliceDelta(a, b)
	var out []MaterialChange
	for _, h := range added {
		out = append(out, MaterialChange{Surface: surface, Op: OpAdded, To: h, Detail: short(h, 12)})
	}
	for _, h := range removed {
		out = append(out, MaterialChange{Surface: surface, Op: OpRemoved, From: h, Detail: short(h, 12)})
	}
	return out
}

// sliceDelta 用集合比较，忽略空串，结果按字典序。
func sliceDelta(a, b []string) (added, removed []string) {
	am, bm := map[string]bool{}, map[string]bool{}
	for _, v := range a {
		if v != "" {
			am[v] = true
		}
	}
	for _, v := range b {
		if v != "" {
			bm[v] = true
		}
	}
	for v := range bm {
		if !am[v] {
			added = append(added, v)
		}
	}
	for v := range am {
		if !bm[v] {
			removed = append(removed, v)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// load 在库为空、哈希为空白或解码失败时返回 false。
func load[T any](cas *Store, hash string) (T, bool) {
	var zero T
	if cas == nil || strings.TrimSpace(hash) == "" {
		return zero, false
	}
	v, _, err := Decode[T](cas, hash)
	if err != nil {
		return zero, false
	}
	return v, true
}

// join 用逗号拼接。
func join(xs []string) string { return strings.Join(xs, ",") }

// itoa 转成十进制文本。
func itoa(n int) string { return strconv.Itoa(n) }

// boolish 写成 true 或 false。
func boolish(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// promptLabel 有槽位时写成「槽位: 正文」。
func promptLabel(slot, text string) string {
	if slot == "" {
		return text
	}
	if text == "" {
		return slot
	}
	return slot + ": " + text
}

// label 有字段名时写成「字段名 值」。
func label(id, v string) string {
	if id == "" {
		return v
	}
	return id + " " + v
}

// short 按字节截到 n 位。n 小于等于 0 或原串更短时原样返回。
func short(hash string, n int) string {
	hash = strings.TrimSpace(hash)
	if n <= 0 || len(hash) <= n {
		return hash
	}
	return hash[:n]
}

// clip 先把空白收成单空格，再按 rune 截断并加省略号。n 小于等于 0 时不截。
func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return s
	}
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
