package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/im"
)

// TestQuotedCode 确认从我们自己写的消息里认出唯一的短编号；同一个编号写两遍算一个，不同编号并存时不认。
func TestQuotedCode(t *testing.T) {
	cases := []struct {
		text, kind, code string
	}{
		{"任务「改登录」在等你：\n先改哪个？\n回 /答 abc 你的回答", "ticket", "abc"},
		{"需要你批准：rm\n回 /批准 k2m · /拒绝 k2m（/批准 k2m 总是：同类以后不再问）", "approval", "k2m"},
		{"明天九点有会\n回 /有用 x7p · /别提醒 x7p", "speak", "x7p"},
		{"Reply /approve q4w · /deny q4w (/approve q4w always: don't ask again)", "approval", "q4w"},
		{"有 2 件在等你：\n· 待批准：rm  /批准 abc\n· 在等你：改登录  /答 def", "", ""},
		{"全文在桌面的「今天」里。回 /状态 看还有什么在等你。", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		kind, code := quotedCode(c.text)
		if kind != c.kind || code != c.code {
			t.Errorf("quotedCode(%q) = %q %q, want %q %q", c.text, kind, code, c.kind, c.code)
		}
	}
}

// TestIMQuoteRemembersPushedCodes 确认发到主通道的一条按平台编号记下短编号；平台编号对不上时从被引用的原文里认。
func TestIMQuoteRemembersPushedCodes(t *testing.T) {
	a, _ := boundApp(t)
	n := 0
	var sent []string
	a.imh.pushID = func(_, _, text string) (string, error) {
		n++
		sent = append(sent, text)
		return fmt.Sprintf("pm-%d", n), nil
	}
	code := a.imIssue(imCode{Kind: "approval", Target: "o1"})
	if err := a.imPush(a.imHomeNow(), approvalIMText(capability.Offer{ID: "o1"}, code, true)); err != nil {
		t.Fatal(err)
	}
	if kind, got := a.imQuotedCode(im.Inbound{AccountID: "lark-1", QuoteID: "pm-1"}); kind != "approval" || got != code {
		t.Fatalf("by platform id = %q %q, want approval %q", kind, got, code)
	}
	if _, got := a.imQuotedCode(im.Inbound{AccountID: "other", QuoteID: "pm-1"}); got != "" {
		t.Fatal("another account's message id must not match")
	}
	if kind, got := a.imQuotedCode(im.Inbound{AccountID: "lark-1", QuoteID: "nope", QuoteText: "回 /答 abc 你的回答"}); kind != "ticket" || got != "abc" {
		t.Fatalf("by quoted text = %q %q", kind, got)
	}
	// 一条里有两个编号的（例如 /状态 的清单）不记。
	a.imRemember("lark-1", "pm-9", "/批准 abc\n/答 def")
	if _, got := a.imQuotedCode(im.Inbound{AccountID: "lark-1", QuoteID: "pm-9"}); got != "" {
		t.Fatal("a message with several codes must not be remembered")
	}
	if len(sent) != 1 {
		t.Fatalf("sent %q", sent)
	}
}

// TestIMQuoteRoute 确认引用回复只认主通道里绑定的人；回复审批的闲话照旧进对话；编号已经处理过时回一句而不是进对话。
func TestIMQuoteRoute(t *testing.T) {
	a, _ := boundApp(t)
	a.imh.pushID = func(_, _, _ string) (string, error) { return "pm-1", nil }
	code := a.imIssue(imCode{Kind: "approval", Target: "o1"})
	if err := a.imPush(a.imHomeNow(), approvalIMText(capability.Offer{ID: "o1"}, code, true)); err != nil {
		t.Fatal(err)
	}
	var got []string
	in := im.Inbound{AccountID: "lark-1", ChatID: "oc_1", SenderID: "u1", QuoteID: "pm-1",
		Reply: func(s string) error { got = append(got, s); return nil }}

	in.Text = "这个命令是干什么的？"
	if a.imQuoteRoute(in) {
		t.Fatal("a question about an approval should go to the conversation")
	}
	stranger := in
	stranger.SenderID = "u2"
	stranger.Text = "批准"
	if a.imQuoteRoute(stranger) {
		t.Fatal("only the bound person can approve by reply")
	}
	// 没有审批门时按「找不到」处理：回一句，不进对话。
	in.Text = "批准。"
	if !a.imQuoteRoute(in) || len(got) == 0 || !strings.Contains(got[len(got)-1], "已经处理过") {
		t.Fatalf("approve by reply: %q", got)
	}
	// 引用了一张已经过期的任务：整句是回答，但编号找不到，回一句。
	ticket := im.Inbound{AccountID: "lark-1", ChatID: "oc_1", SenderID: "u1", QuoteText: "回 /答 zzz 你的回答", Text: "先改登录页",
		Reply: func(s string) error { got = append(got, s); return nil }}
	if !a.imQuoteRoute(ticket) || !strings.Contains(got[len(got)-1], "已经处理过") {
		t.Fatalf("expired ticket: %q", got)
	}
	// 没有引用的话不归这里管。
	plain := im.Inbound{AccountID: "lark-1", ChatID: "oc_1", SenderID: "u1", Text: "批准"}
	if a.imQuoteRoute(plain) {
		t.Fatal("a message without a quote is not a reply")
	}
}

// TestReplyWords 确认短回复的词表。
func TestReplyWords(t *testing.T) {
	for in, want := range map[string]string{"批准": "once", " OK！": "once", "总是": "always", "不行。": "deny", "为什么要删": ""} {
		if got := approvalWord(in); got != want {
			t.Errorf("approvalWord(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"有用": "used", "没用": "ignore", "别提醒": "later", "关掉": "mute", "明天呢": ""} {
		if got := feedbackWord(in); got != want {
			t.Errorf("feedbackWord(%q) = %q, want %q", in, got, want)
		}
	}
}
