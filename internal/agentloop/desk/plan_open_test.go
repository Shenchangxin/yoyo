// plan_open_test.go 锁住桌面回合正常结束、计划却还有未完成步骤时，turn_end 之后补一张 plan_open 卡片；子代理和计划模式不发。

package desk

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// openPlanSteps 先用 update_plan 写一份两步都没做完的计划，再回一句话结束这一轮。
func openPlanSteps() []runtime.Message {
	return []runtime.Message{
		{Role: runtime.RoleAssistant, ToolCalls: []runtime.ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"inspect","status":"in_progress"},{"step":"write","status":"pending"}]}`,
		}}},
		{Role: runtime.RoleAssistant, Content: "plan ready, please confirm"},
	}
}

// runOpenPlan 用 tools 跑一轮 openPlanSteps，返回这一轮发出的全部轨迹事件。
func runOpenPlan(t *testing.T, tools *runtime.WorkspaceTools) []trace.Event {
	t.Helper()
	var evs []trace.Event
	out, err := New().Run(context.Background(), Turn{
		SessionID: "s",
		User:      "build the thing",
		Client:    &runtime.ScriptedClient{Steps: openPlanSteps()},
		Tools:     tools,
		OnEvent:   func(ev trace.Event) { evs = append(evs, ev) },
	})
	if err != nil {
		t.Fatalf("an open plan is still a normal end: %v", err)
	}
	if out != "plan ready, please confirm" {
		t.Fatalf("out %q", out)
	}
	return evs
}

// TestOpenPlanCardFollowsTurnEnd 确认计划还开着时，轨迹里有且只有一张 plan_open 卡片，并且排在最后一条 turn_end 之后。
func TestOpenPlanCardFollowsTurnEnd(t *testing.T) {
	evs := runOpenPlan(t, &runtime.WorkspaceTools{Workspace: t.TempDir()})
	lastEnd, card, cards := -1, -1, 0
	for i, ev := range evs {
		switch ev.Type {
		case trace.TypeTurnEnd:
			lastEnd = i
		case trace.TypeError:
			cards++
			card = i
			if ev.Payload["kind"] != runtime.ErrKindPlanOpen {
				t.Fatalf("card %+v", ev.Payload)
			}
		}
	}
	if cards != 1 {
		t.Fatalf("plan_open cards %d", cards)
	}
	if lastEnd < 0 || card < lastEnd {
		t.Fatalf("turn_end at %d, card at %d: the card must come after turn_end", lastEnd, card)
	}
}

// TestOpenPlanCardSkipsChildAndPlanMode 确认子代理（深度 1）和计划模式下，计划没做完也不发 plan_open 卡片。
func TestOpenPlanCardSkipsChildAndPlanMode(t *testing.T) {
	for name, tools := range map[string]*runtime.WorkspaceTools{
		"child":     {Workspace: t.TempDir(), Depth: 1},
		"plan mode": {Workspace: t.TempDir(), PlanMode: true},
	} {
		for _, ev := range runOpenPlan(t, tools) {
			if ev.Type == trace.TypeError {
				t.Fatalf("%s: unexpected card %+v", name, ev.Payload)
			}
		}
	}
}
