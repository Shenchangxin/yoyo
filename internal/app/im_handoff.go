// im_handoff.go 是 daemon 把领头交给桌面、正在收尾时（见 lead.go 的 drainStep）IM 消息的去处。
// 以前收尾期间 IM 网关照常开着，进来的消息照常开新回合；新回合没跑完之前不让出，最多把收尾拖到三分钟。
// 现在收尾期间进来的、要开回合的消息记进 im/handoff.json，当场回一句「收到，交接完马上处理」；
// 命令（/答、/批准、/状态……）和对某一条的引用回复照旧当场处理，它们不开回合。
// 接手的进程领头、IM 网关起来之后（startRoles），或者桌面又关掉、收尾取消时（cancelDrain），
// 把队列整个取出来：同一个来源的按先后排队开回合，不同来源各走各的，回复经 Runtime.Push 发回原来的聊天。
// 文件锁和写法同 im_codes.go：先拿 handoff.json.lock、重读磁盘、改完先写 .tmp 再改名。
// 等得超过 imHandoffTTL 的不再开回合，只回一句请对方再发一次：半小时前的问题多半已经不是现在的问题。
// 不拥有：开回合本身（im_app.go 的 imTurn）、什么时候算收尾（lead.go）。

package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/im"
	"github.com/Shenchangxin/yoyo/internal/lease"
)

// imHandoffTTL 是一条消息在交接队列里最多等多久还照常处理。
const imHandoffTTL = 30 * time.Minute

// imHandoffMax 是队列最多留几条，超出时丢最旧的。收尾最多三分钟，正常用不到这么多；防的是坏掉的机器人刷屏。
const imHandoffMax = 50

// imHandoffItem 是一条记下来的消息：Inbound 里除了 Reply 的全部字段（Reply 是闭包，换了进程就没了，重放时用 Push 重建）。
type imHandoffItem struct {
	AccountID  string    `json:"account_id"`
	Platform   string    `json:"platform,omitempty"`
	Origin     string    `json:"origin"`
	Kind       string    `json:"kind,omitempty"`
	ChatID     string    `json:"chat_id"`
	SenderID   string    `json:"sender_id,omitempty"`
	SenderName string    `json:"sender_name,omitempty"`
	SelfID     string    `json:"self_id,omitempty"`
	Text       string    `json:"text"`
	Mentioned  bool      `json:"mentioned,omitempty"`
	AtAll      bool      `json:"at_all,omitempty"`
	Admin      bool      `json:"admin,omitempty"`
	MessageID  string    `json:"message_id,omitempty"`
	QuoteID    string    `json:"quote_id,omitempty"`
	QuoteText  string    `json:"quote_text,omitempty"`
	At         time.Time `json:"at"`
}

// imHandoffFile 是 im/handoff.json 的样子，按到达先后排。
type imHandoffFile struct {
	Items []imHandoffItem `json:"items,omitempty"`
}

var errNoHandoffHome = errors.New("im handoff: no home")

// imHandoffPath 是交接队列的位置，和 im/codes.json 放在一起。没有 home 时为空，这时不排队，照旧当场开回合。
func (a *App) imHandoffPath() string {
	if a == nil || a.Home == nil || strings.TrimSpace(a.Home.Root) == "" {
		return ""
	}
	return filepath.Join(a.Home.Root, "im", "handoff.json")
}

// imHandoffEdit 在文件锁里重读队列、执行 fn，fn 返回真时写回。坏文件当空的读，下一次写回盖成好的。
func (a *App) imHandoffEdit(fn func(f *imHandoffFile) bool) error {
	path := a.imHandoffPath()
	if path == "" {
		return errNoHandoffHome
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	unlock, err := lease.Lock(path + ".lock")
	defer unlock()
	if err != nil {
		return err
	}
	var f imHandoffFile
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, &f) != nil {
			f = imHandoffFile{}
		}
	}
	if !fn(&f) {
		return nil
	}
	raw, err := json.Marshal(f)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// draining 在这个进程领头、但已经决定让出、正在收尾时为真。
func (a *App) draining() bool {
	if a == nil {
		return false
	}
	a.lead.mu.Lock()
	defer a.lead.mu.Unlock()
	return a.lead.leading && !a.lead.drainSince.IsZero()
}

