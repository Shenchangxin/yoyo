// 按协议调用搜索端点。端点为空时，jina、exa、exa-mcp 和 duckduckgo 使用 webcontent 的默认地址。

package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

// QuerySearch 按协议请求搜索。端点为空时，jina、exa、exa-mcp、duckduckgo 填上 webcontent 的默认地址，其他协议返回 no search endpoint。ctx 为 nil 时使用 20 秒超时。
// key 传给 brave 的 searchGET，tavily、serper、bocha 和默认分支的 searchJSON 或 searchRaw，以及 exa 的 SearchExaREST。jina、exa-mcp、duckduckgo 的调用不使用它。
// query 去掉空白后为空时返回 empty query。否则按协议放进 q、query，或经 QueryEscape 替换端点里的 {q}，并传给对应的 webcontent 函数。
func QuerySearch(ctx context.Context, c Connection, key, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("empty query")
	}
	endpoint := strings.TrimSpace(c.Endpoint)
	protocol := strings.ToLower(strings.TrimSpace(c.Protocol))
	if protocol == "" {
		protocol = InferProtocol(c.Vendor, CapSearch)
	}
	if endpoint == "" {
		switch protocol {
		case "jina":
			endpoint = webcontent.DefaultJinaSearch
		case "exa":
			endpoint = webcontent.DefaultExaREST
		case "exa-mcp":
			endpoint = webcontent.DefaultExaMCP
		case "duckduckgo", "ddg":
			endpoint = webcontent.DefaultDDGHTML
		default:
			return "", fmt.Errorf("no search endpoint")
		}
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
	}
	switch protocol {
	case "brave":
		return searchGET(ctx, endpoint, query, "X-Subscription-Token", key)
	case "tavily":
		return searchJSON(ctx, endpoint, key, "Authorization", "Bearer "+key, map[string]any{"query": query, "max_results": 8})
	case "serper":
		return searchJSON(ctx, endpoint, key, "X-API-KEY", key, map[string]any{"q": query, "num": 8})
	case "bocha":
		return searchJSON(ctx, endpoint, key, "Authorization", "Bearer "+key, map[string]any{"query": query, "count": 8})
	case "jina":
		return webcontent.SearchJina(ctx, http.DefaultClient, endpoint, query)
	case "exa":
		return webcontent.SearchExaREST(ctx, http.DefaultClient, endpoint, key, query)
	case "exa-mcp":
		return webcontent.SearchExaMCP(ctx, http.DefaultClient, endpoint, query)
	case "duckduckgo", "ddg":
		htmlBase, liteBase := endpoint, ""
		if strings.Contains(strings.ToLower(endpoint), "lite") {
			liteBase = endpoint
			htmlBase = ""
		}
		return webcontent.SearchDuckDuckGo(ctx, http.DefaultClient, htmlBase, liteBase, query)
	default:
		if strings.Contains(endpoint, "{q}") {
			u := strings.ReplaceAll(endpoint, "{q}", url.QueryEscape(query))
			return searchRaw(ctx, http.MethodGet, u, key, "Authorization", "Bearer "+key, nil)
		}
		return searchJSON(ctx, endpoint, key, "Authorization", "Bearer "+key, map[string]any{"query": query, "q": query})
	}
}

// searchGET 把 q 加到查询串上，再按给定头发送钥匙。
// ctx 传给 searchRaw。
// endpoint 交给 url.Parse，查询参数 q 设成 query 后再把地址传给 searchRaw。
// query 设成查询参数 q。
// header 传给 searchRaw，头的值也是 key。
// key 传给 searchRaw，同时作为头的值。
func searchGET(ctx context.Context, endpoint, query, header, key string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	return searchRaw(ctx, http.MethodGet, u.String(), key, header, key, nil)
}

// searchJSON 把 body 编成 JSON 并 POST。
// ctx 传给 searchRaw。
// endpoint 传给 searchRaw。
// key 传给 searchRaw。
// header 传给 searchRaw。
// headerVal 传给 searchRaw。
func searchJSON(ctx context.Context, endpoint, key, header, headerVal string, body map[string]any) (string, error) {
	raw, _ := json.Marshal(body)
	return searchRaw(ctx, http.MethodPost, endpoint, key, header, headerVal, raw)
}

// searchRaw 发出请求。钥匙、头名和头值都非空才设置该头。状态码大于等于 300 时返回错误，正文最多保留 400 字节。
// ctx 传给 NewRequestWithContext。
// method 作为请求方法。
// endpoint 作为请求地址。
// key 去掉空白后非空，且 header 和 headerVal 都非空时，才设置这个头。
// header 非空且 key、headerVal 都非空时作为要设置的头名。
// headerVal 非空且 key、header 都非空时作为头的值。
// body 长度大于 0 时作为请求体，并设置 Content-Type 为 application/json。
func searchRaw(ctx context.Context, method, endpoint, key, header, headerVal string, body []byte) (string, error) {
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, rdr)
	if err != nil {
		return "", err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(key) != "" && header != "" && headerVal != "" {
		req.Header.Set(header, headerVal)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(b))
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return "", fmt.Errorf("search %d: %s", resp.StatusCode, msg)
	}
	return digestSearchJSON(b), nil
}

// digestSearchJSON 从 results、organic、web、data 或 items 里取出最多 8 条标题、链接和摘要。解析失败则返回原文去空白。成功但没有这些字段时，缩进 JSON 最多 8000 字节。
// raw 按 JSON 对象解析。失败时返回去掉空白的原文。
func digestSearchJSON(raw []byte) string {
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		return strings.TrimSpace(string(raw))
	}
	var lines []string
	for _, key := range []string{"results", "organic", "web", "data", "items"} {
		arr := asAnySlice(top[key])
		if key == "web" {
			if m, ok := top["web"].(map[string]any); ok {
				arr = asAnySlice(m["results"])
			}
		}
		if len(arr) == 0 {
			continue
		}
		for i, item := range arr {
			if i >= 8 {
				break
			}
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			title := firstString(m, "title", "name")
			link := firstString(m, "url", "link", "href")
			snip := firstString(m, "content", "snippet", "description", "summary", "text")
			if title == "" && link == "" {
				continue
			}
			line := fmt.Sprintf("%d. %s", i+1, title)
			if link != "" {
				line += "\n   " + link
			}
			if snip != "" {
				line += "\n   " + snip
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			return strings.Join(lines, "\n")
		}
	}
	b, _ := json.MarshalIndent(top, "", "  ")
	s := string(b)
	if len(s) > 8000 {
		return s[:8000]
	}
	return s
}

// asAnySlice 在值是 []any 时返回它，否则返回 nil。
// v 的动态类型是 []any 时返回它，否则返回 nil。
func asAnySlice(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	default:
		return nil
	}
}

// firstString 返回映射里第一个非空白字符串字段。
// m 按 keys 取值，类型是字符串且去掉空白后非空时返回去掉空白的结果。
// keys 按顺序尝试。
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
