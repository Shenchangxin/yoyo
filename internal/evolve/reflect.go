// 把轨迹收成洞察和剧本标签。不调用模型；模型路径在 reflect_llm.go。Curate 消费这里的结果，本文件不改剧本。

package evolve

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Reflection 是确定性的反射结果：洞察、条目标签和排在最前的机制。
type Reflection struct {
	Insights   []string    `json:"insights"`
	BulletTags []BulletTag `json:"bullet_tags"`
	Mechanism  string      `json:"mechanism"`
}

// BulletTag 给一条剧本打 helpful 或 harmful。
type BulletTag struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}

// Reflect 用 Mine 的第一簇机制作为 Mechanism，并把每簇收成一条洞察。只给系统轨迹里出现过正文的剧本打标签。
// 失败表为空时标签是 helpful。否则在机制为 unproductive_retry 或 stalled_tool_loop 时打 harmful，其余打 helpful。
// events 和 failed 一起交给 Mine；events 再和 pb 交给 usedBulletIDs。
// failed 长度为 0 时标签写成 helpful。
// pb 的 Bullets 里，编号出现在 usedBulletIDs 结果中的才打标签。
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

// usedBulletIDs 收集系统事件文本里包含其正文的剧本编号。
// events 里 Type 为 TypeSystem 的，把 Payload 的 text 拼在一起再转小写。
// pb 的 Bullets 中 Text 非空、且上面的小写文本包含该 Text 的小写形式时，记下 ID。
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
