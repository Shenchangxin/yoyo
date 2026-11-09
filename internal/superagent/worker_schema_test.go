package superagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// TestWorkerSchemasCoverTools 锁住五个 orchestrator 工具各有一份 schema，没有多余的；每份都是不许多余字段的对象，必填字段都在属性里，并且能编成 JSON。
func TestWorkerSchemasCoverTools(t *testing.T) {
	ex := &Executor{Engine: NewEngine(autoPolicy())}
	schemas := WorkerToolSchemas()
	tools := ex.orchestratorTools("t1", false)
	if len(tools) != len(schemas) {
		t.Fatalf("%d tools, %d schemas", len(tools), len(schemas))
	}
	for _, tool := range tools {
		schema, ok := schemas[tool.Name].(map[string]any)
		if !ok {
			t.Fatalf("no schema for %s", tool.Name)
		}
		if tool.Parameters != nil {
			t.Fatalf("%s grew dsh parameters; the schema is only for MCP", tool.Name)
		}
		if schema["type"] != "object" || schema["additionalProperties"] != false {
			t.Fatalf("%s schema = %v", tool.Name, schema)
		}
		props := schema["properties"].(map[string]any)
		for _, name := range schema["required"].([]string) {
			if _, ok := props[name]; !ok {
				t.Fatalf("%s requires %s but does not describe it", tool.Name, name)
			}
		}
		for name, prop := range props {
			field := prop.(map[string]any)
			if field["type"] != "string" || field["description"] == "" {
				t.Fatalf("%s.%s = %v", tool.Name, name, field)
			}
		}
		if _, err := json.Marshal(schema); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
	}
}

// TestReportStatesMatchEngine 锁住 schema 里的每个 state 引擎都收，枚举以外的值引擎拒绝。
func TestReportStatesMatchEngine(t *testing.T) {
	for _, state := range ReportStates {
		eng := NewEngine(autoPolicy())
		id := accept(t, eng, "m1")
		if err := eng.Report(id, Report{State: state, Summary: "s"}); err != nil {
			t.Fatalf("%s: %v", state, err)
		}
	}
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.Report(id, Report{State: "done", Summary: "s"}); err == nil || err.Error() != "bad_report" {
		t.Fatalf("done = %v", err)
	}
}

// TestReplyPurposesMatchEngine 锁住 schema 里的每个 purpose 都过得了引擎的用途检查，枚举以外的值引擎拒绝。
func TestReplyPurposesMatchEngine(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	for _, purpose := range ReplyPurposes {
		if _, err := eng.Reply(id, purpose, "m1", "hi"); err != nil && err.Error() == "bad_purpose" {
			t.Fatalf("%s rejected as a purpose", purpose)
		}
	}
	if _, err := eng.Reply(id, "note", "m1", "hi"); err == nil || err.Error() != "bad_purpose" {
		t.Fatalf("note = %v", err)
	}
}

// TestSchemaArgumentsParse 锁住按 schema 填满全部字段的参数，五个工具都能解开：不会因为字段类型不对而回参数错误。
func TestSchemaArgumentsParse(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	ex := &Executor{Engine: eng}
	schemas := WorkerToolSchemas()
	for _, tool := range ex.orchestratorTools(id, false) {
		props := schemas[tool.Name].(map[string]any)["properties"].(map[string]any)
		args := map[string]any{}
		for name, prop := range props {
			if enum, ok := prop.(map[string]any)["enum"].([]string); ok {
				args[name] = enum[0]
				continue
			}
			args[name] = "x"
		}
		data, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		res, err := tool.Run(context.Background(), dsh.ToolCall{ID: "c1", Name: tool.Name, Arguments: string(data)})
		if err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		if strings.Contains(res.Text, "cannot unmarshal") || strings.Contains(res.Text, "invalid character") {
			t.Fatalf("%s did not parse %s: %+v", tool.Name, data, res)
		}
	}
}
