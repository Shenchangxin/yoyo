// 本文件锁住命中格式化后能被 Useful 接受，空文本和没有 http 的 html 不被接受。由 go test 调用。

package webcontent

import (
	"strings"
	"testing"
)

// TestFormatHitsAndUseful 锁住两条命中会带上编号、标题、链接和摘要，空字符串和没有 http 的 html 不是有用结果。
func TestFormatHitsAndUseful(t *testing.T) {
	out := FormatHits([]Hit{
		{Title: "One", URL: "https://example.com/a", Snippet: "alpha"},
		{Title: "Two", URL: "https://example.com/b", Snippet: "beta"},
	})
	if !Useful(out) {
		t.Fatal("expected useful hits")
	}
	for _, p := range []string{"1. One", "https://example.com/a", "alpha", "2. Two"} {
		if !strings.Contains(out, p) {
			t.Fatalf("missing %q in %s", p, out)
		}
	}
	if Useful("") || Useful("<html></html>") {
		t.Fatal("empty/html must not be useful")
	}
}
