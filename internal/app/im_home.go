// im_home.go 把一个 IM 对话升成主通道：人在桌面拿一次性配对码，在飞书、企微、钉钉、Telegram、QQ 里发 /绑定 <码>，
// 之后过了开口门的短讯、晨晚简报、要你批准的事和等你回答的工单都会发到那里，也能在那里直接批、答、反馈。
// 绑定的那个人在那个对话里说的话进主会话，和桌面上是同一段对话、同一份记忆。别的对话照旧走 im 画像和不可信处理。
// 安静时段里的消息先攒着，结束后合成一条发；周末只放审批和出错，和系统通知同一套规则（DeskConfig.HoldToast）。
// 不拥有：平台收发（internal/im）、开口门（initiative）、审批本身（capability.Gate）。

package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/im"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// imPairTTL 是配对码的有效期。
const imPairTTL = 10 * time.Minute

// imCodeTTL 是短编号留多久。过了就要回桌面处理。
const imCodeTTL = 72 * time.Hour

// imSentKeep 是已经发过的工单报告记多久，重启之后不重发。
const imSentKeep = 7 * 24 * time.Hour

// imHeldMax 是安静时段里最多攒几条，多了丢最早的。
const imHeldMax = 30

// IMHome 是绑定的主通道。Sender 是绑定时发码的人；群里只认他。Sent 记着发过的工单报告，值是 RFC3339 时间。
type IMHome struct {
	Account    string            `json:"account"`
	Platform   string            `json:"platform,omitempty"`
	Label      string            `json:"label,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Chat       string            `json:"chat"`
	Sender     string            `json:"sender,omitempty"`
	SenderName string            `json:"senderName,omitempty"`
	At         string            `json:"at,omitempty"`
	Sent       map[string]string `json:"sent,omitempty"`
}

func (h IMHome) bound() bool {
	return strings.TrimSpace(h.Account) != "" && strings.TrimSpace(h.Chat) != ""
}

// IMHomeState 是设置页读到的主通道。Code 只在配对码还有效时有。Held 是安静时段里攒着的条数。
type IMHomeState struct {
	Bound      bool   `json:"bound"`
	Account    string `json:"account,omitempty"`
	Platform   string `json:"platform,omitempty"`
	Label      string `json:"label,omitempty"`
	Kind       string `json:"kind,omitempty"`
	SenderName string `json:"senderName,omitempty"`
	At         string `json:"at,omitempty"`
	Code       string `json:"code,omitempty"`
	CodeUntil  string `json:"codeUntil,omitempty"`
	Held       int    `json:"held"`
	LastErr    string `json:"lastErr,omitempty"`
	LastAt     string `json:"lastAt,omitempty"`
}

// imCode 是发到主通道的一条短编号指向的东西。Kind 取 approval、ticket、speak。
type imCode struct {
	Kind     string
	Target   string
	Class    string
	MemoryID string
	At       time.Time
}

// imHeld 是安静时段里攒着的一条。Target 是审批编号，发的时候已经批过的就不发了。
type imHeld struct {
	Kind   string
	Text   string
	Target string
	At     time.Time
}

// imHomeBox 是主通道在内存里的状态。push 和 sync 只给测试用。
type imHomeBox struct {
	mu        sync.Mutex
	loaded    bool
	home      IMHome
	pair      string
	pairUntil time.Time
	codes     map[string]imCode
	lastSpeak string
	held      []imHeld
	lastErr   string
	lastAt    time.Time
	turnMu    sync.Mutex
	push      func(account, chat, text string) error
	// pushID 只给测试用：和 push 一样，另外返回平台给这条消息的编号。
	pushID func(account, chat, text string) (string, error)
	// quotes 是发到主通道的消息在平台上的编号 → 它带的短编号，人引用回复时用来找回是哪件事。见 im_quote.go。
	quotes map[string]imQuoted
	sync      bool
	stamp     diskStamp
}

func (a *App) imHomePath() string {
	if a == nil || a.Home == nil || strings.TrimSpace(a.Home.Root) == "" {
		return ""
	}
	return filepath.Join(a.Home.Root, "im", "home.json")
}

// imLoadLocked 第一次用时从磁盘读出绑定；之后文件被另一个 yoyo 进程改过（在那边绑定或解除）就重读。调用方持有 imh.mu。
func (a *App) imLoadLocked() {
	path := a.imHomePath()
	st := diskStamp{}
	if path != "" {
		st = stampOf(path)
	}
	if a.imh.loaded && st == a.imh.stamp {
		return
	}
	a.imh.loaded = true
	a.imh.stamp = st
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var h IMHome
	if json.Unmarshal(raw, &h) == nil {
		a.imh.home = h
	}
}

// imSaveLocked 把绑定写回磁盘。调用方持有 imh.mu。
func (a *App) imSaveLocked() error {
	path := a.imHomePath()
	if path == "" {
		return fmt.Errorf("home unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(a.imh.home, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	a.imh.stamp = stampOf(path)
	return nil
}

func (a *App) imHomeNow() IMHome {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	a.imLoadLocked()
	return a.imh.home
}

func (a *App) imBound() bool {
	return a != nil && a.imHomeNow().bound()
}

// IMHomeState 返回主通道此刻的样子。
func (a *App) IMHomeState() IMHomeState {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	a.imLoadLocked()
	h := a.imh.home
	out := IMHomeState{
		Bound: h.bound(), Account: h.Account, Platform: h.Platform, Label: h.Label, Kind: h.Kind,
		SenderName: h.SenderName, At: h.At, Held: len(a.imh.held), LastErr: a.imh.lastErr,
	}
	if !a.imh.lastAt.IsZero() {
		out.LastAt = a.imh.lastAt.UTC().Format(time.RFC3339)
	}
	a.loadPairLocked()
	if a.imh.pair != "" && time.Now().Before(a.imh.pairUntil) {
		out.Code = a.imh.pair
		out.CodeUntil = a.imh.pairUntil.UTC().Format(time.RFC3339)
	}
	return out
}

// IMPair 生成一个十分钟有效的六位配对码。再生成一次时旧码作废。
func (a *App) IMPair() IMHomeState {
	code := pairCode()
	a.imh.mu.Lock()
	a.imh.pair = code
	a.imh.pairUntil = time.Now().Add(imPairTTL)
	a.savePairLocked()
	a.imh.mu.Unlock()
	return a.IMHomeState()
}

// IMUnbind 解除主通道。攒着的消息和短编号一起丢掉。
func (a *App) IMUnbind() error {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	a.imLoadLocked()
	a.imh.home = IMHome{}
	a.imh.held = nil
	// 短编号和引用编号落在 im/codes.json（im_codes.go）：清空并写回，另一个进程重读时也不再认旧号。
	a.imCodesResetLocked()
	return a.imSaveLocked()
}

// IMTest 往主通道发一条测试消息，同步等结果。没有绑定时返回错误。
func (a *App) IMTest() error {
	h := a.imHomeNow()
	if !h.bound() {
		return fmt.Errorf("no im home")
	}
	err := a.imPush(h, pick(a.briefChinese(), "这是一条测试消息。以后简报和要你处理的事会发到这里。", "Test message. Briefs and things that need you will arrive here."))
	a.imRecord(err)
	return err
}

func pairCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%900000+100000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

// imBind 用配对码把发来的这个对话记成主通道。码对上之后立刻作废。
func (a *App) imBind(in im.Inbound, code string) string {
	zh := a.briefChinese()
	code = strings.TrimSpace(code)
	label := a.imAccountLabel(in.AccountID, in.Platform)
	a.imh.mu.Lock()
	a.loadPairLocked()
	ok := code != "" && a.imh.pair != "" && code == a.imh.pair && time.Now().Before(a.imh.pairUntil)
	if !ok {
		a.imh.mu.Unlock()
		return pick(zh, "配对码不对或已经过期。在桌面「设置 · 主动与通知」里重新生成一个。", "That pairing code is wrong or expired. Make a new one in Settings · Proactive on the desktop.")
	}
	a.imLoadLocked()
	a.imh.pair = ""
	a.dropPairLocked()
	a.imh.home = IMHome{
		Account: in.AccountID, Platform: in.Platform, Label: label, Kind: in.Kind, Chat: in.ChatID,
		Sender: in.SenderID, SenderName: in.SenderName, At: time.Now().UTC().Format(time.RFC3339),
	}
	// 绑到新的对话时旧编号作废：清空 im/codes.json（im_codes.go），另一个进程重读时也不再认。
	a.imCodesResetLocked()
	err := a.imSaveLocked()
	a.imh.mu.Unlock()
	if err != nil {
		return pick(zh, "没能记下这个对话：", "Could not save this chat: ") + err.Error()
	}
	return pick(zh,
		"好了，这里就是主通道。以后简报、要你批准的事和等你回答的任务都会发到这里，你在这里说的话进桌面上同一段对话。回 /帮助 看能做什么。",
		"Done, this chat is now home. Briefs, approvals and tasks waiting on you will arrive here, and what you say here goes into the same conversation as the desktop. Reply /help for commands.")
}

func (a *App) imAccountLabel(accountID, platform string) string {
	if a.IM != nil {
		for _, acct := range a.IM.Snapshot().Accounts {
			if acct.ID == accountID && strings.TrimSpace(acct.Label) != "" {
				return acct.Label
			}
		}
	}
	return platform
}

// isIMHome 在这条消息来自主通道、而且是绑定的那个人时为真。
func (a *App) isIMHome(in im.Inbound) bool {
	h := a.imHomeNow()
	if !h.bound() || in.AccountID != h.Account || in.ChatID != h.Chat {
		return false
	}
	return h.Sender == "" || in.SenderID == "" || in.SenderID == h.Sender
}

const imCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func randCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		x, err := rand.Int(rand.Reader, big.NewInt(int64(len(imCodeAlphabet))))
		if err != nil {
			x = big.NewInt(int64(time.Now().UnixNano() % int64(len(imCodeAlphabet))))
		}
		b[i] = imCodeAlphabet[x.Int64()]
	}
	return string(b)
}

// imIssue 给 c 一个短编号。同一个东西再要时给同一个号。
// 编号落在 im/codes.json（见 im_codes.go）：先重读磁盘再发号，发完写回，另一个进程和重启之后都认得出。
func (a *App) imIssue(c imCode) string {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	var out string
	a.imCodesEdit(func() bool {
		if a.imh.codes == nil {
			a.imh.codes = map[string]imCode{}
		}
		now := time.Now()
		for k, v := range a.imh.codes {
			if now.Sub(v.At) > imCodeTTL {
				delete(a.imh.codes, k)
			}
		}
		for k, v := range a.imh.codes {
			if v.Kind == c.Kind && v.Target == c.Target {
				v.At = now
				a.imh.codes[k] = v
				if c.Kind == "speak" {
					a.imh.lastSpeak = k
				}
				out = k
				return true
			}
		}
		code := randCode(3)
		for i := 0; i < 30; i++ {
			if _, taken := a.imh.codes[code]; !taken {
				break
			}
			n := 3
			if i >= 10 {
				n = 4
			}
			code = randCode(n)
		}
		c.At = now
		a.imh.codes[code] = c
		if c.Kind == "speak" {
			a.imh.lastSpeak = code
		}
		out = code
		return true
	})
	return out
}

// imLookup 找回短编号。code 为空且 kind 是 speak 时取最近一条短讯的号。
// 查之前重读 im/codes.json，所以别的进程（或重启之前）发出去的号也找得到。只读，不写盘。
func (a *App) imLookup(code, kind string) (string, imCode, bool) {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	code = strings.ToLower(strings.TrimSpace(code))
	var c imCode
	ok := false
	a.imCodesEdit(func() bool {
		if code == "" && kind == "speak" {
			code = a.imh.lastSpeak
		}
		c, ok = a.imh.codes[code]
		return false
	})
	if !ok || c.Kind != kind || time.Since(c.At) > imCodeTTL {
		return code, imCode{}, false
	}
	return code, c, true
}

// imForget 作废一个短编号（审批批过之后）。删掉后写回磁盘，另一个进程也不会再认它。
func (a *App) imForget(code string) {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	a.imCodesEdit(func() bool {
		delete(a.imh.codes, code)
		return true
	})
}

// imOnce 在 key 还没往主通道发过时为真，并记下来。没绑定时为假。
func (a *App) imOnce(key string) bool {
	a.imh.mu.Lock()
	defer a.imh.mu.Unlock()
	a.imLoadLocked()
	if !a.imh.home.bound() || strings.TrimSpace(key) == "" {
		return false
	}
	if a.imh.home.Sent == nil {
		a.imh.home.Sent = map[string]string{}
	}
	now := time.Now()
	for k, v := range a.imh.home.Sent {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil || now.Sub(t) > imSentKeep {
			delete(a.imh.home.Sent, k)
		}
	}
	if _, ok := a.imh.home.Sent[key]; ok {
		return false
	}
	a.imh.home.Sent[key] = now.UTC().Format(time.RFC3339)
	_ = a.imSaveLocked()
	return true
}

// imSend 往主通道发一条。安静时段里先攒着。没绑定时返回假。
func (a *App) imSend(kind, text, target string) bool {
	return a.imSendAt(kind, text, target, time.Now())
}

func (a *App) imSendAt(kind, text, target string, now time.Time) bool {
	text = strings.TrimSpace(text)
	if a == nil || text == "" {
		return false
	}
	a.imh.mu.Lock()
	a.imLoadLocked()
	h := a.imh.home
	if !h.bound() {
		a.imh.mu.Unlock()
		return false
	}
	if a.Config.Desk.HoldToast(kind, now.In(time.Local)) {
		a.imh.held = append(a.imh.held, imHeld{Kind: kind, Text: text, Target: target, At: now})
		if len(a.imh.held) > imHeldMax {
			a.imh.held = a.imh.held[len(a.imh.held)-imHeldMax:]
		}
		a.imh.mu.Unlock()
		return true
	}
	a.imh.mu.Unlock()
	a.imDeliver(h, text)
	return true
}

// imFlush 在安静时段过去之后把攒着的合成一条发。已经批过的审批不再发。
func (a *App) imFlush(now time.Time) {
	if a == nil {
		return
	}
	a.imh.mu.Lock()
	if len(a.imh.held) == 0 {
		a.imh.mu.Unlock()
		return
	}
	h := a.imh.home
	if !h.bound() {
		a.imh.held = nil
		a.imh.mu.Unlock()
		return
	}
	local := now.In(time.Local)
	var out, keep []imHeld
	for _, m := range a.imh.held {
		if a.Config.Desk.HoldToast(m.Kind, local) {
			keep = append(keep, m)
		} else {
			out = append(out, m)
		}
	}
	a.imh.held = keep
	a.imh.mu.Unlock()
	if len(out) == 0 {
		return
	}
	pending := map[string]bool{}
	if a.Gate != nil {
		for _, o := range a.Gate.Pending() {
			pending[o.ID] = true
		}
	}
	var parts []string
	for _, m := range out {
		if m.Kind == "approval" && m.Target != "" && !pending[m.Target] {
			continue
		}
		parts = append(parts, m.Text)
	}
	switch len(parts) {
	case 0:
		return
	case 1:
		a.imDeliver(h, parts[0])
	default:
		head := pick(a.briefChinese(), fmt.Sprintf("安静时段里攒下了 %d 条：", len(parts)), fmt.Sprintf("%d held during quiet hours:", len(parts)))
		a.imDeliver(h, head+"\n\n"+strings.Join(parts, "\n\n———\n\n"))
	}
}

func (a *App) imDeliver(h IMHome, text string) {
	run := func() {
		a.imRecord(a.imPush(h, text))
	}
	if a.imh.sync {
		run()
		return
	}
	go run()
}

func (a *App) imRecord(err error) {
	a.imh.mu.Lock()
	if err != nil {
		a.imh.lastErr = scrubSense(err.Error())
	} else {
		a.imh.lastErr = ""
		a.imh.lastAt = time.Now()
	}
	a.imh.mu.Unlock()
	if err != nil {
		diaglog.Error("im home push failed", "component", "im", "err", err)
	}
}

// imPush 往主通道发一条。发成功且平台回了编号时，记下这条带的短编号，人之后直接「回复」这条就能处理（见 im_quote.go）。
func (a *App) imPush(h IMHome, text string) error {
	id, err := a.imPushID(h, text)
	if err == nil {
		a.imRemember(h.Account, id, text)
	}
	return err
}

// imSpeak 把一条送到主会话的开口短讯也发到主通道，带上理由和反馈编号。公开渠道的资讯只留在桌面。
func (a *App) imSpeak(s initiative.Signal, note DeskNote) {
	if s.Code == "reach" || s.Code == "welcome" || !a.imBound() {
		return
	}
	zh := a.briefChinese()
	code := a.imIssue(imCode{Kind: "speak", Target: s.ID, Class: s.Class, MemoryID: s.MemoryID})
	var b strings.Builder
	b.WriteString(strings.TrimSpace(note.Text))
	if r := strings.TrimSpace(note.Reason); r != "" {
		b.WriteString("\n" + pick(zh, "为什么：", "Why: ") + r)
	}
	b.WriteString("\n" + pick(zh, "回 /有用 "+code+" · /别提醒 "+code, "Reply /used "+code+" · /later "+code))
	kind := "speak"
	if s.Code == "channel" || s.State == "failed" {
		kind = "error"
	}
	a.imSend(kind, b.String(), "")
}

// imBrief 把一封写成的简报收成几行发到主通道。安静的信不发。
func (a *App) imBrief(chip string, cards []BriefCard) {
	if !a.imBound() {
		return
	}
	if text := briefIMText(chip, cards, a.briefChinese()); text != "" {
		a.imSend("brief", text, "")
	}
}

// briefIMText 是简报在 IM 里的样子：标题加每张卡一行判断、一行为什么，最多五张。
func briefIMText(chip string, cards []BriefCard, zh bool) string {
	var lines []string
	for _, c := range cards {
		if c.ID == "quiet" {
			continue
		}
		title := strings.TrimSpace(c.Title)
		if title == "" {
			continue
		}
		line := "· " + title
		if lede := strings.TrimSpace(c.Lede); lede != "" {
			line += "\n  " + lede
		}
		lines = append(lines, line)
		if len(lines) == 5 {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(chip) + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		pick(zh, "全文在桌面的「今天」里。回 /状态 看还有什么在等你。", "The full brief is in Today on the desktop. Reply /status for what is waiting.")
}

// imApproval 把一条待批的动作发到主通道。像密钥的命令不发原文。
func (a *App) imApproval(o capability.Offer) {
	if !a.imBound() {
		return
	}
	zh := a.briefChinese()
	code := a.imIssue(imCode{Kind: "approval", Target: o.ID})
	a.imSend("approval", approvalIMText(o, code, zh), o.ID)
}

func approvalIMText(o capability.Offer, code string, zh bool) string {
	what := strings.TrimSpace(o.Request.Action)
	if what == "" {
		what = string(o.Request.Level)
	}
	detail := strings.TrimSpace(firstNonEmpty(o.Request.Command, o.Request.Path))
	if detail != "" {
		if scrubSense(detail) == "" {
			detail = pick(zh, "（内容里像有密钥，不发到这里，请在桌面上看）", "(looks like it contains a secret; check the desktop)")
		} else {
			detail = clipBrief(detail, 240)
		}
	}
	text := pick(zh, "需要你批准：", "Needs your approval: ") + what
	if detail != "" {
		text += "\n" + detail
	}
	return text + "\n" + pick(zh,
		"回 /批准 "+code+" · /拒绝 "+code+"（/批准 "+code+" 总是：同类以后不再问）",
		"Reply /approve "+code+" · /deny "+code+" (/approve "+code+" always: don't ask again)")
}

// imTicket 把要人回答、或者已经结束的任务报告发到主通道。同一段只发一次，重启也不重发。
func (a *App) imTicket(ticketID, state, text, question, key string) {
	question = strings.TrimSpace(question)
	ended := taskEndedState(state)
	if (question == "" && !ended) || !a.imBound() || !a.imOnce(key) {
		return
	}
	zh := a.briefChinese()
	title := a.ticketTitle(ticketID)
	body := clipBrief(strings.TrimSpace(text), 700)
	if question != "" {
		code := a.imIssue(imCode{Kind: "ticket", Target: ticketID})
		msg := pick(zh, "任务「"+title+"」在等你：", "Task \""+title+"\" needs you:") + "\n" + body + "\n" +
			pick(zh, "回 /答 "+code+" 你的回答", "Reply /answer "+code+" <your answer>")
		a.imSend("approval", msg, "")
		return
	}
	kind := "speak"
	word := pick(zh, "结束了", "ended")
	switch state {
	case "done", "ready", "delivered":
		word = pick(zh, "做完了", "is done")
	case "failed":
		word = pick(zh, "没做成", "failed")
		kind = "error"
	case "stopped", "interrupted", "dropped":
		word = pick(zh, "停下了", "stopped")
	}
	msg := pick(zh, "任务「"+title+"」"+word+"：", "Task \""+title+"\" "+word+":") + "\n" + body
	a.imSend(kind, msg, "")
}

// liveOrch 是已经打开的编排运行时。还没打开时为空，不为了读一眼去打开数据库。
func (a *App) liveOrch() *superagent.Runtime {
	a.deskMu.Lock()
	defer a.deskMu.Unlock()
	return a.orch
}

func (a *App) ticketTitle(id string) string {
	if rt := a.liveOrch(); rt != nil {
		for _, t := range rt.View().Tickets {
			if t.ID == id {
				if s := scrubSense(t.Title); s != "" {
					return clipBrief(s, 40)
				}
			}
		}
	}
	return id
}

// imCommands 是主通道里认得的命令，中文和英文各一套。
var imCommands = map[string]string{
	"绑定": "bind", "bind": "bind",
	"解绑": "unbind", "unbind": "unbind",
	"批准": "approve", "同意": "approve", "approve": "approve",
	"拒绝": "deny", "deny": "deny",
	"答": "answer", "回答": "answer", "answer": "answer",
	"有用": "used", "用上了": "used", "used": "used",
	"没用": "ignore", "ignore": "ignore",
	"别提醒": "later", "稍后": "later", "later": "later",
	"关掉": "mute", "mute": "mute",
	"简报": "brief", "brief": "brief",
	"状态": "status", "status": "status",
	"帮助": "help", "help": "help",
}

// parseIMCommand 认 /命令 参数…。前面的 @某人 跳过。tail 是第一个参数之后的原文，/答 用它保留回答里的空格和换行。
func parseIMCommand(raw string) (cmd string, args []string, tail string, ok bool) {
	t := strings.TrimSpace(raw)
	for strings.HasPrefix(t, "@") {
		i := strings.IndexAny(t, " \t\n\u3000")
		if i < 0 {
			return "", nil, "", false
		}
		t = strings.TrimSpace(t[i:])
	}
	if !strings.HasPrefix(t, "/") && !strings.HasPrefix(t, "／") {
		return "", nil, "", false
	}
	t = strings.TrimSpace(strings.TrimLeft(t, "/／"))
	fields := strings.Fields(t)
	if len(fields) == 0 {
		return "", nil, "", false
	}
	cmd, ok = imCommands[strings.ToLower(fields[0])]
	if !ok {
		return "", nil, "", false
	}
	args = fields[1:]
	if len(args) > 0 {
		after := strings.TrimSpace(t[len(fields[0]):])
		tail = strings.TrimSpace(strings.TrimPrefix(after, args[0]))
	}
	return cmd, args, tail, true
}

// imCommand 处理主通道命令。处理了返回真，这条消息就不再进对话。raw 是没去掉唤醒前缀的原文。
// 绑定谁都能发，但要配对码；其余命令只认主通道里绑定的那个人。
func (a *App) imCommand(raw, in im.Inbound) bool {
	cmd, args, tail, ok := parseIMCommand(raw.Text)
	if !ok && strings.TrimSpace(raw.Text) != strings.TrimSpace(in.Text) {
		cmd, args, tail, ok = parseIMCommand("/" + strings.TrimSpace(in.Text))
	}
	if !ok {
		return false
	}
	zh := a.briefChinese()
	reply := func(s string) {
		if in.Reply != nil && strings.TrimSpace(s) != "" {
			_ = in.Reply(s)
		}
	}
	first := ""
	if len(args) > 0 {
		first = args[0]
	}
	if cmd == "bind" {
		reply(a.imBind(in, first))
		return true
	}
	if !a.isIMHome(in) {
		if !a.imBound() {
			reply(pick(zh, "还没有主通道。在桌面「设置 · 主动与通知」里生成配对码，然后在这里发 /绑定 <配对码>。", "No home chat yet. Make a pairing code in Settings · Proactive on the desktop, then send /bind <code> here."))
		} else {
			reply(pick(zh, "这个命令只在主通道里、由绑定的人用。", "This command only works in the home chat, from the person who bound it."))
		}
		return true
	}
	notFound := pick(zh, "找不到这个编号，可能已经处理过或过期了。回 /状态 看还在等的。", "No such code; it may be handled or expired. Reply /status for what is waiting.")
	switch cmd {
	case "unbind":
		if err := a.IMUnbind(); err != nil {
			reply(err.Error())
		} else {
			reply(pick(zh, "已解除。之后不会再往这里发。", "Unbound. Nothing more will be sent here."))
		}
	case "help":
		reply(imHelp(zh))
	case "approve", "deny":
		code, c, found := a.imLookup(first, "approval")
		if !found || a.Gate == nil {
			reply(notFound)
			break
		}
		decision := "deny"
		if cmd == "approve" {
			decision = "once"
			if len(args) > 1 {
				switch strings.ToLower(args[1]) {
				case "总是", "always":
					decision = "always"
				case "本次", "会话", "session":
					decision = "session"
				}
			}
		}
		if err := a.ResolveApproval(c.Target, decision); err != nil {
			a.imForget(code)
			reply(notFound)
			break
		}
		a.imForget(code)
		if decision == "deny" {
			reply(pick(zh, "已拒绝。", "Denied."))
		} else {
			reply(pick(zh, "已批准。", "Approved."))
		}
	case "answer":
		_, c, found := a.imLookup(first, "ticket")
		if !found {
			reply(notFound)
			break
		}
		if strings.TrimSpace(tail) == "" {
			reply(pick(zh, "在编号后面写上回答，例如 /答 "+first+" 先改登录页。", "Put the answer after the code, e.g. /answer "+first+" fix the login page first."))
			break
		}
		if err := a.answerTicket(c.Target, tail); err != nil {
			reply(pick(zh, "没送到：", "Not delivered: ") + err.Error())
			break
		}
		reply(pick(zh, "已经交给它了。", "Passed on."))
	case "used", "ignore", "later", "mute":
		_, c, found := a.imLookup(first, "speak")
		if !found {
			reply(notFound)
			break
		}
		if !a.DeskFeedback(c.Class, cmd, c.MemoryID) {
			reply(notFound)
			break
		}
		done := map[string]string{
			"used":   pick(zh, "记下了，这类以后照常说。", "Noted. This kind stays on."),
			"ignore": pick(zh, "记下了。同一类没用三次之后就不再说。", "Noted. After three of these, this kind goes quiet."),
			"later":  pick(zh, "今天这一类不再提。", "Nothing more of this kind today."),
			"mute":   pick(zh, "这一类先不说了，可以在桌面设置里打开。", "This kind is off; turn it back on in desktop settings."),
		}
		reply(done[cmd])
	case "brief":
		reply(a.imTodayBrief(zh))
	case "status":
		reply(a.imStatus(zh))
	}
	return true
}

// answerTicket 把人的回答交给在等的那张任务。
func (a *App) answerTicket(ticketID, text string) error {
	rt := a.orchestrator()
	if rt == nil {
		return fmt.Errorf("orchestrator unavailable")
	}
	return rt.Act("say", ticketID, "", text, "")
}

func imHelp(zh bool) string {
	if zh {
		return "在这里可以：\n/批准 <编号>、/拒绝 <编号>：处理要你批准的事\n/答 <编号> <回答>：回答在等你的任务\n/有用、/没用、/别提醒、/关掉 [编号]：对刚才那条提醒的反应\n/简报：今天的简报\n/状态：还有什么在等你\n/解绑：不再往这里发\n其余的话直接说，会进桌面上同一段对话。"
	}
	return "Here you can:\n/approve <code>, /deny <code>: handle approvals\n/answer <code> <text>: answer a waiting task\n/used, /ignore, /later, /mute [code]: react to the last note\n/brief: today's brief\n/status: what is waiting\n/unbind: stop sending here\nAnything else goes into the same conversation as the desktop."
}

// imTodayBrief 是今天最近一封简报在 IM 里的样子。
func (a *App) imTodayBrief(zh bool) string {
	day := time.Now().Format("2006-01-02")
	var pickRec *BriefRecord
	recs := a.BriefList()
	for i := range recs {
		if recs[i].Day != day {
			continue
		}
		if pickRec == nil || recs[i].Slot == "evening" {
			pickRec = &recs[i]
		}
	}
	if pickRec != nil {
		var body struct {
			Cards []BriefCard `json:"cards"`
		}
		if json.Unmarshal([]byte(pickRec.Brief), &body) == nil {
			if text := briefIMText(pickRec.Chip, body.Cards, zh); text != "" {
				return text
			}
			return pick(zh, "今天的简报写了：没有值得写的事。", "Today's brief: nothing worth writing.")
		}
	}
	return pick(zh, "今天还没有简报。", "No brief yet today.")
}

// imStatus 列出在等人批准的事、在等人回答的任务，以及今天开口了几次、用上了几次。
func (a *App) imStatus(zh bool) string {
	var lines []string
	if a.Gate != nil {
		for _, o := range a.Gate.Pending() {
			code := a.imIssue(imCode{Kind: "approval", Target: o.ID})
			lines = append(lines, "· "+pick(zh, "待批准：", "Approval: ")+clipBrief(firstNonEmpty(o.Request.Action, string(o.Request.Level)), 40)+"  /批准 "+code)
		}
	}
	if rt := a.liveOrch(); rt != nil {
		for _, t := range rt.View().Tickets {
			if !t.Person || t.Stage == "ended" {
				continue
			}
			code := a.imIssue(imCode{Kind: "ticket", Target: t.ID})
			lines = append(lines, "· "+pick(zh, "在等你：", "Waiting: ")+a.ticketTitle(t.ID)+"  /答 "+code)
		}
	}
	head := pick(zh, "没有在等你的事。", "Nothing is waiting on you.")
	if len(lines) > 0 {
		head = pick(zh, fmt.Sprintf("有 %d 件在等你：", len(lines)), fmt.Sprintf("%d things are waiting:", len(lines)))
	}
	out := head
	if len(lines) > 0 {
		out += "\n" + strings.Join(lines, "\n")
	}
	if store, err := a.deskBook(); err == nil {
		day := time.Now().Format("2006-01-02")
		for _, d := range store.Recent(1) {
			if d.Day == day && d.Spoken > 0 {
				out += "\n\n" + pick(zh, fmt.Sprintf("今天主动说了 %d 次，用上了 %d 次。", d.Spoken, d.Used), fmt.Sprintf("Today: %d notes, %d used.", d.Spoken, d.Used))
			}
		}
	}
	a.imh.mu.Lock()
	held := len(a.imh.held)
	a.imh.mu.Unlock()
	if held > 0 {
		out += "\n" + pick(zh, fmt.Sprintf("安静时段里还攒着 %d 条。", held), fmt.Sprintf("%d held for quiet hours.", held))
	}
	return out
}

// imHomeTurn 把主人在主通道里说的话送进主会话，回答发回主通道。一次只跑一段。
func (a *App) imHomeTurn(in im.Inbound) {
	a.imh.turnMu.Lock()
	defer a.imh.turnMu.Unlock()
	reply := func(text string) {
		if in.Reply != nil && strings.TrimSpace(text) != "" {
			_ = in.Reply(text)
		}
	}
	meta, err := a.MainSession()
	if err != nil {
		reply(imFail(in.Text, err))
		return
	}
	zh := a.briefChinese()
	label := firstNonEmpty(a.imHomeNow().Label, in.Platform)
	prefix := pick(zh, "[从 "+label+" 发来，回复会发回那里，写成纯文本。]\n", "[Sent from "+label+"; the reply goes back there, in plain text.]\n")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	out, err := a.SendOpts(ctx, meta.ID, prefix+in.Text, nil, nil, false)
	if err == errBusy {
		reply(pick(zh, "桌面上这段对话正在跑，等它说完再发一次。要处理的批准和任务可以先用 /状态 看。", "The desktop conversation is busy. Send again when it finishes; /status shows what is waiting."))
		return
	}
	if err != nil {
		reply(imFail(in.Text, err))
		return
	}
	for _, part := range a.imParts(in, out) {
		reply(part)
	}
}
