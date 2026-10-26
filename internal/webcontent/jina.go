// 本文件调用 Jina 的无密钥搜索和阅读接口。正文仍受 get 的 maxBody 限制。

package webcontent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	// DefaultJinaSearch 是 Jina 搜索地址 https://s.jina.ai/。
	DefaultJinaSearch = "https://s.jina.ai/"
	// DefaultJinaReader 是 Jina 阅读地址 https://r.jina.ai/。
	DefaultJinaReader = "https://r.jina.ai/"
)

// SearchJina 请求 base 加上路径转义后的 query。base 空白时用 DefaultJinaSearch，没有结尾斜杠会补上。
// 状态码大于等于 300 时错误文案是 jina search 加状态码。能从 JSON 抽出命中且 FormatHits 通过 Useful 就返回格式化文本。
// 否则正文须通过 Useful，不然错误文案是 jina search empty。query 空白时错误文案是 empty query。
func SearchJina(ctx context.Context, client *http.Client, base, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("empty query")
	}
	if strings.TrimSpace(base) == "" {
		base = DefaultJinaSearch
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	u := base + url.PathEscape(query)
	status, _, _, body, err := get(ctx, client, u, "application/json, text/plain;q=0.8")
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("jina search %d", status)
	}
	if hits := hitsFromJSON([]byte(body)); len(hits) > 0 {
		out := FormatHits(hits)
		if Useful(out) {
			return out, nil
		}
	}
	text := strings.TrimSpace(body)
	if !Useful(text) {
		return "", fmt.Errorf("jina search empty")
	}
	return text, nil
}

// ReadJina 请求 base 直接拼接 rawURL，Accept 为 text/plain。base 空白时用 DefaultJinaReader。
// 状态码大于等于 300 时错误文案是 jina reader 加状态码。LooksAntibot 为真时错误文案是 jina reader antibot。
// 去掉空白后短于 40 字节时错误文案是 jina reader empty。rawURL 空白时错误文案是 empty url。
func ReadJina(ctx context.Context, client *http.Client, base, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty url")
	}
	if strings.TrimSpace(base) == "" {
		base = DefaultJinaReader
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	u := base + rawURL
	status, _, _, body, err := get(ctx, client, u, "text/plain")
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("jina reader %d", status)
	}
	if LooksAntibot(body) {
		return "", fmt.Errorf("jina reader antibot")
	}
	text := strings.TrimSpace(body)
	if len(text) < 40 {
		return "", fmt.Errorf("jina reader empty")
	}
	return text, nil
}

// hitsFromJSON 从对象的 data、results、organic、web、items 里取数组。某一项是对象且含 results 数组时用那个数组。
// 每条取 title 或 name、url 或 link 或 href、以及 description、snippet、content、text、summary 里的第一个非空字符串。标题和 URL 都空的丢掉。
func hitsFromJSON(raw []byte) []Hit {
	var top map[string]any
	if json.Unmarshal(raw, &top) != nil {
		return nil
	}
	var arr []any
	for _, key := range []string{"data", "results", "organic", "web", "items"} {
		switch v := top[key].(type) {
		case []any:
			arr = v
		case map[string]any:
			if inner, ok := v["results"].([]any); ok {
				arr = inner
			}
		}
		if len(arr) > 0 {
			break
		}
	}
	var hits []Hit
	for _, item := range arr {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		h := Hit{
			Title:   firstString(m, "title", "name"),
			URL:     firstString(m, "url", "link", "href"),
			Snippet: firstString(m, "description", "snippet", "content", "text", "summary"),
		}
		if h.Title == "" && h.URL == "" {
			continue
		}
		hits = append(hits, h)
	}
	return hits
}

// firstString 按 keys 的顺序返回第一个去掉空白后非空的字符串。
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
