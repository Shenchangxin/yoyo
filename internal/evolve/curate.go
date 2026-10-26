// 按反射结果给剧本做增量。只通过 ApplyDelta 追加或改计数，不把整本剧本当成一块文本重写。Cycle 在提议之前调用。

package evolve

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// Curate 给已有条目按标签加 Helpful 或 Harmful，并把洞察收成新条目。标签找不到原文时跳过。洞察为空或像仓库概览时跳过。
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

// insightToBullet 按洞察里的机制词选一句固定剧本。对不上时用核对工作区产物的那句。
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

// contains 报告 s 是否包含 sub。
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (indexOf(s, sub) >= 0))
}

// indexOf 返回 sub 在 s 中的字节下标，没有时为 -1。
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// shortID 取字符串前 12 个字节；不足 12 个时原样返回。
func shortID(s string) string {
	if len(s) < 12 {
		return s
	}
	return s[:12]
}
