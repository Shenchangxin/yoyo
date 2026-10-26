// 本文件锁住 Validate 会拒绝前缀不是 http 的 URL。
// 不覆盖哈希截断长度。由 go test 调用。

package cite

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateRequiresURL 先用来源通过校验，再把 URL 写成 not-a-url，要求 Validate 返回错误。
func TestValidateRequiresURL(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "citations.json")
	src := HashExcerpt("https://example.com/a", "excerpt about widgets")
	if err := Write(p, []Source{src}); err != nil {
		t.Fatal(err)
	}
	if err := Validate("widgets https://example.com/a", p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`[{"url":"not-a-url","excerpt":"x","hash":"ab"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate("x", p); err == nil {
		t.Fatal("bad url")
	}
}
