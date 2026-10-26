package api

import (
	"context"
	"net/http"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/coder/websocket"
)

// handleWS 接受 WebSocket，来源允许 localhost、127.0.0.1、https://wails.localhost 和任意来源。文本帧包成 net.Conn 后交给 ServeRPC，读写都是这条连接。
func handleWS(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"localhost:*", "127.0.0.1:*", "https://wails.localhost:*", "*"},
		})
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")
		nc := websocket.NetConn(r.Context(), c, websocket.MessageText)
		ctx := r.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		_ = ServeRPC(ctx, a, nc, nc)
	}
}
