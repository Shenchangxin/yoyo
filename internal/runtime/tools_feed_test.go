package runtime

import (
	"net/url"
	"strings"
	"testing"
)

func TestFormatArxivAtomKeepsIDs(t *testing.T) {
	body := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <opensearch:totalResults xmlns:opensearch="http://a9.com/-/spec/opensearch/1.1/">222</opensearch:totalResults>
  <entry>
    <id>http://arxiv.org/abs/2609.29154v1</id>
    <title>A Wrong Turn Does Not Ruin the Journey: Deviation-Guided Skill Self-Evolution</title>
    <published>2026-09-24T07:31:00Z</published>
    <summary>Large language model agents increasingly rely on natural-language skills.</summary>
    <author><name>Yichun Feng</name></author>
    <author><name>Jiawei Wang</name></author>
    <link rel="alternate" href="http://arxiv.org/abs/2609.29154v1"/>
    <link title="pdf" href="http://arxiv.org/pdf/2609.29154v1" rel="related" type="application/pdf"/>
  </entry>
</feed>`
	got, ok := formatFeed(body)
	if !ok {
		t.Fatal("expected feed")
	}
	if !strings.Contains(got, "2609.29154v1") || !strings.Contains(got, "Skill Self-Evolution") {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(got, "totalResults=222") || !strings.Contains(got, "Yichun Feng") {
		t.Fatalf("%s", got)
	}
	stripped := stripTags(body)
	if strings.Contains(stripped, "id:") {
		t.Fatal("stripTags should not be the feed path")
	}
}

func TestRetryHTTPOn406(t *testing.T) {
	u, _ := url.Parse("https://export.arxiv.org/api/query?q=x")
	alt := retryHTTPOn406(u, 406)
	if alt == nil || alt.Scheme != "http" || alt.Host != u.Host {
		t.Fatalf("%v", alt)
	}
	if retryHTTPOn406(u, 200) != nil {
		t.Fatal("200")
	}
	httpU, _ := url.Parse("http://export.arxiv.org/api/query")
	if retryHTTPOn406(httpU, 406) != nil {
		t.Fatal("already http")
	}
}

func TestIsXMLFeed(t *testing.T) {
	if !isXMLFeed("application/atom+xml; charset=utf-8", "") {
		t.Fatal("ct")
	}
	if isXMLFeed("application/json", "<feed>") {
		t.Fatal("json")
	}
	if !isXMLFeed("text/plain", "<?xml version=\"1.0\"?>") {
		t.Fatal("body")
	}
}
