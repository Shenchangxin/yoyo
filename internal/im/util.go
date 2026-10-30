package im

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func modeOf(a Account, def string) string {
	m := strings.ToLower(cfg(a, "connection_mode"))
	if m == "" {
		return def
	}
	return m
}

func joinURL(base, path string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") + "/" + strings.TrimLeft(path, "/")
}

func clipText(s string) string {
	return clipRunes(strings.TrimSpace(s), 3500)
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if n > 0 && len(r) > n {
		return string(r[:n])
	}
	return string(r)
}

func placeholder(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "text":
		return ""
	case "image", "photo", "sticker", "picture", "img":
		return "[image]"
	case "voice", "audio", "record":
		return "[voice]"
	case "video", "shortvideo":
		return "[video]"
	case "file", "document", "attachment":
		return "[file]"
	default:
		return "[" + kind + "]"
	}
}

func joinParts(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(p)
	}
	return b.String()
}

func decodeJSON(b []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func asMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		if t == nil {
			return map[string]any{}
		}
		return t
	case string:
		m, err := decodeJSON([]byte(t))
		if err != nil {
			return map[string]any{}
		}
		return m
	default:
		return map[string]any{}
	}
}

func asList(v any) []any {
	list, _ := v.([]any)
	return list
}

func dig(m map[string]any, path ...string) any {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}

func jstr(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func jnum(v any) int {
	switch t := v.(type) {
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			f, _ := t.Float64()
			return int(f)
		}
		return int(i)
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		return 0
	}
}

func jbool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	case json.Number:
		return t.String() != "0" && t.String() != ""
	case float64:
		return t != 0
	default:
		return false
	}
}

func numOrString(s string) any {
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return json.Number(s)
	}
	return s
}

func sigEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func reqID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func replyCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 45*time.Second)
}

func (rt *Runtime) takeToken(key string) (string, bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	t, ok := rt.tokens[key]
	if !ok || !time.Now().Before(t.until) || t.value == "" {
		return "", false
	}
	return t.value, true
}

func (rt *Runtime) keepToken(key, value string, ttl time.Duration) {
	if value == "" {
		return
	}
	if ttl > 2*time.Minute {
		ttl -= time.Minute
	}
	if ttl < 30*time.Second {
		ttl = 30 * time.Second
	}
	rt.mu.Lock()
	rt.tokens[key] = struct {
		value string
		until time.Time
	}{value, time.Now().Add(ttl)}
	rt.mu.Unlock()
}

func statusErr(code int, body []byte, what string) error {
	msg := fmt.Sprintf("%s http %d: %s", what, code, clipRunes(strings.TrimSpace(string(body)), 180))
	if code == 401 || code == 403 {
		return fatalf("%s", msg)
	}
	return fmt.Errorf("%s", msg)
}

func (rt *Runtime) wsURL(name, fallback string) string {
	b := rt.rootURL(name, fallback)
	b = strings.Replace(b, "https://", "wss://", 1)
	b = strings.Replace(b, "http://", "ws://", 1)
	return b
}

func dialWS(ctx context.Context, raw string, header http.Header) (*websocket.Conn, error) {
	c, _, err := websocket.Dial(ctx, raw, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(1 << 20)
	return c, nil
}

func readWS(ctx context.Context, c *websocket.Conn) ([]byte, error) {
	_, b, err := c.Read(ctx)
	return b, err
}

func closeWS(c *websocket.Conn) {
	if c != nil {
		_ = c.Close(websocket.StatusNormalClosure, "")
	}
}

func gatewayFatal(err error) error {
	var ce websocket.CloseError
	if errors.As(err, &ce) {
		switch ce.Code {
		case 4004, 4010, 4011, 4012, 4013, 4014:
			return fatalf("gateway closed (%d): %s", ce.Code, ce.Reason)
		}
	}
	return err
}

func pumpJSON(ctx context.Context, sock *liveSocket, every time.Duration, frame func() any) {
	if sock == nil || every <= 0 || frame == nil {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = sock.write(ctx, frame())
		}
	}
}

func writePlain(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func writeJSONCode(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func (rt *Runtime) send(ctx context.Context, method, rawURL string, headers map[string]string, body any) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, r)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return rt.do(req)
}

func withQuery(raw string, kv ...string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for i := 0; i+1 < len(kv); i += 2 {
		q.Set(kv[i], kv[i+1])
	}
	u.RawQuery = q.Encode()
	return u.String()
}
