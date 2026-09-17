package session

import (
	"context"
	"fmt"
	"sync"
)

// ErrBusy means a turn is already running on this session.
const ErrBusy errString = "session already running"

type errString string

func (e errString) Error() string { return string(e) }

// Actor is the live mailbox for one session. Turns are serial; steer is allowed.
type Actor struct {
	ID     string
	cancel context.CancelFunc
}

// Manager is the thread manager: one actor per live session, durable run state.
type Manager struct {
	Dir   string
	Index *Index

	mu     sync.Mutex
	actors map[string]*Actor
	runs   map[string]RunState
}

func NewManager(dir string) *Manager {
	return &Manager{
		Dir:    dir,
		Index:  OpenIndex(dir),
		actors: map[string]*Actor{},
		runs:   map[string]RunState{},
	}
}

func (m *Manager) state(id string) RunState {
	if st, ok := m.runs[id]; ok {
		return st
	}
	st, err := LoadRun(m.Dir, id)
	if err != nil {
		st = RunState{Status: StatusIdle}
	}
	m.runs[id] = st
	return st
}

func (m *Manager) persist(id string) {
	st := m.runs[id]
	_ = SaveRun(m.Dir, id, st)
}

func (m *Manager) Load(id string) RunState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state(id)
}

func (m *Manager) SetStatus(id, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.Status = status
	m.runs[id] = st
	m.persist(id)
}

func (m *Manager) Enqueue(id string, turn QueuedTurn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.Queue = append(st.Queue, turn)
	m.runs[id] = st
	m.persist(id)
}

func (m *Manager) QueueList(id string) []QueuedTurn {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	return append([]QueuedTurn(nil), st.Queue...)
}

func (m *Manager) PopQueue(id string) (QueuedTurn, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	if len(st.Queue) == 0 {
		return QueuedTurn{}, false
	}
	turn := st.Queue[0]
	st.Queue = st.Queue[1:]
	m.runs[id] = st
	m.persist(id)
	return turn, true
}

func (m *Manager) PushSteer(id, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.actors[id] == nil {
		return fmt.Errorf("session is not running")
	}
	st := m.state(id)
	st.Steers = append(st.Steers, text)
	m.runs[id] = st
	m.persist(id)
	return nil
}

func (m *Manager) PullSteer(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	if len(st.Steers) == 0 {
		return ""
	}
	items := st.Steers
	st.Steers = nil
	m.runs[id] = st
	m.persist(id)
	out := "User steering (apply now):\n"
	for i, s := range items {
		if i > 0 {
			out += "\n"
		}
		out += s
	}
	return out
}

func (m *Manager) SetCaps(id string, caps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.SessionCaps = dropHighRisk(caps)
	m.runs[id] = st
	m.persist(id)
}

func (m *Manager) Caps(id string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.state(id).SessionCaps...)
}

func (m *Manager) SetApprovals(id string, offers []PendingApproval) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.PendingApprovals = offers
	if len(offers) > 0 {
		st.Status = StatusAwaitingApproval
	}
	m.runs[id] = st
	m.persist(id)
}

func (m *Manager) Running(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.actors[id] != nil
}

func (m *Manager) RunningIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.actors))
	for id := range m.actors {
		ids = append(ids, id)
	}
	return ids
}

func (m *Manager) Acquire(id string, parent context.Context) (context.Context, func(), error) {
	ctx, cancel := context.WithCancel(parent)
	m.mu.Lock()
	if m.actors[id] != nil {
		m.mu.Unlock()
		cancel()
		return nil, nil, ErrBusy
	}
	m.actors[id] = &Actor{ID: id, cancel: cancel}
	st := m.state(id)
	st.Status = StatusRunning
	m.runs[id] = st
	m.persist(id)
	m.mu.Unlock()
	return ctx, func() {
		cancel()
		m.mu.Lock()
		delete(m.actors, id)
		st := m.state(id)
		st.Status = StatusIdle
		st.PendingApprovals = nil
		m.runs[id] = st
		m.persist(id)
		m.mu.Unlock()
	}, nil
}

func (m *Manager) Interrupt(id string) {
	m.mu.Lock()
	a := m.actors[id]
	m.mu.Unlock()
	if a != nil && a.cancel != nil {
		a.cancel()
	}
}

func (m *Manager) Forget(id string) {
	m.Interrupt(id)
	m.mu.Lock()
	delete(m.actors, id)
	delete(m.runs, id)
	m.mu.Unlock()
	_ = RemoveRun(m.Dir, id)
	if m.Index != nil {
		m.Index.Delete(id)
		_ = m.Index.Flush()
	}
}

func (m *Manager) IndexBlob(id, blob string) {
	if m.Index == nil {
		return
	}
	m.Index.Put(id, blob)
	_ = m.Index.Flush()
}
