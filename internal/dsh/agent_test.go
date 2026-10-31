// agent_test.go 锁住回合驱动器的可观察契约：唤醒、下一步投递、取消和后台作业。

package dsh

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// scriptModel 把测试函数当成一次模型完成。
type scriptModel struct {
	fn func(context.Context, Request) (Assistant, error)
}

// Complete 调用 fn。ctx 与 req 原样传进去。
func (m scriptModel) Complete(ctx context.Context, req Request) (Assistant, error) {
	return m.fn(ctx, req)
}

// TestFollowupRunsOneTurn 锁住 followup 只跑一个回合，并且系统提示在最前。
func TestFollowupRunsOneTurn(t *testing.T) {
	var calls atomic.Int32
	agent := New("s", scriptModel{fn: func(_ context.Context, req Request) (Assistant, error) {
		calls.Add(1)
		if len(req.Messages) == 0 || req.Messages[0].Role != "system" || req.Messages[0].Text != "be brief" {
			t.Errorf("system prompt = %#v", req.Messages)
		}
		return Assistant{Text: "ok"}, nil
	}}, nil, Options{Provider: "deepseek", Model: "deepseek-chat", Prompt: "be brief"})
	if agent.Status() != "idle" {
		t.Fatalf("status %s", agent.Status())
	}
	agent.Followup(UserMessage{Text: "hi", Source: "user"})
	agent.WhenIdle()
	if calls.Load() != 1 {
		t.Fatalf("model calls %d", calls.Load())
	}
	if agent.Status() != "idle" {
		t.Fatalf("status after %s", agent.Status())
	}
	var ends []TurnEnd
	var systems int
	for _, ev := range agent.Session().Events() {
		if ev.Type == EventTurnEnd {
			ends = append(ends, ev.Data.(TurnEnd))
		}
		if ev.Type == EventSystem {
			systems++
		}
	}
	if len(ends) != 1 || ends[0].Kind != "completed" {
		t.Fatalf("ends %#v", ends)
	}
	if systems != 1 {
		t.Fatalf("system commits %d", systems)
	}
}

// TestInjectDoesNotWakeUntilFollowup 锁住 inject 不唤醒，followup 时下一步消息先于下一回合消息。
func TestInjectDoesNotWakeUntilFollowup(t *testing.T) {
	var seen []string
	agent := New("s", scriptModel{fn: func(_ context.Context, req Request) (Assistant, error) {
		for _, m := range req.Messages {
			if m.Role == "user" {
				seen = append(seen, m.Text)
			}
		}
		return Assistant{Text: "ok"}, nil
	}}, nil, Options{Prompt: "p"})
	agent.Inject(UserMessage{ID: "i", Text: "context"})
	time.Sleep(20 * time.Millisecond)
	if agent.Status() != "idle" {
		t.Fatal("inject woke the agent")
	}
	agent.Followup(UserMessage{ID: "u", Text: "go"})
	agent.WhenIdle()
	if len(seen) != 2 || seen[0] != "context" || seen[1] != "go" {
		t.Fatalf("seen %v", seen)
	}
}

// TestSteerArrivesAtNextStep 锁住取消后的 steer 会在后续回合里被模型看见。
func TestSteerArrivesAtNextStep(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	var passes atomic.Int32
	agent := New("s", scriptModel{fn: func(ctx context.Context, req Request) (Assistant, error) {
		n := passes.Add(1)
		if n == 1 {
			once.Do(func() { close(started) })
			<-ctx.Done()
			return Assistant{}, ctx.Err()
		}
		for _, m := range req.Messages {
			if m.Role == "user" && m.Text == "steer-me" {
				return Assistant{Text: "saw-steer"}, nil
			}
		}
		t.Errorf("second pass missed steer: %#v", req.Messages)
		return Assistant{Text: "missing"}, nil
	}}, nil, Options{Prompt: "p"})
	agent.Followup(UserMessage{ID: "u", Text: "go"})
	<-started
	agent.Cancel(CancelCause{Kind: "user"}, true)
	agent.Steer(UserMessage{ID: "st", Text: "steer-me"})
	agent.WhenIdle()
	got := ""
	for _, m := range agent.Session().Derive() {
		if m.Role == "assistant" {
			got = m.Text
		}
	}
	if got != "saw-steer" {
		t.Fatalf("assistant %q", got)
	}
}

// TestCancelSkipsUndispatchedTools 锁住取消后还没开始的调用写成 ABORTED_BEFORE_DISPATCH。
func TestCancelSkipsUndispatchedTools(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	var enteredOnce sync.Once
	reg := NewRegistry()
	reg.Add(Tool{
		Name: "slow",
		Mode: func(ToolCall) string { return ModeParallel },
		Run: func(ctx context.Context, _ ToolCall) (Result, error) {
			enteredOnce.Do(func() { close(entered) })
			select {
			case <-release:
				return Result{Text: "finished"}, nil
			case <-ctx.Done():
				return Result{Text: "cancelled", IsError: true}, nil
			}
		},
	})
	reg.Add(Tool{
		Name: "later",
		Mode: func(ToolCall) string { return ModeParallel },
		Run: func(context.Context, ToolCall) (Result, error) {
			t.Fatal("later tool ran")
			return Result{}, nil
		},
	})
	agent := New("s", scriptModel{fn: func(context.Context, Request) (Assistant, error) {
		return Assistant{ToolCalls: []ToolCall{
			{ID: "a", Name: "slow", Arguments: "{}"},
			{ID: "b", Name: "later", Arguments: "{}"},
		}}, nil
	}}, reg, Options{Prompt: "p", MaxParallel: 1})
	agent.Followup(UserMessage{ID: "u", Text: "go"})
	<-entered
	agent.Cancel(CancelCause{Kind: "user"}, false)
	close(release)
	agent.WhenIdle()
	var codes []string
	for _, ev := range agent.Session().Events() {
		if ev.Type != EventToolResult {
			continue
		}
		codes = append(codes, ev.Data.(ToolResultEvent).Code)
	}
	if len(codes) != 2 || codes[1] != ToolAbortedBeforeDispatch {
		t.Fatalf("codes %v", codes)
	}
}

