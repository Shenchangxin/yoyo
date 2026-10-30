package im

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var qqSeq atomic.Uint32

func nextQQSeq() int {
	n := qqSeq.Add(1)
	if n == 0 {
		n = qqSeq.Add(1)
	}
	return int(n)
}

func (rt *Runtime) runQQ(ctx context.Context, acct Account) error {
	token, err := rt.qqToken(ctx, acct)
	if err != nil {
		return err
	}
	raw, code, err := rt.get(ctx, "https://api.bot.qq.com/gateway", map[string]string{"Authorization": "QQBot " + token})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("qq rejected the token (%d)", code)
	}
	if code != 200 {
		return statusErr(code, raw, "qq gateway")
	}
	body, err := decodeJSON(raw)
	if err != nil {
		return err
	}
	gateway := jstr(body["url"])
	if gateway == "" {
		return fmt.Errorf("qq gateway missing url")
	}
	return rt.runOpGateway(ctx, acct, gateway, nil, func() (any, error) {
		access, err := rt.qqToken(ctx, acct)
		if err != nil {
			return nil, err
		}
		return map[string]any{"op": 2, "d": map[string]any{
			"token":   "QQBot " + access,
			"intents": 1 << 25,
			"shard":   []int{0, 1},
		}}, nil
	}, func(t string, d map[string]any) {
		rt.ingestQQ(acct, t, d)
	})
}

func (rt *Runtime) qqToken(ctx context.Context, acct Account) (string, error) {
	if v, ok := rt.takeToken("qq:" + acct.ID); ok {
		return v, nil
	}
	appID := cfg(acct, "app_id")
	secret := rt.secret(acct.ID, "client_secret")
	if appID == "" || secret == "" {
		return "", fatalf("missing qq app id or secret")
	}
	raw, code, err := rt.postJSON(ctx, "https://bots.qq.com/app/getAppAccessToken", nil, map[string]string{
		"appId": appID, "clientSecret": secret,
	})
	if err != nil {
		return "", err
	}
	if code == 401 || code == 403 {
		return "", fatalf("qq rejected the secret (%d)", code)
	}
	m, err := decodeJSON(raw)
	if err != nil {
		return "", err
	}
	access := jstr(m["access_token"])
	if access == "" {
		return "", fatalf("qq token missing: %s", clipRunes(string(raw), 160))
	}
	ttl := time.Duration(jnum(m["expires_in"])) * time.Second
	rt.keepToken("qq:"+acct.ID, access, ttl)
	return access, nil
}

func (rt *Runtime) ingestQQ(acct Account, event string, d map[string]any) {
	kind := "direct"
	chatID := ""
	sender := ""
	mentioned := false
	switch event {
	case "C2C_MESSAGE_CREATE":
		sender = jstr(dig(d, "author", "user_openid"))
		chatID = "c2c:" + sender
	case "GROUP_AT_MESSAGE_CREATE":
		kind = "group"
		mentioned = true
		sender = jstr(dig(d, "author", "member_openid"))
		chatID = "group:" + jstr(d["group_openid"])
	case "AT_MESSAGE_CREATE":
		kind = "group"
		mentioned = true
		sender = jstr(dig(d, "author", "id"))
		chatID = "channel:" + jstr(d["channel_id"])
	case "DIRECT_MESSAGE_CREATE":
		sender = jstr(dig(d, "author", "id"))
		chatID = "channel:" + jstr(d["channel_id"])
	default:
		return
	}
	if sender == "" || chatID == "c2c:" || chatID == "group:" || chatID == "channel:" {
		return
	}
	text := jstr(d["content"])
	if text == "" && len(asList(d["attachments"])) > 0 {
		text = "[file]"
	}
	msgID := jstr(d["id"])
	in := Inbound{
		AccountID: acct.ID, Platform: acct.Platform, Kind: kind, ChatID: chatID,
		SenderID: sender, SenderName: sender, Text: text, Mentioned: mentioned, MessageID: msgID,
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.sendQQ(ctx, acct, chatID, text, msgID)
	}
	rt.emit(in)
}

func (rt *Runtime) sendQQ(ctx context.Context, acct Account, chatID, text, msgID string) error {
	token, err := rt.qqToken(ctx, acct)
	if err != nil {
		return err
	}
	kind, id, _ := strings.Cut(chatID, ":")
	var path string
	switch kind {
	case "group":
		path = "https://api.bot.qq.com/v2/groups/" + id + "/messages"
	case "channel":
		path = "https://api.sgroup.qq.com/channels/" + id + "/messages"
	default:
		path = "https://api.bot.qq.com/v2/users/" + id + "/messages"
	}
	body := map[string]any{"content": clipText(text), "msg_type": 0, "msg_seq": nextQQSeq()}
	if msgID != "" {
		body["msg_id"] = msgID
	}
	raw, code, err := rt.postJSON(ctx, path, map[string]string{"Authorization": "QQBot " + token}, body)
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("qq rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "qq send")
	}
	return nil
}

func (rt *Runtime) pushQQ(ctx context.Context, acct Account, chatID, text string) error {
	return rt.sendQQ(ctx, acct, chatID, text, "")
}

func (rt *Runtime) hookQQ(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	if jnum(m["op"]) == 13 {
		d := asMap(m["d"])
		plain := jstr(d["plain_token"])
		ts := jstr(d["event_ts"])
		secret := rt.secret(acct.ID, "client_secret")
		if secret == "" || plain == "" {
			writePlain(w, 401, "missing secret")
			return
		}
		writeJSONCode(w, 200, map[string]string{
			"plain_token": plain,
			"signature":   hmacSHA256Hex(secret, []byte(ts+plain)),
		})
		return
	}
	writePlain(w, 200, "ok")
	event := jstr(m["t"])
	d := asMap(m["d"])
	if event == "" {
		return
	}
	rt.ingestQQ(acct, event, d)
}
