package im

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (rt *Runtime) runSlack(ctx context.Context, acct Account) error {
	if modeOf(acct, "socket") == "webhook" {
		return rt.runWait(ctx, acct)
	}
	appToken := rt.secret(acct.ID, "app_token")
	if appToken == "" || rt.secret(acct.ID, "bot_token") == "" {
		return fatalf("missing slack app token or bot token")
	}
	self, err := rt.slackSelf(ctx, acct)
	if err != nil {
		return err
	}
	raw, code, err := rt.postJSON(ctx, "https://slack.com/api/apps.connections.open", bearer(appToken), map[string]any{})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("slack rejected the app token (%d)", code)
	}
	body, err := decodeJSON(raw)
	if err != nil {
		return err
	}
	if !jbool(body["ok"]) || jstr(body["url"]) == "" {
		return fmt.Errorf("slack socket: %s", clipRunes(string(raw), 180))
	}
	c, err := dialWS(ctx, jstr(body["url"]), nil)
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
		if env := jstr(frame["envelope_id"]); env != "" {
			_ = sock.write(ctx, map[string]string{"envelope_id": env})
		}
		if jstr(frame["type"]) == "disconnect" {
			return fmt.Errorf("slack disconnected")
		}
		payload := asMap(frame["payload"])
		event := asMap(payload["event"])
		if len(event) == 0 {
			event = asMap(frame["event"])
		}
		if jstr(event["type"]) != "message" {
			continue
		}
		rt.ingestSlack(acct, self, event)
	}
}

func (rt *Runtime) ingestSlack(acct Account, self string, event map[string]any) {
	subtype := jstr(event["subtype"])
	if subtype == "message_changed" || subtype == "message_deleted" || subtype == "bot_message" || jstr(event["bot_id"]) != "" {
		return
	}
	channel := jstr(event["channel"])
	user := jstr(event["user"])
	if channel == "" || user == "" {
		return
	}
	kind := "group"
	switch jstr(event["channel_type"]) {
	case "im", "mpim":
		kind = "direct"
	}
	text := jstr(event["text"])
	if text == "" && len(asList(event["files"])) > 0 {
		text = "[file]"
	}
	mentioned := kind == "direct"
	if self != "" && strings.Contains(text, "<@"+self+">") {
		mentioned = true
	}
	in := Inbound{
		AccountID: acct.ID, Platform: "slack", Kind: kind, ChatID: channel,
		SenderID: user, SenderName: user, SelfID: self,
		Text: text, Mentioned: mentioned, MessageID: channel + ":" + jstr(event["ts"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushSlack(ctx, acct, channel, text)
	}
	rt.emit(in)
}

func (rt *Runtime) pushSlack(ctx context.Context, acct Account, chatID, text string) error {
	bot := rt.secret(acct.ID, "bot_token")
	if bot == "" {
		return fatalf("missing slack bot token")
	}
	raw, code, err := rt.postJSON(ctx, "https://slack.com/api/chat.postMessage", bearer(bot), map[string]string{
		"channel": chatID, "text": clipText(text),
	})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("slack rejected the bot token (%d)", code)
	}
	body, err := decodeJSON(raw)
	if err != nil {
		return err
	}
	if !jbool(body["ok"]) {
		return fmt.Errorf("slack send: %s", jstr(body["error"]))
	}
	return nil
}

func (rt *Runtime) hookSlack(w http.ResponseWriter, r *http.Request, acct Account, body []byte) {
	secret := rt.secret(acct.ID, "signing_secret")
	ts := r.Header.Get("X-Slack-Request-Timestamp")
	got := r.Header.Get("X-Slack-Signature")
	if secret == "" || !sigEqual(slackSig(secret, ts, body), got) {
		writePlain(w, 401, "bad signature")
		return
	}
	if n, err := strconv.ParseInt(ts, 10, 64); err == nil {
		if d := time.Since(time.Unix(n, 0)); d > 5*time.Minute || d < -5*time.Minute {
			writePlain(w, 401, "stale timestamp")
			return
		}
	}
	m, err := decodeJSON(body)
	if err != nil {
		writePlain(w, 400, "bad json")
		return
	}
	if jstr(m["type"]) == "url_verification" {
		writeJSONCode(w, 200, map[string]string{"challenge": jstr(m["challenge"])})
		return
	}
	writePlain(w, 200, "ok")
	if jstr(m["type"]) != "event_callback" {
		return
	}
	event := asMap(m["event"])
	if jstr(event["type"]) != "message" {
		return
	}
	self, _ := rt.slackSelf(r.Context(), acct)
	rt.ingestSlack(acct, self, event)
}

func (rt *Runtime) slackSelf(ctx context.Context, acct Account) (string, error) {
	if id := rt.cursor(acct.ID + ":slack-self"); id != "" {
		return id, nil
	}
	bot := rt.secret(acct.ID, "bot_token")
	if bot == "" {
		return "", nil
	}
	raw, code, err := rt.postJSON(ctx, "https://slack.com/api/auth.test", bearer(bot), map[string]any{})
	if err != nil {
		return "", err
	}
	if code != 200 {
		return "", nil
	}
	body, err := decodeJSON(raw)
	if err != nil || !jbool(body["ok"]) {
		return "", nil
	}
	id := jstr(body["user_id"])
	if id != "" {
		rt.setCursor(acct.ID+":slack-self", id)
	}
	return id, nil
}
