// 本文件调用 Exa 的 JSON 搜索和托管 MCP。两次请求的 numResults 都是 8。REST 的正文上限是 400 个字符。响应仍受 maxBody 限制。

package webcontent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// DefaultExaREST 是 Exa JSON 搜索地址 https://api.exa.ai/search。
	DefaultExaREST = "https://api.exa.ai/search"
	// DefaultExaMCP 是 Exa 托管 MCP 地址 https://mcp.exa.ai/mcp。
	DefaultExaMCP = "https://mcp.exa.ai/mcp"
)

// SearchExaREST 调用 Exa 的 JSON 搜索。key 去掉空白后为空就返回 exa: api key required。endpoint 空白时用 DefaultExaREST。
// 请求体的 numResults 是 8，type 是 auto，contents.text.maxCharacters 是 400。授权头是 Authorization: Bearer 加 key。
// FormatHits 的结果须 Useful，否则错误文案是 exa empty。query 空白时错误文案是 empty query。
func SearchExaREST(ctx context.Context, client *http.Client, endpoint, key, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("empty query")
	}
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("exa: api key required")
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultExaREST
	}
	payload, _ := json.Marshal(map[string]any{
		"query":      query,
		"numResults": 8,
		"type":       "auto",
		"contents":   map[string]any{"text": map[string]any{"maxCharacters": 400}},
	})
	_, body, err := postJSON(ctx, client, endpoint, "Authorization", "Bearer "+strings.TrimSpace(key), payload)
	if err != nil {
		return "", err
	}
	hits := hitsFromJSON([]byte(body))
	out := FormatHits(hits)
	if !Useful(out) {
		return "", fmt.Errorf("exa empty")
	}
	return out, nil
}

// SearchExaMCP 调用 Exa 托管 MCP，不带 API key。先 initialize，再 tools/call，工具名是 web_search_exa，numResults 是 8。
// endpoint 空白时用 DefaultExaMCP。工具文本能抽出命中且 Useful 就返回格式化结果，否则文本本身须 Useful。不行时错误文案是 exa mcp empty。
func SearchExaMCP(ctx context.Context, client *http.Client, endpoint, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("empty query")
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultExaMCP
	}
	session, err := mcpInitialize(ctx, client, endpoint)
	if err != nil {
		return "", err
	}
	call := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "web_search_exa",
			"arguments": map[string]any{
				"query":      query,
				"numResults": 8,
			},
		},
	}
	raw, err := json.Marshal(call)
	if err != nil {
		return "", err
	}
	body, err := mcpPost(ctx, client, endpoint, session, raw)
	if err != nil {
		return "", err
	}
	text := mcpToolText(body)
	if hits := hitsFromJSON([]byte(text)); len(hits) > 0 {
		out := FormatHits(hits)
		if Useful(out) {
			return out, nil
		}
	}
	if !Useful(text) {
		return "", fmt.Errorf("exa mcp empty")
	}
	return text, nil
}

// mcpInitialize 发送 protocolVersion 2025-11-25、空 capabilities，以及 clientInfo 的 name yoyo、version 0.4.1。
// 正文同时含 "error" 和 "code" 时返回 exa mcp initialize failed。随后发送 notifications/initialized，忽略它的错误。返回会话 id。
func mcpInitialize(ctx context.Context, client *http.Client, endpoint string) (string, error) {
	init := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "yoyo", "version": "0.4.1"},
		},
	}
	raw, _ := json.Marshal(init)
	session, body, err := mcpPostSession(ctx, client, endpoint, "", raw)
	if err != nil {
		return "", err
	}
	if strings.Contains(strings.ToLower(body), `"error"`) && strings.Contains(body, `"code"`) {
		return "", fmt.Errorf("exa mcp initialize failed")
	}
	notify, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	_, _ = mcpPost(ctx, client, endpoint, session, notify)
	return session, nil
}

// mcpPost 调用 mcpPostSession 并只返回正文。
func mcpPost(ctx context.Context, client *http.Client, endpoint, session string, payload []byte) (string, error) {
	_, body, err := mcpPostSession(ctx, client, endpoint, session, payload)
	return body, err
}

// mcpPostSession 向 MCP 端点 POST JSON。ctx 为 nil 时用 defaultTimeout。session 非空时设置 Mcp-Session-Id。
// 正文最多读 maxBody。响应头里的 Mcp-Session-Id 非空就改用它。状态码大于等于 300 时把正文截到 400 字节放进错误。成功时返回 decodeSSE 之后的正文。
func mcpPostSession(ctx context.Context, client *http.Client, endpoint, session string, payload []byte) (string, string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	resp, err := httpClient(client).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	raw := string(b)
	if err != nil {
		return resp.Header.Get("Mcp-Session-Id"), "", err
	}
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		session = sid
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(raw)
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return session, raw, fmt.Errorf("exa mcp %d: %s", resp.StatusCode, msg)
	}
	return session, decodeSSE(raw), nil
}

// decodeSSE 在正文含 data: 时返回最后一条 data: 后面的内容。没有这样的行就返回原文。
func decodeSSE(raw string) string {
	if !strings.Contains(raw, "data:") {
		return raw
	}
	var last string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			last = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	if last != "" {
		return last
	}
	return raw
}

// mcpToolText 从 result.content 里拼接 type 项的 text。JSON 无效或没有文本时返回去掉空白的原文。
func mcpToolText(raw string) string {
	var top map[string]any
	if json.Unmarshal([]byte(raw), &top) != nil {
		return strings.TrimSpace(raw)
	}
	if res, ok := top["result"].(map[string]any); ok {
		if content, ok := res["content"].([]any); ok {
			var b strings.Builder
			for _, item := range content {
				m, _ := item.(map[string]any)
				if m == nil {
					continue
				}
				if s, _ := m["text"].(string); strings.TrimSpace(s) != "" {
					if b.Len() > 0 {
						b.WriteByte('\n')
					}
					b.WriteString(strings.TrimSpace(s))
				}
			}
			if b.Len() > 0 {
				return b.String()
			}
		}
	}
	return strings.TrimSpace(raw)
}
