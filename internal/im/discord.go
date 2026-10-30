package im

import (
	"context"
	"fmt"
	"strings"
)

const discordIntents = (1 << 0) | (1 << 9) | (1 << 12) | (1 << 15)

func (rt *Runtime) runDiscord(ctx context.Context, acct Account) error {
	token := rt.secret(acct.ID, "bot_token")
	if token == "" {
		return fatalf("missing discord bot token")
	}
	raw, code, err := rt.get(ctx, "https://discord.com/api/v10/gateway/bot", map[string]string{"Authorization": "Bot " + token})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("discord rejected the token (%d)", code)
	}
	if code != 200 {
		return statusErr(code, raw, "discord gateway")
	}
	body, err := decodeJSON(raw)
	if err != nil {
		return err
	}
	gateway := jstr(body["url"])
	if gateway == "" {
		return fmt.Errorf("discord gateway missing url")
	}
	gateway = withQuery(gateway, "v", "10", "encoding", "json")
	self := ""
	return rt.runOpGateway(ctx, acct, gateway, nil, func() (any, error) {
		return map[string]any{"op": 2, "d": map[string]any{
			"token":   token,
			"intents": discordIntents,
			"properties": map[string]string{
				"os": "windows", "browser": "yoyo", "device": "yoyo",
			},
		}}, nil
	}, func(t string, d map[string]any) {
		if t == "READY" {
			self = jstr(dig(d, "user", "id"))
			return
		}
		if t != "MESSAGE_CREATE" {
			return
		}
		rt.ingestDiscord(acct, self, d)
	})
}

func (rt *Runtime) ingestDiscord(acct Account, self string, d map[string]any) {
	author := asMap(d["author"])
	if jbool(author["bot"]) {
		return
	}
	channel := jstr(d["channel_id"])
	if channel == "" {
		return
	}
	kind := "group"
	if jstr(d["guild_id"]) == "" {
		kind = "direct"
	}
	text := jstr(d["content"])
	if text == "" && len(asList(d["attachments"])) > 0 {
		text = "[file]"
	}
	mentioned := kind == "direct"
	for _, item := range asList(d["mentions"]) {
		if self != "" && jstr(asMap(item)["id"]) == self {
			mentioned = true
		}
	}
	if self != "" && (strings.Contains(text, "<@"+self+">") || strings.Contains(text, "<@!"+self+">")) {
		mentioned = true
	}
	atAll := jbool(d["mention_everyone"])
	in := Inbound{
		AccountID: acct.ID, Platform: "discord", Kind: kind, ChatID: channel,
		SenderID: jstr(author["id"]), SenderName: jstr(author["username"]), SelfID: self,
		Text: text, Mentioned: mentioned, AtAll: atAll, MessageID: jstr(d["id"]),
	}
	in.Reply = func(text string) error {
		ctx, cancel := replyCtx()
		defer cancel()
		return rt.pushDiscord(ctx, acct, channel, text)
	}
	rt.emit(in)
}

func (rt *Runtime) pushDiscord(ctx context.Context, acct Account, chatID, text string) error {
	token := rt.secret(acct.ID, "bot_token")
	if token == "" {
		return fatalf("missing discord bot token")
	}
	raw, code, err := rt.postJSON(ctx, "https://discord.com/api/v10/channels/"+chatID+"/messages", map[string]string{
		"Authorization": "Bot " + token,
	}, map[string]string{"content": clipText(text)})
	if err != nil {
		return err
	}
	if code == 401 || code == 403 {
		return fatalf("discord rejected the token (%d)", code)
	}
	if code >= 300 {
		return statusErr(code, raw, "discord send")
	}
	return nil
}
