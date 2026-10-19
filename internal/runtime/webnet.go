package runtime

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

var webHTTP = http.DefaultClient

var (
	builtinJinaSearch = webcontent.DefaultJinaSearch
	builtinJinaReader = webcontent.DefaultJinaReader
	builtinDDGHTML    = webcontent.DefaultDDGHTML
	builtinDDGLite    = webcontent.DefaultDDGLite
)

func (t *WorkspaceTools) webSearch(query string) ToolResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return ToolResult{Err: fmt.Errorf("web_search: empty query")}
	}
	if looksLikeHTTPURL(query) {
		res := t.webFetch(query)
		if res.Err != nil {
			return res
		}
		res.Content = "query is a URL; fetched instead of searching\n\n" + res.Content
		return res
	}
	if err := t.check(capability.Network, "web_search", "", query); err != nil {
		return ToolResult{Err: err}
	}
	ctx, cancel := t.searchCtx(20 * time.Second)
	defer cancel()
	if t != nil && t.SearchAPI != nil {
		out, err := t.SearchAPI(query)
		if err == nil && webcontent.Useful(out) {
			return ToolResult{Content: "search: " + query + "\n\n" + out}
		}
	}
	if out, err := webcontent.SearchJina(ctx, webHTTP, builtinJinaSearch, query); err == nil && webcontent.Useful(out) {
		return ToolResult{Content: "search (jina): " + query + "\n\n" + out}
	}
	if out, err := webcontent.SearchDuckDuckGo(ctx, webHTTP, builtinDDGHTML, builtinDDGLite, query); err == nil && webcontent.Useful(out) {
		return ToolResult{Content: "search (duckduckgo): " + query + "\n\n" + out}
	}
	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	res := t.webFetch(u)
	if res.Err != nil {
		return res
	}
	if hits := webcontent.ParseDuckDuckGoHTML(res.Content); len(hits) > 0 {
		if out := webcontent.FormatHits(hits); webcontent.Useful(out) {
			return ToolResult{Content: "search (duckduckgo): " + query + "\n\n" + out}
		}
	}
	return ToolResult{Content: "search (fallback scrape): " + query + "\n\n" + res.Content}
}

func (t *WorkspaceTools) searchCtx(d time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if t != nil && t.Ctx != nil {
		ctx = t.Ctx
	}
	return context.WithTimeout(ctx, d)
}

func looksLikeHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != "" && !strings.Contains(s, " ")
}
