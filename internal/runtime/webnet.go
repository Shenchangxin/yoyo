// web_search 的检索顺序：调用方 SearchAPI、Jina、DuckDuckGo，最后用 web_fetch 抓搜索页。出网审批是 Network。抓取与主机拒绝在 tools_codex.go。

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

// webHTTP 是 http.DefaultClient，检索和 Jina 共用它。
var webHTTP = http.DefaultClient

// builtinJinaSearch、builtinJinaReader、builtinDDGHTML、builtinDDGLite 是可被测试替换的默认检索地址。
var (
	// builtinJinaSearch 是默认的 Jina 检索地址，测试可以换掉。
	builtinJinaSearch = webcontent.DefaultJinaSearch
	// builtinJinaReader 是默认的 Jina 阅读器地址，测试可以换掉。
	builtinJinaReader = webcontent.DefaultJinaReader
	// builtinDDGHTML 是默认的 DuckDuckGo HTML 地址，测试可以换掉。
	builtinDDGHTML = webcontent.DefaultDDGHTML
	// builtinDDGLite 是默认的 DuckDuckGo Lite 地址，测试可以换掉。
	builtinDDGLite = webcontent.DefaultDDGLite
)

// webSearch 去掉空白后检索。空查询返回错误。查询本身是 http(s) 地址时改走 webFetch。先过 Network 审批。SearchAPI 有可用结果就返回；否则依次试 Jina 和 DuckDuckGo，最后抓 html.duckduckgo.com。
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

// searchCtx 以工具上下文为父，加上给定超时。工具或上下文为空时用 Background。
func (t *WorkspaceTools) searchCtx(d time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if t != nil && t.Ctx != nil {
		ctx = t.Ctx
	}
	return context.WithTimeout(ctx, d)
}

// looksLikeHTTPURL 在能解析、方案是 http 或 https、有 Host 且不含空格时返回真。
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
