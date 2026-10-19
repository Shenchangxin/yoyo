package webcontent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	DefaultDDGHTML = "https://html.duckduckgo.com/html/"
	DefaultDDGLite = "https://lite.duckduckgo.com/lite/"
)

// SearchDuckDuckGo fetches DDG HTML and returns structured hits.
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
