package im

import (
	"testing"
	"time"
)

// TestOneBotQuote 确认数组格式的 reply 段和字符串格式的 CQ 码都能认出回复的那一条，回复段不留在正文里。
func TestOneBotQuote(t *testing.T) {
	segs := []any{
		map[string]any{"type": "reply", "data": map[string]any{"id": "-2147"}},
		map[string]any{"type": "text", "data": map[string]any{"text": "先改登录页"}},
	}
	if got := onebotQuote(segs, ""); got != "-2147" {
		t.Fatalf("segments = %q", got)
	}
	raw := "[CQ:reply,id=889][CQ:at,qq=10001] 批准"
	if got := onebotQuote(raw, raw); got != "889" {
		t.Fatalf("cq = %q", got)
	}
	if got := onebotQuote("[CQ:reply,seq=3,id=77]好", ""); got != "77" {
		t.Fatalf("id not first = %q", got)
	}
	if got := onebotQuote("没有回复", "没有回复"); got != "" {
		t.Fatalf("plain = %q", got)
	}
	if got := cqReplyAll.ReplaceAllString("[CQ:reply,id=889]批准", ""); got != "批准" {
		t.Fatalf("strip = %q", got)
	}
}

// TestMatrixFallback 确认回复正文开头的引用块被拆成被引用原文和人自己写的话。
func TestMatrixFallback(t *testing.T) {
	body := "> <@yoyo:m.org> 任务在等你\n> 回 /答 abc 你的回答\n\n先改登录页"
	if got := matrixFallback(body); got != "<@yoyo:m.org> 任务在等你\n回 /答 abc 你的回答" {
		t.Fatalf("fallback = %q", got)
	}
	if got := matrixBody(body); got != "先改登录页" {
		t.Fatalf("body = %q", got)
	}
	if got := matrixBody("直接说"); got != "直接说" {
		t.Fatalf("no quote = %q", got)
	}
}

// quoteRuntime 是收消息的运行时，handler 把消息交给通道。
func quoteRuntime(t *testing.T) (*Runtime, chan Inbound) {
	t.Helper()
	rt, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan Inbound, 4)
	rt.SetHandler(func(in Inbound) { ch <- in })
	return rt, ch
}

func recv(t *testing.T, ch chan Inbound) Inbound {
	t.Helper()
	select {
	case in := <-ch:
		return in
	case <-time.After(2 * time.Second):
		t.Fatal("no inbound")
		return Inbound{}
	}
}

// TestTelegramReplyCarriesQuote 确认回复机器人那条时带上被回复的编号和原文，群里也算在叫它。
func TestTelegramReplyCarriesQuote(t *testing.T) {
	rt, ch := quoteRuntime(t)
	acct := Account{ID: "tg-1", Platform: "telegram", Enabled: true}
	rt.ingestTelegram(acct, "42", "yoyobot", map[string]any{"message": map[string]any{
		"message_id": float64(9), "text": "先改登录页",
		"chat":       map[string]any{"id": float64(-100), "type": "supergroup"},
		"from":       map[string]any{"id": float64(7), "first_name": "Ada"},
		"reply_to_message": map[string]any{
			"message_id": float64(5), "text": "回 /答 abc 你的回答",
			"from":       map[string]any{"id": float64(42)},
		},
	}})
	in := recv(t, ch)
	if in.QuoteID != "5" || in.QuoteText != "回 /答 abc 你的回答" || !in.Mentioned {
		t.Fatalf("%+v", in)
	}
}

// TestSlackThreadIsQuote 确认线程里的回复带上父消息的 频道:ts，线程第一条自己不算引用。
func TestSlackThreadIsQuote(t *testing.T) {
	rt, ch := quoteRuntime(t)
	acct := Account{ID: "sl-1", Platform: "slack", Enabled: true}
	rt.ingestSlack(acct, "UBOT", map[string]any{"type": "message", "channel": "D1", "channel_type": "im", "user": "U1",
		"text": "批准", "ts": "171.002", "thread_ts": "171.001"})
	if in := recv(t, ch); in.QuoteID != "D1:171.001" {
		t.Fatalf("thread reply = %+v", in)
	}
	rt.ingestSlack(acct, "UBOT", map[string]any{"type": "message", "channel": "D1", "channel_type": "im", "user": "U1",
		"text": "新话题", "ts": "172.000", "thread_ts": "172.000"})
	if in := recv(t, ch); in.QuoteID != "" {
		t.Fatalf("thread root = %+v", in)
	}
}

// TestDiscordReplyCarriesQuote 确认 message_reference 和 referenced_message 被带上。
func TestDiscordReplyCarriesQuote(t *testing.T) {
	rt, ch := quoteRuntime(t)
	acct := Account{ID: "dc-1", Platform: "discord", Enabled: true}
	rt.ingestDiscord(acct, "B1", map[string]any{"id": "m2", "channel_id": "c1", "content": "不行",
		"author":             map[string]any{"id": "u1", "username": "ada"},
		"message_reference":  map[string]any{"message_id": "m1"},
		"referenced_message": map[string]any{"content": "回 /批准 k2m · /拒绝 k2m"},
	})
	if in := recv(t, ch); in.QuoteID != "m1" || in.QuoteText != "回 /批准 k2m · /拒绝 k2m" {
		t.Fatalf("%+v", in)
	}
}
