package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWireMessagesNestsFunction(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, Content: "looking", ToolCalls: []ToolCall{
			{ID: "c1", Name: "read_file", Arguments: `{"path":"a.go"}`},
			{ID: "c2", Name: "grep", Arguments: ""},
		}},
		{Role: RoleTool, ToolCallID: "c1", Name: "read_file", Content: "ok"},
	}
	raw, err := json.Marshal(wireMessages(msgs))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, `"name":"read_file"`) && !strings.Contains(s, `"function"`) {
		t.Fatalf("flat tool_calls leaked: %s", s)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	asst := rows[2]
	calls, _ := asst["tool_calls"].([]any)
	if len(calls) != 2 {
		t.Fatalf("calls %+v", calls)
	}
	c0, _ := calls[0].(map[string]any)
	fn, _ := c0["function"].(map[string]any)
	if c0["type"] != "function" || fn["name"] != "read_file" {
		t.Fatalf("wire call %+v", c0)
	}
	if fn["arguments"] != `{"path":"a.go"}` {
		t.Fatalf("args %v", fn["arguments"])
	}
	c1, _ := calls[1].(map[string]any)
	fn1, _ := c1["function"].(map[string]any)
	if fn1["arguments"] != "{}" {
		t.Fatalf("empty args should be {}: %v", fn1["arguments"])
	}
	if _, ok := c0["name"]; ok {
		t.Fatalf("flat name on wire: %+v", c0)
	}
}

func TestDisableThinkingPayload(t *testing.T) {
	p := map[string]any{"model": "x"}
	disableThinking(p)
	th, _ := p["thinking"].(map[string]any)
	if th["type"] != "disabled" || p["enable_thinking"] != false {
		t.Fatalf("%+v", p)
	}
	if !thinkingRejected("unknown field: enable_thinking") {
		t.Fatal("rejected")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestShowThinkingOmitsDisableFields(t *testing.T) {
	var got map[string]any
	c := &OpenAIClient{
		BaseURL: "http://example.invalid/v1",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &got)
			return &http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewReader(nil)), Header: make(http.Header)}, nil
		})},
	}
	_, _ = c.doChat(context.Background(), ChatRequest{Model: "x", ShowThinking: true}, false, nil)
	if _, ok := got["enable_thinking"]; ok {
		t.Fatalf("thinking disable leaked: %+v", got)
	}
	if _, ok := got["thinking"]; ok {
		t.Fatalf("thinking object leaked: %+v", got)
	}
}

func TestApplyStreamChunkReasoning(t *testing.T) {
	tools := map[int]*streamAcc{}
	max := -1
	_, text, reason, _, _, _ := applyStreamChunk(
		`{"choices":[{"delta":{"reasoning_content":"think ","content":"hi"}}]}`,
		tools, &max,
	)
	if text != "hi" || reason != "think " {
		t.Fatalf("text=%q reason=%q", text, reason)
	}
	_, _, reason, _, _, _ = applyStreamChunk(
		`{"choices":[{"delta":{"content":[{"type":"thinking","text":"why"},{"type":"text","text":"ok"}]}}]}`,
		tools, &max,
	)
	if reason != "why" {
		t.Fatalf("parts reason %q", reason)
	}
}

func TestClassifyInvalidToolCalls(t *testing.T) {
	err := fmtError(`openai: 500 Internal Server Error: {"code":500,"msg":"Model call failed. Please try again later: invalid_parameter_error: \u003c400\u003e InternalError.Algo.InvalidParameter: Field required: input.messages.4.tool_calls.0.function","error_key":"error.all_channel_models_failed"}`)
	info := ClassifyError(err)
	if info.Kind != ErrKindInvalid {
		t.Fatalf("kind %s title %s detail %s", info.Kind, info.Title, info.Detail)
	}
	if !strings.Contains(info.Detail, "tool_calls") {
		t.Fatalf("detail %s", info.Detail)
	}
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
