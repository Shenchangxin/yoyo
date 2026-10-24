package app

import "testing"

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
