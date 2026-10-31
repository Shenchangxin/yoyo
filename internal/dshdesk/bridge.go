// bridge.go 把 dsh.Completer 接到 runtime.Client。
// 它不持有会话。Host 每一轮换上这一轮的客户端；编排台也可以拿它跑工单。

package dshdesk

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// Bridge 用 runtime.Client 完成一次 dsh 请求。OnStream 非空且客户端实现 Streamer 时走流式。
type Bridge struct {
	Client       runtime.Client
	Model        string
	MaxTokens    int
	ShowThinking bool
	Meter        *runtime.Meter
	UserParts    []runtime.ContentPart
	OnStream     func(runtime.StreamChunk)
	partsUsed    bool
}

// Complete 调用 Client。req 的消息和工具清单转成 runtime 的形状。ctx 取消时把已说出的前缀交回去。
func (b *Bridge) Complete(ctx context.Context, req dsh.Request) (dsh.Assistant, error) {
	if b == nil || b.Client == nil {
		return dsh.Assistant{}, errString("missing model client")
	}
	model := req.Model
	if model == "" {
		model = b.Model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = b.MaxTokens
	}
	chat := runtime.ChatRequest{
		Model:        model,
		Messages:     b.withParts(toRuntimeMessages(req.Messages)),
		Tools:        toToolJSON(req.Tools),
		MaxTokens:    maxTokens,
		ShowThinking: b.ShowThinking,
	}
	var msg runtime.Message
	var err error
	if stream, ok := b.Client.(runtime.Streamer); ok && b.OnStream != nil {
		msg, err = stream.ChatStream(ctx, chat, func(ch runtime.StreamChunk) error {
			b.OnStream(ch)
			return nil
		})
	} else {
		msg, err = b.Client.Chat(ctx, chat)
	}
	if b.Meter != nil {
		b.Meter.Add(msg.PromptTokens, msg.CompletionTokens)
	}
	return toAssistant(msg), err
}

// toRuntimeMessages 把 msgs 转成 runtime 消息。角色、正文、工具调用和工具结果 id 原样保留。
func toRuntimeMessages(msgs []dsh.Message) []runtime.Message {
	out := make([]runtime.Message, 0, len(msgs))
	for _, m := range msgs {
		msg := runtime.Message{
			Role:       runtime.Role(m.Role),
			Content:    m.Text,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		for _, call := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, runtime.ToolCall{
				ID: call.ID, Name: call.Name, Arguments: call.Arguments,
			})
		}
		for _, part := range m.Parts {
			msg.Parts = append(msg.Parts, runtime.ContentPart{Type: part.Type, Text: part.Text, ImageURL: part.URL, MIME: part.MIME})
		}
		out = append(out, msg)
	}
	return out
}

// withParts 把这一轮新附上的图片挂到最后一条用户消息。msgs 里没有用户消息时原样返回。只挂一次。
func (b *Bridge) withParts(msgs []runtime.Message) []runtime.Message {
	if b == nil || b.partsUsed || len(b.UserParts) == 0 {
		return msgs
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != runtime.RoleUser {
			continue
		}
		msgs[i].Parts = append(msgs[i].Parts, b.UserParts...)
		b.partsUsed = true
		break
	}
	return msgs
}

// toToolJSON 把 tools 转成函数工具清单。Parameters 非空时放进 function.parameters。
func toToolJSON(tools []dsh.ToolSchema) []runtime.ToolJSON {
	out := make([]runtime.ToolJSON, 0, len(tools))
	for _, tool := range tools {
		fn := map[string]any{"name": tool.Name, "description": tool.Description}
		if tool.Parameters != nil {
			fn["parameters"] = tool.Parameters
		}
		out = append(out, runtime.ToolJSON{Type: "function", Function: fn})
	}
	return out
}

// toAssistant 把 msg 收成 dsh 的完成结果。Truncated 为真时结束原因是 max-tokens。
func toAssistant(msg runtime.Message) dsh.Assistant {
	out := dsh.Assistant{Text: msg.Content}
	for _, call := range msg.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, dsh.ToolCall{
			ID: call.ID, Name: call.Name, Arguments: call.Arguments,
		})
	}
	if msg.Truncated {
		out.Finish = "max-tokens"
	} else if len(out.ToolCalls) > 0 {
		out.Finish = "tool-calls"
	} else if out.Text != "" || len(msg.ToolCalls) == 0 {
		out.Finish = "stop"
	}
	return out
}

// errString 是桥接层的纯文本错误。
type errString string

// Error 返回错误文本。
func (e errString) Error() string { return string(e) }
