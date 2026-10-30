package im

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (rt *Runtime) runWeixinPersonal(ctx context.Context, acct Account) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	if base == "" {
		base = "http://127.0.0.1:8808"
	}
	token := rt.secret(acct.ID, "token")
	headers := map[string]string{}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	cursor := rt.cursor(acct.ID)
	for {
		if ctx.Err() != nil {
			return nil
		}
		q := url.Values{}
		q.Set("cursor", cursor)
		q.Set("timeout", "25")
		callCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
		raw, code, err := rt.get(callCtx, base+"/yoyo/poll?"+q.Encode(), headers)
		cancel()
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return err
		}
		if code == 401 || code == 403 {
			return fatalf("weixin bridge rejected the token (%d)", code)
		}
		if code == 404 {
			return fmt.Errorf("weixin bridge has no /yoyo/poll")
		}
		if code != 200 {
			return statusErr(code, raw, "weixin bridge")
		}
		body, err := decodeJSON(raw)
		if err != nil {
			return err
		}
		if next := jstr(body["cursor"]); next != "" {
			cursor = next
			rt.setCursor(acct.ID, cursor)
		}
		msgs := asList(body["messages"])
		for _, item := range msgs {
			msg := asMap(item)
			text := jstr(msg["text"])
			if text == "" {
				text = placeholder(jstr(msg["kind"]))
			}
			chatID := jstr(msg["chat_id"])
			if chatID == "" || text == "" {
				continue
			}
			kind := jstr(msg["kind"])
			if kind != "group" {
				kind = "direct"
			}
			in := Inbound{
				AccountID: acct.ID, Platform: "weixin_personal", Kind: kind, ChatID: chatID,
				SenderID: jstr(msg["sender_id"]), SenderName: jstr(msg["sender_name"]),
				SelfID: jstr(msg["self_id"]), Text: text, Mentioned: jbool(msg["mentioned"]) || kind == "direct",
				AtAll: jbool(msg["at_all"]), MessageID: firstNonEmpty(jstr(msg["message_id"]), jstr(msg["id"])),
			}
			in.Reply = func(text string) error {
				cctx, ccancel := replyCtx()
				defer ccancel()
				return rt.pushWeixinPersonal(cctx, acct, chatID, text)
			}
			rt.emit(in)
		}
		if len(msgs) == 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
		}
	}
}

func (rt *Runtime) pushWeixinPersonal(ctx context.Context, acct Account, chatID, text string) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	if base == "" {
		base = "http://127.0.0.1:8808"
	}
	headers := map[string]string{}
	if token := rt.secret(acct.ID, "token"); token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	raw, code, err := rt.postJSON(ctx, base+"/yoyo/send", headers, map[string]string{
		"chat_id": chatID, "text": clipText(text),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("weixin bridge rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "weixin bridge send")
	}
	return nil
}

func (rt *Runtime) runMatrix(ctx context.Context, acct Account) error {
	hs := strings.TrimRight(cfg(acct, "homeserver"), "/")
	token := rt.secret(acct.ID, "access_token")
	if hs == "" || token == "" {
		return fatalf("missing matrix homeserver or access token")
	}
	h := bearer(token)
	raw, code, err := rt.get(ctx, hs+"/_matrix/client/v3/account/whoami", h)
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("matrix rejected the token (%d)", code)
	}
	me, _ := decodeJSON(raw)
	self := jstr(me["user_id"])
	since := rt.cursor(acct.ID)
	primed := since != ""
	directs := map[string]bool{}
	for {
		if ctx.Err() != nil {
			return nil
		}
		q := url.Values{}
		q.Set("timeout", "25000")
		if since != "" {
			q.Set("since", since)
		}
		callCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
		raw, code, err = rt.get(callCtx, hs+"/_matrix/client/v3/sync?"+q.Encode(), h)
		cancel()
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return err
		}
		if code == 401 || code == 403 {
			return fatalf("matrix rejected the token (%d)", code)
		}
		if code != 200 {
			return statusErr(code, raw, "matrix")
		}
		body, err := decodeJSON(raw)
		if err != nil {
			return err
		}
		for _, ev := range asList(dig(body, "account_data", "events")) {
			e := asMap(ev)
			if jstr(e["type"]) != "m.direct" {
				continue
			}
			for _, rooms := range asMap(e["content"]) {
				for _, id := range asList(rooms) {
					if s := jstr(id); s != "" {
						directs[s] = true
					}
				}
			}
		}
		if next := jstr(body["next_batch"]); next != "" {
			since = next
			rt.setCursor(acct.ID, since)
		}
		if !primed {
			primed = true
			continue
		}
		for roomID, rawRoom := range asMap(dig(body, "rooms", "join")) {
			for _, ev := range asList(dig(asMap(rawRoom), "timeline", "events")) {
				rt.ingestMatrix(acct, self, roomID, directs[roomID], asMap(ev))
			}
		}
	}
}

