// 快照的写入、读回，以及按哈希和名字做的字段对照。不解开载荷。内容差异为空时，会话对比会改用这里的文本。

package artifact

import (
	"bytes"
	"fmt"
	"strings"
)

// PutSnapshot 以 KindHarnessSnapshot 写入。ID 为空时用 harness，只影响这一次写入，不改调用方手里的值。
func (s *Store) PutSnapshot(snap HarnessSnapshot) (string, error) {
	if snap.ID == "" {
		snap.ID = "harness"
	}
	return s.Put(KindHarnessSnapshot, snap.ID, snap)
}

// GetSnapshot 按哈希解出 HarnessSnapshot。读取或 JSON 解码失败时返回错误，不另查信封的 Kind。
func (s *Store) GetSnapshot(hash string) (HarnessSnapshot, error) {
	snap, _, err := Decode[HarnessSnapshot](s, hash)
	return snap, err
}

// DiffField 是 SnapshotDiffFields 的一行。From 和 To 是哈希或名字的原文，不是解开的正文。Changed 为零表示这一面相同。
type DiffField struct {
	Field   string `json:"field"`
	From    string `json:"from"`
	To      string `json:"to"`
	Changed bool   `json:"changed"`
}

// SnapshotDiffFields 逐面比较两份快照，始终返回全部面，包含没变的。切片按逗号拼起来比，顺序不同算有变化。这里不读对象库。
func SnapshotDiffFields(a, b HarnessSnapshot) []DiffField {
	join := func(xs []string) string { return strings.Join(xs, ",") }
	rows := []DiffField{
		{"model", a.ModelFingerprint, b.ModelFingerprint, a.ModelFingerprint != b.ModelFingerprint},
		{"parent", a.Parent, b.Parent, a.Parent != b.Parent},
		{"loop", a.LoopPreset, b.LoopPreset, a.LoopPreset != b.LoopPreset},
		{"playbook", a.Playbook, b.Playbook, a.Playbook != b.Playbook},
		{"policy", a.PolicyPack, b.PolicyPack, a.PolicyPack != b.PolicyPack},
		{"eval", a.EvalSuite, b.EvalSuite, a.EvalSuite != b.EvalSuite},
		{"prompts", join(a.PromptFragments), join(b.PromptFragments), join(a.PromptFragments) != join(b.PromptFragments)},
		{"skills", join(a.Skills), join(b.Skills), join(a.Skills) != join(b.Skills)},
		{"tools", join(a.Tools), join(b.Tools), join(a.Tools) != join(b.Tools)},
		{"eval_tools", join(a.EvalTools), join(b.EvalTools), join(a.EvalTools) != join(b.EvalTools)},
		{"wasm", join(a.WASMPlugins), join(b.WASMPlugins), join(a.WASMPlugins) != join(b.WASMPlugins)},
		{"note", a.Note, b.Note, a.Note != b.Note},
	}
	return rows
}

// SnapshotDiff 把 SnapshotDiffFields 打成多行文本。没变的面只写当前值，变了的面写成 from -> to。
func SnapshotDiff(a, b HarnessSnapshot) string {
	var buf bytes.Buffer
	for _, f := range SnapshotDiffFields(a, b) {
		if !f.Changed {
			fmt.Fprintf(&buf, "%s %s\n", f.Field, f.From)
			continue
		}
		fmt.Fprintf(&buf, "%s %s -> %s\n", f.Field, f.From, f.To)
	}
	return buf.String()
}

// CloneSnapshot 复制五份切片，避免改到原来的快照。没有元素时得到 nil。标量按值拷贝。
func CloneSnapshot(s HarnessSnapshot) HarnessSnapshot {
	s.PromptFragments = append([]string(nil), s.PromptFragments...)
	s.Skills = append([]string(nil), s.Skills...)
	s.Tools = append([]string(nil), s.Tools...)
	s.EvalTools = append([]string(nil), s.EvalTools...)
	s.WASMPlugins = append([]string(nil), s.WASMPlugins...)
	return s
}
