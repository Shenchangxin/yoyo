package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestTaskSubagentSummaryOnly(t *testing.T) {
	dir := t.TempDir()
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "1", Name: "task", Arguments: `{"prompt":"Write hello.txt containing hello","isolate":false}`}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "2", Name: "write_file", Arguments: `{"path":"hello.txt","content":"hello"}`}}},
		{Role: RoleAssistant, Content: "child done"},
		{Role: RoleAssistant, Content: "parent done"},
	}}
	out, err := Run(context.Background(), RunRequest{
		User:      "delegate the write",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      DefaultLoop(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "parent done") {
		t.Fatalf("out=%s", out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("%q", b)
	}
}

func TestSubagentOwnJSONL(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	st := trace.NewStore(filepath.Join(home, "sessions"))
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "1", Name: "task", Arguments: `{"prompt":"Write hello.txt containing hello","isolate":false}`}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "2", Name: "write_file", Arguments: `{"path":"hello.txt","content":"hello"}`}}},
		{Role: RoleAssistant, Content: "child done"},
		{Role: RoleAssistant, Content: "parent done"},
	}}
	parentID := "parent"
	_, err := Run(context.Background(), RunRequest{
		SessionID: parentID,
		User:      "delegate the write",
		Workspace: dir,
		Home:      home,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      DefaultLoop(),
		Trace:     st,
	})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := st.Read(parentID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range parent {
		if ev.Type == trace.TypeToolCall {
			if name, _ := ev.Payload["name"].(string); name == "write_file" {
				t.Fatal("child tool_call leaked into parent JSONL")
			}
		}
	}
	found := false
	for _, ev := range parent {
		if ev.Type == trace.TypeSubagent {
			found = true
		}
	}
	if !found {
		t.Fatal("missing parent subagent summary")
	}
}

func TestNestedTaskForbidden(t *testing.T) {
	tools := &WorkspaceTools{Depth: 1}
	res := tools.Call("task", `{"prompt":"nope"}`)
	if res.Err == nil {
		t.Fatal("expected nested deny")
	}
	for _, j := range AllToolJSON(tools) {
		name, _ := j.Function["name"].(string)
		if name == "task" {
			t.Fatal("child schema still lists task")
		}
	}
}

func TestIsolateWorkspaceCopy(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst, cleanup, err := IsolateWorkspace(src)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	b, err := os.ReadFile(filepath.Join(dst, "keep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ok" {
		t.Fatalf("%q", b)
	}
}
