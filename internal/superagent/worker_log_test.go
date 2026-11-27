package superagent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/agentloop"
)

func TestTranscriptLinesSkipAssignment(t *testing.T) {
	lines := TranscriptLines([]agentloop.Event{
		{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: "工单 tkt-1\n原文", Source: "assignment"}},
		{Type: agentloop.EventAssistant, Data: agentloop.Assistant{Text: "找到三家", ToolCalls: []agentloop.ToolCall{{ID: "c1", Name: "shell", Arguments: "ls"}}}},
		{Type: agentloop.EventToolResult, Data: agentloop.ToolResultEvent{ID: "c1", Name: "shell", Text: "ok"}},
		{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: "再看上海", Source: "human"}},
	})
	if len(lines) != 5 {
		t.Fatalf("%+v", lines)
	}
	if lines[0].Role != "user" || lines[0].Source != "assignment" || lines[0].Text != "原文" {
		t.Fatalf("assignment %+v", lines[0])
	}
	if lines[1].Role != "assistant" || lines[2].Role != "tool_call" || lines[2].ID != "c1" || lines[3].Role != "tool_result" || lines[4].Text != "再看上海" {
		t.Fatalf("%+v", lines)
	}
}

func TestTranscriptShowsAPendingSteerAndATurnError(t *testing.T) {
	lines := TranscriptLines([]agentloop.Event{
		{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: "先做完", Source: "human"}},
		{Type: agentloop.EventInbox, Data: agentloop.UserMessage{Text: "把天数延长", Source: "human"}},
		{Type: agentloop.EventAssistant, Data: agentloop.Assistant{Text: "先做完这一轮"}},
		{Type: agentloop.EventTurnEnd, Data: agentloop.TurnEnd{Kind: "error", Reason: `Post "https://example.test/chat/completions": EOF`}},
	})
	if len(lines) != 4 {
		t.Fatalf("%+v", lines)
	}
	if lines[1].Role != "user" || lines[1].Text != "把天数延长" {
		t.Fatalf("pending %+v", lines[1])
	}
	if lines[3].Role != "assistant" || !strings.Contains(lines[3].Text, "EOF") {
		t.Fatalf("error %+v", lines[3])
	}
	claimed := TranscriptLines([]agentloop.Event{
		{Type: agentloop.EventInbox, Data: agentloop.UserMessage{Text: "把天数延长", Source: "human"}},
		{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: "把天数延长", Source: "human"}},
	})
	if len(claimed) != 1 || claimed[0].Text != "把天数延长" {
		t.Fatalf("dup %+v", claimed)
	}
	continued := TranscriptLines([]agentloop.Event{
		{Type: agentloop.EventTurnEnd, Data: agentloop.TurnEnd{Kind: "error", Reason: "EOF"}},
		{Type: agentloop.EventAssistant, Data: agentloop.Assistant{Text: "接着做"}},
	})
	if len(continued) != 1 || continued[0].Text != "接着做" {
		t.Fatalf("continued %+v", continued)
	}
}

func TestScrubRestartNote(t *testing.T) {
	events := []agentloop.Event{
		{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: "工单 tkt-1\n" + hostRestartNote + "\n原文", Source: "assignment"}},
		{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: hostRestartNote, Source: "human"}},
		{Type: agentloop.EventAssistant, Data: agentloop.Assistant{Text: "继续写"}},
	}
	got := scrubRestartNotes(events)
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	msg, ok := got[0].Data.(agentloop.UserMessage)
	if !ok || strings.Contains(msg.Text, "宿主重启了") || !strings.Contains(msg.Text, "原文") {
		t.Fatalf("%+v", got[0].Data)
	}
	lines := TranscriptLines([]agentloop.Event{{Type: agentloop.EventUser, Data: agentloop.UserMessage{Text: hostRestartNote}}})
	if len(lines) != 0 {
		t.Fatalf("%+v", lines)
	}
}

// TestAnsweredQuestionStaysDown 锁住人答过之后，等待报告不再把上一问抬回主会话。
func TestAnsweredQuestionStaysDown(t *testing.T) {
	eng := NewEngine(autoPolicy())
	id := accept(t, eng, "m1")
	if err := eng.Report(id, Report{State: "blocked", Summary: "没找到目录", Question: "找不到目录，要新建吗"}); err != nil {
		t.Fatal(err)
	}
	if eng.Receive(deskHuman("m2", "新建吧")) != "stored" {
		t.Fatal("stored")
	}
	if err := eng.Assign(id, "m2", "新建目录并写初始文档"); err != nil {
		t.Fatal(err)
	}
	if err := eng.Report(id, Report{State: "waiting", Summary: "这一轮做完了，在等下一条消息"}); err != nil {
		t.Fatal(err)
	}
	ticket, _ := eng.Ticket(id)
	if stage := TicketStage(&ticket); stage.Name == "person" {
		t.Fatalf("old question returned %+v", stage)
	}
	question := "是自立题材，还是既有原作？"
	body := rejectedAskLog("call-1", "两份文档已写好", question)
	if _, got, ok := applyRejectedAsk(eng, id, body); !ok || got != question {
		t.Fatalf("not applied: ok=%v question=%q", ok, got)
	}
	ticket, _ = eng.Ticket(id)
	if ticket.Attention != question || TicketStage(&ticket).Action != "answer" {
		t.Fatalf("stage %+v attention %q", TicketStage(&ticket), ticket.Attention)
	}
	if _, _, ok := applyRejectedAsk(eng, id, body); ok {
		t.Fatal("applied twice")
	}
}

// rejectedAskLog 造一条被拒绝的 ready 加提问日志。
func rejectedAskLog(id, summary, question string) []byte {
	args, _ := json.Marshal(map[string]string{"state": "ready", "summary": summary, "question": question})
	call, _ := json.Marshal(agentloop.Assistant{ToolCalls: []agentloop.ToolCall{{ID: id, Name: "orchestrator_report", Arguments: string(args)}}})
	result, _ := json.Marshal(agentloop.ToolResultEvent{ID: id, Name: "orchestrator_report", Text: "questionWithResult", IsError: true})
	raw, _ := json.Marshal([]storedEvent{
		{Type: agentloop.EventAssistant, Data: call},
		{Type: agentloop.EventToolResult, Data: result},
	})
	return raw
}
