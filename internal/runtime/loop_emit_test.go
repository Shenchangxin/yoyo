package runtime

import (
	"context"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

type deltaStreamer struct{}

func (deltaStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return Message{Role: RoleAssistant, Content: "hello"}, nil
}

func (deltaStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	if emit != nil {
		for _, p := range []string{"he", "llo"} {
			if err := emit(StreamDelta{Text: p}); err != nil {
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

func (reasoningStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	if emit != nil {
		_ = emit(StreamDelta{Reasoning: "why "})
		_ = emit(StreamDelta{Reasoning: "not"})
		_ = emit(StreamDelta{Text: "ok"})
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
	var liveDelta, liveFull, diskDelta, diskFull int
	for _, ev := range live {
		if ev.Type != trace.TypeReasoning {
			continue
		}
		if delta, _ := ev.Payload["delta"].(bool); delta {
			liveDelta++
		} else {
			liveFull++
		}
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.Type != trace.TypeReasoning {
			continue
		}
		if delta, _ := ev.Payload["delta"].(bool); delta {
			diskDelta++
		} else {
			diskFull++
		}
	}
	if liveDelta < 2 {
		t.Fatalf("live reasoning deltas %d", liveDelta)
	}
	if diskDelta != 0 {
		t.Fatalf("jsonl must not store reasoning deltas, got %d", diskDelta)
	}
	if liveFull < 1 || diskFull < 1 {
		t.Fatalf("final reasoning live=%d disk=%d", liveFull, diskFull)
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
		if ev.Type != trace.TypeAssistant {
			continue
		}
		if delta, _ := ev.Payload["delta"].(bool); delta {
			liveDelta++
		} else {
			liveFull++
		}
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
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
