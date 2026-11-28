// quote.go 管「回复某一条」这件事的平台这一半：
//   - 发出去时拿回平台给这条消息的编号（PushID），应用层记下「这个编号 → 哪张工单 / 哪条审批」；
//   - 收进来时记下人回复的是哪一条（Inbound.QuoteID），平台给了原文就连原文一起带上（Inbound.QuoteText）。
//
// 能拿回编号的：Telegram（message_id / reply_to_message）、飞书（message_id / parent_id）、Slack（ts / thread_ts，回复即开线程）、
// Discord（id / message_reference）、Matrix（event_id / m.in_reply_to）、OneBot（等 echo 回执 / reply 段）、QQ 官方（id / message_reference）。
// 钉钉的回复只带被引用的原文；企业微信、公众号等不给引用信息，这些平台仍然要用 /答 编号。
// 不拥有：编号和工单的对应关系、按引用路由命令（internal/app/im_quote.go）。

package im

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// onebotEchoWait 是等 OneBot 实现回执的上限。超过了不算发送失败，只是这条以后认不出引用。
const onebotEchoWait = 5 * time.Second

// PushID 和 Push 一样往一个已知会话主动发文本，另外返回平台给这条消息的编号。
// 平台不回编号（或这个平台还没接上编号）时返回空字符串和 nil，调用方照常当作发送成功。
func (rt *Runtime) PushID(ctx context.Context, accountID, chatID, text string) (string, error) {
	acct, ok := rt.account(accountID)
	if !ok {
		return "", fmt.Errorf("unknown account")
	}
	if !acct.Enabled {
		return "", fmt.Errorf("account is stopped")
	}
	text = strings.TrimSpace(text)
	chatID = strings.TrimSpace(chatID)
	if text == "" || chatID == "" {
		return "", fmt.Errorf("chat id and text are required")
	}
	return rt.pushID(ctx, acct, chatID, text)
}

// pushID 按平台选能拿回编号的发送方式；其余平台走原来的 push，编号为空。
func (rt *Runtime) pushID(ctx context.Context, acct Account, chatID, text string) (string, error) {
	text = clipText(text)
	switch acct.Platform {
	case "telegram":
		return rt.sendTelegram(ctx, acct, chatID, text)
	case "lark":
		return rt.sendLark(ctx, acct, chatID, text)
	case "slack":
		return rt.sendSlack(ctx, acct, chatID, text)
	case "discord":
		return rt.sendDiscord(ctx, acct, chatID, text)
	case "matrix":
		return rt.sendMatrix(ctx, acct, chatID, text)
	case "onebot":
		return rt.sendOneBot(ctx, acct, chatID, text)
	case "qq_official", "qq_official_webhook":
		return rt.sendQQ(ctx, acct, chatID, text, "")
	default:
		return "", rt.push(ctx, acct, chatID, text)
	}
}

// sendOneBot 发一条并等实现的回执，返回回执里的 message_id。
// OneBot 的回执走同一条 WebSocket，按 echo 对上（readOneBot 把带 echo 的帧交到 rt.echoes 里等着的通道）。
// 实现回了 failed 时报错；等不到回执时不报错、编号为空（消息多半已经发出去了）。
func (rt *Runtime) sendOneBot(ctx context.Context, acct Account, chatID, text string) (string, error) {
	sock := rt.socket(acct.ID)
	if sock == nil {
		return "", fmt.Errorf("onebot is not connected")
	}
	action, params := onebotTarget(chatID, text)
	echo := reqID()
	ch := make(chan map[string]any, 1)
	rt.echoes.Store(echo, ch)
	defer rt.echoes.Delete(echo)
	if err := sock.write(ctx, map[string]any{"action": action, "params": params, "echo": echo}); err != nil {
		return "", err
	}
	wait := time.NewTimer(onebotEchoWait)
	defer wait.Stop()
	select {
	case m := <-ch:
		if strings.EqualFold(jstr(m["status"]), "failed") {
			return "", fmt.Errorf("onebot send: %s", firstText(jstr(m["wording"]), jstr(m["msg"]), "failed"))
		}
		return jstr(dig(m, "data", "message_id")), nil
	case <-wait.C:
		return "", nil
	case <-ctx.Done():
		return "", nil
	}
}

// onebotTarget 把 private:<qq> / group:<群号> 译成 OneBot 的动作和参数。
func onebotTarget(chatID, text string) (string, map[string]any) {
	action := "send_private_msg"
	params := map[string]any{"message": clipText(text)}
	if strings.HasPrefix(chatID, "group:") {
		action = "send_group_msg"
		params["group_id"] = numOrString(strings.TrimPrefix(chatID, "group:"))
	} else {
		params["user_id"] = numOrString(strings.TrimPrefix(strings.TrimPrefix(chatID, "private:"), "direct:"))
	}
	return action, params
}

var (
	// cqReplyID 认字符串格式消息里的 [CQ:reply,id=123]（id 可能不是第一个参数，也可能是负数）。
	cqReplyID = regexp.MustCompile(`\[CQ:reply,(?:[^\]]*,)?id=(-?\d+)`)
	// cqReplyAll 用来把回复段从正文里去掉，免得「[CQ:reply,id=…]」被当成回答的一部分。
	cqReplyAll = regexp.MustCompile(`\[CQ:reply,[^\]]*\]`)
)

// onebotQuote 找出 OneBot 消息回复的那一条的编号：数组格式看 type=reply 的段，字符串格式看 CQ 码。
func onebotQuote(msg any, raw string) string {
	if segs, ok := msg.([]any); ok {
		for _, item := range segs {
			seg := asMap(item)
			if jstr(seg["type"]) != "reply" {
				continue
			}
			if id := jstr(asMap(seg["data"])["id"]); id != "" {
				return id
			}
		}
	}
	text := raw
	if s, ok := msg.(string); ok && s != "" {
		text = s
	}
	if m := cqReplyID.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

// matrixFallback 取出 Matrix 回复正文开头的引用块（以「> 」开头的那几行），这是规范要求的纯文本回退，内容是被回复那条的原文。
func matrixFallback(body string) string {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, ">") {
			break
		}
		lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, ">")))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// matrixBody 去掉引用块，剩下人自己写的那部分。没有引用块时原样返回。
func matrixBody(body string) string {
	lines := strings.Split(body, "\n")
	i := 0
	for i < len(lines) && strings.HasPrefix(lines[i], ">") {
		i++
	}
	if i == 0 {
		return body
	}
	return strings.TrimSpace(strings.Join(lines[i:], "\n"))
}

// firstText 返回第一个去掉空白后不为空的值。
func firstText(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
