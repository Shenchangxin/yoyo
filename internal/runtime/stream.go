// 与供应商无关的流式块词汇。适配器发出这些块，循环只按这里的块组装。
// 本文件不读网络。流片段在前端由 stream-chunk.ts 的 BlockAssembler 收成内容块。

package runtime

import (
	"context"
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
