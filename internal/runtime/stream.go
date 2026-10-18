package runtime

import (
	"context"
	"encoding/json"
	"strings"
)

// Canonical provider-neutral streaming vocabulary. Adapters emit these chunks;
// BlockAssembler is the only assembly algorithm the loop uses.

const (
	ChunkBlockStart      = "block-start"
	ChunkTextDelta       = "text-delta"
	ChunkReasoningDelta  = "reasoning-delta"
	ChunkToolCallDelta   = "tool-call-delta"
	ChunkBlockEnd        = "block-end"
	ChunkUsage           = "usage"
	ChunkFinish          = "finish"

	BlockText      = "text"
	BlockReasoning = "reasoning"
	BlockToolCall  = "tool-call"

	FinishStop      = "stop"
	FinishToolCalls = "tool-calls"
	FinishMaxTokens = "max-tokens"
	FinishAborted   = "aborted"
	FinishError     = "error"

	StreamFrameStart = "start"
	StreamFrameChunk = "chunk"
	StreamFrameEnd   = "end"

	StreamOutcomeCommitted = "committed"
	StreamOutcomeAbandoned = "abandoned"
)

// ContentBlock is one assembled assistant block. JSON names match the harness.
type ContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// TokenUsage is one model call's accounting. InputTokens is uncached input.
type TokenUsage struct {
	InputTokens     int `json:"inputTokens"`
	OutputTokens    int `json:"outputTokens"`
	TotalTokens     int `json:"totalTokens,omitempty"`
	CacheReadTokens int `json:"cacheReadTokens,omitempty"`
}

// FinishReason is why a model response stopped.
type FinishReason struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
}

// StreamChunk is the closed live-token union. Index is 0-based block identity.
type StreamChunk struct {
	Type           string        `json:"type"`
	Index          int           `json:"index,omitempty"`
	BlockType      string        `json:"blockType,omitempty"`
	Text           string        `json:"text,omitempty"`
	ID             string        `json:"id,omitempty"`
	Name           string        `json:"name,omitempty"`
	ArgumentsDelta string        `json:"argumentsDelta,omitempty"`
	Block          *ContentBlock `json:"block,omitempty"`
	Usage          *TokenUsage   `json:"usage,omitempty"`
	Reason         *FinishReason `json:"reason,omitempty"`
}

// IsToken reports whether the chunk carries visible or tool-call content.
func (c StreamChunk) IsToken() bool {
	switch c.Type {
	case ChunkTextDelta, ChunkReasoningDelta, ChunkToolCallDelta, ChunkBlockStart:
		return true
	default:
		return false
	}
}

// Streamer is an optional Client capability. The loop uses it when present so
// the UI can render tokens without pulling a heavy agent SDK.
type Streamer interface {
	ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error)
}

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

func textFromBlocks(blocks []ContentBlock) string {
	var b strings.Builder
	for _, bl := range blocks {
		if bl.Type == BlockText && bl.Text != "" {
			b.WriteString(bl.Text)
		}
	}
	return b.String()
}

func reasoningFromBlocks(blocks []ContentBlock) string {
	var b strings.Builder
	for _, bl := range blocks {
		if bl.Type == BlockReasoning && bl.Text != "" {
			b.WriteString(bl.Text)
		}
	}
	return b.String()
}

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
