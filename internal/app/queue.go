package app

import "sync"

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
	defer a.queueMu.Unlock()
	a.queue[sessionID] = append(a.queue[sessionID], turn)
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
