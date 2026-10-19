package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

func TestWebSearchUsesSearchAPI(t *testing.T) {
	tools := &WorkspaceTools{SearchAPI: func(q string) (string, error) {
		if q != "hello" {
			t.Fatalf("q=%s", q)
		}
		return "1. Hit\n   https://example.com/paper\n   a longer snippet so Useful accepts it", nil
	}}
	res := tools.webSearch("hello")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "search: hello") || !strings.Contains(res.Content, "https://example.com") {
		t.Fatalf("%s", res.Content)
	}
}

func TestWebSearchJinaFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"title": "JinaHit", "url": "https://jina.example", "description": "ok"}},
		})
	}))
	t.Cleanup(srv.Close)
	prevJina, prevDDG, prevLite := builtinJinaSearch, builtinDDGHTML, builtinDDGLite
	builtinJinaSearch = srv.URL + "/"
	builtinDDGHTML = srv.URL + "/no-ddg"
	builtinDDGLite = srv.URL + "/no-ddg"
	t.Cleanup(func() {
		builtinJinaSearch, builtinDDGHTML, builtinDDGLite = prevJina, prevDDG, prevLite
	})
	res := (&WorkspaceTools{}).webSearch("frameworks")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "search (jina):") || !strings.Contains(res.Content, "JinaHit") {
		t.Fatalf("%s", res.Content)
	}
}

func TestWebSearchEmpty(t *testing.T) {
	res := (&WorkspaceTools{}).webSearch("  ")
	if res.Err == nil {
		t.Fatal("expected empty query error")
	}
}

func TestShouldJinaReaderSkipsLoopback(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:8080/x")
	if shouldJinaReader(u, 403, "text/html", "nope", "nope") {
		t.Fatal("loopback")
	}
	pub, _ := url.Parse("https://example.com/a")
	if !shouldJinaReader(pub, 403, "text/html", "nope", "nope") {
		t.Fatal("403 public")
	}
	if shouldJinaReader(pub, 200, "application/json", `{"a":1}`, `{"a":1}`) {
		t.Fatal("json")
	}
}

func TestWebFetchJinaFallback(t *testing.T) {
	prevSkip := jinaSkipLoopback
	jinaSkipLoopback = false
	t.Cleanup(func() { jinaSkipLoopback = prevSkip })
	var sawJina bool
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(403)
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	t.Cleanup(origin.Close)
	jina := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawJina = true
		_, _ = w.Write([]byte("# Article\n\nEnough markdown from jina reader for the agent to use."))
	}))
	t.Cleanup(jina.Close)
	prev := builtinJinaReader
	builtinJinaReader = jina.URL + "/"
	t.Cleanup(func() { builtinJinaReader = prev })
	res := (&WorkspaceTools{}).webFetch(origin.URL)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !sawJina || !strings.Contains(res.Content, "(jina)") || !strings.Contains(res.Content, "Article") {
		t.Fatalf("sawJina=%v %s", sawJina, res.Content)
	}
}

func TestParseDuckDuckGoUseful(t *testing.T) {
	hits := webcontent.ParseDuckDuckGoHTML(`<a class="result__a" href="https://example.com/p">Paper</a><a class="result__snippet">abs</a>`)
	if len(hits) != 1 || hits[0].Title != "Paper" {
		t.Fatalf("%+v", hits)
	}
}
