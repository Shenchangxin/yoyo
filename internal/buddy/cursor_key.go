// cursor_key.go 拥有 Cursor API key 的预检。Cursor 的 ACP 把「环境里有 CURSOR_API_KEY」直接当成已登录，
// 钥匙其实无效时 session/new 不回错误、一直等到探测超时。启动前先问兑换接口，没通过就不把钥匙交给进程。

package buddy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// cursorAPIKey 是 Cursor 认的钥匙变量。
	cursorAPIKey = "CURSOR_API_KEY"
	// cursorKeyExchangeURL 用钥匙换访问令牌。401 表示钥匙无效，2xx 表示能用。
	cursorKeyExchangeURL = "https://api2.cursor.sh/auth/exchange_user_api_key"
	// cursorKeyCheckTimeout 是一次预检的上限，短于探测的 20 秒，免得预检自己把探测吃完。
	cursorKeyCheckTimeout = 8 * time.Second
	cursorKeyOKTTL        = 10 * time.Minute
	cursorKeyNoTTL        = 10 * time.Minute
	cursorKeyErrTTL       = 15 * time.Second
)

// cursorVerdict 是一把钥匙的缓存结论。err 非空只表示没问成，不保存对方的原文。
type cursorVerdict struct {
	ok  bool
	err bool
	at  time.Time
	ttl time.Duration
}

// vetCursorKey 在钥匙没通过预检时把它从环境里拿掉，并记下给界面的那句话。通过或没配钥匙时清掉旧话。
func (m *Manager) vetCursorKey(id string, env map[string]string) map[string]string {
	key := strings.TrimSpace(env[cursorAPIKey])
	if key == "" {
		m.setCursorNote(id, "")
		return env
	}
	ctx, cancel := context.WithTimeout(m.ctx, cursorKeyCheckTimeout)
	defer cancel()
	ok, err := m.cursorKeyOK(ctx, key)
	if ok {
		m.setCursorNote(id, "")
		return env
	}
	delete(env, cursorAPIKey)
	if err != nil {
		m.setCursorNote(id, m.say(
			"连不上 Cursor 的登录服务，这把 API key 没能完成校验。",
			"Cursor's login service did not answer, so this API key could not be checked.",
		))
		return env
	}
	m.setCursorNote(id, m.say(
		"Cursor 没有接受这把 API key。请换成 Cursor 设置里的 key，或登录 Cursor。",
		"Cursor rejected this API key. Use a key from Cursor settings, or sign in.",
	))
	return env
}

// cursorNote 返回这项最近一次钥匙预检要给界面看的话。没有时是空串。
func (m *Manager) cursorNote(id string) string {
	m.keyMu.Lock()
	defer m.keyMu.Unlock()
	return m.cursorNotes[id]
}

// setCursorNote 记下或清掉界面上的那句话。
func (m *Manager) setCursorNote(id, note string) {
	m.keyMu.Lock()
	defer m.keyMu.Unlock()
	if m.cursorNotes == nil {
		m.cursorNotes = map[string]string{}
	}
	if note == "" {
		delete(m.cursorNotes, id)
		return
	}
	m.cursorNotes[id] = note
}

// cursorKeyOK 返回钥匙能不能用。结论按钥匙摘要缓存，换一把钥匙会重新问。
func (m *Manager) cursorKeyOK(ctx context.Context, key string) (bool, error) {
	sum := sha256.Sum256([]byte(key))
	id := hex.EncodeToString(sum[:])
	now := m.opts.Now()
	m.keyMu.Lock()
	if v, ok := m.cursorVerdicts[id]; ok && now.Sub(v.at) < v.ttl {
		m.keyMu.Unlock()
		if v.err {
			return false, fmt.Errorf("cursor key check unavailable")
		}
		return v.ok, nil
	}
	m.keyMu.Unlock()
	ok, err := m.askCursorKey(ctx, key)
	v := cursorVerdict{ok: ok, at: now, ttl: cursorKeyOKTTL}
	if err != nil {
		v.ok = false
		v.err = true
		v.ttl = cursorKeyErrTTL
	} else if !ok {
		v.ttl = cursorKeyNoTTL
	}
	m.keyMu.Lock()
	if m.cursorVerdicts == nil {
		m.cursorVerdicts = map[string]cursorVerdict{}
	}
	m.cursorVerdicts[id] = v
	m.keyMu.Unlock()
	return ok, err
}

// askCursorKey 问钥匙。测试注入了 CursorKeyCheck 时用它，否则打兑换接口。
func (m *Manager) askCursorKey(ctx context.Context, key string) (bool, error) {
	if m.opts.CursorKeyCheck != nil {
		return m.opts.CursorKeyCheck(ctx, key)
	}
	return exchangeCursorKey(ctx, m.opts.HTTP, cursorKeyExchangeURL, key)
}

// exchangeCursorKey 用钥匙换令牌。2xx 算通过，401 和 403 算拒绝，其余算没问成。
func exchangeCursorKey(ctx context.Context, client *http.Client, endpoint, key string) (bool, error) {
	if client == nil {
		client = http.DefaultClient
	}
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader("{}"))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	res, err := c.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return true, nil
	}
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return false, nil
	}
	return false, fmt.Errorf("cursor key exchange status %d", res.StatusCode)
}
