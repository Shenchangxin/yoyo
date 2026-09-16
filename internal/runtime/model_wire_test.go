package runtime

import (
	"encoding/json"
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
