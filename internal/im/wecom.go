package im

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (rt *Runtime) runWeComAI(ctx context.Context, acct Account) error {
	if modeOf(acct, "long") == "webhook" {
		return rt.runWait(ctx, acct)
	}
	botID := cfg(acct, "bot_id")
	secret := rt.secret(acct.ID, "secret")
	if botID == "" || secret == "" {
		return fatalf("missing wecom ai bot id or secret")
	}
	c, err := dialWS(ctx, rt.wsURL("wecom_ai", "wss://openws.work.weixin.qq.com"), nil)
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
		"cmd":     "aibot_subscribe",
		"headers": map[string]string{"req_id": reqID()},
		"body":    map[string]string{"bot_id": botID, "secret": secret},
	}); err != nil {
		return err
	}
	hbCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go pumpJSON(hbCtx, sock, 30*time.Second, func() any {
		return map[string]any{"cmd": "ping", "headers": map[string]string{"req_id": reqID()}}
	})
	subscribed := false
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
		cmd := jstr(frame["cmd"])
		if !subscribed {
			if code := jnum(frame["errcode"]); code != 0 {
				return fatalf("wecom ai subscribe: %s", jstr(frame["errmsg"]))
			}
			subscribed = true
			if cmd != "aibot_msg_callback" {
				continue
			}
		}
		if cmd == "aibot_msg_callback" {
			rt.ingestWeComAI(acct, sock, frame)
		}
	}
}

