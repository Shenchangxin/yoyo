// 锁住交接收尾时 IM 消息的去处：收尾中记进 im/handoff.json 并回一句收到，同一条平台消息只记一次；
// 不在收尾时不排队；取出时清空文件、按等待时间分成照常处理和请对方重发两类；还原出来的消息带着能发回原聊天的 Reply。

package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/im"
)

func heldInbound(id, text string, replies *[]string) im.Inbound {
	return im.Inbound{
		AccountID: "lark-1", Platform: "lark", Kind: "direct", ChatID: "oc_1", Origin: "lark-1:direct:oc_1",
		SenderID: "u1", Text: text, MessageID: id,
		Reply: func(s string) error { *replies = append(*replies, s); return nil },
	}
}

// TestIMHoldOnlyWhileDraining 确认没在收尾时不排队，调用方照旧当场开回合。
func TestIMHoldOnlyWhileDraining(t *testing.T) {
	a := drainingDaemon(t, false)
	var replies []string
	if a.imHold(heldInbound("m1", "你好", &replies)) {
		t.Fatal("must not hold while leading normally")
	}
	if _, err := os.Stat(a.imHandoffPath()); err == nil {
		t.Fatal("no handoff file should be written outside a drain")
	}
	if len(replies) != 0 {
		t.Fatalf("unexpected replies: %v", replies)
	}
}

// TestIMHoldQueuesAndDedupes 确认收尾中记下消息、回一句收到；平台重投同一条只记一次、不再回。
func TestIMHoldQueuesAndDedupes(t *testing.T) {
	a := drainingDaemon(t, false)
	a.lead.mu.Lock()
	a.lead.drainSince = time.Now()
	a.lead.mu.Unlock()
	var replies []string
	if !a.imHold(heldInbound("m1", "帮我看下明天的日程", &replies)) {
		t.Fatal("must hold while draining")
	}
	if !a.imHold(heldInbound("m1", "帮我看下明天的日程", &replies)) {
		t.Fatal("a redelivery is still consumed")
	}
	if !a.imHold(heldInbound("m2", "and the weather", &replies)) {
		t.Fatal("must hold the second message")
	}
	if len(replies) != 2 || !strings.Contains(replies[0], "收到") || !strings.Contains(replies[1], "Got it") {
		t.Fatalf("acks = %v", replies)
	}
	fresh, stale := a.imHandoffTake(time.Now())
	if len(fresh) != 2 || len(stale) != 0 || fresh[0].MessageID != "m1" || fresh[1].MessageID != "m2" {
		t.Fatalf("take = %+v / %+v", fresh, stale)
	}
	if again, _ := a.imHandoffTake(time.Now()); len(again) != 0 {
		t.Fatal("take must empty the queue")
	}
}

// TestIMHandoffTakeSplitsStale 确认等过 imHandoffTTL 的分到 stale，另一个进程（另一个 App、同一个 home）也取得到。
func TestIMHandoffTakeSplitsStale(t *testing.T) {
	a := drainingDaemon(t, false)
	a.lead.mu.Lock()
	a.lead.drainSince = time.Now()
	a.lead.mu.Unlock()
	var replies []string
	a.imHold(heldInbound("m1", "old question", &replies))
	other := &App{Home: a.Home}
	fresh, stale := other.imHandoffTake(time.Now().Add(imHandoffTTL + time.Minute))
	if len(fresh) != 0 || len(stale) != 1 || stale[0].Text != "old question" {
		t.Fatalf("take = %+v / %+v", fresh, stale)
	}
	if note := imStaleNote(stale[0].Text); !strings.Contains(note, "old question") {
		t.Fatalf("stale note = %q", note)
	}
	if note := imStaleNote("明天几点开会"); !strings.Contains(note, "再发一次") {
		t.Fatalf("zh stale note = %q", note)
	}
}

// TestIMFromHandoffRebuildsOrigin 确认还原出来的消息补齐来源，Reply 在网关没开时报错而不是 panic。
func TestIMFromHandoffRebuildsOrigin(t *testing.T) {
	a := &App{}
	in := a.imFromHandoff(imHandoffItem{AccountID: "tg-1", ChatID: "42", Text: "hi"})
	if in.Origin != "tg-1:direct:42" {
		t.Fatalf("origin = %q", in.Origin)
	}
	if in.Reply == nil || in.Reply("x") == nil {
		t.Fatal("reply without a gateway must return an error")
	}
}

// TestIMHandoffCapsQueue 确认队列最多留 imHandoffMax 条，超出时丢最旧的。
func TestIMHandoffCapsQueue(t *testing.T) {
	a := drainingDaemon(t, false)
	a.lead.mu.Lock()
	a.lead.drainSince = time.Now()
	a.lead.mu.Unlock()
	var replies []string
	for i := 0; i < imHandoffMax+5; i++ {
		a.imHold(heldInbound("m"+strings.Repeat("x", i+1), "msg", &replies))
	}
	fresh, _ := a.imHandoffTake(time.Now())
	if len(fresh) != imHandoffMax || fresh[0].MessageID != "m"+strings.Repeat("x", 6) {
		t.Fatalf("kept %d, first %q", len(fresh), fresh[0].MessageID)
	}
}
