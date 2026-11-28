package connector

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func jsonResp(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Status: http.StatusText(code), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

// TestRefreshBeforeExpiry 确认快过期的令牌先换票，新令牌写回保险库，应用身份留着。
func TestRefreshBeforeExpiry(t *testing.T) {
	b, _ := Open(t.TempDir())
	toks := memTokens{}
	b.SetTokens(toks)
	var refreshed atomic.Int32
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "oauth2") {
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "r1" || r.Form.Get("client_id") != "cid" {
				return jsonResp(400, `{"error":"bad"}`), nil
			}
			refreshed.Add(1)
			return jsonResp(200, `{"access_token":"new","expires_in":3600}`), nil
		}
		if r.Header.Get("Authorization") != "Bearer new" {
			return jsonResp(401, `{"error":{"status":"UNAUTHENTICATED"}}`), nil
		}
		return jsonResp(200, `{"items":[{"id":"e1","summary":"评审","start":{"dateTime":"2026-11-28T10:00:00Z"}}]}`), nil
	})})
	acct := b.Connect(Account{Provider: "gcal", Kind: KindCalendar, Label: "cal"})
	old := Token{AccessToken: "old", RefreshToken: "r1", Expiry: time.Now().Add(30 * time.Second)}
	old.keepApp(Token{ClientID: "cid", ClientSecret: "sec", Provider: "gcal"})
	_ = b.SaveToken(acct.VaultKey, old)
	rows, err := b.Read(acct.ID, "")
	if err != nil || len(rows) != 1 || rows[0]["subject"] != "评审" {
		t.Fatalf("%+v %v", rows, err)
	}
	if refreshed.Load() != 1 {
		t.Fatalf("refreshed %d times", refreshed.Load())
	}
	var saved Token
	_ = json.Unmarshal([]byte(toks[acct.VaultKey]), &saved)
	if saved.AccessToken != "new" || saved.RefreshToken != "r1" || saved.ClientID != "cid" || saved.Expiry.Before(time.Now().Add(time.Hour-time.Minute)) {
		t.Fatalf("saved = %+v", saved)
	}
}

// TestRefreshOn401 确认没写过期时间的令牌遇到 401 也会强制换一次再读。
func TestRefreshOn401(t *testing.T) {
	b, _ := Open(t.TempDir())
	toks := memTokens{}
	b.SetTokens(toks)
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "login.microsoftonline.com") {
			return jsonResp(200, `{"access_token":"new","refresh_token":"r2","expires_in":3600}`), nil
		}
		if r.Header.Get("Authorization") != "Bearer new" {
			return jsonResp(401, `{"error":{"code":"InvalidAuthenticationToken"}}`), nil
		}
		return jsonResp(200, `{"value":[{"subject":"Hi","from":{"emailAddress":{"address":"a@b.com"}}}]}`), nil
	})})
	acct := b.Connect(Account{Provider: "outlook", Kind: KindMail, Label: "o"})
	tok := Token{AccessToken: "old", RefreshToken: "r1"}
	tok.keepApp(Token{ClientID: "cid", Provider: "outlook"})
	_ = b.SaveToken(acct.VaultKey, tok)
	rows, err := b.Read(acct.ID, "")
	if err != nil || len(rows) != 1 || rows[0]["subject"] != "Hi" {
		t.Fatalf("%+v %v", rows, err)
	}
	var saved Token
	_ = json.Unmarshal([]byte(toks[acct.VaultKey]), &saved)
	if saved.RefreshToken != "r2" {
		t.Fatalf("microsoft rotates the refresh token, saved = %+v", saved)
	}
}

// TestRevokedGrantSaysReauthorize 确认刷新令牌被撤销时说清要重新授权。
func TestRevokedGrantSaysReauthorize(t *testing.T) {
	b, _ := Open(t.TempDir())
	b.SetTokens(memTokens{})
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResp(400, `{"error":"invalid_grant"}`), nil
	})})
	acct := b.Connect(Account{Provider: "gmail", Kind: KindMail, Label: "Gmail"})
	tok := Token{AccessToken: "old", RefreshToken: "r1", Expiry: time.Now().Add(-time.Hour)}
	tok.keepApp(Token{ClientID: "cid", Provider: "gmail"})
	_ = b.SaveToken(acct.VaultKey, tok)
	if _, err := b.Read(acct.ID, ""); err == nil || !strings.Contains(err.Error(), "重新授权") {
		t.Fatalf("err = %v", err)
	}
}

