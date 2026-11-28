package connector

import (
	"strings"
	"testing"
)

func TestAliasTagsTicketMailAndReadsItBack(t *testing.T) {
	if got := SubAddress("ada+work@corp.com", "tkt-12"); got != "ada+yoyo-tkt-12@corp.com" {
		t.Fatalf("sub = %q", got)
	}
	if SubAddress("no-at", "tkt-1") != "" || SubAddress("a@b.com", "bad id") != "" {
		t.Fatal("bad input must give no alias")
	}
	acct := Account{ID: "gmail-mail-1", Provider: "gmail", Kind: KindMail, Label: "Ada <Ada@Corp.com>"}
	if AccountAddress(acct) != "ada@corp.com" {
		t.Fatalf("self = %q", AccountAddress(acct))
	}
	d := withAlias(Draft{Subject: "报价确认", Ticket: "tkt-12"}, acct, AccountAddress(acct))
	if d.Subject != "报价确认 [yoyo:tkt-12]" || d.Alias != "ada+yoyo-tkt-12@corp.com" {
		t.Fatalf("draft = %+v", d)
	}
	if again := withAlias(d, acct, "ada@corp.com"); again.Subject != d.Subject {
		t.Fatalf("tag added twice: %q", again.Subject)
	}
	if cal := withAlias(Draft{Subject: "周会", Ticket: "tkt-12"}, Account{Provider: "gcal", Kind: KindCalendar}, "ada@corp.com"); cal.Subject != "周会" || cal.Alias != "" {
		t.Fatalf("calendar touched: %+v", cal)
	}
	if plain := withAlias(Draft{Subject: "hi"}, acct, "ada@corp.com"); plain.Subject != "hi" {
		t.Fatal("no ticket, no tag")
	}
	raw := mimeMail(d)
	if !strings.Contains(raw, "Reply-To: ada+yoyo-tkt-12@corp.com\r\n") || !strings.Contains(raw, "Subject: 报价确认 [yoyo:tkt-12]\r\n") {
		t.Fatalf("mime = %q", raw)
	}
	for text, want := range map[string]string{
		"Re: 报价确认 [yoyo:tkt-12]":        "tkt-12",
		"to ada+yoyo-TKT-7@corp.com":    "tkt-7",
		"Re: 报价确认":                      "",
		"[yoyo:../../etc] path attempt": "",
	} {
		if got := AliasTicket(text); got != want {
			t.Fatalf("AliasTicket(%q) = %q, want %q", text, got, want)
		}
	}
}
