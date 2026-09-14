package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
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
}

type Client interface {
	Chat(ctx context.Context, req ChatRequest) (Message, error)
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
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return Message{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return Message{}, err
	}
	if res.StatusCode >= 300 {
		return Message{}, fmt.Errorf("openai: %s: %s", res.Status, truncate(string(raw), 800))
	}
	var parsed struct {
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
	for _, tc := range m.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
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
