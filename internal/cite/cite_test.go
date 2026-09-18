package cite

import (
	"os"
	"path/filepath"
	"testing"
)

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
