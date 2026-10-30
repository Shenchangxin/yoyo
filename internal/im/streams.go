package im

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

func (rt *Runtime) runMisskey(ctx context.Context, acct Account) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	if base == "" || token == "" {
		return fatalf("missing misskey instance or token")
	}
	ws := strings.Replace(strings.Replace(base, "https://", "wss://", 1), "http://", "ws://", 1)
	c, err := dialWS(ctx, ws+"/streaming?i="+url.QueryEscape(token), nil)
	if err != nil {
		return err
	}
	defer closeWS(c)
	sock := rt.putSocket(acct.ID, c)
	defer func() {
		if cur := rt.socket(acct.ID); cur != nil && cur.c == c {
			rt.sockets.Delete(acct.ID)
		}
	}()
	if err := sock.write(ctx, map[string]any{
		"type": "connect",
		"body": map[string]string{"channel": "main", "id": "yoyo"},
	}); err != nil {
		return err
	}
	for {
		msg, err := readWS(ctx, c)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		frame, err := decodeJSON(msg)
		if err != nil || jstr(frame["type"]) != "channel" {
			continue
		}
		body := asMap(frame["body"])
		if jstr(body["id"]) != "yoyo" {
			continue
		}
		rt.ingestMisskey(acct, jstr(body["type"]), asMap(body["body"]))
	}
}

func (rt *Runtime) ingestMisskey(acct Account, kind string, body map[string]any) {
	switch kind {
	case "mention":
		note := body
		if len(asMap(body["note"])) > 0 {
			note = asMap(body["note"])
		}
		text := jstr(note["text"])
		if text == "" && len(asList(note["files"])) > 0 {
			text = "[file]"
		}
		if text == "" {
			return
		}
		user := asMap(note["user"])
		id := jstr(note["id"])
		in := Inbound{
			AccountID: acct.ID, Platform: "misskey", Kind: "group", ChatID: "note:" + id,
			SenderID:   firstNonEmpty(jstr(note["userId"]), jstr(user["id"])),
			SenderName: firstNonEmpty(jstr(user["name"]), jstr(user["username"])),
			Text:       text, Mentioned: true, MessageID: id,
		}
		in.Reply = func(text string) error {
			ctx, cancel := replyCtx()
			defer cancel()
			return rt.pushMisskey(ctx, acct, "note:"+id, text)
		}
		rt.emit(in)
	case "messagingMessage":
		text := jstr(body["text"])
		if text == "" && jstr(body["fileId"]) != "" {
			text = "[file]"
		}
		if text == "" {
			return
		}
		user := asMap(body["user"])
		chatID := "dm:" + jstr(body["userId"])
		if gid := jstr(body["groupId"]); gid != "" {
			chatID = "grp:" + gid
		}
		in := Inbound{
			AccountID: acct.ID, Platform: "misskey", Kind: "direct", ChatID: chatID,
			SenderID: jstr(body["userId"]), SenderName: firstNonEmpty(jstr(user["name"]), jstr(user["username"])),
			Text: text, MessageID: jstr(body["id"]),
		}
		if strings.HasPrefix(chatID, "grp:") {
			in.Kind = "group"
			in.Mentioned = true
		}
		in.Reply = func(text string) error {
			ctx, cancel := replyCtx()
			defer cancel()
			return rt.pushMisskey(ctx, acct, chatID, text)
		}
		rt.emit(in)
	}
}

func (rt *Runtime) pushMisskey(ctx context.Context, acct Account, chatID, text string) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	if base == "" || token == "" {
		return fatalf("missing misskey instance or token")
	}
	kind, id, _ := strings.Cut(chatID, ":")
	var path string
	body := map[string]any{"i": token, "text": clipText(text)}
	switch kind {
	case "note":
		path = "/api/notes/create"
		body["replyId"] = id
	case "grp":
		path = "/api/messaging/messages/create"
		body["groupId"] = id
	default:
		path = "/api/messaging/messages/create"
		body["userId"] = id
	}
	raw, code, err := rt.postJSON(ctx, base+path, nil, body)
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("misskey rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "misskey send")
	}
	return nil
}

