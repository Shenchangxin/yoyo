// 本文件先请求 DuckDuckGo 的 HTML 端，失败或不像结果时再请求 lite 端。

package webcontent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	// DefaultDDGHTML 是 DuckDuckGo HTML 搜索地址 https://html.duckduckgo.com/html/。
	DefaultDDGHTML = "https://html.duckduckgo.com/html/"
	// DefaultDDGLite 是 DuckDuckGo lite 搜索地址 https://lite.duckduckgo.com/lite/。
	DefaultDDGLite = "https://lite.duckduckgo.com/lite/"
)

// SearchDuckDuckGo 先用 htmlBase，再用 liteBase。空白的 base 换成对应的默认地址。查询放在 q 参数里。
// 某一端返回且 Useful 为真就用它。两边都不行时错误文案是 duckduckgo empty。query 空白时错误文案是 empty query。
// ctx 传给 fetchDDG。
// client 传给 fetchDDG。
func SearchDuckDuckGo(ctx context.Context, client *http.Client, htmlBase, liteBase, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("empty query")
	}
	if strings.TrimSpace(htmlBase) == "" {
		htmlBase = DefaultDDGHTML
	}
	if strings.TrimSpace(liteBase) == "" {
		liteBase = DefaultDDGLite
	}
	if out, err := fetchDDG(ctx, client, htmlBase, query); err == nil && Useful(out) {
		return out, nil
	}
	if out, err := fetchDDG(ctx, client, liteBase, query); err == nil && Useful(out) {
		return out, nil
	}
	return "", fmt.Errorf("duckduckgo empty")
}

// fetchDDG 把 q 设成 query 后 GET，Accept 为 text/html。状态码大于等于 300 时错误文案是 duckduckgo 加状态码。
// 解析结果经 FormatHits 后须 Useful，否则错误文案是 duckduckgo parse empty。
// ctx 传给 get。
// client 传给 get。
// base 交给 url.Parse，查询参数 q 设成 query 后把整段地址传给 get。
func fetchDDG(ctx context.Context, client *http.Client, base, query string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	status, _, _, body, err := get(ctx, client, u.String(), "text/html")
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("duckduckgo %d", status)
	}
	hits := ParseDuckDuckGoHTML(body)
	out := FormatHits(hits)
	if !Useful(out) {
		return "", fmt.Errorf("duckduckgo parse empty")
	}
	return out, nil
}
