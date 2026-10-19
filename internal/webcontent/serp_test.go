package webcontent

import (
	"strings"
	"testing"
)

func TestParseDuckDuckGoHTML(t *testing.T) {
	page := `
<html><body>
<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fpaper">A Paper Title</a>
<a class="result__snippet">The abstract lives here.</a>
<a class="result__a" href="https://example.org/two">Second Hit</a>
<a class="result__snippet">More text.</a>
</body></html>`
	hits := ParseDuckDuckGoHTML(page)
	if len(hits) < 2 {
		t.Fatalf("hits=%d %+v", len(hits), hits)
	}
	if hits[0].URL != "https://example.com/paper" || hits[0].Title != "A Paper Title" {
		t.Fatalf("%+v", hits[0])
	}
	if !strings.Contains(hits[0].Snippet, "abstract") {
		t.Fatalf("snippet %+v", hits[0])
	}
	if hits[1].URL != "https://example.org/two" {
		t.Fatalf("%+v", hits[1])
	}
}

func TestLooksAntibotAndThin(t *testing.T) {
	if !LooksAntibot("Attention Required! | Cloudflare ray id 123") {
		t.Fatal("cloudflare")
	}
	if !ThinExtract("short", "<html>"+strings.Repeat("x", 5000)+"</html>") {
		t.Fatal("thin")
	}
	if ThinExtract(strings.Repeat("word ", 80), "<p>ok</p>") {
		t.Fatal("enough text")
	}
}
