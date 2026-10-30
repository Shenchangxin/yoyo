package im

import (
	"context"
	"crypto/hmac"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func (rt *Runtime) runOneBot(ctx context.Context, acct Account) error {
	host := cfg(acct, "ws_host")
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg(acct, "ws_port")
	if port == "" {
		port = "6700"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return err
	}
	rt.setStatus(acct.ID, "running", "ws://"+ln.Addr().String())
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rt.acceptOneBot(ctx, w, r, acct)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	<-ctx.Done()
	cctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(cctx)
	if s := rt.socket(acct.ID); s != nil {
		closeWS(s.c)
		rt.sockets.Delete(acct.ID)
	}
	return nil
}

func (rt *Runtime) acceptOneBot(ctx context.Context, w http.ResponseWriter, r *http.Request, acct Account) {
	want := rt.secret(acct.ID, "access_token")
	if want != "" {
		got := r.URL.Query().Get("access_token")
		if got == "" {
			got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if !hmac.Equal([]byte(got), []byte(want)) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}, InsecureSkipVerify: true})
	if err != nil {
		return
	}
	c.SetReadLimit(1 << 20)
	if old := rt.socket(acct.ID); old != nil && old.c != nil && old.c != c {
		closeWS(old.c)
	}
	rt.putSocket(acct.ID, c)
	go rt.readOneBot(ctx, acct, c)
}

func (rt *Runtime) readOneBot(ctx context.Context, acct Account, c *websocket.Conn) {
	defer func() {
		if cur := rt.socket(acct.ID); cur != nil && cur.c == c {
			rt.sockets.Delete(acct.ID)
		}
		closeWS(c)
	}()
	for {
		raw, err := readWS(ctx, c)
		if err != nil {
			return
		}
		m, err := decodeJSON(raw)
		if err != nil || jstr(m["post_type"]) != "message" {
			continue
		}
		rt.ingestOneBot(acct, m)
	}
}

func (rt *Runtime) ingestOneBot(acct Account, m map[string]any) {
	self := jstr(m["self_id"])
	sender := asMap(m["sender"])
	user := jstr(m["user_id"])
	if user == "" {
		user = jstr(sender["user_id"])
	}
	kind := "direct"
	chatID := "private:" + user
	if jstr(m["message_type"]) == "group" || jstr(m["group_id"]) != "" && jstr(m["message_type"]) != "private" {
		kind = "group"
		chatID = "group:" + jstr(m["group_id"])
	}
	text, mentioned, atAll := onebotText(m["message"], jstr(m["raw_message"]), self)
	in := Inbound{
		AccountID: acct.ID, Platform: "onebot", Kind: kind, ChatID: chatID,
		SenderID: user, SenderName: jstr(sender["nickname"]), SelfID: self,
		Text: text, Mentioned: mentioned, AtAll: atAll, MessageID: jstr(m["message_id"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushOneBot(ctx, acct, chatID, text)
	}
	rt.emit(in)
}

func onebotText(msg any, raw, self string) (string, bool, bool) {
	mentioned := false
	atAll := false
	switch t := msg.(type) {
	case string:
		raw = t
	case []any:
		var b strings.Builder
		for _, item := range t {
			seg := asMap(item)
			data := asMap(seg["data"])
			switch jstr(seg["type"]) {
			case "text":
				b.WriteString(jstr(data["text"]))
			case "at":
				qq := jstr(data["qq"])
				if qq == "all" {
					atAll = true
				} else if self != "" && qq == self {
					mentioned = true
				} else {
					mentioned = true
				}
			case "image":
				b.WriteString("[image]")
			case "record":
				b.WriteString("[voice]")
			case "video":
				b.WriteString("[video]")
			case "file":
				b.WriteString("[file]")
			}
		}
		if b.Len() > 0 {
			raw = b.String()
		}
	}
	if strings.Contains(raw, "[CQ:at,qq=all") {
		atAll = true
	}
	if self != "" && strings.Contains(raw, "[CQ:at,qq="+self) {
		mentioned = true
	}
	return raw, mentioned, atAll
}

func (rt *Runtime) pushOneBot(ctx context.Context, acct Account, chatID, text string) error {
	sock := rt.socket(acct.ID)
	if sock == nil {
		return fmt.Errorf("onebot is not connected")
	}
	action := "send_private_msg"
	params := map[string]any{"message": clipText(text)}
	if strings.HasPrefix(chatID, "group:") {
		action = "send_group_msg"
		params["group_id"] = numOrString(strings.TrimPrefix(chatID, "group:"))
	} else {
		params["user_id"] = numOrString(strings.TrimPrefix(strings.TrimPrefix(chatID, "private:"), "direct:"))
	}
	return sock.write(ctx, map[string]any{"action": action, "params": params, "echo": reqID()})
}
