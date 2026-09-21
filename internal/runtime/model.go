package runtime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleDeveloper Role = "developer"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleMemory    Role = "working_memory"
)

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	MIME     string `json:"mime,omitempty"`
}

type Message struct {
	Role             Role          `json:"role"`
	Content          string        `json:"content,omitempty"`
	Parts            []ContentPart `json:"parts,omitempty"`
	Name             string        `json:"name,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	PromptTokens     int        `json:"-"`
	CompletionTokens int        `json:"-"`
	CachedTokens     int        `json:"-"`
	CacheReported    bool       `json:"-"`
}

type ToolJSON struct {
	Type     string         `json:"type"`
	Function map[string]any `json:"function"`
}

type ChatRequest struct {
	Model       string     `json:"model"`
	Messages    []Message  `json:"messages"`
	Tools       []ToolJSON `json:"tools,omitempty"`
	Temperature float64    `json:"temperature,omitempty"`
	MaxTokens   int        `json:"max_tokens,omitempty"`
	CacheKey    string     `json:"-"`
}

type StreamDelta struct {
	Text     string
	Tool     ToolCall
	ToolDone bool
}

type Client interface {
	Chat(ctx context.Context, req ChatRequest) (Message, error)
}

// Streamer is an optional capability. The loop uses it when present so the
// UI can render tokens without pulling a heavy agent SDK.
type Streamer interface {
	ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error)
}

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewOpenAIClient(baseURL, apiKey string) *OpenAIClient {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return c.ChatStream(ctx, req, nil)
}

func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	msg, err := c.doChat(ctx, req, true, emit)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "stream") {
		return c.doChat(ctx, req, false, emit)
	}
	return msg, err
}

func (c *OpenAIClient) doChat(ctx context.Context, req ChatRequest, stream bool, emit func(StreamDelta) error) (Message, error) {
	payload := map[string]any{
		"model":    req.Model,
		"messages": wireMessages(req.Messages),
		"stream":   stream,
	}
	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}
	if req.Temperature != 0 {
		payload["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if stream {
		payload["stream_options"] = map[string]any{"include_usage": true}
	}
	if req.CacheKey != "" {
		payload["prompt_cache_key"] = req.CacheKey
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if req.CacheKey != "" {
		httpReq.Header.Set("X-Prompt-Cache-Key", req.CacheKey)
	}
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return Message{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return Message{}, fmt.Errorf("openai: %s: %s", res.Status, truncate(string(raw), 800))
	}
	if !stream {
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return Message{}, err
		}
		msg, err := parseOpenAIJSON(raw)
		if err == nil && emit != nil && msg.Content != "" {
			_ = emit(StreamDelta{Text: msg.Content})
		}
		return msg, err
	}
	return readOpenAIStream(res.Body, emit)
}

type streamAcc struct {
	id, name, args string
	done           bool
}

func readOpenAIStream(r io.Reader, emit func(StreamDelta) error) (Message, error) {
	br := bufio.NewReader(r)
	var content strings.Builder
	tools := map[int]*streamAcc{}
	maxIdx := -1
	sawData := false
	promptTok, completionTok, cachedTok := 0, 0, 0
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			trim := strings.TrimSpace(line)
			if !sawData && trim != "" && !strings.HasPrefix(trim, "data:") {
				rest, _ := io.ReadAll(br)
				return parseOpenAIJSON(append([]byte(line), rest...))
			}
			line = trim
		}
		if strings.HasPrefix(line, "data:") {
			sawData = true
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data != "" && data != "[DONE]" {
				msg, piece, uPrompt, uComp, uCached := applyStreamChunk(data, tools, &maxIdx)
				if uPrompt > 0 {
					promptTok = uPrompt
				}
				if uComp > 0 {
					completionTok = uComp
				}
				if uCached > 0 {
					cachedTok = uCached
				}
				if msg != nil {
					msg.PromptTokens = promptTok
					msg.CompletionTokens = completionTok
					msg.CachedTokens = cachedTok
					msg.CacheReported = cachedTok > 0 || uPrompt > 0
					if emit != nil && msg.Content != "" {
						_ = emit(StreamDelta{Text: msg.Content})
					}
					return *msg, nil
				}
				if piece != "" {
					content.WriteString(piece)
					if emit != nil {
						_ = emit(StreamDelta{Text: piece})
					}
				}
				if emit != nil {
					emitCompletedTools(tools, emit)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return Message{}, err
		}
	}
	out := Message{Role: RoleAssistant, Content: content.String(), PromptTokens: promptTok, CompletionTokens: completionTok, CachedTokens: cachedTok, CacheReported: cachedTok > 0 || promptTok > 0}
	for i := 0; i <= maxIdx; i++ {
		a := tools[i]
		if a == nil {
			continue
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{ID: a.id, Name: a.name, Arguments: a.args})
	}
	return out, nil
}

func emitCompletedTools(tools map[int]*streamAcc, emit func(StreamDelta) error) {
	for i := 0; i < len(tools)+8; i++ {
		a := tools[i]
		if a == nil || a.done || a.name == "" {
			continue
		}
		if a.args != "" && json.Valid([]byte(a.args)) {
			a.done = true
			_ = emit(StreamDelta{Tool: ToolCall{ID: a.id, Name: a.name, Arguments: a.args}, ToolDone: true})
		}
	}
}

func applyStreamChunk(data string, tools map[int]*streamAcc, maxIdx *int) (*Message, string, int, int, int) {
	var chunk struct {
		Usage *struct {
			PromptTokens         int `json:"prompt_tokens"`
			CompletionTokens     int `json:"completion_tokens"`
			CacheReadInputTokens int `json:"cache_read_input_tokens"`
			PromptTokensDetails  *struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
		Choices []struct {
			Delta struct {
				Content   any `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			Message *struct {
				Content   any `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, "", 0, 0, 0
	}
	uPrompt, uComp, uCached := 0, 0, 0
	if chunk.Usage != nil {
		uPrompt, uComp = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		uCached = usageCached(chunk.Usage.CacheReadInputTokens, chunk.Usage.PromptTokensDetails)
	}
	if len(chunk.Choices) == 0 {
		return nil, "", uPrompt, uComp, uCached
	}
	ch := chunk.Choices[0]
	if ch.Message != nil && (stringify(ch.Message.Content) != "" || len(ch.Message.ToolCalls) > 0) {
		msg := Message{Role: RoleAssistant, Content: stringify(ch.Message.Content)}
		for _, tc := range ch.Message.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
		}
		return &msg, "", uPrompt, uComp, uCached
	}
	for _, tc := range ch.Delta.ToolCalls {
		a := tools[tc.Index]
		if a == nil {
			a = &streamAcc{}
			tools[tc.Index] = a
		}
		if tc.Index > *maxIdx {
			*maxIdx = tc.Index
		}
		if tc.ID != "" {
			a.id = tc.ID
		}
		if tc.Function.Name != "" {
			a.name = tc.Function.Name
		}
		a.args += tc.Function.Arguments
	}
	return nil, stringify(ch.Delta.Content), uPrompt, uComp, uCached
}

func parseOpenAIJSON(raw []byte) (Message, error) {
	var parsed struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			CacheReadInputTokens int `json:"cache_read_input_tokens"`
			PromptTokensDetails *struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   any    `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Message{}, err
	}
	if len(parsed.Choices) == 0 {
		return Message{}, fmt.Errorf("openai: empty choices")
	}
	m := parsed.Choices[0].Message
	out := Message{Role: Role(m.Role), Content: stringify(m.Content)}
	if parsed.Usage != nil {
		out.PromptTokens = parsed.Usage.PromptTokens
		out.CompletionTokens = parsed.Usage.CompletionTokens
		out.CachedTokens = usageCached(parsed.Usage.CacheReadInputTokens, parsed.Usage.PromptTokensDetails)
		out.CacheReported = parsed.Usage.PromptTokensDetails != nil || parsed.Usage.CacheReadInputTokens > 0
	}
	for _, tc := range m.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
}

func usageCached(anthropic int, details *struct {
	CachedTokens int `json:"cached_tokens"`
}) int {
	if details != nil && details.CachedTokens > 0 {
		return details.CachedTokens
	}
	return anthropic
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ScriptedClient is a deterministic model used by tests and replay.
type ScriptedClient struct {
	Steps []Message
	i     int
}

func (s *ScriptedClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	if s.i >= len(s.Steps) {
		return Message{Role: RoleAssistant, Content: "done"}, nil
	}
	m := s.Steps[s.i]
	s.i++
	return m, nil
}
