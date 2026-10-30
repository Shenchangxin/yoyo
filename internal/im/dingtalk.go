package im

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (rt *Runtime) runDingTalk(ctx context.Context, acct Account) error {
	if modeOf(acct, "stream") == "webhook" {
		return rt.runWait(ctx, acct)
	}
	clientID := cfg(acct, "client_id")
	secret := rt.secret(acct.ID, "client_secret")
	if clientID == "" || secret == "" {
		return fatalf("missing dingtalk client id or secret")
	}
	raw, code, err := rt.postJSON(ctx, "https://api.dingtalk.com/v1.0/gateway/connections/open", nil, map[string]any{
		"clientId": clientID, "clientSecret": secret,
		"subscriptions": []map[string]string{
			{"type": "EVENT", "topic": "*"},
			{"type": "CALLBACK", "topic": "/v1.0/im/bot/messages/get"},
		},
		"ua": "yoyo",
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("dingtalk rejected the secret (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "dingtalk stream")
	}
	body, err := decodeJSON(raw)
	if err != nil {
		return err
	}
	endpoint := jstr(body["endpoint"])
	ticket := jstr(body["ticket"])
	if endpoint == "" {
		endpoint = jstr(dig(body, "data", "endpoint"))
		ticket = jstr(dig(body, "data", "ticket"))
	}
	if endpoint == "" || ticket == "" {
		return fmt.Errorf("dingtalk stream missing endpoint")
	}
	wsURL := endpoint
	if strings.Contains(wsURL, "?") {
		wsURL += "&ticket=" + url.QueryEscape(ticket)
	} else {
		wsURL += "?ticket=" + url.QueryEscape(ticket)
	}
	c, err := dialWS(ctx, wsURL, nil)
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
		headers := asMap(frame["headers"])
		messageID := jstr(headers["messageId"])
		if messageID != "" {
			_ = sock.write(ctx, map[string]any{
				"code":    200,
				"headers": map[string]string{"messageId": messageID, "contentType": "application/json"},
				"message": "OK",
				"data":    "",
			})
		}
		topic := jstr(headers["topic"])
		kind := strings.ToUpper(jstr(frame["type"]))
		if kind == "SYSTEM" || strings.Contains(strings.ToLower(topic), "ping") {
			continue
		}
		if !strings.Contains(topic, "bot/messages") && kind != "CALLBACK" {
			continue
		}
		data := asMap(frame["data"])
		if jstr(data["msgtype"]) == "" && jstr(data["msgType"]) == "" && jstr(data["text"]) == "" && len(asMap(data["text"])) == 0 {
			continue
		}
		rt.ingestDingTalk(acct, data)
	}
}

func (rt *Runtime) ingestDingTalk(acct Account, data map[string]any) {
	msgType := jstr(data["msgtype"])
	if msgType == "" {
		msgType = jstr(data["msgType"])
	}
	text := jstr(dig(data, "text", "content"))
	if text == "" {
		text = placeholder(msgType)
	}
	conversation := jstr(data["conversationId"])
	staff := jstr(data["senderStaffId"])
	if staff == "" {
		staff = jstr(data["senderId"])
	}
	kind := "group"
	chatID := "group:" + conversation
	if jstr(data["conversationType"]) == "1" {
		kind = "direct"
		chatID = "direct:" + staff
	}
	mentioned := kind == "direct"
	if _, ok := data["isInAtList"]; ok {
		mentioned = jbool(data["isInAtList"]) || kind == "direct"
	} else if kind == "group" {
		mentioned = true
	}
	webhook := jstr(data["sessionWebhook"])
	in := Inbound{
		AccountID: acct.ID, Platform: "dingtalk", Kind: kind, ChatID: chatID,
		SenderID: staff, SenderName: jstr(data["senderNick"]), SelfID: jstr(data["chatbotUserId"]),
		Text: text, Mentioned: mentioned, MessageID: jstr(data["msgId"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		if webhook != "" {
			raw, code, err := rt.postJSON(ctx, webhook, nil, map[string]any{
				"msgtype": "text",
				"text":    map[string]string{"content": clipText(text)},
			})
			if err != nil {
				return err
			}
			if code >= 300 {
				return statusErr(code, raw, "dingtalk reply")
			}
			return nil
		}
		return rt.pushDingTalk(ctx, acct, chatID, text)
	}
	rt.emit(in)
}

func (rt *Runtime) dingToken(ctx context.Context, acct Account) (string, error) {
	if v, ok := rt.takeToken("ding:" + acct.ID); ok {
		return v, nil
	}
	id := cfg(acct, "client_id")
	secret := rt.secret(acct.ID, "client_secret")
	if id == "" || secret == "" {
		return "", fatalf("missing dingtalk client id or secret")
	}
	raw, code, err := rt.postJSON(ctx, "https://api.dingtalk.com/v1.0/oauth2/accessToken", nil, map[string]string{
		"appKey": id, "appSecret": secret,
	})
	if err != nil {
		return "", err
	}
	if code == 401 || code == 403 {
		return "", fatalf("dingtalk rejected the secret (%d)", code)
	}
	m, err := decodeJSON(raw)
	if err != nil {
		return "", err
	}
	access := jstr(m["accessToken"])
	if access == "" {
		return "", fmt.Errorf("dingtalk token missing")
	}
	rt.keepToken("ding:"+acct.ID, access, time.Duration(jnum(m["expireIn"]))*time.Second)
	return access, nil
}

func (rt *Runtime) pushDingTalk(ctx context.Context, acct Account, chatID, text string) error {
	token, err := rt.dingToken(ctx, acct)
	if err != nil {
		return err
	}
	param, _ := json.Marshal(map[string]string{"content": clipText(text)})
	header := map[string]string{"x-acs-dingtalk-access-token": token}
	robot := cfg(acct, "client_id")
	kind, id, _ := strings.Cut(chatID, ":")
	var raw []byte
	var code int
	if kind == "direct" {
		raw, code, err = rt.postJSON(ctx, "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend", header, map[string]any{
			"robotCode": robot, "userIds": []string{id}, "msgKey": "sampleText", "msgParam": string(param),
		})
	} else {
		if kind == "group" {
			id = strings.TrimPrefix(chatID, "group:")
		}
		raw, code, err = rt.postJSON(ctx, "https://api.dingtalk.com/v1.0/robot/groupMessages/send", header, map[string]any{
			"robotCode": robot, "openConversationId": id, "msgKey": "sampleText", "msgParam": string(param),
		})
	}
	if err != nil {
		return err
	}
	if code >= 300 {
		return statusErr(code, raw, "dingtalk send")
	}
	return nil
}

func (rt *Runtime) hookDingTalk(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	writePlain(w, 200, "ok")
	if jstr(m["msgtype"]) == "" && len(asMap(m["text"])) == 0 {
		return
	}
	rt.ingestDingTalk(acct, m)
}
