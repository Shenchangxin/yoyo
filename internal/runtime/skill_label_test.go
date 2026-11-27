// 锁住 load_skill 的实时事件和界面轨迹都保留 skill 标签，不在整理时丢掉。

package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestSlimLiveEventKeepsSkillLabel 锁住实时总线上变瘦的 load_skill 调用仍带 skill 字段，原事件的载荷不被改写。
func TestSlimLiveEventKeepsSkillLabel(t *testing.T) {
	ev := trace.Event{Type: trace.TypeToolCall, SessionID: "s", Payload: map[string]any{
		"name": "load_skill", "arguments": `{"name":"brainstorming"}`, "id": "s1", "skill": "brainstorming",
	}}
	got := slimLiveEvent(ev)
	if got.Payload["name"] != "load_skill" || got.Payload["skill"] != "brainstorming" {
		t.Fatalf("%v", got.Payload)
	}
	if ev.Payload["skill"] != "brainstorming" {
		t.Fatal("slimLiveEvent must not mutate the caller's payload")
	}
}

// TestUITrajectoryKeepsLoadSkill 锁住 UITrajectory 留下 load_skill 的调用和结果，调用上的 skill 标签还在。
func TestUITrajectoryKeepsLoadSkill(t *testing.T) {
	got := UITrajectory([]trace.Event{
		{Type: trace.TypeToolCall, Payload: map[string]any{"id": "s1", "name": "load_skill", "arguments": `{"name":"brainstorming"}`, "skill": "brainstorming"}},
		{Type: trace.TypeToolResult, Payload: map[string]any{"id": "s1", "name": "load_skill", "content": "ok"}},
	})
	if len(got) != 2 {
		t.Fatalf("n=%d", len(got))
	}
	if got[0].Payload["skill"] != "brainstorming" {
		t.Fatalf("%v", got[0].Payload)
	}
}
