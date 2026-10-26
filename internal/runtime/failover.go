// 聊天故障转移：按设置里的提供者和模型依次尝试。
// 不解析 HTTP。已经吐出 token 后不再换下一条。app 的聊天提供者构造这里。

package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

// ChatRoute 是聊天循环可以尝试的一条设置：提供者标识、显示名、模型名和客户端。
type ChatRoute struct {
	ID     string
	Name   string
	Model  string
	Client Client
}

// FailoverClient 先试选中的路由，再试设置里的其它模型和其他提供者。token 已经流出之后，不再把两次模型的输出混在一起。
type FailoverClient struct {
	Routes []ChatRoute
}

// NewFailoverClient 在没有路由时返回 nil，只有一条时直接返回该客户端，多条时包成 FailoverClient。
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

// PrimaryOpenAI 返回聊天路由链里第一个 OpenAI 兼容客户端，供测试和诊断查看选中的提供者。
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

// Chat 走 ChatStream，不向外发出增量。
func (f *FailoverClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return f.ChatStream(ctx, req, nil)
}

// ChatStream 按路由顺序尝试。路由的 Model 非空时覆盖请求里的模型。
// 已经有 IsToken 增量，或错误不值得转移，就停在当前结果。FailoverSkipProvider 为真时跳过同一 ID 的后续路由。
func (f *FailoverClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
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
			wrap = func(d StreamChunk) error {
				if d.IsToken() {
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

// callChat 在客户端实现 Streamer 时走 ChatStream，否则走 Chat。
func callChat(ctx context.Context, c Client, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	if s, ok := c.(Streamer); ok {
		return s.ChatStream(ctx, req, emit)
	}
	return c.Chat(ctx, req)
}

// nextLiveRoute 从 from 起找下一条客户端非空、且 ID 未被跳过的路由。
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
