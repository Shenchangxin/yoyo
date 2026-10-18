package runtime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
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
	ToolCallID       string        `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall    `json:"tool_calls,omitempty"`
	PromptTokens     int           `json:"-"`
	CompletionTokens int           `json:"-"`
	CachedTokens     int           `json:"-"`
	CacheReported    bool          `json:"-"`
	ElapsedMs        int           `json:"-"`
	FirstTokenMs     int           `json:"-"`
	// Truncated is true when the provider stopped the reply at the output-token
	// cap (finish_reason "length"). Tool calls cut off by that stop are not run.
	Truncated bool `json:"-"`
}

type ToolJSON struct {
	Type     string         `json:"type"`
	Function map[string]any `json:"function"`
}

type ChatRequest struct {
	Model        string     `json:"model"`
	Messages     []Message  `json:"messages"`
	Tools        []ToolJSON `json:"tools,omitempty"`
	Temperature  float64    `json:"temperature,omitempty"`
	MaxTokens    int        `json:"max_tokens,omitempty"`
	CacheKey     string     `json:"-"`
	ShowThinking bool       `json:"-"`
}

type Client interface {
	Chat(ctx context.Context, req ChatRequest) (Message, error)
}

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NormalizeBaseURL strips paste whitespace and a trailing slash so
// ".../claw/v1 " does not become ".../claw/v1 /chat/completions".
func NormalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func NewOpenAIClient(baseURL, apiKey string) *OpenAIClient {
	baseURL = NormalizeBaseURL(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		BaseURL:    baseURL,
		APIKey:     strings.TrimSpace(apiKey),
		HTTPClient: openaiHTTPClient(),
	}
}

// openaiHTTPClient keeps streaming unbounded (no Client.Timeout) but raises
// TLSHandshakeTimeout past Go's 10s default. Cross-region gateways such as
// flowyaipc.com otherwise fail before the certificate is even read.
func openaiHTTPClient() *http.Client {
	return &http.Client{Transport: openaiHTTPTransport()}
}

func openaiHTTPTransport() *http.Transport {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 20 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   45 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if dt, ok := http.DefaultTransport.(*http.Transport); ok && dt != nil {
		cloned := dt.Clone()
		cloned.TLSHandshakeTimeout = 45 * time.Second
		if cloned.IdleConnTimeout == 0 {
			cloned.IdleConnTimeout = 90 * time.Second
		}
		return cloned
	}
	return tr
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return c.ChatStream(ctx, req, nil)
}

func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	msg, err := c.doChat(ctx, req, true, emit)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "stream") {
		return c.doChat(ctx, req, false, emit)
	}
	return msg, err
}

func (c *OpenAIClient) doChat(ctx context.Context, req ChatRequest, stream bool, emit func(StreamChunk) error) (Message, error) {
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
	if !req.ShowThinking {
		disableThinking(payload)
	}
	msg, err := c.postChat(ctx, payload, stream, emit)
	if err != nil && thinkingRejected(err.Error()) {
		delete(payload, "thinking")
		delete(payload, "enable_thinking")
		return c.postChat(ctx, payload, stream, emit)
	}
	return msg, err
}

func disableThinking(payload map[string]any) {
	payload["thinking"] = map[string]any{"type": "disabled"}
	payload["enable_thinking"] = false
}

func thinkingRejected(msg string) bool {
	s := strings.ToLower(msg)
	return strings.Contains(s, "thinking") || strings.Contains(s, "enable_thinking") || strings.Contains(s, "reasoning_effort")
}

func (c *OpenAIClient) postChat(ctx context.Context, payload map[string]any, stream bool, emit func(StreamChunk) error) (Message, error) {
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
	if key, _ := payload["prompt_cache_key"].(string); key != "" {
		httpReq.Header.Set("X-Prompt-Cache-Key", key)
	}
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		diaglog.ErrorContext(ctx, "provider request failed", "component", "model", "err", err)
		return Message{}, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusTooManyRequests {
		diaglog.WarnContext(ctx, "provider rate limited", "component", "model", "status", res.StatusCode)
	}
	if res.StatusCode >= 300 {
		if res.StatusCode != http.StatusTooManyRequests {
			diaglog.ErrorContext(ctx, "provider error", "component", "model", "status", res.StatusCode)
		}
		raw, _ := io.ReadAll(res.Body)
		return Message{}, fmt.Errorf("openai: %s: %s", res.Status, truncate(string(raw), 800))
	}
	if !stream {
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return Message{}, err
		}
		msg, err := parseOpenAIJSON(raw)
		if err == nil && emit != nil {
			_ = emitMessageChunks(emit, msg)
		}
		return msg, err
	}
	return readOpenAIStream(ctx, res.Body, emit)
}

type streamAcc struct {
	id, name, args string
}

type openaiEmitter struct {
	emit      func(StreamChunk) error
	next      int
	textIdx   int
	text      strings.Builder
	textOn    bool
	reasonIdx int
	reason    strings.Builder
	reasonOn  bool
	tools     map[int]*toolSlot
}

type toolSlot struct {
	ourIndex int
	id, name, args string
	started  bool
	ended    bool
}

func newOpenAIEmitter(emit func(StreamChunk) error) *openaiEmitter {
	return &openaiEmitter{emit: emit, textIdx: -1, reasonIdx: -1, tools: map[int]*toolSlot{}}
}

func (e *openaiEmitter) out(ch StreamChunk) error {
	if e == nil || e.emit == nil {
		return nil
	}
	return e.emit(ch)
}

func (e *openaiEmitter) textDelta(s string) error {
	if s == "" {
		return nil
	}
	if !e.textOn {
		if err := e.closeReason(); err != nil {
			return err
		}
		e.textIdx = e.next
		e.next++
		e.textOn = true
		if err := e.out(StreamChunk{Type: ChunkBlockStart, Index: e.textIdx, BlockType: BlockText}); err != nil {
			return err
		}
	}
	e.text.WriteString(s)
	return e.out(StreamChunk{Type: ChunkTextDelta, Index: e.textIdx, Text: s})
}

func (e *openaiEmitter) reasonDelta(s string) error {
	if s == "" {
		return nil
	}
	if !e.reasonOn {
		e.reasonIdx = e.next
		e.next++
		e.reasonOn = true
		if err := e.out(StreamChunk{Type: ChunkBlockStart, Index: e.reasonIdx, BlockType: BlockReasoning}); err != nil {
			return err
		}
	}
	e.reason.WriteString(s)
	return e.out(StreamChunk{Type: ChunkReasoningDelta, Index: e.reasonIdx, Text: s})
}

func (e *openaiEmitter) closeText() error {
	if !e.textOn {
		return nil
	}
	err := e.out(StreamChunk{Type: ChunkBlockEnd, Index: e.textIdx, Block: &ContentBlock{Type: BlockText, Text: e.text.String()}})
	e.textOn = false
	return err
}

func (e *openaiEmitter) closeReason() error {
	if !e.reasonOn {
		return nil
	}
	err := e.out(StreamChunk{Type: ChunkBlockEnd, Index: e.reasonIdx, Block: &ContentBlock{Type: BlockReasoning, Text: e.reason.String()}})
	e.reasonOn = false
	return err
}

func (e *openaiEmitter) toolDelta(providerIdx int, id, name, argDelta string) error {
	if err := e.closeReason(); err != nil {
		return err
	}
	if err := e.closeText(); err != nil {
		return err
	}
	slot := e.tools[providerIdx]
	if slot == nil {
		slot = &toolSlot{ourIndex: e.next}
		e.next++
		e.tools[providerIdx] = slot
	}
	if !slot.started {
		slot.started = true
		if err := e.out(StreamChunk{Type: ChunkBlockStart, Index: slot.ourIndex, BlockType: BlockToolCall}); err != nil {
			return err
		}
	}
	if id != "" {
		slot.id = id
	}
	if name != "" {
		slot.name = name
	}
	if argDelta != "" {
		slot.args += argDelta
	}
	ch := StreamChunk{Type: ChunkToolCallDelta, Index: slot.ourIndex, ID: slot.id, ArgumentsDelta: argDelta}
	if name != "" {
		ch.Name = name
	}
	if err := e.out(ch); err != nil {
		return err
	}
	if slot.ended || slot.name == "" || slot.args == "" || !json.Valid([]byte(slot.args)) {
		return nil
	}
	slot.ended = true
	return e.out(StreamChunk{Type: ChunkBlockEnd, Index: slot.ourIndex, Block: &ContentBlock{
		Type: BlockToolCall, ID: slot.id, Name: slot.name, Arguments: slot.args,
	}})
}

func (e *openaiEmitter) closeOpen() error {
	if err := e.closeReason(); err != nil {
		return err
	}
	if err := e.closeText(); err != nil {
		return err
	}
	for i := 0; i < len(e.tools)+8; i++ {
		slot := e.tools[i]
		if slot == nil || !slot.started || slot.ended {
			continue
		}
		slot.ended = true
		if err := e.out(StreamChunk{Type: ChunkBlockEnd, Index: slot.ourIndex, Block: &ContentBlock{
			Type: BlockToolCall, ID: slot.id, Name: slot.name, Arguments: slot.args,
		}}); err != nil {
			return err
		}
	}
	return nil
}

func emitMessageChunks(emit func(StreamChunk) error, msg Message) error {
	if emit == nil {
		return nil
	}
	e := newOpenAIEmitter(emit)
	if err := e.textDelta(msg.Content); err != nil {
		return err
	}
	for i, tc := range msg.ToolCalls {
		if err := e.toolDelta(i, tc.ID, tc.Name, tc.Arguments); err != nil {
			return err
		}
	}
	if err := e.closeOpen(); err != nil {
		return err
	}
	kind := FinishStop
	if len(msg.ToolCalls) > 0 {
		kind = FinishToolCalls
	}
	if msg.Truncated {
		kind = FinishMaxTokens
	}
	return emit(StreamChunk{Type: ChunkFinish, Reason: &FinishReason{Kind: kind}})
}

func readOpenAIStream(ctx context.Context, body io.ReadCloser, emit func(StreamChunk) error) (Message, error) {
	defer body.Close()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = body.Close()
		case <-stop:
		}
	}()
	br := bufio.NewReader(body)
	var content strings.Builder
	tools := map[int]*streamAcc{}
	maxIdx := -1
	sawData := false
	promptTok, completionTok, cachedTok := 0, 0, 0
	finish := ""
	em := newOpenAIEmitter(emit)
	for {
		if err := ctx.Err(); err != nil {
			return Message{}, err
		}
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			trim := strings.TrimSpace(line)
			if !sawData && trim != "" && !strings.HasPrefix(trim, "data:") {
				rest, _ := io.ReadAll(br)
				msg, perr := parseOpenAIJSON(append([]byte(line), rest...))
				if perr == nil {
					_ = emitMessageChunks(emit, msg)
				}
				return msg, perr
			}
			line = trim
		}
		if strings.HasPrefix(line, "data:") {
			sawData = true
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data != "" && data != "[DONE]" {
				if strings.Contains(data, `"finish_reason"`) {
					if r := streamFinishReason(data); r != "" {
						finish = r
					}
				}
				msg, piece, reason, tds, uPrompt, uComp, uCached := applyStreamChunk(data, tools, &maxIdx)
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
					msg.Truncated = isOutputLimitStop(finish)
					_ = emitMessageChunks(emit, *msg)
					return *msg, nil
				}
				if piece != "" {
					content.WriteString(piece)
					_ = em.textDelta(piece)
				}
				if reason != "" {
					_ = em.reasonDelta(reason)
				}
				for _, td := range tds {
					_ = em.toolDelta(td.index, td.id, td.name, td.args)
				}
				if uPrompt > 0 || uComp > 0 {
					_ = em.out(StreamChunk{Type: ChunkUsage, Usage: &TokenUsage{
						InputTokens:     promptTok - cachedTok,
						OutputTokens:    completionTok,
						TotalTokens:     promptTok + completionTok,
						CacheReadTokens: cachedTok,
					}})
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
	_ = em.closeOpen()
	out := Message{Role: RoleAssistant, Content: content.String(), PromptTokens: promptTok, CompletionTokens: completionTok, CachedTokens: cachedTok, CacheReported: cachedTok > 0 || promptTok > 0, Truncated: isOutputLimitStop(finish)}
	for i := 0; i <= maxIdx; i++ {
		a := tools[i]
		if a == nil {
			continue
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{ID: a.id, Name: a.name, Arguments: a.args})
	}
	kind := FinishStop
	if len(out.ToolCalls) > 0 {
		kind = FinishToolCalls
	}
	if out.Truncated {
		kind = FinishMaxTokens
	}
	_ = em.out(StreamChunk{Type: ChunkFinish, Reason: &FinishReason{Kind: kind}})
	return out, nil
}

type toolWireDelta struct {
	index      int
	id, name, args string
}

func applyStreamChunk(data string, tools map[int]*streamAcc, maxIdx *int) (*Message, string, string, []toolWireDelta, int, int, int) {
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
				Content          any `json:"content"`
				ReasoningContent any `json:"reasoning_content"`
				Reasoning        any `json:"reasoning"`
				Thinking         any `json:"thinking"`
				ToolCalls        []struct {
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
		return nil, "", "", nil, 0, 0, 0
	}
	uPrompt, uComp, uCached := 0, 0, 0
	if chunk.Usage != nil {
		uPrompt, uComp = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		uCached = usageCached(chunk.Usage.CacheReadInputTokens, chunk.Usage.PromptTokensDetails)
	}
	if len(chunk.Choices) == 0 {
		return nil, "", "", nil, uPrompt, uComp, uCached
	}
	ch := chunk.Choices[0]
	if ch.Message != nil && (textFromContent(ch.Message.Content) != "" || len(ch.Message.ToolCalls) > 0) {
		msg := Message{Role: RoleAssistant, Content: textFromContent(ch.Message.Content)}
		for _, tc := range ch.Message.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
		}
		return &msg, "", "", nil, uPrompt, uComp, uCached
	}
	var tds []toolWireDelta
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
		tds = append(tds, toolWireDelta{index: tc.Index, id: tc.ID, name: tc.Function.Name, args: tc.Function.Arguments})
	}
	return nil, textFromContent(ch.Delta.Content), reasoningFromDelta(ch.Delta.Content, ch.Delta.ReasoningContent, ch.Delta.Reasoning, ch.Delta.Thinking), tds, uPrompt, uComp, uCached
}

func parseOpenAIJSON(raw []byte) (Message, error) {
	var parsed struct {
		Usage *struct {
			PromptTokens         int `json:"prompt_tokens"`
			CompletionTokens     int `json:"completion_tokens"`
			CacheReadInputTokens int `json:"cache_read_input_tokens"`
			PromptTokensDetails  *struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
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
	out := Message{Role: Role(m.Role), Content: stringify(m.Content), Truncated: isOutputLimitStop(parsed.Choices[0].FinishReason)}
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
	return textFromContent(v)
}

func textFromContent(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []any:
		var b strings.Builder
		for _, p := range t {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := m["type"].(string)
			if typ == "thinking" || typ == "reasoning" {
				continue
			}
			if s, _ := m["text"].(string); s != "" {
				b.WriteString(s)
			}
		}
		return b.String()
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func reasoningFromDelta(content, reasoningContent, reasoning, thinking any) string {
	var b strings.Builder
	b.WriteString(textFromContent(reasoningContent))
	b.WriteString(textFromContent(reasoning))
	b.WriteString(textFromContent(thinking))
	if arr, ok := content.([]any); ok {
		for _, p := range arr {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := m["type"].(string)
			if typ != "thinking" && typ != "reasoning" {
				continue
			}
			if s, _ := m["thinking"].(string); s != "" {
				b.WriteString(s)
			}
			if s, _ := m["text"].(string); s != "" {
				b.WriteString(s)
			}
		}
	}
	return b.String()
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
