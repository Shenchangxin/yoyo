// web_search 的检索顺序：调用方 SearchAPI、Jina、DuckDuckGo，最后在同一次时限内抓搜索页。出网审批是 Network。抓取与主机拒绝在 tools_codex.go。

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

// builtinJinaSearch、builtinJinaReader、builtinDDGHTML、builtinDDGLite、builtinSearchPage 是可被测试替换的默认检索地址。
var (
	// builtinJinaSearch 是默认的 Jina 检索地址，测试可以换掉。
	builtinJinaSearch = webcontent.DefaultJinaSearch
	// builtinJinaReader 是默认的 Jina 阅读器地址，测试可以换掉。
	builtinJinaReader = webcontent.DefaultJinaReader
	// builtinDDGHTML 是默认的 DuckDuckGo HTML 地址，测试可以换掉。
	builtinDDGHTML = webcontent.DefaultDDGHTML
	// builtinDDGLite 是默认的 DuckDuckGo Lite 地址，测试可以换掉。
	builtinDDGLite = webcontent.DefaultDDGLite
	// builtinSearchPage 是检索都没有结果时要抓的搜索页前缀，后面接上转义后的查询。测试可以换掉。
	builtinSearchPage = "https://html.duckduckgo.com/html/?q="
)

// webSearchBudget 是一次检索的总时限，包含最后的搜索页抓取。测试可以改短。
var webSearchBudget = 20 * time.Second

// webSearch 去掉空白后检索。空查询返回错误。查询本身是 http(s) 地址时改走 webFetch。先过 Network 审批。SearchAPI 有可用结果就返回；否则依次试 Jina 和 DuckDuckGo。时限用完就返回已记下的失败，不再另开一次抓取。还有剩余时间才抓搜索页。
// query 先去掉空白，空串返回错误；看起来是 http(s) 地址时交给 webFetch，否则作为检索词传给 SearchAPI、SearchJina 和 SearchDuckDuckGo。
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
	ctx, cancel := t.searchCtx(webSearchBudget)
	defer cancel()
	var fails []string
	note := func(name string, err error) {
		if err != nil {
			fails = append(fails, name+": "+err.Error())
			return
		}
		fails = append(fails, name+": no result")
	}
	if t != nil && t.SearchAPI != nil {
		out, err := t.SearchAPI(query)
		if err == nil && webcontent.Useful(out) {
			return ToolResult{Content: "search: " + query + "\n\n" + out}
		}
		note("search", err)
	}
	if ctx.Err() == nil {
		out, err := webcontent.SearchJina(ctx, webHTTP, builtinJinaSearch, query)
		if err == nil && webcontent.Useful(out) {
			return ToolResult{Content: "search (jina): " + query + "\n\n" + out}
		}
		note("jina", err)
	}
	if ctx.Err() == nil {
		out, err := webcontent.SearchDuckDuckGo(ctx, webHTTP, builtinDDGHTML, builtinDDGLite, query)
		if err == nil && webcontent.Useful(out) {
			return ToolResult{Content: "search (duckduckgo): " + query + "\n\n" + out}
		}
		note("duckduckgo", err)
	}
	if ctx.Err() != nil {
		msg := strings.Join(fails, "; ")
		if msg == "" {
			msg = ctx.Err().Error()
		}
		return ToolResult{Err: fmt.Errorf("web_search: %s", msg)}
	}
	u := builtinSearchPage + url.QueryEscape(query)
	res := t.webFetchCtx(ctx, u)
	if res.Err != nil {
		if len(fails) > 0 {
			res.Err = fmt.Errorf("web_search: %s; fallback: %s", strings.Join(fails, "; "), res.Err.Error())
		}
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
