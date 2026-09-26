package app

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/session"
)

// ErrQueued means the session is already running and the turn was enqueued.
const ErrQueued errString = "queued"

var queueSeq atomic.Uint64

type QueuedTurn struct {
	ID          string       `json:"id,omitempty"`
	Text        string       `json:"text"`
	Plan        bool         `json:"plan"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Path    string `json:"path,omitempty"`
	Name    string `json:"name,omitempty"`
	MIME    string `json:"mime,omitempty"`
	DataB64 string `json:"data_b64,omitempty"`
}

func (a *App) initQueue() {
	a.queueMu = sync.Mutex{}
	a.queue = map[string][]QueuedTurn{}
	a.steers = map[string][]string{}
}

func (a *App) Enqueue(sessionID string, turn QueuedTurn) {
	if turn.ID == "" {
		turn.ID = fmt.Sprintf("q-%d-%d", time.Now().UTC().UnixNano(), queueSeq.Add(1))
	}
	if a.Threads != nil {
		a.Threads.Enqueue(sessionID, toSessionQueued(turn))
		return
	}
	a.queueMu.Lock()
	a.queue[sessionID] = append(a.queue[sessionID], turn)
	a.queueMu.Unlock()
}

func (a *App) QueueList(sessionID string) []QueuedTurn {
	if a.Threads != nil {
		return fromSessionQueuedList(a.Threads.QueueList(sessionID))
	}
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	return append([]QueuedTurn(nil), a.queue[sessionID]...)
}

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

func toSessionQueued(turn QueuedTurn) session.QueuedTurn {
	atts := make([]session.Attachment, 0, len(turn.Attachments))
	for _, x := range turn.Attachments {
		atts = append(atts, session.Attachment{Path: x.Path, Name: x.Name, MIME: x.MIME, DataB64: x.DataB64})
	}
	return session.QueuedTurn{ID: turn.ID, Text: turn.Text, Plan: turn.Plan, Attachments: atts}
}

func fromSessionQueued(t session.QueuedTurn) QueuedTurn {
	atts := make([]Attachment, 0, len(t.Attachments))
	for _, x := range t.Attachments {
		atts = append(atts, Attachment{Path: x.Path, Name: x.Name, MIME: x.MIME, DataB64: x.DataB64})
	}
	return QueuedTurn{ID: t.ID, Text: t.Text, Plan: t.Plan, Attachments: atts}
}

func fromSessionQueuedList(in []session.QueuedTurn) []QueuedTurn {
	out := make([]QueuedTurn, 0, len(in))
	for _, t := range in {
		out = append(out, fromSessionQueued(t))
	}
	return out
}

func (a *App) kickQueue(sessionID string) {
	next, ok := a.popQueue(sessionID)
	if !ok {
		return
	}
	_ = a.StartSendOpts(sessionID, next.Text, next.Plan, next.Attachments)
}

func (a *App) Steer(sessionID, text string) error {
	if text == "" {
		return errString("empty steer")
	}
	if !a.Running(sessionID) {
		return errString("session is not running")
	}
	if a.Threads != nil {
		_ = a.Threads.PushSteer(sessionID, text)
	}
	a.queueMu.Lock()
	a.steers[sessionID] = append(a.steers[sessionID], text)
	a.queueMu.Unlock()
	return nil
}

func (a *App) pullSteer(sessionID string) string {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	items := a.steers[sessionID]
	if len(items) == 0 {
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
	return out
}
