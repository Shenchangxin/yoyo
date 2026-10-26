// unix find -name 映成 glob。Windows 的 FIND 文本搜索不改写。本文件不执行命令。

package runtime

import "testing"

// TestParseUnixFindGlob 确认 find . -name 映成 *.pdf，find src -type f -name 映成 src/**/*.go，文本搜索不改写，只有 -type f 算 misuse。
func TestParseUnixFindGlob(t *testing.T) {
	pat, ok := parseUnixFindGlob(`find . -name "*.pdf"`)
	if !ok || pat != "*.pdf" {
		t.Fatalf("got %q %v", pat, ok)
	}
	pat, ok = parseUnixFindGlob(`find src -type f -name '*.go'`)
	if !ok || pat != "src/**/*.go" {
		t.Fatalf("got %q %v", pat, ok)
	}
	if _, ok := parseUnixFindGlob(`find "needle" notes.txt`); ok {
		t.Fatal("windows FIND text search must not rewrite")
	}
	if !unixFindMisuse(`find . -type f`) {
		t.Fatal("expected unix find misuse")
	}
	if unixFindMisuse(`find "needle" notes.txt`) {
		t.Fatal("text search is valid FIND.exe")
	}
}
