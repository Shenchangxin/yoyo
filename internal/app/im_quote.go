// im_quote.go 让人在主通道里直接「回复」某一条消息来处理它，不用再敲 /答 编号。
// 发到主通道的每一条，如果只带一个短编号（任务的 /答、审批的 /批准、短讯的 /有用），就记下「平台消息编号 → 短编号」。
// 人引用回复时：
//   - 先按平台给的被引用编号（Inbound.QuoteID）找；
//   - 找不到（钉钉这类拿不到机器人消息编号的平台、重启之后）就从被引用的原文（Inbound.QuoteText）里认出我们自己写的命令。
//
// 找到之后按那件事的种类处理：
//   - 任务：整句话就是回答，交给那张工单；
//   - 审批：只认「批准 / 拒绝 / 总是」这类短词，别的话照旧进主会话；
//   - 短讯：只认「有用 / 没用 / 别提醒 / 关掉」，别的话照旧进主会话。
//
// 一条消息里有好几个编号（例如 /状态 的清单）时不记，免得回错事。只有主通道里绑定的那个人能这样回复。
// 不拥有：平台收发与引用字段的解析（internal/im/quote.go）、短编号本身（im_home.go）。

package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/im"
)

// imQuotedMax 是最多记多少条平台编号。多了丢最早的。
const imQuotedMax = 500

// imQuoted 是一条发出去的消息带的短编号。Kind 是 ticket、approval 或 speak，和 imCode.Kind 一致。
type imQuoted struct {
	Kind string
	Code string
	At   time.Time
}

// imQuoteCmd 认消息里我们自己写的命令和编号：/答 abc、／批准 k2m。编号字母表见 imCodeAlphabet，长 3 到 4 位。
var imQuoteCmd = regexp.MustCompile(`[/／]\s*([^\s/／]+)\s+([a-z0-9]{3,4})(?:[^a-z0-9]|$)`)

// commandKind 把命令归到短编号的种类上。不是带编号的命令时为空。
func commandKind(cmd string) string {
	switch imCommands[strings.ToLower(cmd)] {
	case "answer":
		return "ticket"
	case "approve", "deny":
		return "approval"
	case "used", "ignore", "later", "mute":
		return "speak"
	default:
		return ""
	}
}

// quotedCode 从一段文字里认出唯一的那个短编号和它的种类。没有、或者有不止一个编号时都返回空。
// 同一个编号出现多次（「回 /批准 k2m · /拒绝 k2m」）算一个。
func quotedCode(text string) (kind, code string) {
	for _, m := range imQuoteCmd.FindAllStringSubmatch(text, -1) {
		k := commandKind(m[1])
		if k == "" {
			continue
		}
		if code != "" && (m[2] != code || k != kind) {
			return "", ""
		}
		kind, code = k, m[2]
	}
	return kind, code
}

func quoteKey(account, platformID string) string {
	return account + "\n" + platformID
}

// imRemember 记下刚发到主通道的这条带的短编号。平台没回编号、或者这条不带唯一编号时什么都不做。
func (a *App) imRemember(account, platformID, text string) {
	platformID = strings.TrimSpace(platformID)
	if platformID == "" {
		return
	}
	kind, code := quotedCode(text)
	if code == "" {
		return
	}
	now := time.Now()
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	// 引用编号和短编号一起落在 im/codes.json（im_codes.go）：重启或换手之后，人回复重启前发的那条也认得出。
	a.imCodesEdit(func() bool {
		if a.imh.quotes == nil {
			a.imh.quotes = map[string]imQuoted{}
		}
		for k, v := range a.imh.quotes {
			if now.Sub(v.At) > imCodeTTL {
				delete(a.imh.quotes, k)
			}
		}
		if len(a.imh.quotes) >= imQuotedMax {
			oldest, at := "", now
			for k, v := range a.imh.quotes {
				if v.At.Before(at) {
					oldest, at = k, v.At
				}
			}
			delete(a.imh.quotes, oldest)
		}
		a.imh.quotes[quoteKey(account, platformID)] = imQuoted{Kind: kind, Code: code, At: now}
		return true
	})
}

