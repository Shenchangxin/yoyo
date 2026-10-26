// 本文件锁住 DuckDuckGo 链接解码，以及验证码页和过薄正文的判断。由 go test 调用。

package webcontent

import (
	"strings"
	"testing"
)

// TestParseDuckDuckGoHTML 锁住 uddg 参数会解成真实 https 地址，第二条直接使用 href，摘要里留有 abstract。
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

// TestLooksAntibotAndThin 锁住 Cloudflare 标题会被 LooksAntibot 命中，短正文配很长原文算薄，足够长的正文不算薄。
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
