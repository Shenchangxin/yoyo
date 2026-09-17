package evolve

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

const reflectorSystem = `You are the ACE reflector. Diagnose the trajectory. Reply with JSON only:
{"insights":["mechanism: short lesson"],"bullet_tags":[{"id":"bullet-id","tag":"helpful|harmful"}]}
Never rewrite a playbook. Never invent held-out task ids. Incremental insights only.`

// ReflectLLM asks the model for insights, then falls back to Reflect if the
// reply is unusable. The curator remains deterministic.
func ReflectLLM(ctx context.Context, client runtime.Client, model string, events []trace.Event, failed map[string]string, pb artifact.Playbook) (Reflection, error) {
	fallback := Reflect(events, failed, pb)
	if client == nil || ctx == nil {
		return fallback, nil
	}
	raw, _ := json.Marshal(map[string]any{
		"failed":  failed,
		"playbook": pb.Bullets,
		"reflect": fallback,
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