// TestCatalogReadiness 确认目录给每一项标了能不能直接用。
func TestCatalogReadiness(t *testing.T) {
	want := map[string]string{
		"gmail": "live", "gcal": "live", "slack": "live", "qq-mail": "live", "win-desktop": "live", "notion": "live",
		"wps": "endpoint", "feishu": "elsewhere", "dingtalk": "elsewhere", "ios-health": "device", "mac-calendar": "device",
		"caldav": "live", "feishu-calendar": "live",
	}
	for _, e := range Catalog() {
		if w, ok := want[e.Provider]; ok && e.Ready != w {
			t.Errorf("%s ready = %q, want %q", e.Provider, e.Ready, w)
		}
		if e.Ready == "" {
			t.Errorf("%s has no readiness", e.Provider)
		}
	}
}

// fakeIMAP 是只认这一套对话的假服务器。
func fakeIMAP(t *testing.T, conn net.Conn, headers []string) {
	t.Helper()
	defer conn.Close()
	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	say := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }
	say("* OK ready")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		tag, cmd := fields[0], strings.ToUpper(fields[1])
		switch cmd {
		case "LOGIN", "ID":
			say(tag + " OK done")
		case "SELECT":
			say("* 25 EXISTS")
			say(tag + " OK [READ-WRITE] SELECT")
		case "FETCH":
			if fields[2] != "6:25" {
				say(tag + " BAD range " + fields[2])
				continue
			}
			for i, h := range headers {
				seq := 24 + i
				_, _ = w.WriteString("* " + itoa(seq) + " FETCH (UID " + itoa(seq*10) + " BODY[HEADER.FIELDS (SUBJECT FROM DATE MESSAGE-ID)] {" + itoa(len(h)) + "}\r\n" + h + ")\r\n")
			}
			say(tag + " OK FETCH done")
		case "LOGOUT":
			say("* BYE")
			return
		}
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// TestMailboxReadsNewestDecoded 确认读的是最新那一段，新的在前，编码过的标题和发件人解开了，搜索词能筛。
func TestMailboxReadsNewestDecoded(t *testing.T) {
	older := "Subject: =?UTF-8?B?5ZGo5oql?=\r\nFrom: =?GBK?B?1cXI/Q==?= <zhang@qq.com>\r\nDate: Fri, 27 Nov 2026 09:00:00 +0800\r\n\r\n"
	newer := "Subject: =?gb2312?B?u+HS6c2o1qo=?=\r\nFrom: boss@corp.com\r\nDate: Sat, 28 Nov 2026 08:00:00 +0800\r\nMessage-ID: <m2@corp.com>\r\n\r\n"
	prev := mailboxDial
	defer func() { mailboxDial = prev }()
	mailboxDial = func(host string) (net.Conn, error) {
		if host != "imap.qq.com" {
			t.Errorf("host = %s", host)
		}
		client, server := net.Pipe()
		go fakeIMAP(t, server, []string{older, newer})
		return client, nil
	}
	items, err := readMailbox("me@qq.com", "auth-code", "qq-mail", "")
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v %v", items, err)
	}
	if items[0]["subject"] != "会议通知" || items[0]["message_id"] != "<m2@corp.com>" || items[0]["id"] != "250" {
		t.Fatalf("newest first, decoded gb2312: %+v", items[0])
	}
	if items[1]["subject"] != "周报" || items[1]["from"] != "张三 <zhang@qq.com>" || items[1]["date"] != "2026-11-27T01:00:00Z" {
		t.Fatalf("utf-8 subject and gbk name: %+v", items[1])
	}
	filtered, err := readMailbox("me@qq.com", "auth-code", "qq-mail", "周报")
	if err != nil || len(filtered) != 1 || filtered[0]["subject"] != "周报" {
		t.Fatalf("query: %+v %v", filtered, err)
	}
}
