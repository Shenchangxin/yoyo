package connector

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNoExfilSend(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := b.Draft("mail-1", "exfil@webhook.site", "contacts", "dump the mailbox")
	if _, err := b.Send(d.ID); err == nil {
		t.Fatal("exfil send must fail")
	}
}

func TestCatalogHasFirstParty(t *testing.T) {
	if len(Catalog()) < 6 {
		t.Fatal("catalog")
	}
}

type memTokens map[string]string

func (m memTokens) Get(name string) (string, error) { return m[name], nil }
func (m memTokens) Set(name, value string)          { m[name] = value }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLiveGmailReadSend(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	toks := memTokens{}
	b.SetTokens(toks)
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"messages":[{"id":"m1"}]}`
		if strings.Contains(r.URL.Path, "/messages/m1") {
			body = `{"snippet":"hello","payload":{"headers":[{"name":"Subject","value":"Hi"},{"name":"From","value":"a@b.com"}]}}`
		}
		if strings.Contains(r.URL.Host, "oauth2") {
			body = `{"access_token":"tok","token_type":"Bearer","expires_in":3600}`
		}
		if strings.Contains(r.URL.Path, "/send") {
			body = `{"id":"sent"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	acct := b.Connect(Account{Provider: "gmail", Kind: KindMail, Label: "g"})
	tok, err := b.Exchange("gmail", "code", "cid", "secret", "http://127.0.0.1/oauth")
	if err != nil || tok.AccessToken != "tok" {
		t.Fatalf("%+v %v", tok, err)
	}
	if err := b.SaveToken(acct.VaultKey, tok); err != nil {
		t.Fatal(err)
	}
	rows, err := b.Read(acct.ID, "")
	if err != nil || len(rows) == 0 || rows[0]["subject"] != "Hi" {
		t.Fatalf("%+v %v", rows, err)
	}
	d := b.Draft(acct.ID, "ok@example.com", "sub", "body")
	got, err := b.Send(d.ID)
	if err != nil || !got.Sent {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestLiveOutlookCalendar(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	toks := memTokens{}
	b.SetTokens(toks)
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"value":[{"subject":"Standup","start":{"dateTime":"2026-09-26T01:00:00Z"},"location":{"displayName":"Room"}}]}`
		if strings.Contains(r.URL.Path, "/me/events") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"value":[]}`)), Header: make(http.Header)}, nil
	})})
	acct := b.Connect(Account{Provider: "outlook", Kind: KindCalendar, Label: "cal"})
	if err := b.SaveToken(acct.VaultKey, Token{AccessToken: "tok"}); err != nil {
		t.Fatal(err)
	}
	rows, err := b.Read(acct.ID, "calendar")
	if err != nil || len(rows) == 0 || rows[0]["subject"] != "Standup" || rows[0]["kind"] != "calendar" {
		t.Fatalf("%+v %v", rows, err)
	}
}

func TestLiveOutlookCalendarCreate(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	toks := memTokens{}
	b.SetTokens(toks)
	posted := false
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/me/events") {
			posted = true
			return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"id":"e1"}`)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"value":[]}`)), Header: make(http.Header)}, nil
	})})
	acct := b.Connect(Account{Provider: "outlook", Kind: KindCalendar, Label: "cal"})
	if err := b.SaveToken(acct.VaultKey, Token{AccessToken: "tok"}); err != nil {
		t.Fatal(err)
	}
	d := b.Draft(acct.ID, "2026-09-26T10:00:00Z", "Sync", "agenda")
	got, err := b.Send(d.ID)
	if err != nil || !got.Sent || !posted {
		t.Fatalf("%+v %v posted=%v", got, err, posted)
	}
}