func (rt *Runtime) ingestMatrix(acct Account, self, roomID string, direct bool, ev map[string]any) {
	if jstr(ev["type"]) != "m.room.message" {
		return
	}
	sender := jstr(ev["sender"])
	if sender == "" || sender == self {
		return
	}
	content := asMap(ev["content"])
	text := jstr(content["body"])
	if jstr(content["msgtype"]) != "m.text" && jstr(content["msgtype"]) != "" {
		text = joinParts(placeholder(strings.TrimPrefix(jstr(content["msgtype"]), "m.")), text)
	}
	if text == "" {
		return
	}
	kind := "group"
	if direct {
		kind = "direct"
	}
	mentioned := direct
	for _, id := range asList(dig(content, "m.mentions", "user_ids")) {
		if jstr(id) == self {
			mentioned = true
		}
	}
	if self != "" && strings.Contains(text, self) {
		mentioned = true
	}
	in := Inbound{
		AccountID: acct.ID, Platform: "matrix", Kind: kind, ChatID: roomID,
		SenderID: sender, SenderName: sender, SelfID: self, Text: text, Mentioned: mentioned,
		MessageID: jstr(ev["event_id"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushMatrix(ctx, acct, roomID, text)
	}
	rt.emit(in)
}

func (rt *Runtime) pushMatrix(ctx context.Context, acct Account, chatID, text string) error {
	hs := strings.TrimRight(cfg(acct, "homeserver"), "/")
	token := rt.secret(acct.ID, "access_token")
	if hs == "" || token == "" {
		return fatalf("missing matrix homeserver or access token")
	}
	path := hs + "/_matrix/client/v3/rooms/" + url.PathEscape(chatID) + "/send/m.room.message/" + reqID()
	raw, code, err := rt.send(ctx, http.MethodPut, path, bearer(token), map[string]string{
		"msgtype": "m.text", "body": clipText(text),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("matrix rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "matrix send")
	}
	return nil
}

func (rt *Runtime) runRocket(ctx context.Context, acct Account) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	user := cfg(acct, "user_id")
	room := cfg(acct, "room_id")
	if base == "" || token == "" || user == "" || room == "" {
		return fatalf("missing rocketchat base, token, user id, or room id")
	}
	headers := map[string]string{"X-Auth-Token": token, "X-User-Id": user}
	cursor := rt.cursor(acct.ID + ":ts")
	primed := cursor != ""
	for {
		if ctx.Err() != nil {
			return nil
		}
		raw, code, route, err := rt.rocketHistory(ctx, base, room, cursor, headers)
		if err != nil {
			return err
		}
		if code == 401 || code == 403 {
			return fatalf("rocketchat rejected the token (%d)", code)
		}
		if route == "" {
			return fmt.Errorf("rocketchat history http %d", code)
		}
		body, err := decodeJSON(raw)
		if err != nil {
			return err
		}
		newest := cursor
		for _, item := range asList(body["messages"]) {
			msg := asMap(item)
			ts := rcTime(msg["ts"])
			if ts > newest {
				newest = ts
			}
			if !primed || (cursor != "" && ts <= cursor) {
				continue
			}
			if jstr(dig(msg, "u", "_id")) == user {
				continue
			}
			text := jstr(msg["msg"])
			if text == "" {
				text = "[file]"
			}
			in := Inbound{
				AccountID: acct.ID, Platform: "rocketchat", Kind: "group", ChatID: room,
				SenderID: jstr(dig(msg, "u", "_id")), SenderName: jstr(dig(msg, "u", "username")),
				SelfID: user, Text: text, Mentioned: true, MessageID: jstr(msg["_id"]),
			}
			in.Reply = func(text string) error {
				cctx, ccancel := replyCtx()
				defer ccancel()
				return rt.pushRocket(cctx, acct, room, text)
			}
			rt.emit(in)
		}
		primed = true
		if newest != "" && newest != cursor {
			cursor = newest
			rt.setCursor(acct.ID+":ts", cursor)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(4 * time.Second):
		}
	}
}

func (rt *Runtime) rocketHistory(ctx context.Context, base, room, oldest string, headers map[string]string) ([]byte, int, string, error) {
	routes := []string{"channels.history", "groups.history", "im.history"}
	var lastCode int
	var lastBody []byte
	var lastErr error
	for _, route := range routes {
		q := url.Values{}
		q.Set("roomId", room)
		q.Set("count", "20")
		if oldest != "" {
			q.Set("oldest", oldest)
		}
		raw, code, err := rt.get(ctx, base+"/api/v1/"+route+"?"+q.Encode(), headers)
		if err != nil {
			return nil, 0, "", err
		}
		lastCode, lastBody, lastErr = code, raw, nil
		if code == 200 {
			return raw, code, route, nil
		}
		if code == 401 || code == 403 {
			return raw, code, "", nil
		}
	}
	return lastBody, lastCode, "", lastErr
}

func rcTime(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		return jstr(t["$date"])
	default:
		return jstr(v)
	}
}

func (rt *Runtime) pushRocket(ctx context.Context, acct Account, chatID, text string) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	user := cfg(acct, "user_id")
	raw, code, err := rt.postJSON(ctx, base+"/api/v1/chat.postMessage", map[string]string{
		"X-Auth-Token": token, "X-User-Id": user,
	}, map[string]string{"roomId": chatID, "text": clipText(text)})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("rocketchat rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "rocketchat send")
	}
	return nil
}
