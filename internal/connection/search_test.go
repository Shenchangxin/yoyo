// 锁住搜索协议、哪些供应商要钥匙，以及模板里包含这些搜索站。

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

// TestQuerySearchVendors 确认 Tavily 带 Bearer 并 POST，Brave 把钥匙放在 X-Subscription-Token，模板端点的路径被用上。
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

// TestInferProtocolSearchVendors 确认 tavily、jina、exa、duckduckgo 和未知供应商的搜索协议。
func TestInferProtocolSearchVendors(t *testing.T) {
	if InferProtocol("tavily", CapSearch) != "tavily" {
		t.Fatal(InferProtocol("tavily", CapSearch))
	}
	if InferProtocol("jina", CapSearch) != "jina" {
		t.Fatal(InferProtocol("jina", CapSearch))
	}
	if InferProtocol("exa", CapSearch) != "exa" {
		t.Fatal(InferProtocol("exa", CapSearch))
	}
	if InferProtocol("duckduckgo", CapSearch) != "duckduckgo" {
		t.Fatal(InferProtocol("duckduckgo", CapSearch))
	}
	if InferProtocol("unknown", CapSearch) != "search" {
		t.Fatal(InferProtocol("unknown", CapSearch))
	}
}

// TestNeedsKeySearchVendors 确认 tavily 和 exa 要钥匙，通用 search、jina 和 exa-mcp 不要。
func TestNeedsKeySearchVendors(t *testing.T) {
	if !needsKey(Connection{Protocol: "tavily", Capabilities: []string{CapSearch}}) {
		t.Fatal("tavily needs key")
	}
	if !needsKey(Connection{Protocol: "exa", Capabilities: []string{CapSearch}}) {
		t.Fatal("exa rest needs key")
	}
	if needsKey(Connection{Protocol: "search", Capabilities: []string{CapSearch}}) {
		t.Fatal("generic search does not need key")
	}
	if needsKey(Connection{Protocol: "jina", Capabilities: []string{CapSearch}}) {
		t.Fatal("jina does not need key")
	}
	if needsKey(Connection{Protocol: "exa-mcp", Capabilities: []string{CapSearch}}) {
		t.Fatal("exa mcp does not need key")
	}
}

// TestTemplatesIncludeSearchVendors 确认模板里有这些搜索供应商，包括 exa-mcp。
func TestTemplatesIncludeSearchVendors(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Templates() {
		if HasCap(c, CapSearch) {
			seen[c.Vendor] = true
			seen[c.Protocol] = true
		}
	}
	for _, want := range []string{"search", "tavily", "brave", "serper", "bocha", "jina", "exa", "duckduckgo"} {
		if !seen[want] {
			t.Fatalf("missing %s in %v", want, seen)
		}
	}
	if !seen["exa-mcp"] {
		t.Fatalf("missing exa-mcp in %v", seen)
	}
}

// TestQuerySearchJina 确认空端点的 jina 连接仍能返回搜索摘要。
func TestQuerySearchJina(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"title": "Jina", "url": "https://jina.example", "description": "hit"}},
		})
	}))
	t.Cleanup(srv.Close)
	out, err := QuerySearch(context.Background(), Connection{Protocol: "jina", Endpoint: srv.URL + "/"}, "", "q")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Jina") {
		t.Fatalf("%s", out)
	}
}
