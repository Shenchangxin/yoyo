// 本文件锁住 index.html 和目录根是同一份文档，以及多个标签里优先选正在看的那一页。
// 由 go test 调用。

package browser

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSameDocumentURLTreatsIndexAsRoot 锁住 / 与 index.html、以及子目录里的 index.html 是同一文档，myindex.html 和其他文件不是。
func TestSameDocumentURLTreatsIndexAsRoot(t *testing.T) {
	root := "http://127.0.0.1:9/"
	index := "http://127.0.0.1:9/index.html"
	if !sameDocumentURL(root, index) || !sameDocumentURL(index, root) {
		t.Fatal("index.html and / must be the same document")
	}
	if !sameDocumentURL("http://127.0.0.1:9/app/", "http://127.0.0.1:9/app/index.html") {
		t.Fatal("nested index.html must match its directory")
	}
	if sameDocumentURL("http://127.0.0.1:9/myindex.html", "http://127.0.0.1:9/index.html") {
		t.Fatal("only a path segment named index.html is the index")
	}
	if sameDocumentURL("http://127.0.0.1:9/other.html", "http://127.0.0.1:9/index.html") {
		t.Fatal("different documents must still differ")
	}
}

// TestShouldReloadLocalSkipsWhenOnlyIndexSpellingDiffers 锁住活着的页面只是 index 写法不同时不重新加载。
func TestShouldReloadLocalSkipsWhenOnlyIndexSpellingDiffers(t *testing.T) {
	live := "http://127.0.0.1:9/"
	want := "http://127.0.0.1:9/index.html"
	if shouldReloadLocal(true, live, want) {
		t.Fatal("a live page that is the same index must not reload on click or screenshot")
	}
}

// TestPageWSPrefersServedIndexOverOtherTabs 锁住想要 index.html 时，会选中被报成 / 的那个 page，而不是另一个标签。
func TestPageWSPrefersServedIndexOverOtherTabs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/list" && r.URL.Path != "/json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]cdpTarget{
			{Type: "page", URL: "http://127.0.0.1:9/stale.html", WS: "ws://127.0.0.1:9/devtools/page/stale"},
			{Type: "page", URL: "http://127.0.0.1:9/", WS: "ws://127.0.0.1:9/devtools/page/game"},
		})
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")
	got, err := pageWSPrefer(addr, "http://127.0.0.1:9/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/devtools/page/game") {
		t.Fatalf("expected the served index, got %s", got)
	}
}

// TestNormalizeDocPath 锁住根和 index.html 收成空路径，子目录的 index 收成目录，百分号编码会解开。
func TestNormalizeDocPath(t *testing.T) {
	cases := map[string]string{
		"/":                 "",
		"":                  "",
		"/index.html":       "",
		"/app/":             "/app",
		"/app/index.html":   "/app",
		"/app/index.htm":    "/app",
		"/myindex.html":     "/myindex.html",
		"/app/game%20a.htm": "/app/game a.htm",
	}
	for in, want := range cases {
		if got := normalizeDocPath(in); got != want {
			t.Errorf("normalizeDocPath(%q)=%q want %q", in, got, want)
		}
	}
}
