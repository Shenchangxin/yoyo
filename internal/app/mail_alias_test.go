package app

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// TestMailReplyDropsOwnCopyAndDeliversOnce 锁住：不带标记的信照旧收成触发；自己发出去的那份副本丢掉；
// 一封回信只交一次，交失败的下次还能再试。
func TestMailReplyDropsOwnCopyAndDeliversOnce(t *testing.T) {
	a := &App{Home: &home.Dir{Root: t.TempDir()}}
	acct := connector.Account{ID: "gmail-1", Provider: "gmail", Kind: connector.KindMail, Label: "ada@corp.com"}
	if a.mailReply(acct, map[string]string{"from": "bob@x.com"}, "m1", "你好", "正文") {
		t.Fatal("an untagged mail is a normal trigger")
	}
	if !a.mailReply(acct, map[string]string{"from": "Ada <ada@corp.com>"}, "m2", "报价确认 [yoyo:tkt-1]", "正文") {
		t.Fatal("our own sent copy must be dropped")
	}
	if !a.aliasOnce("gmail-1:m3", true) || a.aliasOnce("gmail-1:m3", true) {
		t.Fatal("one reply is delivered once")
	}
	a.aliasOnce("gmail-1:m3", false)
	if !a.aliasOnce("gmail-1:m3", true) {
		t.Fatal("a failed delivery may be retried")
	}
}

func TestFinishTextAndPayload(t *testing.T) {
	f := &superagent.Finish{What: "付款下单", URL: "https://shop.example/cart", Price: "¥199"}
	text := finishText("选好了\n\n最后一步交给你：付款下单", f)
	if !strings.HasSuffix(text, "价格：¥199\n打开：https://shop.example/cart") {
		t.Fatalf("text = %q", text)
	}
	if p := finishPayload(f); p["what"] != "付款下单" || p["url"] != "https://shop.example/cart" || p["deadline"] != nil {
		t.Fatalf("payload = %+v", p)
	}
}
