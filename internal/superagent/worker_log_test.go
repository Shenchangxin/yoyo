package superagent

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

func TestTranscriptLinesSkipAssignment(t *testing.T) {
	lines := TranscriptLines([]dsh.Event{
		{Type: dsh.EventUser, Data: dsh.UserMessage{Text: "工单 tkt-1\n原文", Source: "assignment"}},
		{Type: dsh.EventAssistant, Data: dsh.Assistant{Text: "找到三家", ToolCalls: []dsh.ToolCall{{ID: "c1", Name: "shell", Arguments: "ls"}}}},
		{Type: dsh.EventToolResult, Data: dsh.ToolResultEvent{Name: "shell", Text: "ok"}},
		{Type: dsh.EventUser, Data: dsh.UserMessage{Text: "再看上海", Source: "human"}},
	})
	if len(lines) != 4 {
		t.Fatalf("%+v", lines)
	}
	if lines[0].Role != "assistant" || lines[1].Role != "tool_call" || lines[2].Role != "tool_result" || lines[3].Text != "再看上海" {
		t.Fatalf("%+v", lines)
	}
}