// imHold 在收尾期间把一条要开回合的消息记进交接队列，回一句「收到」，返回真；不在收尾或记不下来时返回假，调用方照旧当场开回合。
// 同一个账号的同一个平台消息编号只记一次（平台重投）。
func (a *App) imHold(in im.Inbound) bool {
	if !a.draining() {
		return false
	}
	it := imHandoffItem{
		AccountID: in.AccountID, Platform: in.Platform, Origin: in.Origin, Kind: in.Kind, ChatID: in.ChatID,
		SenderID: in.SenderID, SenderName: in.SenderName, SelfID: in.SelfID, Text: in.Text,
		Mentioned: in.Mentioned, AtAll: in.AtAll, Admin: in.Admin,
		MessageID: in.MessageID, QuoteID: in.QuoteID, QuoteText: in.QuoteText, At: time.Now(),
	}
	dup := false
	err := a.imHandoffEdit(func(f *imHandoffFile) bool {
		if it.MessageID != "" {
			for _, old := range f.Items {
				if old.AccountID == it.AccountID && old.MessageID == it.MessageID {
					dup = true
					return false
				}
			}
		}
		f.Items = append(f.Items, it)
		if n := len(f.Items); n > imHandoffMax {
			f.Items = append([]imHandoffItem(nil), f.Items[n-imHandoffMax:]...)
		}
		return true
	})
	if err != nil {
		diaglog.Info("im handoff hold failed", "component", "lead", "err", err.Error())
		return false
	}
	if !dup && in.Reply != nil {
		_ = in.Reply(pick(imTextZh(in.Text),
			"收到。yoyo 正在换到桌面上接着跑，交接完马上处理这条，不用重发。",
			"Got it. yoyo is moving over to the desktop app; this message will be handled right after, no need to resend."))
	}
	return true
}

// imHandoffTake 把队列整个取出来并清空：没等太久的放进 fresh，等过 imHandoffTTL 的放进 stale。
func (a *App) imHandoffTake(now time.Time) (fresh, stale []imHandoffItem) {
	_ = a.imHandoffEdit(func(f *imHandoffFile) bool {
		if len(f.Items) == 0 {
			return false
		}
		for _, it := range f.Items {
			if now.Sub(it.At) > imHandoffTTL {
				stale = append(stale, it)
			} else {
				fresh = append(fresh, it)
			}
		}
		f.Items = nil
		return true
	})
	return fresh, stale
}

// imReplay 在领头且不在收尾时处理交接队列。调用方用 go 调它：回合可能跑好几分钟，不能卡住选主那一拍。
func (a *App) imReplay() {
	if a == nil || a.IM == nil || !a.leading() || a.draining() {
		return
	}
	fresh, stale := a.imHandoffTake(time.Now())
	if len(fresh)+len(stale) == 0 {
		return
	}
	diaglog.Info("im handoff replay", "component", "lead", "fresh", len(fresh), "stale", len(stale))
	for _, it := range stale {
		in := a.imFromHandoff(it)
		_ = in.Reply(imStaleNote(it.Text))
	}
	byOrigin := map[string][]im.Inbound{}
	var order []string
	for _, it := range fresh {
		in := a.imFromHandoff(it)
		if _, ok := byOrigin[in.Origin]; !ok {
			order = append(order, in.Origin)
		}
		byOrigin[in.Origin] = append(byOrigin[in.Origin], in)
	}
	for _, origin := range order {
		go func(ins []im.Inbound) {
			for _, in := range ins {
				a.imTurn(in)
			}
		}(byOrigin[origin])
	}
}

// imFromHandoff 把记下的消息还原成 Inbound，回复改走 Push 发回原来的聊天。
func (a *App) imFromHandoff(it imHandoffItem) im.Inbound {
	acct, chat := it.AccountID, it.ChatID
	origin := it.Origin
	if origin == "" {
		kind := it.Kind
		if kind == "" {
			kind = "direct"
		}
		origin = acct + ":" + kind + ":" + chat
	}
	return im.Inbound{
		AccountID: acct, Platform: it.Platform, Origin: origin, Kind: it.Kind, ChatID: chat,
		SenderID: it.SenderID, SenderName: it.SenderName, SelfID: it.SelfID, Text: it.Text,
		Mentioned: it.Mentioned, AtAll: it.AtAll, Admin: it.Admin,
		MessageID: it.MessageID, QuoteID: it.QuoteID, QuoteText: it.QuoteText,
		Reply: func(text string) error {
			if a.IM == nil {
				return errors.New("im handoff: gateway closed")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			return a.IM.Push(ctx, acct, chat, text)
		},
	}
}

// imStaleNote 是等太久的那条的回话：说明没处理，附上原话开头，请对方再发一次。
func imStaleNote(text string) string {
	snip := []rune(strings.TrimSpace(text))
	if len(snip) > 40 {
		snip = append(snip[:40], '…')
	}
	if imTextZh(text) {
		return "这条消息在交接时等得太久，没有处理，还需要的话请再发一次：「" + string(snip) + "」"
	}
	return "This message waited too long during a handover and was not handled. Please send it again if you still need it: \"" + string(snip) + "\""
}

// imTextZh 在文本里有汉字时为真，用来挑回话的语言。
func imTextZh(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
