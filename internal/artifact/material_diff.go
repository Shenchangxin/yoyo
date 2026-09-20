package artifact

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	OpAdded   = "added"
	OpRemoved = "removed"
	OpChanged = "changed"
)

// MaterialChange is a decoded, operator-facing delta between two snapshots.
// Surface names match SnapshotDiffFields so the UI can group hash diffs and
// content diffs together.
type MaterialChange struct {
	Surface string `json:"surface"`
	Op      string `json:"op"`
	ID      string `json:"id,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Detail  string `json:"detail,omitempty"`
	L3      bool   `json:"l3,omitempty"`
}

// MaterialDiff walks CAS payloads so a hash-only snapshot diff is not a black box.
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

func noteOp(from, to string) string {
	if from == "" {
		return OpAdded
	}
	if to == "" {
		return OpRemoved
	}
	return OpChanged
}

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

func diffNamedList(surface string, a, b []string) []MaterialChange {
	return listChanges(surface, "", a, b, false)
}

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

func join(xs []string) string { return strings.Join(xs, ",") }

func itoa(n int) string { return strconv.Itoa(n) }

func boolish(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func promptLabel(slot, text string) string {
	if slot == "" {
		return text
	}
	if text == "" {
		return slot
	}
	return slot + ": " + text
}

func label(id, v string) string {
	if id == "" {
		return v
	}
	return id + " " + v
}

func short(hash string, n int) string {
	hash = strings.TrimSpace(hash)
	if n <= 0 || len(hash) <= n {
		return hash
	}
	return hash[:n]
}

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
