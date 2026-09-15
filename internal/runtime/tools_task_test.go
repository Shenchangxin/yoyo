package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
