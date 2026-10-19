package webcontent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const userAgent = "Yoyo/0.3 (+https://github.com/Shenchangxin/yoyo)"

const defaultTimeout = 20 * time.Second

const maxBody = 256 << 10

func httpClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return http.DefaultClient
}

func get(ctx context.Context, client *http.Client, rawURL, accept string) (status int, finalURL, contentType, body string, err error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, rawURL, "", "", err
	}
	req.Header.Set("User-Agent", userAgent)
	if accept == "" {
		accept = "text/html, application/json;q=0.9, text/plain;q=0.8, */*;q=0.7"
	}
	req.Header.Set("Accept", accept)
	resp, err := httpClient(client).Do(req)
	if err != nil {
		return 0, rawURL, "", "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type"), "", err
	}
	return resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type"), string(raw), nil
}

func postJSON(ctx context.Context, client *http.Client, rawURL, authHeader, authVal string, payload []byte) (status int, body string, err error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(string(payload)))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if strings.TrimSpace(authHeader) != "" && strings.TrimSpace(authVal) != "" {
		req.Header.Set(authHeader, authVal)
	}
	resp, err := httpClient(client).Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return resp.StatusCode, "", err
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return resp.StatusCode, msg, fmt.Errorf("http %d: %s", resp.StatusCode, msg)
	}
	return resp.StatusCode, string(raw), nil
}
