package dsh

import "testing"

// TestRepairOpenToolsBeforeUser 锁住工具调用后面先接到用户消息时，结果会补到用户前面。
func TestRepairOpenToolsBeforeUser(t *testing.T) {
	s := NewSession()
	s.Append(EventAssistant, 3, 11, Assistant{
		Finish:    "tool-calls",
		ToolCalls: []ToolCall{{ID: "call_web", Name: "web_search", Arguments: `{"q":"gugu"}`}},
	})
	s.Append(EventUser, 4, 1, UserMessage{Text: "继续", Source: "assignment"})
	s.Append(EventUser, 5, 1, UserMessage{Text: "上一回合中断了", Source: "retry"})
	if n := s.RepairOpenTools(); n != 1 {
		t.Fatalf("repaired %d", n)
	}
	msgs := s.Derive()
	if len(msgs) != 4 {
		t.Fatalf("messages %d", len(msgs))
	}
	if msgs[0].Role != "assistant" || len(msgs[0].ToolCalls) != 1 || msgs[0].ToolCalls[0].ID != "call_web" {
		t.Fatalf("assistant %#v", msgs[0])
	}
	if msgs[1].Role != "tool" || msgs[1].ToolCallID != "call_web" || msgs[1].Name != "web_search" {
		t.Fatalf("tool %#v", msgs[1])
	}
	if msgs[2].Role != "user" || msgs[2].Text != "继续" || msgs[3].Role != "user" {
		t.Fatalf("users %#v %#v", msgs[2], msgs[3])
	}
	if s.RepairOpenTools() != 0 {
		t.Fatal("second repair")
	}
}

// TestRepairOpenToolsMovesLateResult 锁住隔在用户消息后面的结果会挪回调用旁边，不会再补一条。
func TestRepairOpenToolsMovesLateResult(t *testing.T) {
	s := NewSession()
	s.Append(EventAssistant, 1, 1, Assistant{
		Finish:    "tool-calls",
		ToolCalls: []ToolCall{{ID: "c1", Name: "shell", Arguments: "{}"}},
	})
	s.Append(EventUser, 1, 2, UserMessage{Text: "等一下", Source: "human"})
	s.Append(EventToolResult, 1, 1, ToolResultEvent{ID: "c1", Name: "shell", Text: "ok"})
	if n := s.RepairOpenTools(); n != 1 {
		t.Fatalf("moved %d", n)
	}
	msgs := s.Derive()
	if len(msgs) != 3 || msgs[1].Role != "tool" || msgs[1].Text != "ok" || msgs[2].Role != "user" {
		t.Fatalf("order %#v", msgs)
	}
	if s.RepairOpenTools() != 0 {
		t.Fatal("second repair")
	}
}

// TestRepairOpenToolsLeavesPairs 锁住已经配对的调用不会被改写。
func TestRepairOpenToolsLeavesPairs(t *testing.T) {
	s := NewSession()
	s.Append(EventAssistant, 1, 1, Assistant{
		Finish:    "tool-calls",
		ToolCalls: []ToolCall{{ID: "c1", Name: "shell", Arguments: "{}"}},
	})
	s.Append(EventToolResult, 1, 1, ToolResultEvent{ID: "c1", Name: "shell", Text: "ok"})
	s.Append(EventUser, 2, 1, UserMessage{Text: "下一句", Source: "human"})
	if s.RepairOpenTools() != 0 {
		t.Fatal("healthy log changed")
	}
	msgs := s.Derive()
	if len(msgs) != 3 || msgs[1].ToolCallID != "c1" || msgs[1].Text != "ok" {
		t.Fatalf("messages %#v", msgs)
	}
}
