// 转移：401 换提供者，model_not_found 试同提供者的兄弟模型，已流出的 token 不再换路由。

package runtime

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// scriptClient 按调用顺序返回预设的消息或错误。
type scriptClient struct {
	err    error
	msg    Message
	stream []StreamChunk
	calls  *int
	model  *string
}

// Chat 把请求交给 ChatStream，不发出增量。
func (s *scriptClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return s.ChatStream(ctx, req, nil)
}

// ChatStream 记下调用次数和模型名，先把预设增量交给 emit，再返回预设消息和错误。
func (s *scriptClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
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

// TestFailoverClientAuthFallsToNextProvider 确认 401 跳到下一提供者，同 ID 的 m2 不被调用。
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

// TestFailoverClientModelNotFoundTriesSibling 确认 model_not_found 改试同一提供者的 alive。
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

// TestFailoverClientDoesNotMixStreamedTokens 确认已经流出 hello 之后，502 不再换路由。
func TestFailoverClientDoesNotMixStreamedTokens(t *testing.T) {
	c := NewFailoverClient([]ChatRoute{
		{ID: "a", Name: "A", Model: "m1", Client: &scriptClient{
			stream: []StreamChunk{{Type: ChunkTextDelta, Index: 0, Text: "hello"}},
			err:    fmt.Errorf("openai: 502 Bad Gateway"),
		}},
		{ID: "b", Name: "B", Model: "n", Client: &scriptClient{msg: Message{Content: "other"}}},
	})
	var got strings.Builder
	_, err := c.(Streamer).ChatStream(context.Background(), ChatRequest{Model: "m1"}, func(d StreamChunk) error {
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

// TestFailoverClientStopsOnInvalidRequest 确认 invalid_request 不转移到下一条路由。
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

// TestPrimaryOpenAI 确认取到链上第一个 OpenAI 客户端。
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
