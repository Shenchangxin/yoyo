// 向模型要最多 k 条 L1 提议，失败时退回按簇生成的启发式片段。准入在 quality.go，写入在 admit.go。本文件不跑 Harbor。

package evolve

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// Proposal 是一条待准入的 L1 改动。五种载荷至多一种：片段、剧本条目、技能、指令文本、中间件文本。
type Proposal struct {
	ID              string                   `json:"id"`
	TargetCluster   string                   `json:"target_cluster"`
	Surface         string                   `json:"surface"`
	Expected        string                   `json:"expected"`
	Risk            string                   `json:"risk"`
	Audit           string                   `json:"audit"`
	PredictedFixes  []string                 `json:"predicted_fixes,omitempty"`
	AtRisk          []string                 `json:"at_risk,omitempty"`
	Fragment        *artifact.PromptFragment `json:"fragment,omitempty"`
	PlaybookBullet  *artifact.PlaybookBullet `json:"playbook_bullet,omitempty"`
	SkillMD         string                   `json:"skill_md,omitempty"`
	InstructionSlot string                   `json:"instruction_slot,omitempty"`
	InstructionText string                   `json:"instruction_text,omitempty"`
	MiddlewareN     int                      `json:"middleware_n,omitempty"`
	MiddlewareText  string                   `json:"middleware_text,omitempty"`
}

// PassSummary 是证据里一条已通过题的编号和备注。
type PassSummary struct {
	TaskID string `json:"task_id"`
	Note   string `json:"note,omitempty"`
}

// PriorTrial 是档案里先前一次试用的摘要，供提议者看见。
type PriorTrial struct {
	Hash     string `json:"hash"`
	Surface  string `json:"surface"`
	Reason   string `json:"reason"`
	Accepted bool   `json:"accepted"`
}

// proposerSystem 是提议者的系统提示。正文是发给模型的英文约束，这里不改它。
const proposerSystem = `You propose bounded L1 harness edits. Reply with JSON array of 1-3 objects:
[{"surface":"prompt_fragment|playbook|skill|instruction|middleware","slot":"verification|failure_recovery|runtime|bootstrap","text":"...","expected":"...","risk":"...","audit":"...","predicted_fixes":["task-id"],"at_risk":["task-id"]}]
Rules: one surface per object; predicted_fixes must come from repeated fail-vs-pass behavior across tasks, not from copying a task instruction; do not rewrite the control loop; do not touch evaluator/secrets/kernel; do not mention held-out task ids.`

// Propose 向模型要 JSON 提议。k 小于等于 0 时用 3。客户端为 nil、对话失败或解析结果为空时改用 HeuristicPropose。结果长于 k 时截到 k。
// ctx 传给 client.Chat。
// client 为 nil 时改走 HeuristicPropose；否则用它 Chat，失败或解析为空时也改走。
// model 写入 ChatRequest 的 Model。
// bundle 序列化后放进用户消息；解析为空时交给 HeuristicPropose。
func Propose(ctx context.Context, client runtime.Client, model string, bundle EvidenceBundle, k int) ([]Proposal, error) {
	if k <= 0 {
		k = 3
	}
	if client == nil {
		return HeuristicPropose(bundle, k), nil
	}
	raw, _ := json.Marshal(bundle)
	msg, err := client.Chat(ctx, runtime.ChatRequest{
		Model: model,
		Messages: []runtime.Message{
			{Role: runtime.RoleSystem, Content: proposerSystem},
			{Role: runtime.RoleUser, Content: "Evidence:\n" + string(raw)},
		},
	})
	if err != nil {
		return HeuristicPropose(bundle, k), nil
	}
	parsed := parseProposals(msg.Content)
	if len(parsed) == 0 {
		return HeuristicPropose(bundle, k), nil
	}
	if len(parsed) > k {
		parsed = parsed[:k]
	}
	return parsed, nil
}

// HeuristicPropose 为前 k 个簇各做一条 prompt_fragment。没有簇时返回一条槽位为 verification 的默认片段。
// bundle 的 Clusters 取前 k 个；一个都没有时返回默认片段。
func HeuristicPropose(bundle EvidenceBundle, k int) []Proposal {
	var out []Proposal
	for i, c := range bundle.Clusters {
		if i >= k {
			break
		}
		text := heuristicText(c.Signature.AgentMechanism)
		out = append(out, Proposal{
			ID:             fmt.Sprintf("p%d", i+1),
			TargetCluster:  c.Signature.Key(),
			Surface:        "prompt_fragment",
			Expected:       "address " + c.Signature.AgentMechanism,
			Risk:           "may overfit held-in tasks",
			Audit:          "heuristic L1 fragment",
			PredictedFixes: append([]string{}, c.TaskIDs...),
			Fragment: &artifact.PromptFragment{
				ID:      fmt.Sprintf("auto-%s", c.Signature.AgentMechanism),
				Slot:    slotFor(c.Signature.AgentMechanism),
				Text:    text,
				Surface: "prompt_fragment",
			},
		})
	}
	if len(out) == 0 {
		out = append(out, Proposal{
			ID:      "p1",
			Surface: "prompt_fragment",
			Audit:   "default verification reminder",
			Fragment: &artifact.PromptFragment{
				ID:   "verify-early",
				Slot: "verification",
				Text: "Create required output artifacts early and re-read them before stopping.",
			},
		})
	}
	return out
}

