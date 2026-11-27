// 锁住 fileChangeOf 把行 diff 的结果正确填进 FileChange。算法本身的测试在 internal/textdiff。

package runtime

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestFileChangeOfCreatedFile 锁住新建文件的 Created 为真、有新增、没有删除，补丁含加号行。
func TestFileChangeOfCreatedFile(t *testing.T) {
	ch := fileChangeOf("web/app.ts", nil, []byte("export const n = 1\n"), true)
	if !ch.Created || ch.Added < 1 || ch.Removed != 0 {
		t.Fatalf("%+v", ch)
	}
	if !strings.Contains(ch.Patch, "+export const n = 1") {
		t.Fatalf("patch %s", ch.Patch)
	}
}

// TestFileChangeOfSlashesPath 锁住路径按 filepath.ToSlash 规范化后，同时写进 Paths 和补丁头。
// Windows 上反斜杠会变成斜杠；其他平台上反斜杠是合法文件名字符，ToSlash 不转换，期望值按同一规则算。
func TestFileChangeOfSlashesPath(t *testing.T) {
	ch := fileChangeOf("web\\app.ts", []byte("a\n"), []byte("b\n"), false)
	want := filepath.ToSlash("web\\app.ts")
	if len(ch.Paths) != 1 || ch.Paths[0] != want {
		t.Fatalf("paths %v, want %q", ch.Paths, want)
	}
	if !strings.Contains(ch.Patch, "--- a/"+want) {
		t.Fatalf("patch %s", ch.Patch)
	}
}
