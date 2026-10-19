package webcontent

import (
	"strings"
	"testing"
)

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
