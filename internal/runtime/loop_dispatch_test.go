package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/safeguard"
)

func TestExclusiveShellFailureDoesNotAbortSiblingWrite(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir, Spill: NewSpill(filepath.Join(dir, "spill"))}
	req := RunRequest{SessionID: "s", Tools: tools, Loop: DefaultLoop()}
	out := dispatchTools(context.Background(), req, []ToolCall{
		{ID: "s1", Name: "shell", Arguments: `{"command":"not-a-real-yoyo-cmd-xyz"}`},
		{ID: "w1", Name: "write_file", Arguments: `{"path":"ok.txt","content":"hello"}`},
	}, "s:r1", false)
	if len(out) != 2 {
		t.Fatalf("len %d", len(out))
	}
	if !strings.Contains(out[0].Content, "ERROR") {
		t.Fatalf("expected shell error, got %q", out[0].Content)
	}
	if strings.Contains(out[1].Content, "aborted") || strings.Contains(out[1].Content, "ERROR") {
		t.Fatalf("write aborted after exclusive shell failure: %s", out[1].Content)
	}
	b, err := os.ReadFile(filepath.Join(dir, "ok.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("wrote %q", b)
	}
}

func TestWriteFileWithFormatCommentIsNotDenied(t *testing.T) {
	args := `{"path":"api/export.go","content":"// export format is JSON\npackage api\n"}`
	if deny, reason := safeguard.PreTool("write_file", args); deny {
		t.Fatalf("write_file denied: %s", reason)
	}
	if deny, _ := safeguard.PreTool("apply_patch", args); deny {
		t.Fatal("apply_patch denied")
	}
	if deny, _ := safeguard.PreTool("str_replace", args); deny {
		t.Fatal("str_replace denied")
	}
	if deny, reason := safeguard.PreTool("shell", `{"command":"format c:"}`); !deny {
		t.Fatalf("format c: must still deny, got deny=%v %s", deny, reason)
	}
}

func TestStrReplaceMissIncludesFileHint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package demo\n\nfunc Hello() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("str_replace", `{"path":"a.go","old_str":"func Missing() {}","new_str":"func Hello() {}"}`)
	if res.Err == nil {
		t.Fatal("expected miss")
	}
	msg := res.Err.Error()
	if !strings.Contains(msg, "old_str not found") || !strings.Contains(msg, "func Hello") {
		t.Fatalf("hint missing: %s", msg)
	}
}

func TestWriteFileEmitsFileChange(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("write_file", `{"path":"n.txt","content":"x"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.FileChange == nil || len(res.FileChange.Paths) != 1 || res.FileChange.Paths[0] != "n.txt" {
		t.Fatalf("file change %+v", res.FileChange)
	}
}

func TestWriteFileUnchangedSkipsFileChange(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	first := tools.Call("write_file", `{"path":"n.txt","content":"x"}`)
	if first.Err != nil {
		t.Fatal(first.Err)
	}
	again := tools.Call("write_file", `{"path":"n.txt","content":"x"}`)
	if again.Err != nil {
		t.Fatal(again.Err)
	}
	if again.FileChange != nil {
		t.Fatalf("identical rewrite still emitted change %+v", again.FileChange)
	}
	if !strings.Contains(again.Content, "unchanged") {
		t.Fatalf("content %q", again.Content)
	}
}

func TestReplaceUnchangedSkipsFileChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("str_replace", `{"path":"a.go","old_str":"package demo","new_str":"package demo"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.FileChange != nil {
		t.Fatalf("noop replace still emitted change %+v", res.FileChange)
	}
	if !strings.Contains(res.Content, "unchanged") {
		t.Fatalf("content %q", res.Content)
	}
}