// heuristicText 按机制返回一句固定片段文案。
// mech 为 missing_artifact、unproductive_retry 或 stalled_tool_loop 时各返回一句固定文案，其余返回核对工作区的那句。
func heuristicText(mech string) string {
	switch mech {
	case "missing_artifact":
		return "Identify the required output path first and write a placeholder file before exploring further."
	case "unproductive_retry":
		return "If the same command fails twice, change strategy; do not retry identical invocations."
	case "stalled_tool_loop":
		return "After prolonged tool use without new evidence, stop exploring and implement plus verify."
	default:
		return "Verify outcomes against the actual workspace before concluding."
	}
}

// slotFor 把重试和空转机制放到 failure_recovery，把缺产物放到 verification，其余放到 runtime。
// mech 为 unproductive_retry 或 stalled_tool_loop 时返回 failure_recovery，为 missing_artifact 时返回 verification，其余返回 runtime。
func slotFor(mech string) string {
	switch mech {
	case "unproductive_retry", "stalled_tool_loop":
		return "failure_recovery"
	case "missing_artifact":
		return "verification"
	default:
		return "runtime"
	}
}

// parseProposals 从回复里截出 JSON 数组。解析失败时改走 parseProposalsLegacy。没有 text 的对象丢弃。中间件的 MiddlewareN 写成 2。未知 surface 收成 prompt_fragment。
// content 去掉两端空白后，从第一个 [ 截到最后一个 ]；截不出返回 nil，这段解析失败则交给 parseProposalsLegacy。
func parseProposals(content string) []Proposal {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end <= start {
		return nil
	}
	var raw []map[string]any
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return parseProposalsLegacy(content[start : end+1])
	}
	var out []Proposal
	for i, m := range raw {
		p := Proposal{
			ID:             fmt.Sprintf("m%d", i+1),
			Surface:        strMap(m, "surface"),
			Expected:       strMap(m, "expected"),
			Risk:           strMap(m, "risk"),
			Audit:          strMap(m, "audit"),
			PredictedFixes: strSlice(m["predicted_fixes"]),
			AtRisk:         strSlice(m["at_risk"]),
		}
		text := strMap(m, "text")
		if text == "" {
			continue
		}
		switch p.Surface {
		case "playbook":
			p.PlaybookBullet = &artifact.PlaybookBullet{ID: p.ID, Text: text, Helpful: 1}
		case "skill":
			p.SkillMD = "---\nname: auto-" + p.ID + "\ndescription: " + strings.ReplaceAll(text, "\n", " ") + "\n---\n\n" + text + "\n"
		case "instruction":
			p.InstructionSlot = or(strMap(m, "slot"), "verification")
			p.InstructionText = text
		case "middleware":
			p.MiddlewareN = 2
			p.MiddlewareText = text
		default:
			p.Surface = "prompt_fragment"
			p.Fragment = &artifact.PromptFragment{ID: p.ID, Slot: or(strMap(m, "slot"), "runtime"), Text: text}
		}
		out = append(out, p)
	}
	return out
}

// parseProposalsLegacy 解析字符串到字符串的对象数组。技能和剧本同样落载荷，其余收成 prompt_fragment。
// raw 按字符串到字符串的对象数组解开，失败返回 nil。
func parseProposalsLegacy(raw string) []Proposal {
	var maps []map[string]string
	if err := json.Unmarshal([]byte(raw), &maps); err != nil {
		return nil
	}
	var out []Proposal
	for i, m := range maps {
		p := Proposal{
			ID:       fmt.Sprintf("m%d", i+1),
			Surface:  m["surface"],
			Expected: m["expected"],
			Risk:     m["risk"],
			Audit:    m["audit"],
		}
		text := m["text"]
		if text == "" {
			continue
		}
		switch p.Surface {
		case "playbook":
			p.PlaybookBullet = &artifact.PlaybookBullet{ID: p.ID, Text: text, Helpful: 1}
		case "skill":
			p.SkillMD = "---\nname: auto-" + p.ID + "\ndescription: " + strings.ReplaceAll(text, "\n", " ") + "\n---\n\n" + text + "\n"
		default:
			p.Surface = "prompt_fragment"
			p.Fragment = &artifact.PromptFragment{ID: p.ID, Slot: or(m["slot"], "runtime"), Text: text}
		}
		out = append(out, p)
	}
	return out
}

// strMap 取出字符串字段，类型不对时返回空串。
// m 用 k 取值，类型不是 string 时返回空串。
// k 作为 m 的键。
func strMap(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

// strSlice 把任意值里的非空字符串收成切片。不是数组时返回 nil。
// v 不是 []any 时返回 nil；否则收集其中非空字符串。
func strSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, x := range arr {
		if s, ok := x.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// or 在 a 为空时返回 b。
func or(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

// sanitizeHeldOut 从 PredictedFixes 和 AtRisk 里去掉 held 编号。不改提议正文。held 为空时原样返回。
// p 在 held 非空时，把 PredictedFixes 和 AtRisk 交给 dropHeld 后返回。
func sanitizeHeldOut(p Proposal, held map[string]bool) Proposal {
	if len(held) == 0 {
		return p
	}
	p.PredictedFixes = dropHeld(p.PredictedFixes, held)
	p.AtRisk = dropHeld(p.AtRisk, held)
	return p
}

// dropHeld 丢掉出现在 held 里的编号。
// ids 里不在 held 中的项按原顺序保留。
func dropHeld(ids []string, held map[string]bool) []string {
	var out []string
	for _, id := range ids {
		if held[id] {
			continue
		}
		out = append(out, id)
	}
	return out
}
