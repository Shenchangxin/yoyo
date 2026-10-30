package im

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (rt *Runtime) runLark(ctx context.Context, acct Account) error {
	if modeOf(acct, "webhook") != "socket" {
		return rt.runWait(ctx, acct)
	}
	return fatalf("feishu socket frames are not public; use webhook")
}

func (rt *Runtime) larkBase(acct Account) string {
	b := cfg(acct, "api_base")
	if b == "" {
		b = "https://open.feishu.cn"
	}
	return strings.TrimRight(b, "/")
}

func (rt *Runtime) larkToken(ctx context.Context, acct Account) (string, error) {
	if v, ok := rt.takeToken("lark:" + acct.ID); ok {
		return v, nil
	}
	appID := cfg(acct, "app_id")
	secret := rt.secret(acct.ID, "app_secret")
	if appID == "" || secret == "" {
		return "", fatalf("missing feishu app id or secret")
	}
	raw, code, err := rt.postJSON(ctx, rt.larkBase(acct)+"/open-apis/auth/v3/tenant_access_token/internal", nil, map[string]string{
		"app_id": appID, "app_secret": secret,
	})
	if err != nil {
		return "", err
	}
	if code == 401 || code == 403 {
		return "", fatalf("feishu rejected the secret (%d)", code)
	}
	m, err := decodeJSON(raw)
	if err != nil {
		return "", err
	}
	access := jstr(m["tenant_access_token"])
	if access == "" {
		return "", fmt.Errorf("feishu token: %s", jstr(m["msg"]))
	}
	rt.keepToken("lark:"+acct.ID, access, time.Duration(jnum(m["expire"]))*time.Second)
	return access, nil
}

func (rt *Runtime) pushLark(ctx context.Context, acct Account, chatID, text string) error {
	token, err := rt.larkToken(ctx, acct)
	if err != nil {
		return err
	}
	content, _ := json.Marshal(map[string]string{"text": clipText(text)})
	raw, code, err := rt.postJSON(ctx, rt.larkBase(acct)+"/open-apis/im/v1/messages?receive_id_type=chat_id", map[string]string{
		"Authorization": "Bearer " + token,
	}, map[string]string{
		"receive_id": chatID, "msg_type": "text", "content": string(content),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("feishu rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "feishu send")
	}
	return nil
}

func (rt *Runtime) hookLark(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	raw := body
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	if enc := jstr(m["encrypt"]); enc != "" {
		key := rt.secret(acct.ID, "encrypt_key")
		if key == "" {
			writePlain(w, 400, "missing encrypt key")
			return
		}
		plain, err := decryptFeishu(key, enc)
		if err != nil {
			writePlain(w, 400, "bad crypt")
			return
		}
		raw = []byte(plain)
		m, err = decodeJSON(raw)
		if err != nil {
			writePlain(w, 400, "bad inner json")
			return
		}
	}
	verify := rt.secret(acct.ID, "verify_token")
	got := jstr(m["token"])
	if got == "" {
		got = jstr(dig(m, "header", "token"))
	}
	if verify != "" && !sigEqual(verify, got) {
		writePlain(w, 401, "bad token")
		return
	}
	if jstr(m["type"]) == "url_verification" || jstr(m["challenge"]) != "" && jstr(dig(m, "header", "event_type")) == "" {
		writeJSONCode(w, 200, map[string]string{"challenge": jstr(m["challenge"])})
		return
	}
	writePlain(w, 200, "ok")
	if jstr(dig(m, "header", "event_type")) != "im.message.receive_v1" {
		return
	}
	event := asMap(m["event"])
	message := asMap(event["message"])
	sender := jstr(dig(event, "sender", "sender_id", "open_id"))
	chatID := jstr(message["chat_id"])
	if chatID == "" || sender == "" {
		return
	}
	kind := "direct"
	if jstr(message["chat_type"]) == "group" {
		kind = "group"
	}
	text := larkText(jstr(message["message_type"]), jstr(message["content"]))
	in := Inbound{
		AccountID: acct.ID, Platform: "lark", Kind: kind, ChatID: chatID,
		SenderID: sender, SenderName: sender, Text: text, Mentioned: true,
		MessageID: jstr(message["message_id"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushLark(ctx, acct, chatID, text)
	}
	rt.emit(in)
}

func larkText(kind, content string) string {
	if kind == "" || kind == "text" {
		m, err := decodeJSON([]byte(content))
		if err == nil {
			if text := jstr(m["text"]); text != "" {
				return text
			}
		}
		if strings.TrimSpace(content) != "" && kind == "text" {
			return content
		}
	}
	if note := placeholder(kind); note != "" {
		return note
	}
	return strings.TrimSpace(content)
}
