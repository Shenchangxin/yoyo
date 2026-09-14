package evolve

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

type Proposal struct {
	ID             string                   `json:"id"`
	TargetCluster  string                   `json:"target_cluster"`
	Surface        string                   `json:"surface"`
	Expected       string                   `json:"expected"`
	Risk           string                   `json:"risk"`
	Audit          string                   `json:"audit"`
	Fragment       *artifact.PromptFragment `json:"fragment,omitempty"`
	PlaybookBullet *artifact.PlaybookBullet `json:"playbook_bullet,omitempty"`
	SkillMD        string                   `json:"skill_md,omitempty"`
}

const proposerSystem = `You propose bounded L1 harness edits. Reply with JSON array of 1-3 objects:
[{"surface":"prompt_fragment|playbook|skill","slot":"verification|failure_recovery|runtime","text":"...","expected":"...","risk":"...","audit":"..."}]
Rules: minimal, diverse, do not rewrite the control loop, do not touch evaluator/secrets/kernel.`

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

func HeuristicPropose(bundle EvidenceBundle, k int) []Proposal {
	var out []Proposal
	for i, c := range bundle.Clusters {
		if i >= k {
			break
		}
		text := heuristicText(c.Signature.AgentMechanism)
		out = append(out, Proposal{
			ID:            fmt.Sprintf("p%d", i+1),
			TargetCluster: c.Signature.Key(),
			Surface:       "prompt_fragment",
			Expected:      "address " + c.Signature.AgentMechanism,
			Risk:          "may overfit held-in tasks",
			Audit:         "heuristic L1 fragment",
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

func parseProposals(content string) []Proposal {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end <= start {
		return nil
	}
	var raw []map[string]string
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return nil
	}
	var out []Proposal
	for i, m := range raw {
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

func or(a, b string) string {
	if a == "" {
		return b
	}
	return a
}
