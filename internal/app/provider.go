package app

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func (a *App) TestProvider() map[string]any {
	base := strings.TrimRight(a.Config.BaseURL, "/")
	if base == "" {
		return map[string]any{"ok": false, "error": "base_url empty"}
	}
	url := base + "/models"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	if key, err := a.Vault.Get("default"); err == nil && key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	ok := res.StatusCode >= 200 && res.StatusCode < 300
	out := map[string]any{"ok": ok, "status": res.StatusCode}
	if !ok {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 240 {
			msg = msg[:240]
		}
		out["error"] = msg
	}
	return out
}
