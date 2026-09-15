package api

import (
	"context"
	"net/http"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/coder/websocket"
)

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
