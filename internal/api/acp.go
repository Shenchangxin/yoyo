package api

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// ServeACP 把一小段 Agent Client Protocol 适配到本进程的 JSON-RPC，不改写代理核心。读到 EOF 返回 nil，上下文已取消则返回该错误。
// initialize 直接写出 protocolVersion 为 1、serverInfo.name 为 yoyo、capabilities.loadSession 为 true，然后继续读下一条，不进入 Dispatch。
// session/new 把方法改成 thread.start。session/prompt 把方法改成 turn.start。session/cancel 把方法改成 turn.interrupt。其余方法保持原名。改写后的请求交给 Dispatch。
// session/prompt 不补 wait。turn.start 只在 wait 为真时同步走 SendOpts，所以这条 ACP 提示走 StartSendOpts。
// ctx 在每次解码前看 Err，非 nil 时返回该错误，并把 ctx 传给 Dispatch。
func ServeACP(ctx context.Context, a *app.App, r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var req RPCRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		mapped := req
		switch req.Method {
		case "initialize":
			// 在这里直接回答并 continue，不交给后面的 Dispatch。
			_ = enc.Encode(RPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"protocolVersion": "1",
				"serverInfo":      map[string]any{"name": "yoyo"},
				"capabilities":    map[string]any{"loadSession": true},
			}})
			continue
		case "session/new":
			mapped.Method = "thread.start"
		case "session/prompt":
			// 只改方法名，不补 wait。turn.start 看到 wait 为假就走 StartSendOpts，这条提示不会等模型跑完。
			mapped.Method = "turn.start"
		case "session/cancel":
			mapped.Method = "turn.interrupt"
		}
		_ = enc.Encode(Dispatch(ctx, a, mapped))
	}
}
