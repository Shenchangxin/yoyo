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

func (e *Engine) doJSON(req *http.Request) (map[string]any, int, []byte, error) {
	res, err := e.HTTP.Do(req)
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
