package webcontent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseFeedReadsRSSAndAtom(t *testing.T) {
	rss := `<?xml version="1.0"?><rss xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel>
<item><title><![CDATA[智能体周报]]></title><link>https://example.com/rss</link><description><p>摘要</p></description></item>
<item><title>无链接</title><description>跳过</description></item>
</channel></rss>`
	hits, err := ParseFeed(rss, 5)
	if err != nil || len(hits) != 1 || hits[0].URL != "https://example.com/rss" || hits[0].Snippet != "摘要" || hits[0].Title != "智能体周报" {
		t.Fatalf("%v %#v", err, hits)
	}
	atom := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><entry>
<title>Agent paper</title>
<id>http://arxiv.org/abs/2401.00001v1</id>
<link rel="alternate" href="http://arxiv.org/abs/2401.00001v1"/>
<summary>论文</summary>
</entry></feed>`
	papers, err := ParseFeed(atom, 5)
	if err != nil || len(papers) != 1 || papers[0].URL != "https://arxiv.org/abs/2401.00001v1" || papers[0].Snippet != "论文" {
		t.Fatalf("%v %#v", err, papers)
	}
}

func TestHackerNewsAndGitHubQueries(t *testing.T) {
	var hnQuery, ghQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "search_by_date"):
			hnQuery = r.URL.Query().Get("query")
			_, _ = w.Write([]byte(`{"hits":[{"title":"Ask AI","url":null,"story_text":"正文","objectID":"42"},{"title":"外链","url":"https://example.com/ai","objectID":"7"}]}`))
		default:
			ghQuery = r.URL.Query().Get("q")
			_, _ = w.Write([]byte(`{"items":[{"full_name":"acme/agent","html_url":"https://github.com/acme/agent","description":"AI agent"}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	prevHN, prevGH := HNSearchBase, GitHubAPIBase
	HNSearchBase = srv.URL + "/search_by_date"
	GitHubAPIBase = srv.URL
	t.Cleanup(func() {
		HNSearchBase = prevHN
		GitHubAPIBase = prevGH
	})
	client := srv.Client()
	hn, err := HackerNews(context.Background(), client, "AI agent", 5)
	if err != nil || hnQuery != "AI agent" || len(hn) != 2 || hn[0].URL != "https://news.ycombinator.com/item?id=42" || hn[1].URL != "https://example.com/ai" {
		t.Fatalf("query %s err %v hits %#v", hnQuery, err, hn)
	}
	gh, err := GitHubUpdated(context.Background(), client, "AI agent", 5)
	if err != nil || !strings.Contains(ghQuery, "pushed:>") || !strings.Contains(ghQuery, "in:name,description") || gh[0].URL != "https://github.com/acme/agent" {
		t.Fatalf("q %s err %v hits %#v", ghQuery, err, gh)
	}
}

func TestDevToPicksTheTopicTag(t *testing.T) {
	var tag string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tag = r.URL.Query().Get("tag")
		_, _ = w.Write([]byte(`[{"title":"LLM notes","url":"https://dev.to/x/llm","description":"notes"}]`))
	}))
	t.Cleanup(srv.Close)
	prev := DevToArticlesBase
	DevToArticlesBase = srv.URL + "/articles"
	t.Cleanup(func() { DevToArticlesBase = prev })
	hits, err := DevToHot(context.Background(), srv.Client(), "大模型", 5)
	if err != nil || tag != "llm" || len(hits) != 1 {
		t.Fatalf("tag %s err %v %#v", tag, err, hits)
	}
}

func TestLobstersFallsBackToTheShortLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"title":"Agents","url":"http://insecure.example/a","short_id_url":"https://lobste.rs/s/abc","description":"ai"}]`))
	}))
	t.Cleanup(srv.Close)
	prev := LobstersAIBase
	LobstersAIBase = srv.URL
	t.Cleanup(func() { LobstersAIBase = prev })
	hits, err := LobstersAI(context.Background(), srv.Client(), 5)
	if err != nil || len(hits) != 1 || hits[0].URL != "https://lobste.rs/s/abc" {
		t.Fatalf("%v %#v", err, hits)
	}
}
