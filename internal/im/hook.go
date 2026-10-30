package im

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func (rt *Runtime) onHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writePlain(w, http.StatusMethodNotAllowed, "method")
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/im/"), "/")
	if i := strings.IndexByte(id, '/'); i >= 0 {
		id = id[:i]
	}
	acct, ok := rt.accountByPath(id)
	if !ok || !acct.Enabled {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writePlain(w, 400, "body")
		return
	}
	switch acct.Platform {
	case "lark":
		rt.hookLark(w, r, acct, body)
	case "dingtalk":
		rt.hookDingTalk(w, r, acct, body)
	case "wecom", "wecom_ai_bot", "weixin_official_account":
		rt.hookWX(w, r, acct, body)
	case "line":
		rt.hookLine(w, r, acct, body)
	case "slack":
		rt.hookSlack(w, r, acct, body)
	case "qq_official_webhook":
		rt.hookQQ(w, r, acct, body)
	case "whatsapp":
		rt.hookWhatsApp(w, r, acct, body)
	case "vocechat":
		rt.hookVoce(w, r, acct, body)
	default:
		http.NotFound(w, r)
	}
}

func (rt *Runtime) hookLine(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	secret := rt.secret(acct.ID, "channel_secret")
	if secret == "" || !sigEqual(hmacSHA256(secret, body), r.Header.Get("X-Line-Signature")) {
		writePlain(w, 401, "bad signature")
		return
	}
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	writePlain(w, 200, "ok")
	for _, item := range asList(m["events"]) {
		ev := asMap(item)
		if jstr(ev["type"]) != "message" {
			continue
		}
		msg := asMap(ev["message"])
		text := jstr(msg["text"])
		if text == "" {
			text = placeholder(jstr(msg["type"]))
		}
		source := asMap(ev["source"])
		kind := "direct"
		chatID := jstr(source["userId"])
		switch jstr(source["type"]) {
		case "group":
			kind = "group"
			chatID = jstr(source["groupId"])
		case "room":
			kind = "group"
			chatID = jstr(source["roomId"])
		}
		if chatID == "" || text == "" {
			continue
		}
		mentioned := kind == "direct"
		for _, who := range asList(dig(msg, "mention", "mentionees")) {
			one := asMap(who)
			if jbool(one["isSelf"]) || jstr(one["type"]) == "all" {
				mentioned = true
			}
		}
		atAll := false
		for _, who := range asList(dig(msg, "mention", "mentionees")) {
			if jstr(asMap(who)["type"]) == "all" {
				atAll = true
			}
		}
		user := jstr(source["userId"])
		in := Inbound{
			AccountID: acct.ID, Platform: "line", Kind: kind, ChatID: chatID,
			SenderID: user, SenderName: user, Text: text, Mentioned: mentioned, AtAll: atAll,
			MessageID: jstr(msg["id"]),
		}
		in.Reply = func(text string) error {
			ctx, cancel := replyCtx()
			defer cancel()
			return rt.pushLine(ctx, acct, chatID, text)
		}
		rt.emit(in)
	}
}

