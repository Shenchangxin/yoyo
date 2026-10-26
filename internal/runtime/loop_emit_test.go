// 锁住实时流和落盘的差别：文本增量和推理增量只走实时总线，JSONL 只留结算后的助手消息。
// 工具参数进度也只走实时总线，并且不带文件正文。seed 会把重复的用户原文折成一份。

package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// deltaStreamer 把 hello 拆成两段文本增量。
type deltaStreamer struct{}

// Chat 返回完整的 hello，不走增量。
func (deltaStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "hello"}, nil
}

// ChatStream 先发出 he 和 llo 两段增量，再返回拼好的 hello。
func (deltaStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	if emit != nil {
		for _, p := range []string{"he", "llo"} {
			if err := emit(StreamChunk{Type: ChunkTextDelta, Index: 0, Text: p}); err != nil {
				return Message{}, err
			}
		}
	}
	return Message{Role: RoleAssistant, Content: "hello"}, nil
}

// reasoningStreamer 发出两段推理增量和一段可见文本。
type reasoningStreamer struct{}

// Chat 返回可见文本 ok。
func (reasoningStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "ok"}, nil
}

// ChatStream 发出 why 和 not 两段推理，以及可见文本 ok。
func (reasoningStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	if emit != nil {
		_ = emit(StreamChunk{Type: ChunkReasoningDelta, Index: 0, Text: "why "})
		_ = emit(StreamChunk{Type: ChunkReasoningDelta, Index: 0, Text: "not"})
		_ = emit(StreamChunk{Type: ChunkTextDelta, Index: 1, Text: "ok"})
	}
	return Message{Role: RoleAssistant, Content: "ok"}, nil
}

// TestReasoningDeltasAreLiveOnly 确认 ShowThinking 时推理增量只出现在实时总线，JSONL 不存推理事件，结算后的助手消息带推理块。
func TestReasoningDeltasAreLiveOnly(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	var live []trace.Event
	_, err := Run(context.Background(), RunRequest{
		SessionID:    "s",
		User:         "hi",
		Workspace:    t.TempDir(),
		Loop:         DefaultLoop(),
		Client:       reasoningStreamer{},
		Trace:        st,
		ShowThinking: true,
		OnEvent:      func(ev trace.Event) { live = append(live, ev) },
	})
	if err != nil {
		t.Fatal(err)
	}
	var liveDelta, diskReasoning, diskAssistant int
	for _, ev := range live {
		if streamChunkType(ev) == ChunkReasoningDelta {
			liveDelta++
		}
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.Type == trace.TypeReasoning {
			diskReasoning++
		}
		if ev.Type == trace.TypeAssistant {
			diskAssistant++
			if !settlementHasBlock(ev, BlockReasoning) {
				t.Fatal("settlement missing reasoning block")
			}
		}
	}
	if liveDelta < 2 {
		t.Fatalf("live reasoning deltas %d", liveDelta)
	}
	if diskReasoning != 0 {
		t.Fatalf("jsonl must not store reasoning events, got %d", diskReasoning)
	}
	if diskAssistant < 1 {
		t.Fatalf("final assistant disk=%d", diskAssistant)
	}
}

// writeProgressStreamer 第一次把一段很长的 write_file 参数拆成两次增量，工具调用本身却是 read_file；第二次只回答 done。
type writeProgressStreamer struct{ n int }

// Chat 转发到 ChatStream，不另发增量。
func (s *writeProgressStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return s.ChatStream(ctx, req, nil)
}

// ChatStream 第一次发出两段 write_file 参数增量，返回的工具调用却是 read_file。之后返回 done。
func (s *writeProgressStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	s.n++
	if s.n > 1 {
		return Message{Role: RoleAssistant, Content: "done"}, nil
	}
	args := `{"path":"a.html","content":"` + strings.Repeat("x", 3000) + `"}`
	if emit != nil {
		_ = emit(StreamChunk{Type: ChunkToolCallDelta, Index: 0, ID: "c1", Name: "write_file", ArgumentsDelta: args[:20]})
		_ = emit(StreamChunk{Type: ChunkToolCallDelta, Index: 0, ID: "c1", Name: "write_file", ArgumentsDelta: args[20:]})
	}
	return Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "read_file", Arguments: `{"path":"a.txt"}`}}}, nil
}

