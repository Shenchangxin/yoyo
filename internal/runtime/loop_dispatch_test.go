// 锁住工具批的并行和 exclusive：多个 task 同时跑，shell 失败不取消同批写入。
// 也锁住格式注释不被误拦，以及没有变化的写入不发 FileChange。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/safeguard"
)

// TestWriteFileWithFormatCommentIsNotDenied 确认正文里的 format 注释不拦截 write_file、apply_patch 和 str_replace，命令 format c: 仍被拒绝。
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

// TestStrReplaceMissIncludesFileHint 确认 old_str 找不到时，错误里带有文件里实际存在的函数文本。
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

// TestWriteFileEmitsFileChange 确认新建文件的结果带上路径 n.txt。
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

// TestWriteFileUnchangedSkipsFileChange 确认内容相同的再次写入没有 FileChange，正文含 unchanged。
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

// TestReplaceUnchangedSkipsFileChange 确认 old_str 与 new_str 相同的替换没有 FileChange，正文含 unchanged。
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
