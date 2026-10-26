// httpx.go 拼接供应商地址，并收 JSON 或原始响应。

package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// joinURL 去掉 base 末尾的斜杠。prefix 非空、base 还没以它结尾、也不包含 prefix 加斜杠时，把 prefix 接到 base 后面。suffix 为空就返回 base。suffix 以 http 开头时原样返回。其他 suffix 前面补上斜杠再拼到 base。
func joinURL(base, prefix, suffix string) string {
	base = strings.TrimRight(base, "/")
	if prefix != "" && !strings.HasSuffix(base, prefix) && !strings.Contains(base, prefix+"/") {
		base += prefix
	}
	if suffix == "" {
		return base
	}
	if strings.HasPrefix(suffix, "http") {
		return suffix
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return base + suffix
}

// httpGet 构造 GET。路径为空或只有斜杠时改成 models。key 非空时同时写上 Authorization Bearer 和 x-goog-api-key。
func httpGet(raw, key string) (*http.Request, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = path.Join(u.Path, "models")
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("x-goog-api-key", key)
	}
	return req, nil
}

// doRaw 用 Engine.HTTP 发请求，客户端为空时用 http.DefaultClient。limit 小于等于 0 时按 32 MiB 截断正文。状态码大于等于 400 时返回正文或状态文本作为错误，同时仍带回状态码、Content-Type 和正文。
func (e *Engine) doRaw(req *http.Request, limit int64) (int, string, []byte, error) {
	client := e.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	if limit <= 0 {
		limit = 32 << 20
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, limit))
	if err != nil {
		return res.StatusCode, res.Header.Get("Content-Type"), nil, err
	}
	if res.StatusCode >= 400 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = res.Status
		}
		return res.StatusCode, res.Header.Get("Content-Type"), raw, fmt.Errorf("%s", msg)
	}
	return res.StatusCode, res.Header.Get("Content-Type"), raw, nil
}

// doJSON 读取至多 32 MiB。正文以 { 开头时按对象解析。状态码大于等于 400 时依次取 error 字符串、error.message、message，都没有再用状态文本。成功且没有对象时返回空 map。
func (e *Engine) doJSON(req *http.Request) (map[string]any, int, []byte, error) {
	client := e.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return nil, res.StatusCode, nil, err
	}
	var m map[string]any
	if len(raw) > 0 && raw[0] == '{' {
		_ = json.Unmarshal(raw, &m)
	}
	if res.StatusCode >= 400 {
		msg := strings.TrimSpace(string(raw))
		if m != nil {
			if e, ok := m["error"].(string); ok && e != "" {
				msg = e
			} else if em, ok := m["error"].(map[string]any); ok {
				if s, _ := em["message"].(string); s != "" {
					msg = s
				}
			} else if s, _ := m["message"].(string); s != "" {
				msg = s
			}
		}
		if msg == "" {
			msg = res.Status
		}
		return m, res.StatusCode, raw, fmt.Errorf("%s", msg)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, res.StatusCode, raw, nil
}

// jsonReq 把 body 编成 JSON。body 为 nil 时没有正文。Content-Type 是 application/json。key 非空时写上 Authorization Bearer。
func jsonReq(method, url, key string, body any) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return req, nil
}

// strAny 在给定键里找第一个非空字符串。值如果是对象，再从 url、video_url、image_url、message 里找。
func strAny(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if t != "" {
					return t
				}
			case map[string]any:
				if s := strAny(t, "url", "video_url", "image_url", "message"); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// firstURL 从字符串、对象或切片的第一项里取出 URL。对象看 url、video_url 和 image_url。
func firstURL(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		return strAny(t, "url", "video_url", "image_url")
	case []any:
		if len(t) > 0 {
			return firstURL(t[0])
		}
	}
	return ""
}

// pickData 优先返回 data，其次 output，再次 content。都不是对象时返回 m 本身。
func pickData(m map[string]any) map[string]any {
	if d, ok := m["data"].(map[string]any); ok {
		return d
	}
	if o, ok := m["output"].(map[string]any); ok {
		return o
	}
	if c, ok := m["content"].(map[string]any); ok {
		return c
	}
	return m
}