// TestToolArgProgressIsLiveOnly 确认参数进度只出现在实时总线，arguments 里没有文件正文，JSONL 不存 progress。
func TestToolArgProgressIsLiveOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := trace.NewStore(t.TempDir())
	var live []trace.Event
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      "hi",
		Workspace: dir,
		Loop:      DefaultLoop(),
		Client:    &writeProgressStreamer{},
		Tools:     &WorkspaceTools{Workspace: dir},
		Trace:     st,
		OnEvent:   func(ev trace.Event) { live = append(live, ev) },
	})
	if err != nil {
		t.Fatal(err)
	}
	var prog int
	for _, ev := range live {
		if ev.Type == trace.TypeToolCall && payloadProgress(ev.Payload) {
			prog++
			if _, ok := ev.Payload["content"]; ok {
				t.Fatal("progress must not carry file bodies")
			}
			if args, _ := ev.Payload["arguments"].(string); strings.Contains(args, strings.Repeat("x", 100)) {
				t.Fatal("progress arguments must stay a path stub")
			}
		}
	}
	if prog == 0 {
		t.Fatal("expected live tool-arg progress")
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.Type == trace.TypeToolCall && payloadProgress(ev.Payload) {
			t.Fatal("jsonl must not store tool-arg progress")
		}
	}
}

// TestAssistantDeltasAreLiveOnly 确认文本增量只走实时总线，JSONL 不存 assistant_stream 和 delta，两边都有结算后的助手消息。
func TestAssistantDeltasAreLiveOnly(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	var live []trace.Event
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      "hi",
		Workspace: t.TempDir(),
		Loop:      DefaultLoop(),
		Client:    deltaStreamer{},
		Trace:     st,
		OnEvent:   func(ev trace.Event) { live = append(live, ev) },
	})
	if err != nil {
		t.Fatal(err)
	}
	var liveDelta, liveFull, diskDelta, diskFull int
	for _, ev := range live {
		if streamChunkType(ev) == ChunkTextDelta {
			liveDelta++
		}
		if ev.Type == trace.TypeAssistant {
			if delta, _ := ev.Payload["delta"].(bool); delta {
				liveDelta++
			} else {
				liveFull++
			}
		}
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.Type == trace.TypeAssistantStream {
			t.Fatal("jsonl must not store assistant_stream")
		}
		if ev.Type != trace.TypeAssistant {
			continue
		}
		if delta, _ := ev.Payload["delta"].(bool); delta {
			diskDelta++
		} else {
			diskFull++
		}
	}
	if liveDelta < 2 {
		t.Fatalf("live deltas %d", liveDelta)
	}
	if diskDelta != 0 {
		t.Fatalf("jsonl must not store token deltas, got %d", diskDelta)
	}
	if liveFull < 1 || diskFull < 1 {
		t.Fatalf("final assistant live=%d disk=%d", liveFull, diskFull)
	}
}

// streamChunkType 取出 assistant_stream 事件里 chunk 的 type。不是这类事件时返回空串。
func streamChunkType(ev trace.Event) string {
	if ev.Type != trace.TypeAssistantStream || ev.Payload == nil {
		return ""
	}
	ch, _ := ev.Payload["chunk"].(map[string]any)
	t, _ := ch["type"].(string)
	return t
}

// settlementHasBlock 看结算事件的 blocks 里有没有指定 kind。
func settlementHasBlock(ev trace.Event, kind string) bool {
	raw, ok := ev.Payload["blocks"]
	if !ok {
		return false
	}
	b, _ := json.Marshal(raw)
	return strings.Contains(string(b), `"type":"`+kind+`"`)
}

// TestSeedCollapsesDoubledUser 确认用户原文连写两遍时，落盘的 TypeUser 只留一遍。
func TestSeedCollapsesDoubledUser(t *testing.T) {
	once := "本项目是一个基于golang开发的RBAC权限管理系统，请完善前端界面。"
	st := trace.NewStore(t.TempDir())
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      once + once,
		Workspace: t.TempDir(),
		Loop:      DefaultLoop(),
		Client:    &recordingClient{},
		Trace:     st,
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for _, ev := range evs {
		if ev.Type == trace.TypeUser {
			text, _ = ev.Payload["text"].(string)
			break
		}
	}
	if text != once {
		t.Fatalf("user %q", text)
	}
}
