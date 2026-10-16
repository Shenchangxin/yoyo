package connection

import (
	"bytes"
	"encoding/json"
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
	key = strings.TrimSpace(key)
	if key == "" && r != nil && r.vault != nil {
		if k, err := r.vault.Get("default"); err == nil {
			key = strings.TrimSpace(k)
		}
	}
	if needsKey(c) && key == "" {
		return map[string]any{"ok": false, "error": "missing vault key"}
	}
	url := probeURL(c)
	if url == "" {
		return map[string]any{"ok": false, "error": "endpoint empty"}
	}
	if HasCap(c, CapChat) || c.Protocol == "openai-chat-completions" {
		return ProbeChatCompletions(strings.TrimRight(strings.TrimSpace(c.Endpoint), "/"), key, ModelFor(c, CapChat), c.Vendor)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	applyProbeAuth(req, c, key)
	cli := &http.Client{Timeout: 8 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	ok := res.StatusCode >= 200 && res.StatusCode < 300
	out := map[string]any{"ok": ok, "status": res.StatusCode, "reachable": ok}
	if !ok {
		out["error"] = ProbeError(res.StatusCode, string(raw))
	}
	return out
}

// ProbeChatCompletions hits POST {base}/chat/completions — the contract every
// OpenAI-compatible chat host implements. GET /models is optional and some
// gateways 401 that route even with a valid Bearer token.
func ProbeChatCompletions(base, key, model, vendor string) map[string]any {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	key = strings.TrimSpace(key)
	model = strings.TrimSpace(model)
	if model == "" {
		model = "default"
	}
	if base == "" {
		return map[string]any{"ok": false, "error": "endpoint empty"}
	}
	if key == "" {
		return map[string]any{"ok": false, "error": "missing vault key"}
	}
	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
		"stream":     false,
	})
	req, err := http.NewRequest(http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	if chatVendorUsesGoogleKey(vendor, base) {
		req.Header.Set("x-goog-api-key", key)
	}
	cli := &http.Client{Timeout: 8 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	ok := chatProbeOK(res.StatusCode, raw)
	out := map[string]any{"ok": ok, "status": res.StatusCode, "reachable": ok}
	if !ok {
		out["error"] = ProbeError(res.StatusCode, string(raw))
	}
	return out
}

func chatProbeOK(status int, raw []byte) bool {
	if status >= 200 && status < 300 {
		return true
	}
	if status == http.StatusBadRequest {
		msg := strings.ToLower(string(raw))
		if strings.Contains(msg, "authorization") || strings.Contains(msg, "unauthoriz") ||
			strings.Contains(msg, "api key") || strings.Contains(msg, "invalid token") ||
			strings.Contains(msg, "expired token") || strings.Contains(msg, "authentication") {
			return false
		}
		// Gateway accepted the Bearer and rejected the tiny ping payload
		// (unknown model, max_tokens, etc). That still proves reachability.
		return true
	}
	return false
}

func ProbeError(status int, raw string) string {
	msg := strings.TrimSpace(raw)
	var wrap struct {
		Msg     string          `json:"msg"`
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal([]byte(msg), &wrap) == nil {
		switch {
		case strings.TrimSpace(wrap.Msg) != "":
			msg = strings.TrimSpace(wrap.Msg)
		case strings.TrimSpace(wrap.Message) != "":
			msg = strings.TrimSpace(wrap.Message)
		default:
			if s := jsonErrorString(wrap.Error); s != "" {
				msg = s
			}
		}
	}
	if len(msg) > 240 {
		msg = msg[:240]
	}
	if msg == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	return fmt.Sprintf("HTTP %d: %s", status, msg)
}

func jsonErrorString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Type    string `json:"type"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		if strings.TrimSpace(obj.Message) != "" {
			return strings.TrimSpace(obj.Message)
		}
		if strings.TrimSpace(obj.Msg) != "" {
			return strings.TrimSpace(obj.Msg)
		}
	}
	return strings.TrimSpace(string(raw))
}

func probeURL(c Connection) string {
	base := strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	if base == "" {
		return ""
	}
	v := strings.ToLower(strings.TrimSpace(c.Vendor))
	p := strings.ToLower(strings.TrimSpace(c.Protocol))
	switch {
	case v == "pexels" || p == "pexels":
		if strings.Contains(base, "/videos") {
			return base + "/popular?per_page=1"
		}
		return base + "/curated?per_page=1"
	case v == "pixabay" || p == "pixabay":
		return base
	case v == "unsplash" || p == "unsplash":
		return base + "/photos/random"
	case v == "giphy" || p == "giphy":
		return base + "/gifs/trending?limit=1"
	case v == "deepgram" || p == "deepgram":
		if strings.HasSuffix(base, "/v1") {
			return base + "/projects"
		}
		return base + "/v1/projects"
	case v == "assemblyai" || p == "assemblyai":
		if strings.Contains(base, "/v2") {
			return base + "/transcript"
		}
		return base + "/v2/transcript"
	default:
		return base
	}
}

func applyProbeAuth(req *http.Request, c Connection, key string) {
	if req == nil || strings.TrimSpace(key) == "" {
		return
	}
	v := strings.ToLower(strings.TrimSpace(c.Vendor))
	p := strings.ToLower(strings.TrimSpace(c.Protocol))
	switch {
	case v == "pexels" || p == "pexels":
		req.Header.Set("Authorization", key)
	case v == "pixabay" || p == "pixabay":
		q := req.URL.Query()
		q.Set("key", key)
		if q.Get("q") == "" {
			q.Set("q", "yellow")
		}
		if q.Get("per_page") == "" {
			q.Set("per_page", "3")
		}
		req.URL.RawQuery = q.Encode()
	case v == "unsplash" || p == "unsplash":
		req.Header.Set("Authorization", "Client-ID "+key)
	case v == "giphy" || p == "giphy":
		q := req.URL.Query()
		q.Set("api_key", key)
		req.URL.RawQuery = q.Encode()
	case v == "deepgram" || p == "deepgram":
		req.Header.Set("Authorization", "Token "+key)
	case v == "assemblyai" || p == "assemblyai":
		req.Header.Set("Authorization", key)
	case v == "elevenlabs":
		req.Header.Set("xi-api-key", key)
	case v == "azure" || v == "microsoft":
		req.Header.Set("Ocp-Apim-Subscription-Key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
		if chatVendorUsesGoogleKey(c.Vendor, c.Endpoint) {
			req.Header.Set("x-goog-api-key", key)
		}
	}
}

func chatVendorUsesGoogleKey(vendor, base string) bool {
	v := strings.ToLower(strings.TrimSpace(vendor))
	if v == "gemini" || v == "google" {
		return true
	}
	return strings.Contains(strings.ToLower(base), "generativelanguage.googleapis.com")
}

func truncateProbe(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func needsKey(c Connection) bool {
	if c.Protocol == "cas" {
		return false
	}
	if HasCap(c, CapOTEL) {
		return false
	}
	if HasCap(c, CapSearch) {
		switch strings.ToLower(strings.TrimSpace(c.Protocol)) {
		case "tavily", "brave", "serper", "bocha":
			return true
		default:
			return false
		}
	}
	if HasCap(c, CapStorage) && c.Protocol == "cas" {
		return false
	}
	return true
}