func (rt *Runtime) pushLine(ctx context.Context, acct Account, chatID, text string) error {
	token := rt.secret(acct.ID, "channel_access_token")
	if token == "" {
		return fatalf("missing line channel access token")
	}
	raw, code, err := rt.postJSON(ctx, "https://api.line.me/v2/bot/message/push", map[string]string{
		"Authorization": "Bearer " + token,
	}, map[string]any{
		"to": chatID,
		"messages": []map[string]string{{
			"type": "text", "text": clipText(text),
		}},
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("line rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "line push")
	}
	return nil
}

func (rt *Runtime) hookWhatsApp(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	q := r.URL.Query()
	verify := rt.secret(acct.ID, "verify_token")
	if r.Method == http.MethodGet {
		if verify != "" && !sigEqual(verify, q.Get("hub.verify_token")) {
			writePlain(w, 403, "bad token")
			return
		}
		writePlain(w, 200, q.Get("hub.challenge"))
		return
	}
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	writePlain(w, 200, "ok")
	for _, entry := range asList(m["entry"]) {
		for _, change := range asList(asMap(entry)["changes"]) {
			value := asMap(asMap(change)["value"])
			phoneID := jstr(dig(value, "metadata", "phone_number_id"))
			if phoneID != "" {
				rt.setCursor(acct.ID+":wa-phone", phoneID)
			}
			for _, item := range asList(value["messages"]) {
				msg := asMap(item)
				from := jstr(msg["from"])
				if from == "" {
					continue
				}
				kind := "direct"
				chatID := from
				if group := jstr(msg["group_id"]); group != "" {
					kind = "group"
					chatID = group
				}
				text := jstr(dig(msg, "text", "body"))
				if text == "" {
					text = placeholder(jstr(msg["type"]))
				}
				if text == "" {
					continue
				}
				in := Inbound{
					AccountID: acct.ID, Platform: "whatsapp", Kind: kind, ChatID: chatID,
					SenderID: from, SenderName: from, Text: text, Mentioned: kind == "direct",
					MessageID: jstr(msg["id"]),
				}
				in.Reply = func(text string) error {
					ctx, cancel := replyCtx()
					defer cancel()
					return rt.pushWhatsApp(ctx, acct, chatID, text)
				}
				rt.emit(in)
			}
		}
	}
}

func (rt *Runtime) pushWhatsApp(ctx context.Context, acct Account, chatID, text string) error {
	token := rt.secret(acct.ID, "access_token")
	phone := cfg(acct, "phone_number_id")
	if phone == "" {
		phone = rt.cursor(acct.ID + ":wa-phone")
	}
	if token == "" || phone == "" {
		return fatalf("missing whatsapp access token or phone number id")
	}
	raw, code, err := rt.postJSON(ctx, "https://graph.facebook.com/v21.0/"+phone+"/messages", bearer(token), map[string]any{
		"messaging_product": "whatsapp",
		"to":                chatID,
		"type":              "text",
		"text":              map[string]string{"body": clipText(text)},
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("whatsapp rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "whatsapp send")
	}
	return nil
}

func (rt *Runtime) hookVoce(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	writePlain(w, 200, "ok")
	detail := asMap(m["detail"])
	text := jstr(detail["content"])
	if text == "" {
		text = firstNonEmpty(jstr(m["content"]), jstr(m["text"]))
	}
	if text == "" {
		text = placeholder(jstr(detail["content_type"]))
	}
	if text == "" {
		return
	}
	target := asMap(m["target"])
	kind := "direct"
	chatID := "user:" + jstr(m["from_uid"])
	if gid := firstNonEmpty(jstr(target["gid"]), jstr(m["gid"])); gid != "" {
		kind = "group"
		chatID = "group:" + gid
	}
	user := firstNonEmpty(jstr(m["from_uid"]), jstr(m["uid"]))
	in := Inbound{
		AccountID: acct.ID, Platform: "vocechat", Kind: kind, ChatID: chatID,
		SenderID: user, SenderName: user, Text: text, Mentioned: kind == "direct",
		MessageID: jstr(m["mid"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushVoce(ctx, acct, chatID, text)
	}
	rt.emit(in)
}

func (rt *Runtime) pushVoce(ctx context.Context, acct Account, chatID, text string) error {
	base := strings.TrimRight(cfg(acct, "base_url"), "/")
	token := rt.secret(acct.ID, "token")
	if base == "" || token == "" {
		return fatalf("missing vocechat base or api key")
	}
	kind, id, ok := strings.Cut(chatID, ":")
	path := "/api/bot/send_to_user"
	body := map[string]any{"text": clipText(text)}
	if ok && kind == "group" {
		path = "/api/bot/send_to_group"
		body["gid"] = numOrString(id)
	} else {
		if ok {
			body["uid"] = numOrString(id)
		} else {
			body["uid"] = numOrString(chatID)
		}
	}
	raw, code, err := rt.postJSON(ctx, base+path, map[string]string{"x-api-key": token}, body)
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("vocechat rejected the key (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "vocechat send")
	}
	return nil
}
