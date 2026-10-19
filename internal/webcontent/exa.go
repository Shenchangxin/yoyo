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
	DefaultExaREST = "https://api.exa.ai/search"
	DefaultExaMCP  = "https://mcp.exa.ai/mcp"
)

// SearchExaREST calls Exa's JSON search API. key is required.
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

// SearchExaMCP calls Exa's hosted MCP endpoint (no API key). Handshake is
// initialize + tools/call web_search_exa.
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

func mcpPost(ctx context.Context, client *http.Client, endpoint, session string, payload []byte) (string, error) {
	_, body, err := mcpPostSession(ctx, client, endpoint, session, payload)
	return body, err
}

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
