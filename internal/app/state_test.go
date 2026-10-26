// 锁住状态订阅能收到 Touch，以及读方法不产生重绘主题。

package app

import "testing"

// TestTouchReachesSubscriber 锁住第一次 Touch 的 Epoch 为 1，订阅方收到的 Topic、Object 与传入一致。
func TestTouchReachesSubscriber(t *testing.T) {
	a := &App{}
	ch, unsub := a.SubscribeState()
	defer unsub()
	if got := a.Touch("session", "s1"); got != 1 {
		t.Fatalf("epoch %d", got)
	}
	change := <-ch
	if change.Topic != "session" || change.Object != "s1" || change.Epoch != 1 {
		t.Fatalf("%+v", change)
	}
}

// TestRPCTopicSkipsReads 锁住 health 和 thread.list 不是变更，thread.model.set 的主题是 session，key.set 的主题是 config。
func TestRPCTopicSkipsReads(t *testing.T) {
	if _, ok := RPCTopic("health"); ok {
		t.Fatal("health is a poll")
	}
	if _, ok := RPCTopic("thread.list"); ok {
		t.Fatal("list is a read")
	}
	topic, ok := RPCTopic("thread.model.set")
	if !ok || topic != "session" {
		t.Fatalf("model set %q %v", topic, ok)
	}
	topic, ok = RPCTopic("key.set")
	if !ok || topic != "config" {
		t.Fatalf("key %q %v", topic, ok)
	}
}
