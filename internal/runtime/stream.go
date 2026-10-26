// 与供应商无关的流式块词汇。适配器发出这些块，循环只按这里的块组装。
// BlockAssembler 在 assembler.go。本文件不读网络。

package runtime

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	// ChunkBlockStart 是一个块的开始。
	ChunkBlockStart = "block-start"
	// ChunkTextDelta 是可见文本的增量。
	ChunkTextDelta = "text-delta"
	// ChunkReasoningDelta 是推理文本的增量。ShowThinking 为假时 chat 丢掉它。
	ChunkReasoningDelta = "reasoning-delta"
	// ChunkToolCallDelta 是工具调用参数的增量。
	ChunkToolCallDelta = "tool-call-delta"
	// ChunkBlockEnd 是一个块的结束。
	ChunkBlockEnd = "block-end"
	// ChunkUsage 是这一次调用的用量。
	ChunkUsage = "usage"
	// ChunkFinish 是模型结束原因。
	ChunkFinish = "finish"

	// BlockText 是可见文本块。
	BlockText = "text"
	// BlockReasoning 是推理块。
	BlockReasoning = "reasoning"
	// BlockToolCall 是工具调用块。
	BlockToolCall = "tool-call"

	// FinishStop 是模型自己停住。
	FinishStop = "stop"
	// FinishToolCalls 是模型停下来等工具结果。
	FinishToolCalls = "tool-calls"
	// FinishMaxTokens 是停在输出 token 上限。
	FinishMaxTokens = "max-tokens"
	// FinishAborted 是流被中止。
	FinishAborted = "aborted"
	// FinishError 是流以错误结束。
	FinishError = "error"

	// StreamFrameStart 是一次尝试的开始帧。
	StreamFrameStart = "start"
	// StreamFrameChunk 是一次尝试中的块帧。
	StreamFrameChunk = "chunk"
	// StreamFrameEnd 是一次尝试的结束帧。
	StreamFrameEnd = "end"

	// StreamOutcomeCommitted 表示这次尝试的块已经收下。
	StreamOutcomeCommitted = "committed"
	// StreamOutcomeAbandoned 表示这次尝试的块丢弃。
	StreamOutcomeAbandoned = "abandoned"
)

// ContentBlock 是组装好的一块助手内容。JSON 名字和 harness 一致。
type ContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// TokenUsage 是一次模型调用的用量。
type TokenUsage struct {
	InputTokens     int `json:"inputTokens"` // 未命中缓存的输入 token
	OutputTokens    int `json:"outputTokens"`
	TotalTokens     int `json:"totalTokens,omitempty"`
	CacheReadTokens int `json:"cacheReadTokens,omitempty"`
}

// FinishReason 是模型这次停下的原因。
type FinishReason struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
}

// StreamChunk 是实时 token 的封闭联合。适配器只发这些字段。
type StreamChunk struct {
	Type           string        `json:"type"`
	Index          int           `json:"index,omitempty"` // 从 0 起的块序号；省略时是 0
	BlockType      string        `json:"blockType,omitempty"`
	Text           string        `json:"text,omitempty"`
	ID             string        `json:"id,omitempty"`
	Name           string        `json:"name,omitempty"`
	ArgumentsDelta string        `json:"argumentsDelta,omitempty"`
	Block          *ContentBlock `json:"block,omitempty"`
	Usage          *TokenUsage   `json:"usage,omitempty"`
	Reason         *FinishReason `json:"reason,omitempty"`
}

// IsToken 在块是文本增量、推理增量、工具参数增量或块开始时返回 true。块结束、用量和结束原因返回 false。
func (c StreamChunk) IsToken() bool {
	switch c.Type {
	case ChunkTextDelta, ChunkReasoningDelta, ChunkToolCallDelta, ChunkBlockStart:
		return true
	default:
		return false
	}
}

// Streamer 是 Client 的可选能力。循环发现它就走流式，界面可以逐块画，不必再接一套代理 SDK。
type Streamer interface {
	ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error)
}

// chunkMap 把块编成 JSON 对象。编解码失败时只留下 type。
func chunkMap(ch StreamChunk) map[string]any {
	b, err := json.Marshal(ch)
	if err != nil {
		return map[string]any{"type": ch.Type}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{"type": ch.Type}
	}
	return m
}

// textFromBlocks 按顺序拼出文本块。推理块和空文本不进入结果。
// blocks 里 Type 为文本且 Text 非空的块按顺序拼起来。
func textFromBlocks(blocks []ContentBlock) string {
	var b strings.Builder
	for _, bl := range blocks {
		if bl.Type == BlockText && bl.Text != "" {
			b.WriteString(bl.Text)
		}
	}
	return b.String()
}

// reasoningFromBlocks 按顺序拼出推理块。空文本不进入结果。
// blocks 里 Type 为推理且 Text 非空的块按顺序拼起来。
func reasoningFromBlocks(blocks []ContentBlock) string {
	var b strings.Builder
	for _, bl := range blocks {
		if bl.Type == BlockReasoning && bl.Text != "" {
			b.WriteString(bl.Text)
		}
	}
	return b.String()
}

// toolCallsFromBlocks 按块顺序抽出工具调用。没有工具块时返回 nil。
// blocks 里 Type 为工具调用的块按顺序收成 ToolCall。
func toolCallsFromBlocks(blocks []ContentBlock) []ToolCall {
	var out []ToolCall
	for _, bl := range blocks {
		if bl.Type != BlockToolCall {
			continue
		}
		out = append(out, ToolCall{ID: bl.ID, Name: bl.Name, Arguments: bl.Arguments})
	}
	return out
}

// blocksFromMessage 把一条助手消息收成块。正文去掉空白后非空时先放文本块，再按原顺序放工具调用。没有正文也没有工具调用时返回 nil。
// msg 的 Content 去掉空白后非空时先放文本块，再按 ToolCalls 的原顺序放工具调用块。
func blocksFromMessage(msg Message) []ContentBlock {
	var out []ContentBlock
	if strings.TrimSpace(msg.Content) != "" {
		out = append(out, ContentBlock{Type: BlockText, Text: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		out = append(out, ContentBlock{Type: BlockToolCall, ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
	}
	return out
}
