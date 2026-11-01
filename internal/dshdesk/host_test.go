// host_test.go 锁住桌面回合能走 dsh 代理：纯文本回复，以及一次写文件会留下工具事件。

package dshdesk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestDeskReturnsAssistant 确认没有工具调用时，Run 返回模型正文。
func TestDeskReturnsAssistant(t *testing.T) {
	host := New()
	out, err := host.Run(context.Background(), Turn{
		SessionID: "s",
		User:      "ping",
		Client:    &runtime.ScriptedClient{Steps: []runtime.Message{{Role: runtime.RoleAssistant, Content: "pong"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "pong" {
		t.Fatalf("out %q", out)
	}
}

// seeingClient 记下模型真正收到的消息，再回一句固定正文。
type seeingClient struct {
	seen  [][]runtime.Message
	reply string
}

// Chat 记录 req 的消息并返回 reply。ctx 取消时返回 ctx.Err。
func (c *seeingClient) Chat(ctx context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	if err := ctx.Err(); err != nil {
		return runtime.Message{}, err
	}
	c.seen = append(c.seen, append([]runtime.Message(nil), req.Messages...))
	return runtime.Message{Role: runtime.RoleAssistant, Content: c.reply}, nil
}

// TestDeskReusesAgent 确认同一会话的第二次发送不再重灌历史，模型仍能看见上一轮。
func TestDeskReusesAgent(t *testing.T) {
	host := New()
	client := &seeingClient{reply: "pong"}
	if _, err := host.Run(context.Background(), Turn{SessionID: "s", User: "ping", Client: client}); err != nil {
		t.Fatal(err)
	}
	client.reply = "next"
	if _, err := host.Run(context.Background(), Turn{SessionID: "s", User: "again", Client: client}); err != nil {
		t.Fatal(err)
	}
	if len(client.seen) < 2 {
		t.Fatalf("calls %d", len(client.seen))
	}
	second := client.seen[1]
	var ping, again, pong int
	for _, msg := range second {
		if msg.Role == runtime.RoleUser && strings.Contains(msg.Content, "ping") {
			ping++
		}
		if msg.Role == runtime.RoleUser && strings.Contains(msg.Content, "again") {
			again++
		}
		if msg.Role == runtime.RoleAssistant && strings.Contains(msg.Content, "pong") {
			pong++
		}
	}
	if ping != 1 || again != 1 || pong != 1 {
		t.Fatalf("ping %d again %d pong %d", ping, again, pong)
	}
}

// TestDeskWritesFile 确认启发式模型的 write_file 会落盘，并发出工具调用事件。
func TestDeskWritesFile(t *testing.T) {
	dir := t.TempDir()
	host := New()
	var saw bool
	out, err := host.Run(context.Background(), Turn{
		SessionID:   "s",
		User:        "Write hello.txt containing hello",
		HarnessHash: "h1",
		Client:      runtime.HeuristicSolver{},
		Tools:       &runtime.WorkspaceTools{Workspace: dir, ChatOverlay: true},
		OnEvent: func(ev trace.Event) {
			if ev.Type == trace.TypeToolCall {
				if ev.HarnessSnapshot != "h1" {
					t.Errorf("hash %q", ev.HarnessSnapshot)
				}
				saw = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "done" {
		t.Fatalf("out %q", out)
	}
	body, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("body %q", body)
	}
	if !saw {
		t.Fatal("no tool call")
	}
}

// textStreamer 把正文拆成几段增量，块下标始终是 0。
type textStreamer struct {
	parts []string
	reply string
}

// Chat 返回完整回复，不发增量。
func (s textStreamer) Chat(ctx context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	return runtime.Message{Role: runtime.RoleAssistant, Content: s.reply}, nil
}

// ChatStream 按 parts 发文本增量，块下标都是 0，最后发结束块。
func (s textStreamer) ChatStream(ctx context.Context, req runtime.ChatRequest, emit func(runtime.StreamChunk) error) (runtime.Message, error) {
	if emit != nil {
		for _, part := range s.parts {
			if err := emit(runtime.StreamChunk{Type: runtime.ChunkTextDelta, Index: 0, Text: part}); err != nil {
				return runtime.Message{}, err
			}
		}
		if err := emit(runtime.StreamChunk{Type: runtime.ChunkFinish, Reason: &runtime.FinishReason{Kind: runtime.FinishStop}}); err != nil {
			return runtime.Message{}, err
		}
	}
	return runtime.Message{Role: runtime.RoleAssistant, Content: s.reply}, nil
}

// TestDeskProjectsTurnAndStream 确认回合开始会被投影出来，文本增量用递增的帧序号，结算和流用同一个尝试标识，JSONL 不存流帧。
func TestDeskProjectsTurnAndStream(t *testing.T) {
	store := trace.NewStore(t.TempDir())
	var evs []trace.Event
	host := New()
	out, err := host.Run(context.Background(), Turn{
		SessionID: "s",
		TurnID:    "tid",
		User:      "ping",
		Client:    textStreamer{parts: []string{"he", "llo"}, reply: "hello"},
		Trace:     store,
		OnEvent: func(ev trace.Event) {
			evs = append(evs, ev)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Fatalf("out %q", out)
	}
	var starts, indexes []int
	var attempt, settled string
	sawTurn := false
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeTurnStart:
			sawTurn = true
			if numberOf(ev.Payload["turn"]) != 1 {
				t.Fatalf("turn %v", ev.Payload["turn"])
			}
		case trace.TypeAssistantStream:
			kind, _ := ev.Payload["kind"].(string)
			switch kind {
			case "start":
				starts = append(starts, 1)
				attempt, _ = ev.Payload["attempt_id"].(string)
			case "chunk":
				indexes = append(indexes, numberOf(ev.Payload["index"]))
			case "end":
				if ev.Payload["outcome"] != runtime.StreamOutcomeCommitted {
					t.Fatalf("outcome %v", ev.Payload["outcome"])
				}
			}
		case trace.TypeAssistant:
			settled, _ = ev.Payload["attempt_id"].(string)
			if ev.Payload["text"] != "hello" {
				t.Fatalf("text %v", ev.Payload["text"])
			}
		}
	}
	if !sawTurn {
		t.Fatal("missing turn_start")
	}
	if len(starts) != 1 || attempt == "" || settled != attempt {
		t.Fatalf("starts %d attempt %q settled %q", len(starts), attempt, settled)
	}
	if len(indexes) != 2 || indexes[0] != 0 || indexes[1] != 1 {
		t.Fatalf("frame indexes %v", indexes)
	}
	saved, err := store.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range saved {
		if ev.Type == trace.TypeAssistantStream {
			t.Fatal("jsonl stored assistant_stream")
		}
	}
}

// stepStreamer 第一次调用发两段正文并要 list_dir，第二次发 ok。
type stepStreamer struct {
	n int
}

// Chat 转发到 ChatStream，不另发增量。
func (s *stepStreamer) Chat(ctx context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	return s.ChatStream(ctx, req, nil)
}

// ChatStream 第一次发 he、llo 和 list_dir，第二次发 ok。
func (s *stepStreamer) ChatStream(ctx context.Context, req runtime.ChatRequest, emit func(runtime.StreamChunk) error) (runtime.Message, error) {
	s.n++
	if s.n == 1 {
		if emit != nil {
			for _, part := range []string{"he", "llo"} {
				if err := emit(runtime.StreamChunk{Type: runtime.ChunkTextDelta, Index: 0, Text: part}); err != nil {
					return runtime.Message{}, err
				}
			}
		}
		return runtime.Message{
			Role:    runtime.RoleAssistant,
			Content: "looking",
			ToolCalls: []runtime.ToolCall{{
				ID: "c1", Name: "list_dir", Arguments: `{"path":"."}`,
			}},
		}, nil
	}
	if emit != nil {
		if err := emit(runtime.StreamChunk{Type: runtime.ChunkTextDelta, Index: 0, Text: "ok"}); err != nil {
			return runtime.Message{}, err
		}
	}
	return runtime.Message{Role: runtime.RoleAssistant, Content: "ok"}, nil
}

// TestDeskStreamsEachModelCall 确认同一回合的第二次模型调用会重新开始一条流。
func TestDeskStreamsEachModelCall(t *testing.T) {
	var ids []string
	host := New()
	client := &stepStreamer{}
	out, err := host.Run(context.Background(), Turn{
		SessionID: "s",
		User:      "look",
		Client:    client,
		Tools:     &runtime.WorkspaceTools{Workspace: t.TempDir(), ChatOverlay: true},
		OnEvent: func(ev trace.Event) {
			if ev.Type != trace.TypeAssistantStream {
				return
			}
			if ev.Payload["kind"] == "start" {
				id, _ := ev.Payload["attempt_id"].(string)
				ids = append(ids, id)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "ok" {
		t.Fatalf("out %q", out)
	}
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("attempts %v", ids)
	}
}

// TestJobCompletionStaysOnTheTool 确认后台完成写回工具结果，不另开一条用户消息。
func TestJobCompletionStaysOnTheTool(t *testing.T) {
	var got []trace.Event
	turn := Turn{SessionID: "s", OnEvent: func(ev trace.Event) { got = append(got, ev) }}
	emitDsh(turn, dsh.Event{Type: dsh.EventUser, Data: dsh.UserMessage{
		Source: "job",
		Text:   "job call_00_GfJM0puRqpg7UAH3S2uq8586 finished\n14:15:29 快照已保存\n",
	}})
	emitDsh(turn, dsh.Event{Type: dsh.EventUser, Data: dsh.UserMessage{Source: "steer", Text: "停一下"}})
	if len(got) != 2 {
		t.Fatalf("events %d", len(got))
	}
	if got[0].Type != trace.TypeToolResult || got[0].Payload["id"] != "call_00_GfJM0puRqpg7UAH3S2uq8586" || got[0].Payload["content"] != "14:15:29 快照已保存" {
		t.Fatalf("tool %+v", got[0])
	}
	if got[1].Type != trace.TypeUser || got[1].Source != "steer" {
		t.Fatalf("steer %+v", got[1])
	}
}

// TestDeskResumesTurnNumber 确认新代理从轨迹里已有的最大回合号继续，下一回合是 5。
func TestDeskResumesTurnNumber(t *testing.T) {
	store := trace.NewStore(t.TempDir())
	if err := store.Append(trace.Event{
		SessionID: "s",
		Type:      trace.TypeTurnEnd,
		Payload:   map[string]any{"turn": 4, "kind": "completed"},
	}); err != nil {
		t.Fatal(err)
	}
	var got int
	var saw bool
	host := New()
	if _, err := host.Run(context.Background(), Turn{
		SessionID: "s",
		User:      "again",
		Client:    &runtime.ScriptedClient{Steps: []runtime.Message{{Role: runtime.RoleAssistant, Content: "pong"}}},
		Trace:     store,
		OnEvent: func(ev trace.Event) {
			if ev.Type != trace.TypeTurnStart || saw {
				return
			}
			saw = true
			got = numberOf(ev.Payload["turn"])
		},
	}); err != nil {
		t.Fatal(err)
	}
	if !saw || got != 5 {
		t.Fatalf("saw %v turn %d", saw, got)
	}
}