// imQuotedCode 找出人这条消息引用的那一条带的短编号：先按平台编号查（重读 im/codes.json，只读），再从被引用的原文里认。
func (a *App) imQuotedCode(in im.Inbound) (kind, code string) {
	if id := strings.TrimSpace(in.QuoteID); id != "" {
		var q imQuoted
		ok := false
		a.imh.mu.Lock()
		a.imCodesEdit(func() bool {
			q, ok = a.imh.quotes[quoteKey(in.AccountID, id)]
			return false
		})
		a.imh.mu.Unlock()
		if ok && time.Since(q.At) <= imCodeTTL {
			return q.Kind, q.Code
		}
	}
	return quotedCode(in.QuoteText)
}

// imPushID 往主通道发一条，返回平台给它的编号（拿不到时为空）。测试替身优先。
func (a *App) imPushID(h IMHome, text string) (string, error) {
	if a.imh.pushID != nil {
		return a.imh.pushID(h.Account, h.Chat, text)
	}
	if a.imh.push != nil {
		return "", a.imh.push(h.Account, h.Chat, text)
	}
	if a.IM == nil {
		return "", fmt.Errorf("im unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.IM.PushID(ctx, h.Account, h.Chat, text)
}

// shortWord 把一句短回复收成好比较的样子：小写、去掉首尾空白和常见标点。
func shortWord(text string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(text), " \t\r\n。.!！~～,，、?？"))
}

// approvalWord 把回复审批的一句短话译成决定：once、always、deny。认不出时为空。
func approvalWord(text string) string {
	switch shortWord(text) {
	case "批准", "同意", "可以", "好", "好的", "行", "允许", "ok", "okay", "yes", "y", "approve", "批":
		return "once"
	case "总是", "总是批准", "总是允许", "以后都行", "always":
		return "always"
	case "拒绝", "不", "不行", "不要", "不同意", "别", "no", "n", "deny":
		return "deny"
	default:
		return ""
	}
}

// feedbackWord 把回复短讯的一句短话译成反馈：used、ignore、later、mute。认不出时为空。
func feedbackWord(text string) string {
	switch shortWord(text) {
	case "有用", "用上了", "谢谢", "收到", "used", "useful", "thanks":
		return "used"
	case "没用", "无用", "ignore", "useless":
		return "ignore"
	case "别提醒", "稍后", "今天别提", "later":
		return "later"
	case "关掉", "别再说", "mute":
		return "mute"
	default:
		return ""
	}
}

// imQuoteRoute 处理主通道里对某一条的回复。处理了返回真，这条就不再进主会话。
// 不是主通道、没有引用、引用的那条不带编号、或者回复审批 / 短讯的话认不出时返回假，照旧进对话。
func (a *App) imQuoteRoute(in im.Inbound) bool {
	if in.QuoteID == "" && in.QuoteText == "" {
		return false
	}
	if !a.isIMHome(in) {
		return false
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return false
	}
	kind, code := a.imQuotedCode(in)
	if code == "" {
		return false
	}
	zh := a.briefChinese()
	reply := func(s string) {
		if in.Reply != nil && strings.TrimSpace(s) != "" {
			_ = in.Reply(s)
		}
	}
	notFound := pick(zh, "这一条已经处理过或过期了。回 /状态 看还在等的。", "That one is already handled or expired. Reply /status for what is waiting.")
	switch kind {
	case "ticket":
		_, c, ok := a.imLookup(code, "ticket")
		if !ok {
			reply(notFound)
			return true
		}
		if err := a.answerTicket(c.Target, text); err != nil {
			reply(pick(zh, "没送到：", "Not delivered: ") + err.Error())
			return true
		}
		reply(pick(zh, "已经交给它了。", "Passed on."))
		return true
	case "approval":
		decision := approvalWord(text)
		if decision == "" {
			return false
		}
		_, c, ok := a.imLookup(code, "approval")
		if !ok || a.Gate == nil {
			reply(notFound)
			return true
		}
		err := a.ResolveApproval(c.Target, decision)
		a.imForget(code)
		switch {
		case err != nil:
			reply(notFound)
		case decision == "deny":
			reply(pick(zh, "已拒绝。", "Denied."))
		default:
			reply(pick(zh, "已批准。", "Approved."))
		}
		return true
	case "speak":
		fb := feedbackWord(text)
		if fb == "" {
			return false
		}
		_, c, ok := a.imLookup(code, "speak")
		if !ok || !a.DeskFeedback(c.Class, fb, c.MemoryID) {
			reply(notFound)
			return true
		}
		reply(pick(zh, "记下了。", "Noted."))
		return true
	}
	return false
}
