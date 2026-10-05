package connection

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (r *Registry) Test(id string) map[string]any {
	c, err := r.Get(id)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return TestConnection(c, r)
}

func TestConnection(c Connection, r *Registry) map[string]any {
	if c.Protocol == "cas" {
		return map[string]any{"ok": true, "status": 200, "reachable": true}
	}
	key := ""
	if r != nil && c.VaultKey != "" {
		key, _ = r.Lease(c)
	}
	if needsKey(c) && key == "" {
		return map[string]any{"ok": false, "error": "missing vault key"}
	}
	url := strings.TrimRight(c.Endpoint, "/")
	if url == "" {
		return map[string]any{"ok": false, "error": "endpoint empty"}
	}
	if HasCap(c, CapChat) || c.Protocol == "openai-chat-completions" {
		if !strings.HasSuffix(url, "/models") {
			url = url + "/models"
		}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("x-goog-api-key", key)
	}
	cli := &http.Client{Timeout: 8 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 2048))
	ok := res.StatusCode >= 200 && res.StatusCode < 300
	out := map[string]any{"ok": ok, "status": res.StatusCode, "reachable": ok}
	if !ok {
		out["error"] = fmt.Sprintf("HTTP %d", res.StatusCode)
	}
	return out
}

func needsKey(c Connection) bool {
	if c.Protocol == "cas" {
		return false
	}
	if HasCap(c, CapOTEL) || HasCap(c, CapSearch) {
		return false
	}
	if HasCap(c, CapStorage) && c.Protocol == "cas" {
		return false
	}
	return true
}
