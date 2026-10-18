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

type deltaStreamer struct{}

func (deltaStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "hello"}, nil
}

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

type reasoningStreamer struct{}

func (reasoningStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "ok"}, nil
}

func (reasoningStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	if emit != nil {
		_ = emit(StreamChunk{Type: ChunkReasoningDelta, Index: 0, Text: "why "})
		_ = emit(StreamChunk{Type: ChunkReasoningDelta, Index: 0, Text: "not"})
		_ = emit(StreamChunk{Type: ChunkTextDelta, Index: 1, Text: "ok"})
	}
	return Message{Role: RoleAssistant, Content: "ok"}, nil
}

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

type writeProgressStreamer struct{ n int }

func (s *writeProgressStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return s.ChatStream(ctx, req, nil)
}

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

func streamChunkType(ev trace.Event) string {
	if ev.Type != trace.TypeAssistantStream || ev.Payload == nil {
		return ""
	}
	ch, _ := ev.Payload["chunk"].(map[string]any)
	t, _ := ch["type"].(string)
	return t
}

func settlementHasBlock(ev trace.Event, kind string) bool {
	raw, ok := ev.Payload["blocks"]
	if !ok {
		return false
	}
	b, _ := json.Marshal(raw)
	return strings.Contains(string(b), `"type":"`+kind+`"`)
}

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
