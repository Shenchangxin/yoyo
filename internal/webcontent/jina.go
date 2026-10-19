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
	DefaultJinaSearch = "https://s.jina.ai/"
	DefaultJinaReader = "https://r.jina.ai/"
)

// SearchJina hits Jina's no-key search endpoint and returns formatted hits.
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

// ReadJina returns markdown for a public URL via Jina Reader.
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

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
