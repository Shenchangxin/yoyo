// 本文件锁住 Jina、DuckDuckGo、Exa 和两个站点提取器对测试服务器的解析。由 go test 调用。

package webcontent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSearchJinaJSON 锁住查询会进路径，并且 data 里的标题和 URL 出现在结果里。
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

// TestReadJinaRejectsAntibot 锁住含 requiring captcha 的正文让 ReadJina 返回错误。
func TestReadJinaRejectsAntibot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("Warning: this site is requiring captcha. Title: Just a moment..."))
	}))
	t.Cleanup(srv.Close)
	if _, err := ReadJina(context.Background(), srv.Client(), srv.URL+"/", "https://example.com"); err == nil {
		t.Fatal("expected antibot error")
	}
}

// TestSearchDuckDuckGoStructured 锁住 result__a 页面能抽出标题 Alpha。
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

// TestSearchExaREST 锁住请求带 Bearer 密钥，并且结果标题出现在输出里。
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

// TestSearchExaMCP 锁住 initialize、notifications/initialized 和 tools/call 之后，工具文本里的标题 MCP 出现在输出里。
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

// TestSiteTextGitHub 锁住仓库根会请求 /repos/acme/tool，输出含 full_name 和 description。
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

// TestBilibiliText 锁住查看接口的查询含 bvid，输出含标题和 up 主名字。
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
