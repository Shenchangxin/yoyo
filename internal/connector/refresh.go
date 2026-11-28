// refresh.go 让 OAuth 令牌过期之后还能用：快过期时先换一张，接口回 401 时强制换一张再试一次。
// 换票要的应用身份在第一次换票时跟令牌一起放进 TokenStore（保险库），不进 connectors.json。
// 不拥有：第一次授权（oauth.go 的 AuthURL 和 live.go 的 Exchange）。

package connector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
)

// refreshEarly 是离过期还有多久就先换票。
const refreshEarly = 2 * time.Minute

// refreshMu 让同一时刻只有一次换票，免得两次读同时拿旧的刷新令牌去换，后一次被供应商作废。
var refreshMu sync.Mutex

// tokenEndpoint 是供应商换票的地址。不支持刷新的供应商为空。
func tokenEndpoint(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "gmail", "gcal":
		return "https://oauth2.googleapis.com/token"
	case "outlook":
		return "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	case "feishu-calendar", "feishu-docs", "feishu-task":
		// 飞书 v2 换票，请求体是 JSON（tokenRequest 负责编），每次都给一张新的刷新令牌，见 feishu_oauth.go。
		return feishuTokenURL()
	default:
		return ""
	}
}

// keepApp 把换票用的应用身份从 from 抄到 t 上。
func (t *Token) keepApp(from Token) {
	t.ClientID, t.ClientSecret, t.Provider = from.ClientID, from.ClientSecret, from.Provider
}

// freshToken 返回能用的令牌。force 为假时只在快过期时换；force 为真时一定换。
// 没有刷新令牌或应用身份时：不强制就原样返回；强制时返回错误，说清要重新授权。
func (b *Broker) freshToken(a Account, force bool) (Token, error) {
	tok, err := b.loadToken(a.VaultKey)
	if err != nil {
		return tok, err
	}
	if !force && (tok.Expiry.IsZero() || time.Until(tok.Expiry) > refreshEarly) {
		return tok, nil
	}
	endpoint := tokenEndpoint(a.Provider)
	if endpoint == "" || tok.RefreshToken == "" || tok.ClientID == "" {
		if force {
			return tok, fmt.Errorf("connector: %s 的授权过期了，请在「连接」里重新授权", firstNonEmpty(a.Label, a.Provider))
		}
		return tok, nil
	}
	refreshMu.Lock()
	defer refreshMu.Unlock()
	// 等锁的时候别人可能已经换过了。
	if again, err := b.loadToken(a.VaultKey); err == nil && again.AccessToken != tok.AccessToken && !again.Expiry.IsZero() && time.Until(again.Expiry) > refreshEarly {
		return again, nil
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tok.RefreshToken},
		"client_id":     {tok.ClientID},
	}
	if tok.ClientSecret != "" {
		form.Set("client_secret", tok.ClientSecret)
	}
	// 飞书日历的换票接口只收 JSON，其余供应商收表单，见 feishu_oauth.go。
	req, err := tokenRequest(a.Provider, endpoint, form)
	if err != nil {
		return tok, err
	}
	resp, err := b.client().Do(req)
	if err != nil {
		return tok, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		if strings.Contains(string(body), "invalid_grant") {
			return tok, fmt.Errorf("connector: %s 的授权被撤销或过期了，请在「连接」里重新授权", firstNonEmpty(a.Label, a.Provider))
		}
		return tok, fmt.Errorf("connector: refresh %s: %s", resp.Status, truncate(string(body), 200))
	}
	var next Token
	if err := json.Unmarshal(body, &next); err != nil || next.AccessToken == "" {
		return tok, fmt.Errorf("connector: refresh returned no access token")
	}
	var extra struct {
		ExpiresIn int `json:"expires_in"`
	}
	_ = json.Unmarshal(body, &extra)
	if extra.ExpiresIn > 0 {
		next.Expiry = time.Now().UTC().Add(time.Duration(extra.ExpiresIn) * time.Second)
	}
	// Google 换票时不再给刷新令牌，留着原来的。微软会给一张新的，用新的。
	if next.RefreshToken == "" {
		next.RefreshToken = tok.RefreshToken
	}
	next.keepApp(tok)
	if err := b.SaveToken(a.VaultKey, next); err != nil {
		return tok, err
	}
	return next, nil
}

// authExpired 在错误看起来是令牌失效时为真：401、UNAUTHENTICATED、InvalidAuthenticationToken。
func authExpired(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, word := range []string{"401", "unauthenticated", "invalidauthenticationtoken", "invalid_token", "token_expired", "invalid credentials"} {
		if strings.Contains(msg, word) {
			return true
		}
	}
	return false
}
