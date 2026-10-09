package runtime

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type scriptClient struct {
	err    error
	msg    Message
	stream []StreamDelta
	calls  *int
	model  *string
}

func (s *scriptClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return s.ChatStream(ctx, req, nil)
}

func (s *scriptClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	if s.calls != nil {
		*s.calls++
	}
	if s.model != nil {
		*s.model = req.Model
	}
	for _, d := range s.stream {
		if emit != nil {
			if err := emit(d); err != nil {
				return Message{}, err
			}
		}
	}
	return s.msg, s.err
}

func TestFailoverClientAuthFallsToNextProvider(t *testing.T) {
	var calls int
	second := &scriptClient{msg: Message{Content: "ok"}, calls: &calls}
	c := NewFailoverClient([]ChatRoute{
		{ID: "a", Name: "A", Model: "m1", Client: &scriptClient{err: fmt.Errorf("openai: 401 Unauthorized: invalid or expired token")}},
		{ID: "a", Name: "A", Model: "m2", Client: &scriptClient{msg: Message{Content: "same-provider"}}},
		{ID: "b", Name: "B", Model: "n", Client: second},
	})
	msg, err := c.Chat(context.Background(), ChatRequest{Model: "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "ok" {
		t.Fatalf("content %q", msg.Content)
	}
	if calls != 1 {
		t.Fatalf("second provider calls %d", calls)
	}
}

func TestFailoverClientModelNotFoundTriesSibling(t *testing.T) {
	var model string
	c := NewFailoverClient([]ChatRoute{
		{ID: "a", Name: "A", Model: "gone", Client: &scriptClient{err: fmt.Errorf("openai: 404: model_not_found")}},
		{ID: "a", Name: "A", Model: "alive", Client: &scriptClient{msg: Message{Content: "ok"}, model: &model}},
	})
	msg, err := c.Chat(context.Background(), ChatRequest{Model: "gone"})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "ok" || model != "alive" {
		t.Fatalf("content=%q model=%q", msg.Content, model)
	}
}

func TestFailoverClientDoesNotMixStreamedTokens(t *testing.T) {
	c := NewFailoverClient([]ChatRoute{
		{ID: "a", Name: "A", Model: "m1", Client: &scriptClient{
			stream: []StreamDelta{{Text: "hello"}},
			err:    fmt.Errorf("openai: 502 Bad Gateway"),
		}},
		{ID: "b", Name: "B", Model: "n", Client: &scriptClient{msg: Message{Content: "other"}}},
	})
	var got strings.Builder
	_, err := c.(Streamer).ChatStream(context.Background(), ChatRequest{Model: "m1"}, func(d StreamDelta) error {
		got.WriteString(d.Text)
		return nil
	})
	if err == nil {
		t.Fatal("expected stream error")
	}
	if got.String() != "hello" {
		t.Fatalf("streamed %q", got.String())
	}
}

func TestFailoverClientStopsOnInvalidRequest(t *testing.T) {
	var calls int
	c := NewFailoverClient([]ChatRoute{
		{ID: "a", Name: "A", Model: "m1", Client: &scriptClient{err: fmt.Errorf("invalid_request: tool_calls")}},
		{ID: "b", Name: "B", Model: "n", Client: &scriptClient{msg: Message{Content: "nope"}, calls: &calls}},
	})
	_, err := c.Chat(context.Background(), ChatRequest{Model: "m1"})
	if err == nil {
		t.Fatal("expected invalid request")
	}
	if calls != 0 {
		t.Fatalf("should not failover, calls %d", calls)
	}
}

func TestPrimaryOpenAI(t *testing.T) {
	inner := NewOpenAIClient("https://example.com/v1", "k")
	if PrimaryOpenAI(inner) != inner {
		t.Fatal("direct")
	}
	wrapped := NewFailoverClient([]ChatRoute{
		{ID: "a", Client: inner},
		{ID: "b", Client: NewOpenAIClient("https://other/v1", "x")},
	})
	got := PrimaryOpenAI(wrapped)
	if got == nil || got.APIKey != "k" {
		t.Fatalf("%+v", got)
	}
}
