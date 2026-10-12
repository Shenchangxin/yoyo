package connection

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuerySearchVendors(t *testing.T) {
	var gotMethod, gotAuth, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		if gotAuth == "" {
			gotAuth = r.Header.Get("X-API-KEY")
			if gotAuth == "" {
				gotAuth = r.Header.Get("X-Subscription-Token")
			}
		}
		gotPath = r.URL.String()
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"title": "Insight", "url": "https://example.com", "content": "note"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	out, err := QuerySearch(ctx, Connection{Protocol: "tavily", Endpoint: srv.URL}, "tv-key", "护肤成分")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotAuth, "tv-key") || !strings.Contains(gotBody, "护肤成分") {
		t.Fatalf("tavily method=%s auth=%s body=%s", gotMethod, gotAuth, gotBody)
	}
	if !strings.Contains(out, "Insight") || !strings.Contains(out, "https://example.com") {
		t.Fatalf("digest %s", out)
	}

	_, err = QuerySearch(ctx, Connection{Protocol: "brave", Endpoint: srv.URL}, "brave-key", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotAuth != "brave-key" || !strings.Contains(gotPath, "q=hello") {
		t.Fatalf("brave method=%s auth=%s path=%s", gotMethod, gotAuth, gotPath)
	}

	_, err = QuerySearch(ctx, Connection{Protocol: "search", Endpoint: srv.URL + "/find?q={q}"}, "", "foo bar")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || !strings.Contains(gotPath, "foo") {
		t.Fatalf("template path=%s", gotPath)
	}
}

func TestInferProtocolSearchVendors(t *testing.T) {
	if InferProtocol("tavily", CapSearch) != "tavily" {
		t.Fatal(InferProtocol("tavily", CapSearch))
	}
	if InferProtocol("unknown", CapSearch) != "search" {
		t.Fatal(InferProtocol("unknown", CapSearch))
	}
}

func TestNeedsKeySearchVendors(t *testing.T) {
	if !needsKey(Connection{Protocol: "tavily", Capabilities: []string{CapSearch}}) {
		t.Fatal("tavily needs key")
	}
	if needsKey(Connection{Protocol: "search", Capabilities: []string{CapSearch}}) {
		t.Fatal("generic search does not need key")
	}
}

func TestTemplatesIncludeSearchVendors(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Templates() {
		if HasCap(c, CapSearch) {
			seen[c.Vendor] = true
		}
	}
	for _, want := range []string{"search", "tavily", "brave", "serper", "bocha"} {
		if !seen[want] {
			t.Fatalf("missing %s in %v", want, seen)
		}
	}
}
