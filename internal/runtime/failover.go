package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// ChatRoute is one Settings provider + model the chat loop can try.
type ChatRoute struct {
	ID     string
	Name   string
	Model  string
	Client Client
}

// FailoverClient tries the selected route first, then other Settings models
// and providers. It does not mix two models after tokens have already streamed.
type FailoverClient struct {
	Routes []ChatRoute
}

func NewFailoverClient(routes []ChatRoute) Client {
	switch len(routes) {
	case 0:
		return nil
	case 1:
		return routes[0].Client
	default:
		return &FailoverClient{Routes: routes}
	}
}

// PrimaryOpenAI returns the first OpenAI-compatible client in a chat route
// chain so tests and diagnostics can inspect the selected provider.
func PrimaryOpenAI(c Client) *OpenAIClient {
	switch t := c.(type) {
	case *OpenAIClient:
		return t
	case *FailoverClient:
		if t == nil {
			return nil
		}
		for _, r := range t.Routes {
			if o, ok := r.Client.(*OpenAIClient); ok {
				return o
			}
		}
	}
	return nil
}

func (f *FailoverClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return f.ChatStream(ctx, req, nil)
}

func (f *FailoverClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	if f == nil || len(f.Routes) == 0 {
		return Message{}, fmt.Errorf("no chat providers")
	}
	skip := map[string]bool{}
	var last error
	for i, route := range f.Routes {
		if err := ctx.Err(); err != nil {
			return Message{}, err
		}
		if route.Client == nil {
			continue
		}
		if route.ID != "" && skip[route.ID] {
			continue
		}
		attempt := req
		if m := strings.TrimSpace(route.Model); m != "" {
			attempt.Model = m
		}
		streamed := false
		wrap := emit
		if emit != nil {
			wrap = func(d StreamDelta) error {
				if d.Text != "" || d.Reasoning != "" || d.Tool.Name != "" {
					streamed = true
				}
				return emit(d)
			}
		}
		msg, err := callChat(ctx, route.Client, attempt, wrap)
		if err == nil {
			return msg, nil
		}
		last = err
		if streamed || !FailoverWorthy(err) {
			return msg, err
		}
		if FailoverSkipProvider(err) && route.ID != "" {
			skip[route.ID] = true
		}
		if next, ok := nextLiveRoute(f.Routes, i+1, skip); ok {
			diaglog.WarnContext(ctx, "chat failover",
				"from", route.Name, "model", route.Model, "err", err,
				"to", next.Name, "next_model", next.Model)
		}
	}
	if last == nil {
		return Message{}, fmt.Errorf("no chat providers")
	}
	return Message{}, last
}

func callChat(ctx context.Context, c Client, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	if s, ok := c.(Streamer); ok {
		return s.ChatStream(ctx, req, emit)
	}
	return c.Chat(ctx, req)
}

func nextLiveRoute(routes []ChatRoute, from int, skip map[string]bool) (ChatRoute, bool) {
	if from < 0 {
		from = 0
	}
	for _, r := range routes[from:] {
		if r.Client == nil {
			continue
		}
		if r.ID != "" && skip[r.ID] {
			continue
		}
		return r, true
	}
	return ChatRoute{}, false
}
