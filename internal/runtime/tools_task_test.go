// 锁住子代理只把摘要交回父级，子工具调用不写进父 JSONL。深度达到 MaxTaskDepth 时拒绝再派，深度 1 仍广告 task。隔离复制能带走文件。

package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestTaskSubagentSummaryOnly 锁住父级最终输出含 parent done，子级写入的 hello.txt 在工作区里。
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

// TestSubagentOwnJSONL 锁住父轨迹里没有子级的 write_file 调用，但有带 child 的子代理 start 和 complete 事件。
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
	sawStart := false
	for _, ev := range parent {
		if ev.Type != trace.TypeSubagent {
			continue
		}
		found = true
		phase, _ := ev.Payload["phase"].(string)
		if phase == "start" {
			sawStart = true
		}
		if _, ok := ev.Payload["child"].(string); !ok {
			t.Fatal("subagent event missing child")
		}
	}
	if !found {
		t.Fatal("missing parent subagent summary")
	}
	if !sawStart {
		t.Fatal("missing subagent start event")
	}
}

// TestChildSessionIDIsPathSafe 锁住子会话 ID 不含斜杠，IsChildSessionID 和 ParentOfSession 能来回解析，旧的 /tasks/ 形式仍能认出父 ID。
func TestChildSessionIDIsPathSafe(t *testing.T) {
	id := ChildSessionID("parent")
	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		t.Fatalf("child id must not split HTTP paths: %s", id)
	}
	if !IsChildSessionID(id) || ParentOfSession(id) != "parent" {
		t.Fatalf("%s parent=%s", id, ParentOfSession(id))
	}
	legacy := "abc/tasks/x"
	if !IsChildSessionID(legacy) || ParentOfSession(legacy) != "abc" {
		t.Fatalf("legacy %s", legacy)
	}
}

// TestSubagentLiveEventsUseChildSession 锁住子级工具的实时事件使用子会话 ID，而不是父会话 ID。
func TestSubagentLiveEventsUseChildSession(t *testing.T) {
	dir := t.TempDir()
	var live []trace.Event
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "1", Name: "task", Arguments: `{"prompt":"Write hello.txt containing hello","isolate":false}`}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "2", Name: "write_file", Arguments: `{"path":"hello.txt","content":"hello"}`}}},
		{Role: RoleAssistant, Content: "child done"},
		{Role: RoleAssistant, Content: "parent done"},
	}}
	_, err := Run(context.Background(), RunRequest{
		SessionID: "parent",
		User:      "delegate",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      DefaultLoop(),
		OnEvent: func(ev trace.Event) {
			live = append(live, ev)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	childLive := 0
	for _, ev := range live {
		if ev.Type == trace.TypeToolCall {
			name, _ := ev.Payload["name"].(string)
			if name == "write_file" && !IsChildSessionID(ev.SessionID) {
				t.Fatal("child tool_call live event used parent session")
			}
		}
		if IsChildSessionID(ev.SessionID) {
			childLive++
		}
	}
	if childLive == 0 {
		t.Fatal("expected child live events on Hub")
	}
}

// TestReadOnlySubagentDeniesWrites 锁住 ReadOnly 子代理的 write_file 被拒绝。
func TestReadOnlySubagentDeniesWrites(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir, ReadOnly: true}
	if res := tools.Call("read_file", `{"path":"a.txt"}`); res.Err != nil {
		t.Fatal(res.Err)
	}
	res := tools.Call("write_file", `{"path":"b.txt","content":"x"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "read-only") {
		t.Fatalf("%+v", res)
	}
	res = tools.Call("shell", `{"command":"echo x"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "read-only") {
		t.Fatalf("shell %+v", res)
	}
	for _, j := range AllToolJSON(tools) {
		name, _ := j.Function["name"].(string)
		if name == "shell" || name == "write_file" {
			t.Fatalf("read-only schema still lists %s", name)
		}
	}
}

// TestNestedTaskForbidden 锁住 Depth 等于 MaxTaskDepth 时 task 失败，且 schema 里不再列出 task。
func TestNestedTaskForbidden(t *testing.T) {
	tools := &WorkspaceTools{Depth: MaxTaskDepth}
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

// TestTaskListedAtDepthOne 锁住 Depth 为 1 时 schema 仍包含 task。
func TestTaskListedAtDepthOne(t *testing.T) {
	tools := &WorkspaceTools{Depth: 1}
	found := false
	for _, j := range AllToolJSON(tools) {
		name, _ := j.Function["name"].(string)
		if name == "task" {
			found = true
		}
	}
	if !found {
		t.Fatal("depth 1 must still advertise task (nesting cap is 2)")
	}
}

// TestIsolateWorkspaceCopy 锁住 IsolateWorkspace 的目标里能读到源文件。
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

// TestPersistWorktreeCopy 锁住 PersistWorktree 能复制文件，目标已存在时再次调用成功，RemoveWorktree 会删掉目标。
func TestPersistWorktreeCopy(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "wt")
	if err := PersistWorktree(src, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dest, "keep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ok" {
		t.Fatalf("%q", b)
	}
	if err := PersistWorktree(src, dest); err != nil {
		t.Fatal(err)
	}
	RemoveWorktree(src, dest)
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("worktree still present: %v", err)
	}
}
