package api

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// ServeACP maps a subset of Agent Client Protocol methods onto Yoyo JSON-RPC.
// The agent core is not rewritten as ACP; this is a compatibility adapter.
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
			_ = enc.Encode(RPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"protocolVersion": "1",
				"serverInfo":      map[string]any{"name": "yoyo"},
				"capabilities":    map[string]any{"loadSession": true},
			}})
			continue
		case "session/new":
			mapped.Method = "thread.start"
		case "session/prompt":
			mapped.Method = "turn.start"
		case "session/cancel":
			mapped.Method = "turn.interrupt"
		}
		_ = enc.Encode(Dispatch(ctx, a, mapped))
	}
}
