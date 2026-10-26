// 模型反射。回复不能用时退回 Reflect。不改剧本，Curate 仍只消费 Reflection。

package evolve

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// reflectorSystem 是反射模型的系统提示。正文发给模型，这里不改它。
const reflectorSystem = `You are the ACE reflector. Diagnose the trajectory. Reply with JSON only:
{"insights":["mechanism: short lesson"],"bullet_tags":[{"id":"bullet-id","tag":"helpful|harmful"}]}
Never rewrite a playbook. Never invent held-out task ids. Incremental insights only.`

// ReflectLLM 先算确定性的 Reflect。客户端或 ctx 为 nil、对话失败、正文为空、带工具调用、截不出对象、解析失败，或洞察和标签都空时，返回那份确定性结果且错误为 nil。
// 解析成功但 Mechanism 为空时，用确定性结果的机制补上。
// client 为 nil 时直接返回确定性结果；否则用它 Chat。
// model 写入 ChatRequest 的 Model。
// events 和 failed、pb 一起交给 Reflect。
// failed 放进发给模型的 JSON，键名是 failed。
// pb 的 Bullets 放进该 JSON 的 playbook，并交给 Reflect。
func ReflectLLM(ctx context.Context, client runtime.Client, model string, events []trace.Event, failed map[string]string, pb artifact.Playbook) (Reflection, error) {
	fallback := Reflect(events, failed, pb)
	if client == nil || ctx == nil {
		return fallback, nil
	}
	raw, _ := json.Marshal(map[string]any{
		"failed":   failed,
		"playbook": pb.Bullets,
		"reflect":  fallback,
	})
	msg, err := client.Chat(ctx, runtime.ChatRequest{
		Model: model,
		Messages: []runtime.Message{
			{Role: runtime.RoleSystem, Content: reflectorSystem},
			{Role: runtime.RoleUser, Content: string(raw)},
		},
	})
	if err != nil || strings.TrimSpace(msg.Content) == "" || len(msg.ToolCalls) > 0 {
		return fallback, nil
	}
	start := strings.Index(msg.Content, "{")
	end := strings.LastIndex(msg.Content, "}")
	if start < 0 || end <= start {
		return fallback, nil
	}
	var parsed Reflection
	if err := json.Unmarshal([]byte(msg.Content[start:end+1]), &parsed); err != nil {
		return fallback, nil
	}
	if len(parsed.Insights) == 0 && len(parsed.BulletTags) == 0 {
		return fallback, nil
	}
	if parsed.Mechanism == "" {
		parsed.Mechanism = fallback.Mechanism
	}
	return parsed, nil
}
