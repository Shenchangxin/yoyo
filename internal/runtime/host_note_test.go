// host_note_test.go 锁住主机记录：工人回报、通知和简报回放成一条用户角色的记录而不是助手消息，
// 放在下一句人话前面，不插进工具调用和结果中间；记录不被当成人的要求；行数有上限。

package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func hostEv(typ trace.EventType, source string, payload map[string]any) trace.Event {
	return trace.Event{Type: typ, Source: source, SessionID: "s", TS: time.Unix(1_700_000_000, 0), Payload: payload}
}

// TestHostLinesReplayAsOneNoteBeforeTheNextUserMessage 锁住回放：模型的回复还是助手消息，主机代写的几行并成一条记录，落在下一句人话前面。
func TestHostLinesReplayAsOneNoteBeforeTheNextUserMessage(t *testing.T) {
	evs := []trace.Event{
		hostEv(trace.TypeUser, "", map[string]any{"text": "把 CI 修一下"}),
		hostEv(trace.TypeAssistant, "model", map[string]any{"text": "交给任务了。"}),
		hostEv(trace.TypeAssistant, "worker", map[string]any{"id": "handoff:tkt-1:a", "kind": "status", "ticket": "tkt-1", "state": "ready", "text": "改好了，\n本地测试通过。"}),
		hostEv(trace.TypeAssistant, "notice", map[string]any{"id": "notice:s:1", "text": "更新已下载。"}),
		hostEv(trace.TypeAssistant, "desk", map[string]any{"id": "brief:morning:2026-11-12", "kind": "brief", "slot": "morning", "day": "2026-11-12", "text": "早上好。"}),
		hostEv(trace.TypeUser, "", map[string]any{"text": "它改了哪里？"}),
	}
	msgs := MessagesFromEvents(evs)
	if len(msgs) != 4 {
		t.Fatalf("messages %d: %+v", len(msgs), msgs)
	}
	if msgs[1].Role != RoleAssistant || msgs[1].Content != "交给任务了。" {
		t.Fatalf("model reply %+v", msgs[1])
	}
	note := msgs[2]
	if note.Role != RoleUser || !IsHostNote(note.Content) {
		t.Fatalf("host note %+v", note)
	}
	for _, want := range []string{
		"任务 tkt-1的回报 · 做完了：改好了， 本地测试通过。",
		"通知：更新已下载。",
		"晨间简报（2026-11-12）：早上好。",
	} {
		if !strings.Contains(note.Content, want) {
			t.Fatalf("note misses %q:\n%s", want, note.Content)
		}
	}
	if msgs[3].Role != RoleUser || msgs[3].Content != "它改了哪里？" {
		t.Fatalf("user %+v", msgs[3])
	}
	for _, m := range msgs {
		if m.Role == RoleAssistant && strings.Contains(m.Content, "改好了") {
			t.Fatalf("worker report replayed as the model's own words: %+v", m)
		}
	}
}

// TestHostNoteStaysOutOfToolPairs 锁住工具调用还没结果时到的回报不插在调用和结果中间。
func TestHostNoteStaysOutOfToolPairs(t *testing.T) {
	evs := []trace.Event{
		hostEv(trace.TypeUser, "", map[string]any{"text": "做两件事"}),
		hostEv(trace.TypeToolCall, "tool", map[string]any{"id": "c1", "name": "task_route", "arguments": "{}"}),
		hostEv(trace.TypeAssistant, "worker", map[string]any{"id": "handoff:tkt-2:b", "kind": "status", "ticket": "tkt-2", "state": "ready", "text": "另一件做完了。"}),
		hostEv(trace.TypeToolResult, "tool", map[string]any{"id": "c1", "name": "task_route", "content": "accepted"}),
		hostEv(trace.TypeAssistant, "model", map[string]any{"text": "交出去了。"}),
	}
	msgs := MessagesFromEvents(evs)
	roles := make([]string, 0, len(msgs))
	for _, m := range msgs {
		roles = append(roles, string(m.Role))
	}
	if got := strings.Join(roles, ","); got != "user,assistant,tool,assistant,user" {
		t.Fatalf("roles %s: %+v", got, msgs)
	}
	if msgs[1].ToolCalls[0].ID != "c1" || msgs[2].ToolCallID != "c1" || msgs[2].Content != "accepted" {
		t.Fatalf("pair broken: %+v", msgs[1:3])
	}
	if !IsHostNote(msgs[4].Content) || !strings.Contains(msgs[4].Content, "另一件做完了。") {
		t.Fatalf("trailing note %+v", msgs[4])
	}
}