func (rt *Runtime) runMattermost(ctx context.Context, acct Account) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	if base == "" || token == "" {
		return fatalf("missing mattermost server or token")
	}
	meRaw, code, err := rt.get(ctx, base+"/api/v4/users/me", bearer(token))
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("mattermost rejected the token (%d)", code)
	}
	me, _ := decodeJSON(meRaw)
	self := jstr(me["id"])
	username := jstr(me["username"])
	ws := strings.Replace(strings.Replace(base, "https://", "wss://", 1), "http://", "ws://", 1)
	c, err := dialWS(ctx, ws+"/api/v4/websocket", http.Header{"Authorization": []string{"Bearer " + token}})
	if err != nil {
		return err
	}
	defer closeWS(c)
	sock := rt.putSocket(acct.ID, c)
	defer func() {
		if cur := rt.socket(acct.ID); cur != nil && cur.c == c {
			rt.sockets.Delete(acct.ID)
		}
	}()
	_ = sock
	for {
		msg, err := readWS(ctx, c)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		frame, err := decodeJSON(msg)
		if err != nil || jstr(frame["event"]) != "posted" {
			continue
		}
		data := asMap(frame["data"])
		post, err := decodeJSON([]byte(jstr(data["post"])))
		if err != nil {
			continue
		}
		if jstr(post["user_id"]) == self {
			continue
		}
		text := jstr(post["message"])
		if text == "" {
			text = "[file]"
		}
		channel := jstr(post["channel_id"])
		kind := "group"
		if jstr(data["channel_type"]) == "D" {
			kind = "direct"
		}
		mentioned := kind == "direct"
		if username != "" && strings.Contains(strings.ToLower(text), "@"+strings.ToLower(username)) {
			mentioned = true
		}
		in := Inbound{
			AccountID: acct.ID, Platform: "mattermost", Kind: kind, ChatID: channel,
			SenderID: jstr(post["user_id"]), SenderName: jstr(data["sender_name"]),
			SelfID: self, Text: text, Mentioned: mentioned, MessageID: jstr(post["id"]),
		}
		in.Reply = func(text string) error {
			cctx, ccancel := replyCtx()
			defer ccancel()
			return rt.pushMattermost(cctx, acct, channel, text)
		}
		rt.emit(in)
	}
}

func (rt *Runtime) pushMattermost(ctx context.Context, acct Account, chatID, text string) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	raw, code, err := rt.postJSON(ctx, base+"/api/v4/posts", bearer(token), map[string]string{
		"channel_id": chatID, "message": clipText(text),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("mattermost rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "mattermost send")
	}
	return nil
}

func (rt *Runtime) runKook(ctx context.Context, acct Account) error {
	token := rt.secret(acct.ID, "token")
	if token == "" {
		return fatalf("missing kook token")
	}
	auth := map[string]string{"Authorization": "Bot " + token}
	raw, code, err := rt.get(ctx, "https://www.kookapp.cn/api/v3/user/me", auth)
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("kook rejected the token (%d)", code)
	}
	me, _ := decodeJSON(raw)
	self := jstr(dig(me, "data", "id"))
	raw, code, err = rt.get(ctx, "https://www.kookapp.cn/api/v3/gateway/index", auth)
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("kook rejected the token (%d)", code)
	}
	body, err := decodeJSON(raw)
	if err != nil {
		return err
	}
	gateway := withQuery(jstr(dig(body, "data", "url")), "compress", "0")
	if gateway == "" || gateway == "?compress=0" {
		return fmt.Errorf("kook gateway missing url")
	}
	c, err := dialWS(ctx, gateway, nil)
	if err != nil {
		return err
	}
	defer closeWS(c)
	sock := rt.putSocket(acct.ID, c)
	defer func() {
		if cur := rt.socket(acct.ID); cur != nil && cur.c == c {
			rt.sockets.Delete(acct.ID)
		}
	}()
	var sn atomic.Int64
	hbCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go pumpJSON(hbCtx, sock, 30*time.Second, func() any {
		return map[string]any{"s": 2, "sn": sn.Load()}
	})
	for {
		msg, err := readWS(ctx, c)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		frame, err := decodeJSON(msg)
		if err != nil {
			continue
		}
		if n := int64(jnum(frame["sn"])); n > sn.Load() {
			sn.Store(n)
		}
		if jnum(frame["s"]) != 0 {
			continue
		}
		d := asMap(frame["d"])
		rt.ingestKook(acct, self, d)
	}
}

