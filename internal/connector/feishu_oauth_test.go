package connector

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestFeishuAuthURLAsksForOfflineAccess 确认飞书日历的授权地址带上日历权限和 offline_access（没有它拿不到刷新令牌）。
func TestFeishuAuthURLAsksForOfflineAccess(t *testing.T) {
	b, _ := Open(t.TempDir())
	raw, state, err := b.AuthURL("feishu-calendar", "cli_a1", "http://127.0.0.1:3080/oauth")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	q := u.Query()
	if !strings.HasPrefix(raw, feishuAuthorizeURL+"?") || q.Get("client_id") != "cli_a1" || q.Get("state") != state ||
		!strings.Contains(q.Get("scope"), "offline_access") || !strings.Contains(q.Get("scope"), "calendar:calendar") {
		t.Fatalf("auth url = %s", raw)
	}
}

// TestFeishuExchangeAndRefreshUseJSON 确认授权码换票和刷新都按 JSON 发到飞书 v2 换票地址；
// 刷新后飞书给的新刷新令牌和应用身份都写回保险库；日历接口回 401 时强制换一次再读成功。
func TestFeishuExchangeAndRefreshUseJSON(t *testing.T) {
	prev := feishuOpenBase
	defer func() { feishuOpenBase = prev }()
	feishuOpenBase = "https://feishu.test"

	var grants []map[string]string
	b, _ := Open(t.TempDir())
	toks := memTokens{}
	b.SetTokens(toks)
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/open-apis/authen/v2/oauth/token" {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				t.Errorf("token request content-type = %q", r.Header.Get("Content-Type"))
			}
			raw, _ := io.ReadAll(r.Body)
			var m map[string]string
			_ = json.Unmarshal(raw, &m)
			grants = append(grants, m)
			if m["grant_type"] == "authorization_code" {
				return jsonResp(200, `{"code":0,"access_token":"u1","refresh_token":"r1","expires_in":7200}`), nil
			}
			return jsonResp(200, `{"code":0,"access_token":"u2","refresh_token":"r2","expires_in":7200}`), nil
		}
		// 日历接口：旧令牌报 99991677（令牌过期），新令牌读到一条。
		if r.Header.Get("Authorization") != "Bearer u2" {
			return jsonResp(200, `{"code":99991677,"msg":"token expired"}`), nil
		}
		return jsonResp(200, `{"code":0,"data":{"items":[{"event_id":"e1","summary":"评审","start_time":{"timestamp":"1795999200"}}]}}`), nil
	})})

	tok, err := b.Exchange("feishu-calendar", "code-1", "cli_a1", "sec", "http://127.0.0.1:3080/oauth")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "u1" || tok.RefreshToken != "r1" || tok.ClientID != "cli_a1" || tok.Expiry.Before(time.Now().Add(time.Hour)) {
		t.Fatalf("exchanged = %+v", tok)
	}
	if g := grants[0]; g["code"] != "code-1" || g["client_id"] != "cli_a1" || g["client_secret"] != "sec" || g["redirect_uri"] == "" {
		t.Fatalf("exchange body = %+v", g)
	}

	acct := b.Connect(Account{Provider: "feishu-calendar", Kind: KindCalendar, Label: "飞书日历", Endpoint: "cal_1"})
	_ = b.SaveToken(acct.VaultKey, tok)
	rows, err := b.Read(acct.ID, "")
	if err != nil || len(rows) != 1 || rows[0]["id"] != "e1" {
		t.Fatalf("read after forced refresh = %+v %v", rows, err)
	}
	if len(grants) != 2 || grants[1]["grant_type"] != "refresh_token" || grants[1]["refresh_token"] != "r1" {
		t.Fatalf("refresh body = %+v", grants)
	}
	var saved Token
	_ = json.Unmarshal([]byte(toks[acct.VaultKey]), &saved)
	if saved.AccessToken != "u2" || saved.RefreshToken != "r2" || saved.ClientID != "cli_a1" || saved.ClientSecret != "sec" {
		t.Fatalf("feishu rotates the refresh token; saved = %+v", saved)
	}
}
