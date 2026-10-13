package browser

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Chrome reports a served index.html as "/" after the file server redirects
// it. Those must be the same document, or every click reloaded the game.
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

func TestShouldReloadLocalSkipsWhenOnlyIndexSpellingDiffers(t *testing.T) {
	live := "http://127.0.0.1:9/"
	want := "http://127.0.0.1:9/index.html"
	if shouldReloadLocal(true, live, want) {
		t.Fatal("a live page that is the same index must not reload on click or screenshot")
	}
}

// With several tabs open, the preferred target must be the document we
// navigated to even when Chrome reports it without "index.html".
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