func (rt *Runtime) ingestWeComAI(acct Account, sock *liveSocket, frame map[string]any) {
	body := asMap(frame["body"])
	req := jstr(dig(frame, "headers", "req_id"))
	user := jstr(dig(body, "from", "userid"))
	kind := "direct"
	chatID := "single:" + user
	if jstr(body["chattype"]) == "group" {
		kind = "group"
		chatID = "group:" + jstr(body["chatid"])
	}
	text := jstr(dig(body, "text", "content"))
	if text == "" {
		text = placeholder(jstr(body["msgtype"]))
	}
	stream := jstr(body["msgid"])
	if stream == "" {
		stream = reqID()
	}
	in := Inbound{
		AccountID: acct.ID, Platform: "wecom_ai_bot", Kind: kind, ChatID: chatID,
		SenderID: user, SenderName: user, Text: text, Mentioned: true, MessageID: jstr(body["msgid"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return sock.write(ctx, map[string]any{
			"cmd":     "aibot_respond_msg",
			"headers": map[string]string{"req_id": req},
			"body": map[string]any{
				"msgtype": "stream",
				"stream":  map[string]any{"id": stream, "finish": true, "content": clipText(text)},
			},
		})
	}
	rt.emit(in)
}

func (rt *Runtime) pushWeComAI(ctx context.Context, acct Account, chatID, text string) error {
	sock := rt.socket(acct.ID)
	if sock == nil {
		return fmt.Errorf("wecom ai bot is not connected")
	}
	kind, id, ok := strings.Cut(chatID, ":")
	chatType := 0
	if ok && (kind == "single" || kind == "direct") {
		chatType = 1
	} else if ok && kind == "group" {
		chatType = 2
	} else {
		id = chatID
	}
	return sock.write(ctx, map[string]any{
		"cmd":     "aibot_send_msg",
		"headers": map[string]string{"req_id": reqID()},
		"body": map[string]any{
			"chatid": id, "chat_type": chatType, "msgtype": "markdown",
			"markdown": map[string]string{"content": clipText(text)},
		},
	})
}

func (rt *Runtime) wecomToken(ctx context.Context, acct Account) (string, error) {
	if v, ok := rt.takeToken("wecom:" + acct.ID); ok {
		return v, nil
	}
	corp := cfg(acct, "corp_id")
	secret := rt.secret(acct.ID, "corp_secret")
	if corp == "" || secret == "" {
		return "", fatalf("missing wecom corp id or secret")
	}
	q := url.Values{}
	q.Set("corpid", corp)
	q.Set("corpsecret", secret)
	raw, code, err := rt.get(ctx, "https://qyapi.weixin.qq.com/cgi-bin/gettoken?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	m, err := decodeJSON(raw)
	if err != nil {
		return "", err
	}
	if code == 401 || jnum(m["errcode"]) == 40001 || jnum(m["errcode"]) == 40013 {
		return "", fatalf("wecom rejected the secret")
	}
	access := jstr(m["access_token"])
	if access == "" {
		return "", fmt.Errorf("wecom token: %s", jstr(m["errmsg"]))
	}
	rt.keepToken("wecom:"+acct.ID, access, time.Duration(jnum(m["expires_in"]))*time.Second)
	return access, nil
}

func (rt *Runtime) pushWeCom(ctx context.Context, acct Account, chatID, text string) error {
	token, err := rt.wecomToken(ctx, acct)
	if err != nil {
		return err
	}
	agent, _ := strconv.Atoi(cfg(acct, "agent_id"))
	raw, code, err := rt.postJSON(ctx, "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+url.QueryEscape(token), nil, map[string]any{
		"touser": chatID, "msgtype": "text", "agentid": agent,
		"text": map[string]string{"content": clipText(text)},
	})
	if err != nil {
		return err
	}
	if code >= 300 {
		return statusErr(code, raw, "wecom send")
	}
	m, _ := decodeJSON(raw)
	if jnum(m["errcode"]) != 0 && len(m) > 0 {
		return fmt.Errorf("wecom send: %s", jstr(m["errmsg"]))
	}
	return nil
}

func (rt *Runtime) weixinToken(ctx context.Context, acct Account) (string, error) {
	if v, ok := rt.takeToken("wxoa:" + acct.ID); ok {
		return v, nil
	}
	appID := cfg(acct, "app_id")
	secret := rt.secret(acct.ID, "app_secret")
	if appID == "" || secret == "" {
		return "", fatalf("missing weixin app id or secret")
	}
	q := url.Values{}
	q.Set("grant_type", "client_credential")
	q.Set("appid", appID)
	q.Set("secret", secret)
	raw, code, err := rt.get(ctx, "https://api.weixin.qq.com/cgi-bin/token?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	m, err := decodeJSON(raw)
	if err != nil {
		return "", err
	}
	if code == 401 || jnum(m["errcode"]) == 40001 || jnum(m["errcode"]) == 40125 {
		return "", fatalf("weixin rejected the secret")
	}
	access := jstr(m["access_token"])
	if access == "" {
		return "", fmt.Errorf("weixin token: %s", jstr(m["errmsg"]))
	}
	rt.keepToken("wxoa:"+acct.ID, access, time.Duration(jnum(m["expires_in"]))*time.Second)
	return access, nil
}

func (rt *Runtime) pushWeixin(ctx context.Context, acct Account, chatID, text string) error {
	token, err := rt.weixinToken(ctx, acct)
	if err != nil {
		return err
	}
	raw, code, err := rt.postJSON(ctx, "https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token="+url.QueryEscape(token), nil, map[string]any{
		"touser": chatID, "msgtype": "text",
		"text": map[string]string{"content": clipText(text)},
	})
	if err != nil {
		return err
	}
	if code >= 300 {
		return statusErr(code, raw, "weixin send")
	}
	m, _ := decodeJSON(raw)
	if jnum(m["errcode"]) != 0 && len(m) > 0 {
		return fmt.Errorf("weixin send: %s", jstr(m["errmsg"]))
	}
	return nil
}

type wxXML struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        string   `xml:"MsgId"`
	Encrypt      string   `xml:"Encrypt"`
	Recognition  string   `xml:"Recognition"`
}

func wxSigned(token, ts, nonce, extra, got string) bool {
	if strings.TrimSpace(token) == "" {
		return true
	}
	if strings.TrimSpace(got) == "" {
		return false
	}
	parts := []string{token, ts, nonce}
	if extra != "" {
		parts = append(parts, extra)
	}
	return sigEqual(strings.ToLower(sha1Join(parts...)), strings.ToLower(strings.TrimSpace(got)))
}

func (rt *Runtime) hookWX(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	token := rt.secret(acct.ID, "token")
	aesKey := rt.secret(acct.ID, "encoding_aes_key")
	q := r.URL.Query()
	if r.Method == http.MethodGet {
		echo := q.Get("echostr")
		extra := ""
		got := q.Get("signature")
		if ms := q.Get("msg_signature"); ms != "" {
			got = ms
			extra = echo
		}
		if !wxSigned(token, q.Get("timestamp"), q.Get("nonce"), extra, got) {
			writePlain(w, 403, "bad signature")
			return
		}
		if aesKey != "" && echo != "" {
			plain, err := decryptWeCom(aesKey, echo)
			if err != nil {
				writePlain(w, 400, "bad echo")
				return
			}
			writePlain(w, 200, plain)
			return
		}
		writePlain(w, 200, echo)
		return
	}
	var outer wxXML
	if xml.Unmarshal(body, &outer) != nil {
		trimmed := bytes.TrimSpace(body)
		if acct.Platform == "wecom_ai_bot" && len(trimmed) > 0 && trimmed[0] == '{' {
			rt.hookWeComAIJSON(w, acct, trimmed)
			return
		}
		writePlain(w, 400, "bad xml")
		return
	}
	if outer.Encrypt != "" {
		extra := outer.Encrypt
		got := q.Get("msg_signature")
		if got == "" {
			got = q.Get("signature")
			extra = ""
		}
		if !wxSigned(token, q.Get("timestamp"), q.Get("nonce"), extra, got) {
			writePlain(w, 403, "bad signature")
			return
		}
		plain, err := decryptWeCom(aesKey, outer.Encrypt)
		if err != nil {
			writePlain(w, 400, "bad crypt")
			return
		}
		outer = wxXML{}
		if xml.Unmarshal([]byte(plain), &outer) != nil {
			writePlain(w, 400, "bad inner xml")
			return
		}
	} else if !wxSigned(token, q.Get("timestamp"), q.Get("nonce"), "", firstNonEmpty(q.Get("signature"), q.Get("msg_signature"))) {
		writePlain(w, 403, "bad signature")
		return
	}
	writePlain(w, 200, "success")
	if strings.EqualFold(outer.MsgType, "event") || outer.FromUserName == "" {
		return
	}
	text := strings.TrimSpace(outer.Content)
	if text == "" {
		text = strings.TrimSpace(outer.Recognition)
	}
	if text == "" {
		text = placeholder(outer.MsgType)
	}
	if text == "" {
		return
	}
	chatID := outer.FromUserName
	in := Inbound{
		AccountID: acct.ID, Platform: acct.Platform, Kind: "direct", ChatID: chatID,
		SenderID: chatID, SenderName: chatID, Text: text, MessageID: outer.MsgId,
	}
	switch acct.Platform {
	case "weixin_official_account":
		in.Reply = func(text string) error {
			ctx, cancel := replyCtx()
			defer cancel()
			return rt.pushWeixin(ctx, acct, chatID, text)
		}
	case "wecom":
		in.Reply = func(text string) error {
			ctx, cancel := replyCtx()
			defer cancel()
			return rt.pushWeCom(ctx, acct, chatID, text)
		}
	default:
		in.Reply = func(string) error {
			return fmt.Errorf("wecom ai webhook cannot wait for the model; use long connection")
		}
	}
	rt.emit(in)
}

func (rt *Runtime) hookWeComAIJSON(w http.ResponseWriter, acct Account, body []byte) {
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	writePlain(w, 200, "success")
	text := jstr(dig(m, "text", "content"))
	if text == "" {
		text = placeholder(jstr(m["msgtype"]))
	}
	if text == "" {
		return
	}
	user := jstr(dig(m, "from", "userid"))
	kind := "direct"
	chatID := "single:" + user
	if jstr(m["chattype"]) == "group" {
		kind = "group"
		if id := jstr(m["chatid"]); id != "" {
			chatID = "group:" + id
		}
	}
	in := Inbound{
		AccountID: acct.ID, Platform: acct.Platform, Kind: kind, ChatID: chatID,
		SenderID: user, Text: text, Mentioned: true, MessageID: jstr(m["msgid"]),
	}
	in.Reply = func(string) error {
		return fmt.Errorf("wecom ai webhook cannot wait for the model; use long connection")
	}
	rt.emit(in)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
