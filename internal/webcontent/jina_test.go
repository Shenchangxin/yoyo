package webcontent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchJinaJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "LLM") {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"title": "Docs", "url": "https://example.com/docs", "description": "official"},
			},
		})
	}))
	t.Cleanup(srv.Close)
	out, err := SearchJina(context.Background(), srv.Client(), srv.URL+"/", "LLM frameworks")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Docs") || !strings.Contains(out, "https://example.com/docs") {
		t.Fatalf("%s", out)
	}
}

func TestReadJinaRejectsAntibot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("Warning: this site is requiring captcha. Title: Just a moment..."))
	}))
	t.Cleanup(srv.Close)
	if _, err := ReadJina(context.Background(), srv.Client(), srv.URL+"/", "https://example.com"); err == nil {
		t.Fatal("expected antibot error")
	}
}

func TestSearchDuckDuckGoStructured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a class="result__a" href="https://example.com/a">Alpha</a><a class="result__snippet">note</a>`))
	}))
	t.Cleanup(srv.Close)
	out, err := SearchDuckDuckGo(context.Background(), srv.Client(), srv.URL, srv.URL, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Alpha") {
		t.Fatalf("%s", out)
	}
}

func TestSearchExaREST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer exa-key" {
			t.Fatalf("auth %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{"title": "Exa Hit", "url": "https://exa.example", "text": "snippet"}},
		})
	}))
	t.Cleanup(srv.Close)
	out, err := SearchExaREST(context.Background(), srv.Client(), srv.URL, "exa-key", "q")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Exa Hit") {
		t.Fatalf("%s", out)
	}
}

func TestSearchExaMCP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "sess-1")
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		switch method {
		case "initialize":
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{}})
		case "notifications/initialized":
			_, _ = w.Write([]byte(`{}`))
		case "tools/call":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": 2,
				"result": map[string]any{
					"content": []map[string]any{{
						"type": "text",
						"text": `{"results":[{"title":"MCP","url":"https://mcp.example","snippet":"ok"}]}`,
					}},
				},
			})
		default:
			t.Fatalf("method %s", method)
		}
	}))
	t.Cleanup(srv.Close)
	out, err := SearchExaMCP(context.Background(), srv.Client(), srv.URL, "q")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "MCP") {
		t.Fatalf("%s", out)
	}
}

func TestSiteTextGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/tool" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"full_name": "acme/tool", "description": "does things",
			"html_url": "https://github.com/acme/tool", "language": "Go", "stargazers_count": 12,
		})
	}))
	t.Cleanup(srv.Close)
	prev := GitHubAPIBase
	GitHubAPIBase = srv.URL
	t.Cleanup(func() { GitHubAPIBase = prev })
	out, ok, err := SiteText(context.Background(), srv.Client(), "https://github.com/acme/tool")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !strings.Contains(out, "acme/tool") || !strings.Contains(out, "does things") {
		t.Fatalf("%s", out)
	}
}

func TestBilibiliText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "bvid=BV1xx411c7mD") {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"title": "Demo", "desc": "hello", "bvid": "BV1xx411c7mD",
				"owner": map[string]any{"name": "up"},
			},
		})
	}))
	t.Cleanup(srv.Close)
	prev := BilibiliViewBase
	BilibiliViewBase = srv.URL
	t.Cleanup(func() { BilibiliViewBase = prev })
	out, ok, err := SiteText(context.Background(), srv.Client(), "https://www.bilibili.com/video/BV1xx411c7mD")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !strings.Contains(out, "Demo") || !strings.Contains(out, "up") {
		t.Fatalf("%s", out)
	}
}