func (rt *Runtime) ingestKook(acct Account, self string, d map[string]any) {
	switch jnum(d["type"]) {
	case 1, 2, 3, 4, 8, 9, 10:
	default:
		return
	}
	target := jstr(d["target_id"])
	author := jstr(d["author_id"])
	if target == "" || author == "" || author == self {
		return
	}
	text := jstr(d["content"])
	if text == "" {
		text = placeholder(map[int]string{2: "image", 3: "video", 4: "file", 8: "file", 10: "file"}[jnum(d["type"])])
	}
	if text == "" {
		return
	}
	kind := "group"
	chatID := "group:" + target
	if strings.EqualFold(jstr(d["channel_type"]), "PERSON") {
		kind = "direct"
		chatID = "direct:" + target
	}
	mentioned := kind == "direct"
	for _, id := range asList(dig(d, "extra", "mention")) {
		if jstr(id) == self {
			mentioned = true
		}
	}
	if jbool(dig(d, "extra", "mention_all")) {
		mentioned = true
	}
	in := Inbound{
		AccountID: acct.ID, Platform: "kook", Kind: kind, ChatID: chatID,
		SenderID: author, SenderName: author, SelfID: self, Text: text, Mentioned: mentioned,
		MessageID: jstr(d["msg_id"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushKook(ctx, acct, chatID, text)
	}
	rt.emit(in)
}

func (rt *Runtime) pushKook(ctx context.Context, acct Account, chatID, text string) error {
	token := rt.secret(acct.ID, "token")
	if token == "" {
		return fatalf("missing kook token")
	}
	kind, id, _ := strings.Cut(chatID, ":")
	path := "https://www.kookapp.cn/api/v3/message/create"
	if kind == "direct" {
		path = "https://www.kookapp.cn/api/v3/direct-message/create"
	}
	raw, code, err := rt.postJSON(ctx, path, map[string]string{"Authorization": "Bot " + token}, map[string]any{
		"type": 1, "target_id": id, "content": clipText(text),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("kook rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "kook send")
	}
	return nil
}

func (rt *Runtime) runSatori(ctx context.Context, acct Account) error {
	base := strings.TrimRight(cfg(acct, "endpoint"), "/")
	token := rt.secret(acct.ID, "token")
	if base == "" || token == "" {
		return fatalf("missing satori endpoint or token")
	}
	root := strings.TrimSuffix(base, "/v1")
	ws := strings.Replace(strings.Replace(root, "https://", "wss://", 1), "http://", "ws://", 1) + "/v1/events"
	header := http.Header{"Authorization": []string{"Bearer " + token}}
	c, err := dialWS(ctx, ws, header)
	if err != nil {
		return err
	}
	defer closeWS(c)
	rt.putSocket(acct.ID, c)
	defer func() {
		if cur := rt.socket(acct.ID); cur != nil && cur.c == c {
			rt.sockets.Delete(acct.ID)
		}
	}()
	for {
		msg, err := readWS(ctx, c)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		frame, err := decodeJSON(msg)
		if err != nil {
			continue
		}
		event := frame
		if body := asMap(frame["body"]); jstr(body["type"]) != "" {
			event = body
		}
		if jstr(event["type"]) != "message-created" {
			continue
		}
		rt.ingestSatori(acct, event)
	}
}

func (rt *Runtime) ingestSatori(acct Account, event map[string]any) {
	channel := asMap(event["channel"])
	user := asMap(event["user"])
	if len(user) == 0 {
		user = asMap(dig(event, "member", "user"))
	}
	message := asMap(event["message"])
	chatID := jstr(channel["id"])
	text := jstr(message["content"])
	if text == "" {
		text = "[file]"
	}
	if chatID == "" {
		return
	}
	kind := "group"
	if jnum(channel["type"]) == 1 {
		kind = "direct"
	}
	self := jstr(event["self_id"])
	platform := jstr(event["platform"])
	if platform != "" {
		rt.setCursor(acct.ID+":satori-platform", platform)
	}
	if self != "" {
		rt.setCursor(acct.ID+":satori-self", self)
	}
	mentioned := kind == "direct" || strings.Contains(text, "<at")
	in := Inbound{
		AccountID: acct.ID, Platform: "satori", Kind: kind, ChatID: chatID,
		SenderID: jstr(user["id"]), SenderName: firstNonEmpty(jstr(user["name"]), jstr(user["nick"])),
		SelfID: self, Text: text, Mentioned: mentioned, MessageID: jstr(message["id"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushSatori(ctx, acct, chatID, text)
	}
	rt.emit(in)
}

func (rt *Runtime) pushSatori(ctx context.Context, acct Account, chatID, text string) error {
	base := strings.TrimRight(cfg(acct, "endpoint"), "/")
	token := rt.secret(acct.ID, "token")
	if base == "" || token == "" {
		return fatalf("missing satori endpoint or token")
	}
	root := strings.TrimSuffix(base, "/v1")
	headers := map[string]string{"Authorization": "Bearer " + token}
	if p := rt.cursor(acct.ID + ":satori-platform"); p != "" {
		headers["Satori-Platform"] = p
	}
	if self := rt.cursor(acct.ID + ":satori-self"); self != "" {
		headers["Satori-User-ID"] = self
	}
	raw, code, err := rt.postJSON(ctx, root+"/v1/message.create", headers, map[string]string{
		"channel_id": chatID, "content": clipText(text),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("satori rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "satori send")
	}
	return nil
}