// TestToolBatchExclusiveBarrier 锁住可并行调用可以重叠，独占调用要等它们结束。
func TestToolBatchExclusiveBarrier(t *testing.T) {
	var mu sync.Mutex
	var current, maxSeen int
	order := []string{}
	reg := NewRegistry()
	add := func(name, mode string) {
		reg.Add(Tool{
			Name: name,
			Mode: func(ToolCall) string { return mode },
			Run: func(_ context.Context, call ToolCall) (Result, error) {
				mu.Lock()
				current++
				if current > maxSeen {
					maxSeen = current
				}
				order = append(order, "start:"+call.Name)
				mu.Unlock()
				time.Sleep(30 * time.Millisecond)
				mu.Lock()
				current--
				order = append(order, "end:"+call.Name)
				mu.Unlock()
				return Result{Text: call.Name}, nil
			},
		})
	}
	add("a", ModeParallel)
	add("b", ModeParallel)
	add("c", ModeExclusive)
	var passes atomic.Int32
	agent := New("s", scriptModel{fn: func(_ context.Context, req Request) (Assistant, error) {
		if passes.Add(1) > 1 {
			return Assistant{Text: "done"}, nil
		}
		_ = req
		return Assistant{ToolCalls: []ToolCall{
			{ID: "1", Name: "a"},
			{ID: "2", Name: "b"},
			{ID: "3", Name: "c"},
		}}, nil
	}}, reg, Options{Prompt: "p", MaxParallel: 4})
	agent.Followup(UserMessage{ID: "u", Text: "go"})
	agent.WhenIdle()
	mu.Lock()
	defer mu.Unlock()
	if maxSeen < 2 {
		t.Fatalf("parallel calls did not overlap, max %d order %v", maxSeen, order)
	}
	cStart := -1
	aEnd, bEnd := -1, -1
	for i, ev := range order {
		switch ev {
		case "start:c":
			cStart = i
		case "end:a":
			aEnd = i
		case "end:b":
			bEnd = i
		}
	}
	if cStart < 0 || aEnd > cStart || bEnd > cStart {
		t.Fatalf("exclusive barrier broken: %v", order)
	}
}

// TestBackgroundJobSteersNextStep 锁住超过前台预算的作业完成后，用 steer 把输出送进下一步。
func TestBackgroundJobSteersNextStep(t *testing.T) {
	release := make(chan struct{})
	var passes atomic.Int32
	agent := New("s", scriptModel{fn: func(_ context.Context, req Request) (Assistant, error) {
		n := passes.Add(1)
		if n == 1 {
			return Assistant{Text: "started"}, nil
		}
		for _, m := range req.Messages {
			if m.Role == "user" && len(m.Text) >= 3 && m.Text[:3] == "job" {
				return Assistant{Text: "job-seen"}, nil
			}
		}
		t.Errorf("pass %d messages %#v", n, req.Messages)
		return Assistant{Text: "no"}, nil
	}}, nil, Options{Prompt: "p"})
	agent.Followup(UserMessage{ID: "u", Text: "go"})
	agent.WhenIdle()
	view := agent.RunJob("j1", 15*time.Millisecond, func(ctx context.Context) (string, error) {
		select {
		case <-release:
			return "output", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})
	if !view.Background {
		t.Fatal("expected background job")
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	got := ""
	for time.Now().Before(deadline) {
		agent.WhenIdle()
		got = ""
		for _, m := range agent.Session().Derive() {
			if m.Role == "assistant" {
				got = m.Text
			}
		}
		if got == "job-seen" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("last assistant %q", got)
}

// TestProjectSystemAppendsInHistory 锁住历史内的提示变化会追加，替换模式则改写头部。
func TestProjectSystemAppendsInHistory(t *testing.T) {
	first := ProjectSystem(nil, "v1", true, true)
	if len(first) != 1 || first[0].Op != "append" || first[0].Text != "v1" {
		t.Fatalf("first %#v", first)
	}
	same := ProjectSystem([]string{"v1"}, "v1", true, false)
	if len(same) != 0 {
		t.Fatalf("unchanged %#v", same)
	}
	appended := ProjectSystem([]string{"v1"}, "v2", true, false)
	if len(appended) != 1 || appended[0].Op != "append" || appended[0].Text != "v2" {
		t.Fatalf("append %#v", appended)
	}
	replaced := ProjectSystem([]string{"v1", "v2"}, "v3", false, true)
	if len(replaced) != 2 || replaced[0].Op != "replace" || replaced[0].Index != 1 || replaced[1].Index != 0 || replaced[1].Text != "v3" {
		t.Fatalf("replace %#v", replaced)
	}
}
