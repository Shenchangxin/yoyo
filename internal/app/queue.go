// 忙时把回合入队。正在跑的会话把 Steer 交给 dsh 代理；代理还没建好时仍先放进内存，由这一轮开场收走。

package app

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// ErrQueued 表示会话已经在跑，这一轮已经进入队列。
const ErrQueued errString = "queued"

// queueSeq 给入队项追加递增序号，避免同一纳秒里的 id 撞上。
var queueSeq atomic.Uint64

// QueuedTurn 是排队等待的一轮：正文、是否计划、是否续跑，以及附件。
type QueuedTurn struct {
	ID          string       `json:"id,omitempty"`
	Text        string       `json:"text"`
	Plan        bool         `json:"plan"`
	Resume      bool         `json:"resume,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment 是路径、名字、MIME 或 base64 正文。
type Attachment struct {
	Path    string `json:"path,omitempty"`
	Name    string `json:"name,omitempty"`
	MIME    string `json:"mime,omitempty"`
	DataB64 string `json:"data_b64,omitempty"`
}

// initQueue 准备按会话存放的回合队列和 steer 列表。
func (a *App) initQueue() {
	a.queueMu = sync.Mutex{}
	a.queue = map[string][]QueuedTurn{}
	a.steers = map[string][]string{}
}

// Enqueue 把一轮放进会话队列。有线程管理器时写入那里，否则放进内存。并记一条 next-turn 的入队。
// sessionID 传给 Threads.Enqueue 或用作 queue 的键，并传给 recordInbox。
func (a *App) Enqueue(sessionID string, turn QueuedTurn) {
	if turn.ID == "" {
		turn.ID = fmt.Sprintf("q-%d-%d", time.Now().UTC().UnixNano(), queueSeq.Add(1))
	}
	if a.Threads != nil {
		a.Threads.Enqueue(sessionID, toSessionQueued(turn))
	} else {
		a.queueMu.Lock()
		a.queue[sessionID] = append(a.queue[sessionID], turn)
		a.queueMu.Unlock()
	}
	a.recordInbox(sessionID, "enqueue", "next-turn", turn.ID, turn.Text)
}

// QueueList 返回该会话的排队回合副本。
// sessionID 传给 Threads.QueueList，或用作 queue 的键。
func (a *App) QueueList(sessionID string) []QueuedTurn {
	if a.Threads != nil {
		return fromSessionQueuedList(a.Threads.QueueList(sessionID))
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	return append([]QueuedTurn(nil), a.queue[sessionID]...)
}

// QueueCancel 按 id 拿掉一条排队回合。找到则返回真。
// sessionID 传给 Threads.CancelQueue，或用作 queue 的键。
// itemID 传给 Threads.CancelQueue，或用来匹配排队回合的 ID。
func (a *App) QueueCancel(sessionID, itemID string) bool {
	if a.Threads != nil {
		return a.Threads.CancelQueue(sessionID, itemID)
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	q := a.queue[sessionID]
	out := q[:0]
	ok := false
	for _, t := range q {
		if t.ID == itemID {
			ok = true
			continue
		}
		out = append(out, t)
	}
	a.queue[sessionID] = out
	return ok
}

// QueueReorder 把一条排队回合按 delta 与相邻项对调。越界或找不到则返回假。
// sessionID 传给 Threads.ReorderQueue，或用作 queue 的键。
// itemID 传给 Threads.ReorderQueue，或用来匹配排队回合的 ID。
func (a *App) QueueReorder(sessionID, itemID string, delta int) bool {
	if a.Threads != nil {
		return a.Threads.ReorderQueue(sessionID, itemID, delta)
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	q := a.queue[sessionID]
	i := -1
	for n, t := range q {
		if t.ID == itemID {
			i = n
			break
		}
	}
	if i < 0 {
		return false
	}
	j := i + delta
	if j < 0 || j >= len(q) {
		return false
	}
	q[i], q[j] = q[j], q[i]
	a.queue[sessionID] = q
	return true
}

// clearQueue 丢掉 sessionID 上还没开跑的排队回合，以及还没被代理领走的插话。
func (a *App) clearQueue(sessionID string) {
	if a == nil {
		return
	}
	if a.Threads != nil {
		a.Threads.ClearQueue(sessionID)
	}
	a.queueMu.Lock()
	delete(a.queue, sessionID)
	delete(a.steers, sessionID)
	a.queueMu.Unlock()
}

// takeQueue 按 itemID 取出并删掉一条排队回合。找不到时 ok 为假。
func (a *App) takeQueue(sessionID, itemID string) (QueuedTurn, bool) {
	if a.Threads != nil {
		turn, ok := a.Threads.TakeQueue(sessionID, itemID)
		if !ok {
			return QueuedTurn{}, false
		}
		return fromSessionQueued(turn), true
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	q := a.queue[sessionID]
	for i, turn := range q {
		if turn.ID != itemID {
			continue
		}
		a.queue[sessionID] = append(q[:i], q[i+1:]...)
		return turn, true
	}
	return QueuedTurn{}, false
}

// SteerQueued 把 itemID 从下一轮队列拿出来，插进 sessionID 正在跑的这一步。会话没在跑或找不到时返回错误。
func (a *App) SteerQueued(sessionID, itemID string) error {
	if !a.Running(sessionID) {
		return errString("session is not running")
	}
	turn, ok := a.takeQueue(sessionID, itemID)
	if !ok {
		return errString("queue item not found")
	}
	text := steerText(turn)
	if text == "" {
		return nil
	}
	return a.Steer(sessionID, text)
}

// SteerQueue 按队列顺序把 sessionID 上全部下一轮插进正在跑的这一步。会话没在跑时返回错误。
func (a *App) SteerQueue(sessionID string) error {
	if !a.Running(sessionID) {
		return errString("session is not running")
	}
	var first error
	for {
		turn, ok := a.popQueue(sessionID)
		if !ok {
			return first
		}
		text := steerText(turn)
		if text == "" {
			continue
		}
		if err := a.Steer(sessionID, text); err != nil && first == nil {
			first = err
		}
	}
}

// popQueue 取出队列第一条并把它从队列里删掉。空队列返回假。
// sessionID 传给 Threads.PopQueue，或用作 queue 的键。
func (a *App) popQueue(sessionID string) (QueuedTurn, bool) {
	if a.Threads != nil {
		t, ok := a.Threads.PopQueue(sessionID)
		if !ok {
			return QueuedTurn{}, false
		}
		return fromSessionQueued(t), true
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	q := a.queue[sessionID]
	if len(q) == 0 {
		return QueuedTurn{}, false
	}
	turn := q[0]
	a.queue[sessionID] = q[1:]
	return turn, true
}

// toSessionQueued 把本包的排队回合转成 session 包的结构。
// turn 用来读 ID、Text、Plan、Resume 和 Attachments。
func toSessionQueued(turn QueuedTurn) session.QueuedTurn {
	atts := make([]session.Attachment, 0, len(turn.Attachments))
	for _, x := range turn.Attachments {
		atts = append(atts, session.Attachment{Path: x.Path, Name: x.Name, MIME: x.MIME, DataB64: x.DataB64})
	}
	return session.QueuedTurn{ID: turn.ID, Text: turn.Text, Plan: turn.Plan, Resume: turn.Resume, Attachments: atts}
}

// fromSessionQueued 把 session 包的排队回合转回本包结构。
// t 用来读 ID、Text、Plan、Resume 和 Attachments。
func fromSessionQueued(t session.QueuedTurn) QueuedTurn {
	atts := make([]Attachment, 0, len(t.Attachments))
	for _, x := range t.Attachments {
		atts = append(atts, Attachment{Path: x.Path, Name: x.Name, MIME: x.MIME, DataB64: x.DataB64})
	}
	return QueuedTurn{ID: t.ID, Text: t.Text, Plan: t.Plan, Resume: t.Resume, Attachments: atts}
}

// fromSessionQueuedList 转换一整条排队列表。
// in 逐项传给 fromSessionQueued。
func fromSessionQueuedList(in []session.QueuedTurn) []QueuedTurn {
	out := make([]QueuedTurn, 0, len(in))
	for _, t := range in {
		out = append(out, fromSessionQueued(t))
	}
	return out
}

// kickQueue 取出下一条排队回合并发送。Resume 为真时按续跑启动，不把排队正文当成新的用户消息。
// sessionID 传给 popQueue 和 recordInbox，再传给 launchSend 或 StartSendOpts。
func (a *App) kickQueue(sessionID string) {
	next, ok := a.popQueue(sessionID)
	if !ok {
		return
	}
	a.recordInbox(sessionID, "claim", "next-turn", next.ID, next.Text)
	if next.Resume {
		_ = a.launchSend(sessionID, "", next.Plan, nil, true)
		return
	}
	_ = a.StartSendOpts(sessionID, next.Text, next.Plan, next.Attachments)
}

// Steer 在会话正在跑时把 text 交给 dsh 代理。代理还没建好时追加进内存 steers，由这一轮开场收走。空文本或会话没在跑则返回错误。
// sessionID 传给 Running。代理收下时写入收件箱；没收下时传给 PushSteer，并用作 steers 的键。
// text 为空则返回错误。代理收下时不再写入 steers，避免同一句被投递两次。
func (a *App) Steer(sessionID, text string) error {
	if text == "" {
		return errString("empty steer")
	}
	if !a.Running(sessionID) {
		return errString("session is not running")
	}
	if a.agentDesk().Steer(sessionID, text) {
		a.recordInbox(sessionID, "enqueue", "next-step", "", text)
		return nil
	}
	if a.Threads != nil {
		_ = a.Threads.PushSteer(sessionID, text)
	}
	a.queueMu.Lock()
	a.steers[sessionID] = append(a.steers[sessionID], text)
	a.queueMu.Unlock()
	a.recordInbox(sessionID, "enqueue", "next-step", "", text)
	return nil
}

// steerText 用排队正文。正文为空但有附件时，改成点出附件名字的一行。两者都空时返回空串。
func steerText(turn QueuedTurn) string {
	text := strings.TrimSpace(turn.Text)
	if text != "" {
		return text
	}
	var names []string
	for _, att := range turn.Attachments {
		name := strings.TrimSpace(att.Name)
		if name == "" {
			name = strings.TrimSpace(att.Path)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "Attachments: " + strings.Join(names, ", ")
}

// pullSteer 一次取走该会话内存里的全部 steer，拼成一段，并记一条 next-step 的领取。没有则返回空串。运行时在下一次模型调用前调用它。
// sessionID 用作 steers 的键，并传给 recordInbox。
func (a *App) pullSteer(sessionID string) string {
	a.queueMu.Lock()
	items := a.steers[sessionID]
	if len(items) == 0 {
		a.queueMu.Unlock()
		return ""
	}
	a.steers[sessionID] = nil
	out := "User steering (apply now):\n"
	for i, s := range items {
		if i > 0 {
			out += "\n"
		}
		out += s
	}
	a.queueMu.Unlock()
	a.recordInbox(sessionID, "claim", "next-step", "", out)
	return out
}

// recordInbox 往轨迹追加一条收件箱事件，记下入队或领取。没有轨迹或会话 id 则直接返回。
// sessionID 去掉首尾空白后为空则返回，否则写入事件的 SessionID。
// kind 写入 payload 的 kind。
// lane 写入 payload 的 lane。
// text 非空时写入 payload 的 text。
func (a *App) recordInbox(sessionID, kind, lane, id, text string) {
	if a == nil || a.Traces == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	payload := map[string]any{"kind": kind, "lane": lane}
	if id != "" {
		payload["id"] = id
	}
	if text != "" {
		payload["text"] = text
	}
	_ = a.Traces.Append(trace.Event{
		Type:      trace.TypeInbox,
		Source:    "runtime",
		SessionID: sessionID,
		Payload:   payload,
	})
}
