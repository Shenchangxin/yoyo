// guard_test.go 锁住桌面回合的安全网：同一条工具错误先提醒、再停下，停止卡片种类是 stuck；模型说完话的回合不会被改成 blocked。

package desk

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/agentloop"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// loopingClient 每次都读同一个不存在的文件，最多 limit 次，之后回一句 done。安全网失效时测试也能结束。
type loopingClient struct {
	calls int
	limit int
}

// Chat 返回一次 read_file 调用；调用 id 每次不同，参数完全相同。ctx 取消时返回 ctx.Err。
func (c *loopingClient) Chat(ctx context.Context, _ runtime.ChatRequest) (runtime.Message, error) {
	if err := ctx.Err(); err != nil {
		return runtime.Message{}, err
	}
	c.calls++
	if c.calls > c.limit {
		return runtime.Message{Role: runtime.RoleAssistant, Content: "done"}, nil
	}
	return runtime.Message{Role: runtime.RoleAssistant, ToolCalls: []runtime.ToolCall{{
		ID: fmt.Sprintf("c%d", c.calls), Name: "read_file", Arguments: `{"path":"missing.txt"}`,
	}}}, nil
}

// TestGuardStopsRepeatedToolError 确认同一条工具错误重复时，先落一条 guard 注入，再在第 errorStopHits 次之后停下：
// Run 返回 StopStuck，轨迹里有一张 stuck 卡片，模型被叫的次数不超过 errorStopHits 加 1。
func TestGuardStopsRepeatedToolError(t *testing.T) {
	var evs []trace.Event
	client := &loopingClient{limit: 40}
	_, err := New().Run(context.Background(), Turn{
		SessionID: "s",
		User:      "读一下 missing.txt",
		Client:    client,
		Tools:     &runtime.WorkspaceTools{Workspace: t.TempDir()},
		OnEvent:   func(ev trace.Event) { evs = append(evs, ev) },
	})
	if runtime.ReasonOf(err) != runtime.StopStuck {
		t.Fatalf("err %v reason %s", err, runtime.ReasonOf(err))
	}
	if client.calls > errorStopHits+1 {
		t.Fatalf("model called %d times", client.calls)
	}
	// 同一个调用反复失败时，错误计数和重复调用计数会同时到阈值，两条提醒都可能出现；都要跟着操作者用中文。
	var nudges, errorNudges, cards int
	for _, ev := range evs {
		switch {
		case ev.Type == trace.TypeInject && ev.Source == sourceGuard:
			nudges++
			text, _ := ev.Payload["text"].(string)
			if !hasHan(text) {
				t.Fatalf("nudge should follow the operator's language: %q", text)
			}
			if strings.Contains(text, "同一个工具错误") {
				errorNudges++
			}
		case ev.Type == trace.TypeError:
			cards++
			if ev.Payload["kind"] != runtime.ErrKindStuck {
				t.Fatalf("card %+v", ev.Payload)
			}
		}
	}
	if nudges == 0 || errorNudges == 0 {
		t.Fatalf("guard nudges %d, error nudges %d before stopping", nudges, errorNudges)
	}
	if cards != 1 {
		t.Fatalf("stuck cards %d", cards)
	}
}

// stepClient 每次都调用一次不同参数的 list_dir，永远不说完。只有硬步数上限能让这一轮停下。
type stepClient struct{ calls int }

// Chat 返回一次 list_dir 调用，参数每次不同，避开重复调用的提醒和停止。ctx 取消时返回 ctx.Err。
func (c *stepClient) Chat(ctx context.Context, _ runtime.ChatRequest) (runtime.Message, error) {
	if err := ctx.Err(); err != nil {
		return runtime.Message{}, err
	}
	c.calls++
	return runtime.Message{Role: runtime.RoleAssistant, ToolCalls: []runtime.ToolCall{{
		ID: fmt.Sprintf("s%d", c.calls), Name: "list_dir", Arguments: fmt.Sprintf(`{"path":".","n":%d}`, c.calls),
	}}}, nil
}

// TestGuardHardStepLimit 确认 Turn.MaxSteps 是硬上限：模型被叫满 MaxSteps 次后停下，Run 返回 StopMaxTurns，轨迹里有一张 max_turns 卡片。
func TestGuardHardStepLimit(t *testing.T) {
	var evs []trace.Event
	client := &stepClient{}
	_, err := New().Run(context.Background(), Turn{
		SessionID: "s",
		User:      "list the workspace",
		Client:    client,
		Tools:     &runtime.WorkspaceTools{Workspace: t.TempDir()},
		OnEvent:   func(ev trace.Event) { evs = append(evs, ev) },
		MaxSteps:  3,
	})
	if runtime.ReasonOf(err) != runtime.StopMaxTurns {
		t.Fatalf("err %v reason %s", err, runtime.ReasonOf(err))
	}
	if client.calls != 3 {
		t.Fatalf("model called %d times, want 3", client.calls)
	}
	cards := 0
	for _, ev := range evs {
		if ev.Type == trace.TypeError {
			cards++
			if ev.Payload["kind"] != runtime.ErrKindMaxTurns {
				t.Fatalf("card %+v", ev.Payload)
			}
		}
	}
	if cards != 1 {
		t.Fatalf("max_turns cards %d", cards)
	}
}

// TestGuardLetsFinishedTurnEnd 确认模型已经不再调用工具时，即使前面错误次数够了，也不拦下这一步，回合照常结束。
func TestGuardLetsFinishedTurnEnd(t *testing.T) {
	s := agentloop.NewSession()
	g := newGuard(s, "go", 0)
	for i := 0; i < errorStopHits; i++ {
		id := fmt.Sprintf("c%d", i)
		call := agentloop.ToolCall{ID: id, Name: "read_file", Arguments: `{"path":"x"}`}
		s.Append(agentloop.EventStepStart, 1, i+1, nil)
		s.Append(agentloop.EventAssistant, 1, i+1, agentloop.Assistant{ToolCalls: []agentloop.ToolCall{call}})
		s.Append(agentloop.EventToolCall, 1, i+1, call)
		s.Append(agentloop.EventToolResult, 1, i+1, agentloop.ToolResultEvent{ID: id, Name: "read_file", Text: "no such file", IsError: true})
	}
	s.Append(agentloop.EventAssistant, 1, errorStopHits+1, agentloop.Assistant{Text: "我放弃了，文件不存在。"})
	enter, msgs, err := g.preStep(context.Background(), nil)
	if err != nil || !enter || msgs != nil {
		t.Fatalf("enter %v msgs %v err %v", enter, msgs, err)
	}
	if _, _, ok := g.halted(); ok {
		t.Fatal("finished turn was halted")
	}
}

// TestGuardKeys 锁住调用签名区分参数，错误签名只取第一行并截到 160 个 rune，top 在计数相同时取字典序靠前的键。
func TestGuardKeys(t *testing.T) {
	a := callKey(agentloop.ToolCall{Name: "shell", Arguments: `{"command":"ls"}`})
	b := callKey(agentloop.ToolCall{Name: "shell", Arguments: `{"command":"ls -a"}`})
	if a == b || callName(a) != "shell" {
		t.Fatalf("%q %q", a, b)
	}
	long := strings.Repeat("x", 400) + "\nstack"
	if key := errorKey(agentloop.ToolResultEvent{Name: "shell", Text: long}); strings.Contains(key, "stack") || len([]rune(key)) != len("shell: ")+160 {
		t.Fatalf("key %q", key)
	}
	if k, n := top(map[string]int{"b": 2, "a": 2, "c": 1}); k != "a" || n != 2 {
		t.Fatalf("top %s %d", k, n)
	}
}
