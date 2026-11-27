// 本文件锁住 idol_live.go 里进行中回合的状态：通话中或刚开过一轮时她不先开口，以及界面长轮询拿新段。
// 厂商协议（语音、识别、生图）的测试在 internal/idol/media。由 go test 调用。

package app

import (
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/idol"
)

// TestIdolTalking 锁住用户正在和她通话、或刚开过一轮时，她不先开口；挂断并过了一段时间后才行。
func TestIdolTalking(t *testing.T) {
	now := time.Now()
	if idolTalking("talk-test", now) {
		t.Fatal("idle")
	}
	noteIdolCall("talk-test", true)
	if !idolTalking("talk-test", now) {
		t.Fatal("in a call")
	}
	noteIdolCall("talk-test", false)
	noteIdolTurn("talk-test")
	if !idolTalking("talk-test", time.Now()) || idolTalking("talk-test", time.Now().Add(idolTalkingWindow+time.Second)) {
		t.Fatal("turn window")
	}
}

// TestLiveTurnWait 锁住长轮询：有新段立刻醒来只给新的那些，跑完时带上结果，没有进展时到时返回。
func TestLiveTurnWait(t *testing.T) {
	lt := &idolLiveTurn{id: "x", wake: make(chan struct{}), stop: make(chan struct{}), cancel: func() {}}
	go func() {
		time.Sleep(30 * time.Millisecond)
		lt.update(func() { lt.segs = append(lt.segs, idol.Segment{Spoken: "一"}) })
	}()
	v := lt.wait(0, time.Second)
	if len(v.Segments) != 1 || v.Done {
		t.Fatalf("%+v", v)
	}
	start := time.Now()
	if v := lt.wait(1, 50*time.Millisecond); len(v.Segments) != 0 || time.Since(start) < 40*time.Millisecond {
		t.Fatalf("timeout %+v", v)
	}
	lt.update(func() {
		lt.segs = append(lt.segs, idol.Segment{Spoken: "二"})
		lt.done = true
		lt.result = &idol.TurnResult{RecordID: "r"}
	})
	if v := lt.wait(1, time.Second); !v.Done || len(v.Segments) != 1 || v.Result.RecordID != "r" {
		t.Fatalf("%+v", v)
	}
}