// TestHostLineKinds 锁住几种行各自的写法：等人回答、外部 agent、任务组卡片；正文为空的不算。
func TestHostLineKinds(t *testing.T) {
	ask, ok := HostLineOf(hostEv(trace.TypeAssistant, "worker", map[string]any{"id": "h1", "kind": "handoff", "ticket": "tkt-3", "state": "question", "agentName": "Codex", "text": "用哪个端口？"}))
	if !ok || ask.ID != "h1" || ask.Text != "任务 tkt-3（Codex）的回报 · 在等人回答：用哪个端口？" {
		t.Fatalf("ask %+v %v", ask, ok)
	}
	group, ok := HostLineOf(hostEv(trace.TypeAssistant, "worker", map[string]any{"id": "group:grp-1", "kind": "group", "group": "grp-1", "title": "做 todo 应用", "mode": "split", "text": "Task group"}))
	if !ok || group.Text != "任务组 grp-1「做 todo 应用」已开出（分工）" {
		t.Fatalf("group %+v %v", group, ok)
	}
	if _, ok := HostLineOf(hostEv(trace.TypeAssistant, "notice", map[string]any{"id": "n", "text": "  "})); ok {
		t.Fatal("blank notice became a line")
	}
	if _, ok := HostLineOf(hostEv(trace.TypeAssistant, "model", map[string]any{"text": "我说的"})); ok {
		t.Fatal("the model's own reply became a host line")
	}
	long, _ := HostLineOf(hostEv(trace.TypeAssistant, "notice", map[string]any{"id": "n2", "text": strings.Repeat("长", 2000)}))
	if n := len([]rune(long.Text)); n > hostLineRunes+1 {
		t.Fatalf("line not clipped: %d runes", n)
	}
}

// TestHostNoteIsNotAnOperatorAsk 锁住主机记录是控制行：不会成为笔记里的目标，也不会被当成人最近说的话。
func TestHostNoteIsNotAnOperatorAsk(t *testing.T) {
	note := HostNote([]HostLine{{ID: "a", Text: "通知：更新已下载。"}})
	if !isControlUser(note) {
		t.Fatal("host note is not a control line")
	}
	notes := NotesFromMessages([]Message{{Role: RoleUser, Content: note}, {Role: RoleUser, Content: "把 CI 修一下"}})
	if notes.Objective != "把 CI 修一下" {
		t.Fatalf("objective %q", notes.Objective)
	}
	if voice := OperatorVoice("", []Message{{Role: RoleUser, Content: "fix the build"}, {Role: RoleUser, Content: note}}); voice != "fix the build" {
		t.Fatalf("voice %q", voice)
	}
	if HostNote(nil) != "" {
		t.Fatal("empty note is not empty")
	}
}

// TestHostNoteKeepsTheLatestLines 锁住人离开很久之后攒下的行只留最近的，前面报个数。
func TestHostNoteKeepsTheLatestLines(t *testing.T) {
	var lines []HostLine
	for i := 0; i < hostNoteLines+5; i++ {
		lines = append(lines, HostLine{ID: string(rune('a' + i)), Text: "通知：第 " + string(rune('a'+i)) + " 条"})
	}
	note := HostNote(lines)
	if !strings.Contains(note, "更早的 5 条略去") {
		t.Fatalf("no skipped count:\n%s", note)
	}
	if strings.Contains(note, "第 a 条") || !strings.Contains(note, "第 f 条") {
		t.Fatalf("wrong lines kept:\n%s", note)
	}
	if got := strings.Count(note, "\n- "); got != hostNoteLines {
		t.Fatalf("lines %d", got)
	}
}
