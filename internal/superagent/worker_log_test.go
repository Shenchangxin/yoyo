package superagent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

func TestTranscriptLinesSkipAssignment(t *testing.T) {
	lines := TranscriptLines([]dsh.Event{
		{Type: dsh.EventUser, Data: dsh.UserMessage{Text: "工单 tkt-1\n原文", Source: "assignment"}},
		{Type: dsh.EventAssistant, Data: dsh.Assistant{Text: "找到三家", ToolCalls: []dsh.ToolCall{{ID: "c1", Name: "shell", Arguments: "ls"}}}},
		{Type: dsh.EventToolResult, Data: dsh.ToolResultEvent{ID: "c1", Name: "shell", Text: "ok"}},
		{Type: dsh.EventUser, Data: dsh.UserMessage{Text: "再看上海", Source: "human"}},
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

func TestScrubRestartNote(t *testing.T) {
	events := []dsh.Event{
		{Type: dsh.EventUser, Data: dsh.UserMessage{Text: "工单 tkt-1\n" + hostRestartNote + "\n原文", Source: "assignment"}},
		{Type: dsh.EventUser, Data: dsh.UserMessage{Text: hostRestartNote, Source: "human"}},
		{Type: dsh.EventAssistant, Data: dsh.Assistant{Text: "继续写"}},
	}
	got := scrubRestartNotes(events)
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	msg, ok := got[0].Data.(dsh.UserMessage)
	if !ok || strings.Contains(msg.Text, "宿主重启了") || !strings.Contains(msg.Text, "原文") {
		t.Fatalf("%+v", got[0].Data)
	}
	lines := TranscriptLines([]dsh.Event{{Type: dsh.EventUser, Data: dsh.UserMessage{Text: hostRestartNote}}})
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
	if !applyRejectedAsk(eng, id, body) {
		t.Fatal("not applied")
	}
	ticket, _ = eng.Ticket(id)
	if ticket.Attention != question || TicketStage(&ticket).Action != "answer" {
		t.Fatalf("stage %+v attention %q", TicketStage(&ticket), ticket.Attention)
	}
	if applyRejectedAsk(eng, id, body) {
		t.Fatal("applied twice")
	}
}

// rejectedAskLog 造一条被拒绝的 ready 加提问日志。
func rejectedAskLog(id, summary, question string) []byte {
	args, _ := json.Marshal(map[string]string{"state": "ready", "summary": summary, "question": question})
	call, _ := json.Marshal(dsh.Assistant{ToolCalls: []dsh.ToolCall{{ID: id, Name: "orchestrator_report", Arguments: string(args)}}})
	result, _ := json.Marshal(dsh.ToolResultEvent{ID: id, Name: "orchestrator_report", Text: "questionWithResult", IsError: true})
	raw, _ := json.Marshal([]storedEvent{
		{Type: dsh.EventAssistant, Data: call},
		{Type: dsh.EventToolResult, Data: result},
	})
	return raw
}
