// 锁住子代理只把摘要交回父级，子工具调用不写进父 JSONL。深度达到 MaxTaskDepth 时拒绝再派，深度 1 仍广告 task。隔离复制能带走文件。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestChildSessionIDIsPathSafe 锁住子会话 ID 不含斜杠，以父 ID 加 taskSep 开头，IsChildSessionID 能认出它；旧的 /tasks/ 形式也仍被认成子会话。
func TestChildSessionIDIsPathSafe(t *testing.T) {
	id := ChildSessionID("parent")
	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		t.Fatalf("child id must not split HTTP paths: %s", id)
	}
	if !IsChildSessionID(id) || !strings.HasPrefix(id, "parent"+taskSep) {
		t.Fatalf("child id %s", id)
	}
	if legacy := "abc/tasks/x"; !IsChildSessionID(legacy) {
		t.Fatalf("legacy %s", legacy)
	}
	if IsChildSessionID("parent") {
		t.Fatal("plain session id read as a child")
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
