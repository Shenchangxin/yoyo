// 用租约或 default 钥匙探测端点是否可达。探测请求带上钥匙，连接 JSON 仍然不保存钥匙原文。

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

// Test 读取 id 对应的连接并做一次探测。读不到时返回 ok 为 false。
func (r *Registry) Test(id string) map[string]any {
	c, err := r.Get(id)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return TestConnection(c, r)
}

// TestConnection 对 cas 协议直接返回可达。其他协议用 Lease 取钥匙，仍为空则再 Get default。聊天或 openai-chat-completions 走 ProbeChatCompletions。
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

// ProbeChatCompletions 向 base/chat/completions 发 POST，正文 max_tokens 为 1。不请求 GET /models。钥匙为空返回 missing vault key。
// vendor 和整理后的 base 传给 chatVendorUsesGoogleKey。为真时设置 x-goog-api-key。
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

// chatProbeOK 在 2xx 时返回真。400 且正文不像鉴权失败时也返回真。其他状态返回假。
// status 大于等于 200 且小于 300 时返回真。等于 http.StatusBadRequest 时再看 raw。其他值返回假。
// raw 在 status 等于 http.StatusBadRequest 时转成小写。含 authorization、unauthoriz、api key、invalid token、expired token 或 authentication 则返回假，否则返回真。
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
		// 网关收下了 Bearer，拒绝的是这条极短探测体。这仍算可达。
		return true
	}
	return false
}

// ProbeError 从 JSON 的 msg、message 或 error 里抽出说明，最长 240 字节。抽不出时返回 HTTP 状态码。
// status 在说明为空时写成 HTTP 加这个状态码。否则写在 HTTP、状态码和冒号后面。
// raw 去掉空白后尝试按 JSON 取出 msg、message 或 error。结果长于 240 字节时截到 240。
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

// jsonErrorString 把 error 字段解析成字符串，或对象里的 message、msg。
// raw 长度为 0 时返回空字符串。先尝试解成字符串，再尝试解成带 message 或 msg 的对象，都不行则返回去掉空白的原文。
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

// probeURL 按供应商或协议给非聊天探测补上路径。端点为空返回空串。
// c 的 Endpoint 去掉空白和末尾斜杠。空则返回空串。Vendor 和 Protocol 转成小写后决定要不要补路径。
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
	case v == "removebg" || p == "removebg":
		if strings.Contains(base, "/v1.0") {
			return base + "/account"
		}
		return base + "/v1.0/account"
	default:
		return base
	}
}

// applyProbeAuth 按供应商把钥匙放进 Authorization、查询参数或专用头。钥匙为空则不改请求。
// req 为 nil 或 key 去掉空白后为空时直接返回。否则按供应商改 req 的头或查询。
// c 的 Vendor 和 Protocol 用来选择怎么放钥匙。默认分支还把 c 的 Vendor 和 Endpoint 传给 chatVendorUsesGoogleKey。
// key 去掉空白后为空时直接返回。否则写入 Authorization、查询参数或专用头。
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
	case v == "removebg" || p == "removebg":
		req.Header.Set("X-Api-Key", key)
	case v == "photoroom" || p == "photoroom":
		req.Header.Set("x-api-key", key)
	case v == "azure" || v == "microsoft":
		req.Header.Set("Ocp-Apim-Subscription-Key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
		if chatVendorUsesGoogleKey(c.Vendor, c.Endpoint) {
			req.Header.Set("x-goog-api-key", key)
		}
	}
}

// chatVendorUsesGoogleKey 在供应商是 gemini 或 google，或端点包含 generativelanguage.googleapis.com 时返回真。
// vendor 去掉空白并转小写。是 gemini 或 google 时返回真。
// base 转成小写后含 generativelanguage.googleapis.com 时返回真。
func chatVendorUsesGoogleKey(vendor, base string) bool {
	v := strings.ToLower(strings.TrimSpace(vendor))
	if v == "gemini" || v == "google" {
		return true
	}
	return strings.Contains(strings.ToLower(base), "generativelanguage.googleapis.com")
}

// truncateProbe 按字节把字符串截到 n。本文件没有调用它。
// s 的字节长度不大于 n 时原样返回，否则截到前 n 个字节。
func truncateProbe(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// needsKey 对 cas 和 otel 返回假。搜索里 tavily、brave、serper、bocha、exa 要钥匙，其余搜索协议不要。其他协议要钥匙。
func needsKey(c Connection) bool {
	if c.Protocol == "cas" {
		return false
	}
	if HasCap(c, CapOTEL) {
		return false
	}
	if HasCap(c, CapSearch) {
		switch strings.ToLower(strings.TrimSpace(c.Protocol)) {
		case "tavily", "brave", "serper", "bocha", "exa":
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
