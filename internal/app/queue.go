package app

import (
	"sync"

	"github.com/Shenchangxin/yoyo/internal/session"
)

// ErrQueued means the session is already running and the turn was enqueued.
const ErrQueued errString = "queued"

type QueuedTurn struct {
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
	a.queueMu.Lock()
	a.queue[sessionID] = append(a.queue[sessionID], turn)
	a.queueMu.Unlock()
	if a.Threads != nil {
		atts := make([]session.Attachment, 0, len(turn.Attachments))
		for _, x := range turn.Attachments {
			atts = append(atts, session.Attachment{Path: x.Path, Name: x.Name, MIME: x.MIME, DataB64: x.DataB64})
		}
		a.Threads.Enqueue(sessionID, session.QueuedTurn{Text: turn.Text, Plan: turn.Plan, Attachments: atts})
	}
}

func (a *App) QueueList(sessionID string) []QueuedTurn {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()
	out := append([]QueuedTurn(nil), a.queue[sessionID]...)
	return out
}

func (a *App) popQueue(sessionID string) (QueuedTurn, bool) {
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
