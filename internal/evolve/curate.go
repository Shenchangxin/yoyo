package evolve

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// Curate applies ACE delta updates. It never rewrites the playbook in place
// as a blob — only ADD (via ApplyDelta) and Helpful/Harmful counters.
func Curate(pb artifact.Playbook, ref Reflection) artifact.Playbook {
	var add []artifact.PlaybookBullet
	for _, tag := range ref.BulletTags {
		delta := artifact.PlaybookBullet{ID: tag.ID}
		for _, b := range pb.Bullets {
			if b.ID == tag.ID {
				delta.Text = b.Text
				break
			}
		}
		if delta.Text == "" {
			continue
		}
		if tag.Tag == "harmful" {
			delta.Harmful = 1
		} else {
			delta.Helpful = 1
		}
		add = append(add, delta)
	}
	for _, insight := range ref.Insights {
		if insight == "" || looksLikeRepoOverview(strings.ToLower(insight)) {
			continue
		}
		add = append(add, artifact.PlaybookBullet{
			ID:      "learn-" + shortID(insight),
			Text:    insightToBullet(insight),
			Helpful: 1,
			Source:  "ace-curator",
		})
	}
	return pb.ApplyDelta(add, nil)
}

func insightToBullet(insight string) string {
	switch {
	case contains(insight, "missing_artifact"):
		return "Write required output files before exploring further; verify they exist before stopping."
	case contains(insight, "unproductive_retry"):
		return "If the same command fails twice, change strategy instead of retrying it."
	case contains(insight, "stalled_tool_loop"):
		return "After prolonged tool use without new evidence, implement and verify."
	default:
		return "Verify workspace artifacts against the task before claiming success."
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func shortID(s string) string {
	if len(s) < 12 {
		return s
	}
	return s[:12]
}
