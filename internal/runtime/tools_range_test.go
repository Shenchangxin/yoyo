// 锁住越界读取会写明文件实际行数，空文件会说明是空的。非法正则按字面量搜索，而不是返回错误让模型原样重试。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadFileRangeNamesTheRealExtent 锁住 offset 超过文件末尾时写明总行数，空文件写明 0 行，两者都不是无输出的成功。
func TestReadFileRangeNamesTheRealExtent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}

	past := tools.Call("read_file", `{"path":"app.js","offset":270,"limit":20}`)
	if past.Err != nil {
		t.Fatalf("past-EOF read must not error: %v", past.Err)
	}
	if !strings.Contains(past.Content, "has 3 lines") || !strings.Contains(past.Content, "past the end") {
		t.Fatalf("past-EOF read should name the file length, got %q", past.Content)
	}

	mid := tools.Call("read_file", `{"path":"app.js","offset":2,"limit":1}`)
	if mid.Err != nil || !strings.Contains(mid.Content, "     2|b") || strings.Contains(mid.Content, "c") {
		t.Fatalf("in-range window changed: %+v", mid)
	}

	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	empty := tools.Call("read_file", `{"path":"empty.txt","offset":1,"limit":5}`)
	if empty.Err != nil || !strings.Contains(empty.Content, "is empty") {
		t.Fatalf("empty file should say so: %+v", empty)
	}
}

// TestGrepFallsBackToLiteralOnInvalidRegex 锁住 })(); 这种非法正则按字面量命中，合法正则不会带 literal text 说明。
func TestGrepFallsBackToLiteralOnInvalidRegex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("function a() {}\n})();\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}

	res := tools.Call("grep", `{"pattern":"})();"}`)
	if res.Err != nil {
		t.Fatalf("invalid regex must fall back to literal search, got error: %v", res.Err)
	}
	if !strings.Contains(res.Content, "literal text") || !strings.Contains(res.Content, "app.js") {
		t.Fatalf("expected literal-match note and hit, got %q", res.Content)
	}

	valid := tools.Call("grep", `{"pattern":"function a"}`)
	if valid.Err != nil || strings.Contains(valid.Content, "literal text") {
		t.Fatalf("valid regex must not get the fallback note: %+v", valid)
	}
}
