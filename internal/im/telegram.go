package im

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (rt *Runtime) runTelegram(ctx context.Context, acct Account) error {
	token := rt.secret(acct.ID, "token")
	if token == "" {
		return fatalf("missing bot token")
	}
	me, err := rt.telegramCall(ctx, token, "getMe", nil)
	if err != nil {
		return err
	}
	self := jstr(dig(me, "result", "id"))
	username := jstr(dig(me, "result", "username"))
	offset := rt.cursor(acct.ID)
	for {
		if ctx.Err() != nil {
			return nil
		}
		q := url.Values{}
		q.Set("timeout", "25")
		if offset != "" {
			q.Set("offset", offset)
		}
		callCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
		raw, code, err := rt.get(callCtx, rt.telegramURL(token, "getUpdates")+"?"+q.Encode(), nil)
		cancel()
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return err
		}
		if code == 401 || code == 404 {
			return fatalf("telegram rejected the token (%d)", code)
		}
		body, err := decodeJSON(raw)
		if err != nil {
			return err
		}
		if !jbool(body["ok"]) {
			if jnum(body["error_code"]) == 401 {
				return fatalf("telegram rejected the token")
			}
			return fmt.Errorf("telegram: %s", jstr(body["description"]))
		}
		for _, item := range asList(body["result"]) {
			up := asMap(item)
			if id := jstr(up["update_id"]); id != "" {
				if n, err := strconv.ParseInt(id, 10, 64); err == nil {
					offset = strconv.FormatInt(n+1, 10)
				}
			}
			rt.ingestTelegram(acct, self, username, up)
		}
		if offset != "" {
			rt.setCursor(acct.ID, offset)
		}
	}
}

func (rt *Runtime) telegramCall(ctx context.Context, token, method string, body any) (map[string]any, error) {
	var raw []byte
	var code int
	var err error
	if body == nil {
		raw, code, err = rt.get(ctx, rt.telegramURL(token, method), nil)
	} else {
		raw, code, err = rt.postJSON(ctx, rt.telegramURL(token, method), nil, body)
	}
	if err != nil {
		return nil, err
	}
	if code == 401 || code == 404 {
		return nil, fatalf("telegram rejected the token (%d)", code)
	}
	m, err := decodeJSON(raw)
	if err != nil {
		return nil, err
	}
	if !jbool(m["ok"]) {
		if code == 401 || jnum(m["error_code"]) == 401 {
			return nil, fatalf("telegram rejected the token")
		}
		return nil, statusErr(code, raw, "telegram")
	}
	return m, nil
}

func (rt *Runtime) telegramURL(token, method string) string {
	return rt.rootURL("telegram", "https://api.telegram.org") + "/bot" + token + "/" + method
}

func (rt *Runtime) ingestTelegram(acct Account, self, username string, up map[string]any) {
	if cb := asMap(up["callback_query"]); len(cb) > 0 && jstr(cb["id"]) != "" {
		msg := asMap(cb["message"])
		if len(msg) == 0 {
			msg = map[string]any{"chat": map[string]any{"id": dig(cb, "from", "id"), "type": "private"}, "from": cb["from"]}
		}
		msg["text"] = jstr(cb["data"])
		msg["message_id"] = jstr(cb["id"])
		rt.emitTelegram(acct, self, username, msg)
		return
	}
	for _, key := range []string{"message", "edited_message", "channel_post"} {
		msg := asMap(up[key])
		if len(msg) == 0 {
			continue
		}
		rt.emitTelegram(acct, self, username, msg)
		return
	}
}

func (rt *Runtime) emitTelegram(acct Account, self, username string, msg map[string]any) {
	chat := asMap(msg["chat"])
	from := asMap(msg["from"])
	chatID := jstr(chat["id"])
	if chatID == "" {
		return
	}
	kind := "direct"
	switch jstr(chat["type"]) {
	case "group", "supergroup", "channel":
		kind = "group"
	}
	text := jstr(msg["text"])
	if text == "" {
		text = jstr(msg["caption"])
	}
	if text == "" {
		switch {
		case len(asList(msg["photo"])) > 0 || len(asMap(msg["sticker"])) > 0:
			text = "[image]"
		case len(asMap(msg["voice"])) > 0 || len(asMap(msg["audio"])) > 0:
			text = "[voice]"
		case len(asMap(msg["video"])) > 0 || len(asMap(msg["video_note"])) > 0:
			text = "[video]"
		case len(asMap(msg["document"])) > 0:
			text = "[file]"
		}
	}
	mentioned := false
	uname := strings.ToLower(strings.TrimPrefix(username, "@"))
	if uname != "" && strings.Contains(strings.ToLower(text), "@"+uname) {
		mentioned = true
	}
	for _, ent := range asList(msg["entities"]) {
		switch jstr(asMap(ent)["type"]) {
		case "mention", "text_mention":
			mentioned = true
		}
	}
	name := joinParts(jstr(from["first_name"]), jstr(from["last_name"]))
	if name == "" {
		name = jstr(from["username"])
	}
	in := Inbound{
		AccountID: acct.ID, Platform: "telegram", Kind: kind, ChatID: chatID,
		SenderID: jstr(from["id"]), SenderName: name, SelfID: self,
		Text: text, Mentioned: mentioned, MessageID: jstr(msg["message_id"]),
	}
	token := rt.secret(acct.ID, "token")
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		_, err := rt.telegramCall(ctx, token, "sendMessage", map[string]any{
			"chat_id": numOrString(chatID),
			"text":    clipText(text),
		})
		return err
	}
	rt.emit(in)
}

func (rt *Runtime) pushTelegram(ctx context.Context, acct Account, chatID, text string) error {
	token := rt.secret(acct.ID, "token")
	if token == "" {
		return fatalf("missing bot token")
	}
	_, err := rt.telegramCall(ctx, token, "sendMessage", map[string]any{
		"chat_id": numOrString(chatID),
		"text":    text,
	})
	return err
}
