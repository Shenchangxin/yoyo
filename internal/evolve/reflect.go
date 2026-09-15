package evolve

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Reflection is the ACE Reflector output: diagnose trajectories, tag bullets.
type Reflection struct {
	Insights   []string    `json:"insights"`
	BulletTags []BulletTag `json:"bullet_tags"`
	Mechanism  string      `json:"mechanism"`
}

type BulletTag struct {
	ID  string `json:"id"`
	Tag string `json:"tag"` // helpful | harmful
}

// Reflect is deterministic: no extra model call. The LLM proposer still
// generates candidate text; the reflector only attributes failure mechanisms
// and playbook counters so we never collapse the book into a rewrite.
func Reflect(events []trace.Event, failed map[string]string, pb artifact.Playbook) Reflection {
	bundle := Mine(events, failed)
	ref := Reflection{}
	if len(bundle.Clusters) > 0 {
		ref.Mechanism = bundle.Clusters[0].Signature.AgentMechanism
	}
	for _, c := range bundle.Clusters {
		ref.Insights = append(ref.Insights, c.Signature.AgentMechanism+": "+strings.Join(c.Symptoms, "; "))
	}
	used := usedBulletIDs(events, pb)
	for _, b := range pb.Bullets {
		if !used[b.ID] {
			continue
		}
		tag := "helpful"
		if ref.Mechanism == "unproductive_retry" || ref.Mechanism == "stalled_tool_loop" {
			tag = "harmful"
		}
		if len(failed) == 0 {
			tag = "helpful"
		}
		ref.BulletTags = append(ref.BulletTags, BulletTag{ID: b.ID, Tag: tag})
	}
	return ref
}

func usedBulletIDs(events []trace.Event, pb artifact.Playbook) map[string]bool {
	out := map[string]bool{}
	var sys string
	for _, ev := range events {
		if ev.Type == trace.TypeSystem {
			if t, _ := ev.Payload["text"].(string); t != "" {
				sys += t
			}
		}
	}
	lower := strings.ToLower(sys)
	for _, b := range pb.Bullets {
		if b.Text != "" && strings.Contains(lower, strings.ToLower(b.Text)) {
			out[b.ID] = true
		}
	}
	return out
}
