package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type RunRequest struct {
	SessionID        string
	TaskID           string
	User             string
	Workspace        string
	Harness          artifact.HarnessSnapshot
	HarnessHash      string
	ModelFingerprint string
	Model            string
	Loop             artifact.LoopPreset
	Fragments        []artifact.PromptFragment
	Playbook         artifact.Playbook
	Skills           []artifact.Skill
	Tools            *WorkspaceTools
	Client           Client
	Trace            *trace.Store
	OnEvent          func(trace.Event)
}

func Run(ctx context.Context, req RunRequest) (string, error) {
	if req.Loop.MaxTurns <= 0 {
		req.Loop.MaxTurns = 24
	}
	system := AssembleSystem(req.Loop, req.Fragments, req.Playbook, req.Skills)
	emit(req, trace.TypeSystem, "runtime", map[string]any{"text": system})
	emit(req, trace.TypeUser, "user", map[string]any{"text": req.User})

	messages := []Message{
		{Role: RoleSystem, Content: system},
		{Role: RoleUser, Content: req.User},
	}
	toolsJSON := BuiltinToolJSON()
	var last string
	toolCount := 0
	for turn := 0; turn < req.Loop.MaxTurns; turn++ {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		if req.Loop.MaxToolMessages > 0 && toolCount >= req.Loop.MaxToolMessages {
			messages = append(messages, Message{Role: RoleUser, Content: "Stop using tools and produce the final answer now."})
		}
		msg, err := req.Client.Chat(ctx, ChatRequest{
			Model:    req.Model,
			Messages: compact(messages, req.Loop.CompactionKeep),
			Tools:    toolsJSON,
		})
		if err != nil {
			emit(req, trace.TypeError, "model", map[string]any{"error": err.Error()})
			return last, err
		}
		if msg.Content != "" {
			last = msg.Content
			emit(req, trace.TypeAssistant, "model", map[string]any{"text": msg.Content})
		}
		if len(msg.ToolCalls) == 0 {
			return strings.TrimSpace(msg.Content), nil
		}
		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			toolCount++
			emit(req, trace.TypeToolCall, "agent", map[string]any{"name": tc.Name, "arguments": tc.Arguments, "id": tc.ID})
			res := req.Tools.Call(tc.Name, tc.Arguments)
			content := res.Content
			if res.Err != nil {
				content = "ERROR: " + res.Err.Error()
				if res.Content != "" {
					content += "\n" + res.Content
				}
			}
			if tc.Name == "load_skill" && res.Err == nil {
				emit(req, trace.TypeInject, "skill", map[string]any{"name": tc.Name, "text": content})
			}
			emit(req, trace.TypeToolResult, "tool", map[string]any{"name": tc.Name, "id": tc.ID, "content": content, "untrusted": true})
			messages = append(messages, Message{
				Role:       RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Name,
				Content:    content,
			})
		}
	}
	return last, fmt.Errorf("max turns reached")
}

func emit(req RunRequest, typ trace.EventType, source string, payload map[string]any) {
	ev := trace.Event{
		Type:             typ,
		Source:           source,
		SessionID:        req.SessionID,
		HarnessSnapshot:  req.HarnessHash,
		ModelFingerprint: req.ModelFingerprint,
		TaskID:           req.TaskID,
		Payload:          payload,
	}
	if req.Trace != nil {
		_ = req.Trace.Append(ev)
	}
	if req.OnEvent != nil {
		req.OnEvent(ev)
	}
}

func compact(msgs []Message, keep int) []Message {
	if keep <= 0 || len(msgs) <= keep+1 {
		return msgs
	}
	out := []Message{msgs[0]}
	out = append(out, msgs[len(msgs)-keep:]...)
	return out
}
