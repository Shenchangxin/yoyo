// 锁住交接收尾时引擎的两件事：Hold 之后不领新任务、松开后照常领；Busy 数的是已领取还没结束的任务。

package personal

import (
	"testing"
	"time"
)

// TestHoldStopsClaiming 确认 Hold(true) 期间排着的任务一直不被领走，Hold(false) 之后下一拍就被领走。
func TestHoldStopsClaiming(t *testing.T) {
	e, _ := testEngine(t)
	e.Hold(true)
	if !e.Held() {
		t.Fatal("Held must report the hold")
	}
	task, err := e.CreateTask("plan", "plan the week", KindPlan, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusQueued {
		t.Skipf("plan tasks start as %s here; this test needs a queued task", task.Status)
	}
	time.Sleep(150 * time.Millisecond)
	if got, ok := e.store.task(task.ID); !ok || got.Status != StatusQueued || got.Attempts != 0 {
		t.Fatalf("held engine claimed a task: %+v", got)
	}
	e.Hold(false)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got, ok := e.store.task(task.ID); ok && got.Attempts > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("released engine never claimed the queued task")
}

// TestBusyCountsInFlight 确认 Busy 是在跑的任务数，nil 引擎为 0、Hold 不出错。
func TestBusyCountsInFlight(t *testing.T) {
	var none *Engine
	none.Hold(true)
	if none.Busy() != 0 || none.Held() {
		t.Fatal("nil engine must be idle and not held")
	}
	e, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if e.Busy() != 0 {
		t.Fatal("fresh engine must be idle")
	}
	e.mu.Lock()
	e.inFlight["a"] = struct{}{}
	e.inFlight["b"] = struct{}{}
	e.mu.Unlock()
	if got := e.Busy(); got != 2 {
		t.Fatalf("Busy = %d, want 2", got)
	}
}
