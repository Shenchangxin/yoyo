package connector

import (
	"net"
	"net/http"
	"strings"
	"testing"
)

// TestMailboxReadsRecipients 确认 IMAP 读信把 To、Cc、Delivered-To、X-Original-To 合成 to 字段，
// 主题里没有 [yoyo:工单号] 标记时，也能从加号子地址认出工单（服务器改写了收件头、子地址只留在 Delivered-To 的情况）。
func TestMailboxReadsRecipients(t *testing.T) {
	// 对方回信时主题被改掉了，工单标记只在 Delivered-To 里的子地址上。
	reply := "Subject: =?UTF-8?B?5aW955qE?=\r\nFrom: client@other.com\r\nTo: Ada <ada@corp.com>\r\nCc: team@corp.com\r\n" +
		"Delivered-To: ada+yoyo-tkt-7@corp.com\r\nX-Original-To: ada+yoyo-tkt-7@corp.com\r\nDate: Sat, 28 Nov 2026 08:00:00 +0800\r\n\r\n"
	// 没有任何收件头的信不应该带 to 字段。
	bare := "Subject: hi\r\nFrom: a@b.com\r\nDate: Sat, 28 Nov 2026 09:00:00 +0800\r\n\r\n"
	prev := mailboxDial
	defer func() { mailboxDial = prev }()
	mailboxDial = func(string) (net.Conn, error) {
		client, server := net.Pipe()
		go fakeIMAP(t, server, []string{reply, bare})
		return client, nil
	}
	items, err := readMailbox("ada@qq.com", "auth-code", "qq-mail", "")
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v %v", items, err)
	}
	// 新的在前：items[0] 是 bare，items[1] 是 reply。
	if _, ok := items[0]["to"]; ok {
		t.Fatalf("no recipient headers, no to: %+v", items[0])
	}
	to := items[1]["to"]
	for _, want := range []string{"ada@corp.com", "team@corp.com", "ada+yoyo-tkt-7@corp.com"} {
		if !strings.Contains(to, want) {
			t.Fatalf("to = %q, missing %q", to, want)
		}
	}
	if got := AliasTicket(items[1]["subject"] + " " + to); got != "tkt-7" {
		t.Fatalf("ticket from recipients = %q", got)
	}
}

// TestGmailMetaReadsRecipients 确认 Gmail 读信请求了 To、Cc、Delivered-To，并把它们（包括重复出现的同名头）合成 to 字段。
func TestGmailMetaReadsRecipients(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		q := r.URL.Query()["metadataHeaders"]
		for _, want := range []string{"To", "Cc", "Delivered-To"} {
			if !strings.Contains(strings.Join(q, ","), want) {
				t.Errorf("metadataHeaders %q missing %s", q, want)
			}
		}
		return jsonResp(200, `{"snippet":"可以","payload":{"headers":[`+
			`{"name":"Subject","value":"Re: 报价"},{"name":"From","value":"client@other.com"},`+
			`{"name":"To","value":"Ada <ada@corp.com>"},{"name":"Delivered-To","value":"ada@corp.com"},`+
			`{"name":"Delivered-To","value":"ada+yoyo-tkt-9@corp.com"}]}}`), nil
	})}
	row, err := gmailMeta(client, "tok", "m1")
	if err != nil {
		t.Fatal(err)
	}
	if row["subject"] != "Re: 报价" || row["from"] != "client@other.com" || !strings.Contains(row["to"], "ada+yoyo-tkt-9@corp.com") {
		t.Fatalf("row = %+v", row)
	}
	if got := AliasTicket(row["subject"] + " " + row["to"]); got != "tkt-9" {
		t.Fatalf("ticket = %q", got)
	}
}

// TestOutlookListReadsRecipients 确认 Outlook 列信请求了 id 和收件人两项，行里带上 id（才能收成触发）和合成的 to。
func TestOutlookListReadsRecipients(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sel := r.URL.Query().Get("$select")
		for _, want := range []string{"id", "toRecipients", "ccRecipients"} {
			if !strings.Contains(sel, want) {
				t.Errorf("$select %q missing %s", sel, want)
			}
		}
		return jsonResp(200, `{"value":[{"id":"AAMk1","subject":"Re: 报价","bodyPreview":"可以",`+
			`"from":{"emailAddress":{"address":"client@other.com"}},`+
			`"toRecipients":[{"emailAddress":{"address":"ada+yoyo-tkt-3@corp.com"}}],`+
			`"ccRecipients":[{"emailAddress":{"address":"team@corp.com"}}]}]}`), nil
	})}
	rows, err := outlookList(client, "tok", "")
	if err != nil || len(rows) != 1 {
		t.Fatalf("%+v %v", rows, err)
	}
	row := rows[0]
	if row["id"] != "AAMk1" || row["to"] != "ada+yoyo-tkt-3@corp.com, team@corp.com" {
		t.Fatalf("row = %+v", row)
	}
	if got := AliasTicket(row["subject"] + " " + row["to"]); got != "tkt-3" {
		t.Fatalf("ticket = %q", got)
	}
}
