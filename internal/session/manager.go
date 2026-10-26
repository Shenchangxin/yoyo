// Package session 管一条会话的身份、运行状态和可重建的搜索索引。
// Acquire 在同一 id 已有 Actor 时返回 ErrBusy，错误文案是 session already running。
// LoadRun 把 running、compacting、awaiting_approval 收成 idle，并丢掉 high_risk、send_as_you、computer_use。
// 频道常量只有 agent 和 video。马具策略只有 follow_active 和 pin。
package session

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ErrBusy 表示这条会话已经有一轮在跑。Error 的文案是 session already running。
const ErrBusy errString = "session already running"

// errString 是可以用常量表示的错误文案。
type errString string

// Error 返回常量里的文案。
func (e errString) Error() string { return string(e) }

// Actor 是一条会话正在跑的回合。同一时刻只有一个，插入转向文本不占用新的 Actor。
type Actor struct {
	// ID 是会话 id。
	ID string
	// cancel 取消这一轮的 context。Interrupt 调用它。
	cancel context.CancelFunc
}

// Manager 按会话 id 保存正在跑的 Actor，并把 RunState 写到磁盘。
type Manager struct {
	// Dir 是运行状态 json 和搜索索引所在的目录。
	Dir string
	// Index 是 OpenIndex 打开的搜索边车。Forget 和 IndexBlob 会用它。
	Index *Index

	// mu 保护 actors 和 runs。
	mu sync.Mutex
	// actors 是正在跑的会话。没有条目表示没在跑。
	actors map[string]*Actor
	// runs 是已经读进内存的运行状态。
	runs map[string]RunState
}

// NewManager 用 dir 打开索引，并准备空的 actors 和 runs。
func NewManager(dir string) *Manager {
	return &Manager{
		Dir:    dir,
		Index:  OpenIndex(dir),
		actors: map[string]*Actor{},
		runs:   map[string]RunState{},
	}
}

// state 返回内存里的状态。没有时 LoadRun。LoadRun 出错就用 StatusIdle。调用方须已持有 mu。
// id 用来查 runs，没有时和 Dir 一起传给 LoadRun。
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

// persist 把内存里的该 id 状态交给 SaveRun。写盘错误被丢掉。调用方须已持有 mu。
func (m *Manager) persist(id string) {
	st := m.runs[id]
	_ = SaveRun(m.Dir, id, st)
}

// Load 返回该会话的运行状态。
// id 传给 state。
func (m *Manager) Load(id string) RunState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state(id)
}

// SetStatus 写入状态。新状态不是 running 时清空 ResumeText、ResumePlan 和 ResumeInFlight。
// id 传给 state 和 persist。
// status 写入 Status。不等于 StatusRunning 时清空 ResumeText、ResumePlan 和 ResumeInFlight。
func (m *Manager) SetStatus(id, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.Status = status
	if status != StatusRunning {
		st.ResumeText = ""
		st.ResumePlan = false
		st.ResumeInFlight = false
	}
	m.runs[id] = st
	m.persist(id)
}

// SetResume 按文本是否非空或 plan 是否为真，调用 MarkTurnInFlight。文本本身不写入 ResumeText。
// id 传给 MarkTurnInFlight。
// text 去掉空白后是否为空，和 plan 一起决定 MarkTurnInFlight 的 inFlight 实参。
func (m *Manager) SetResume(id, text string, plan bool) {
	m.MarkTurnInFlight(id, strings.TrimSpace(text) != "" || plan, plan)
}

// MarkTurnInFlight 记下是否有未完成回合和是否带计划。ResumeText 总是写成空字符串，用户原文留在 JSONL。
// id 传给 state 和 persist。
// inFlight 写入 ResumeInFlight。
// plan 写入 ResumePlan。
func (m *Manager) MarkTurnInFlight(id string, inFlight, plan bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.ResumeInFlight = inFlight
	st.ResumePlan = plan
	// 用户原文在 JSONL。写进这里会在崩溃后把同一条再入队一次。
	st.ResumeText = ""
	m.runs[id] = st
	m.persist(id)
}

// Enqueue 把回合加到队列末尾。ID 为空时用 q-、UTC 纳秒和当前长度加 1。
// id 传给 state 和 persist。
// turn 追加到 Queue。ID 为空时写成 q-、UTC 纳秒和当前长度加 1。
func (m *Manager) Enqueue(id string, turn QueuedTurn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	if turn.ID == "" {
		turn.ID = fmt.Sprintf("q-%d-%d", time.Now().UTC().UnixNano(), len(st.Queue)+1)
	}
	st.Queue = append(st.Queue, turn)
	m.runs[id] = st
	m.persist(id)
}

// PrependQueue 把回合插到队列最前。ID 为空时写成 q-resume。
// id 传给 state 和 persist。
// turn 插到 Queue 最前。ID 为空时写成 q-resume。
func (m *Manager) PrependQueue(id string, turn QueuedTurn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	if turn.ID == "" {
		turn.ID = "q-resume"
	}
	st.Queue = append([]QueuedTurn{turn}, st.Queue...)
	m.runs[id] = st
	m.persist(id)
}

// CancelQueue 按 itemID 删掉一条队列项。找到返回 true。
// id 传给 state 和 persist。
func (m *Manager) CancelQueue(id, itemID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	out := st.Queue[:0]
	ok := false
	for _, t := range st.Queue {
		if t.ID == itemID {
			ok = true
			continue
		}
		out = append(out, t)
	}
	st.Queue = out
	m.runs[id] = st
	m.persist(id)
	return ok
}

// ReorderQueue 把 itemID 的下标加上 delta 后与该位置交换。找不到或越界返回 false。
// id 传给 state 和 persist。
func (m *Manager) ReorderQueue(id, itemID string, delta int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	i := -1
	for n, t := range st.Queue {
		if t.ID == itemID {
			i = n
			break
		}
	}
	if i < 0 {
		return false
	}
	j := i + delta
	if j < 0 || j >= len(st.Queue) {
		return false
	}
	st.Queue[i], st.Queue[j] = st.Queue[j], st.Queue[i]
	m.runs[id] = st
	m.persist(id)
	return true
}

// SetConnectorAllow 用 accounts 的副本替换 ConnectorAllow。
// id 传给 state 和 persist。
func (m *Manager) SetConnectorAllow(id string, accounts []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.ConnectorAllow = append([]string(nil), accounts...)
	m.runs[id] = st
	m.persist(id)
}

// ConnectorAllow 返回 ConnectorAllow 的副本。
// id 传给 state。
func (m *Manager) ConnectorAllow(id string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.state(id).ConnectorAllow...)
}

// QueueList 返回队列副本。
// id 传给 state。
func (m *Manager) QueueList(id string) []QueuedTurn {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	return append([]QueuedTurn(nil), st.Queue...)
}

// PopQueue 取出并删掉队列第一项。队列空时返回零值和 false。
// id 传给 state。队列非空时也传给 persist。
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

// PushSteer 在会话正在跑时把 text 加进 Steers。没有 Actor 时错误文案是 session is not running。
// id 用来查 actors。有 Actor 时传给 state 和 persist。
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

// PullSteer 取走全部 Steers 并清空。没有时返回空字符串。有内容时前面加上 User steering (apply now): 和换行，各项用换行拼接。
// id 传给 state。Steers 非空时也传给 persist。
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

// SetCaps 把能力名单经 dropHighRisk 后写入 SessionCaps。
// id 传给 state 和 persist。
// caps 传给 dropHighRisk，结果写入 SessionCaps。
func (m *Manager) SetCaps(id string, caps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.SessionCaps = dropHighRisk(caps)
	m.runs[id] = st
	m.persist(id)
}

// SetAuthMode 写入授权模式，并把 caps 经 dropHighRisk 后写入 SessionCaps。
// id 传给 state 和 persist。
// mode 写入 AuthMode。
func (m *Manager) SetAuthMode(id, mode string, caps []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.AuthMode = mode
	st.SessionCaps = dropHighRisk(caps)
	m.runs[id] = st
	m.persist(id)
}

// AuthMode 返回该会话的授权模式。
// id 传给 state。
func (m *Manager) AuthMode(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state(id).AuthMode
}

// Caps 返回 SessionCaps 的副本。
// id 传给 state。
func (m *Manager) Caps(id string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.state(id).SessionCaps...)
}

// SetApprovals 写入待审批列表。列表非空时状态改为 awaiting_approval。
// 列表为空且当前是 awaiting_approval 时，有 Actor 就改回 running，否则改回 idle。
// offers 写入 PendingApprovals。长度大于 0 时 Status 改为 awaiting_approval。长度等于 0 且当前是 awaiting_approval 时，有 Actor 就改回 running，否则改回 idle。
func (m *Manager) SetApprovals(id string, offers []PendingApproval) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.state(id)
	st.PendingApprovals = offers
	if len(offers) > 0 {
		st.Status = StatusAwaitingApproval
	} else if st.Status == StatusAwaitingApproval {
		if m.actors[id] != nil {
			st.Status = StatusRunning
		} else {
			st.Status = StatusIdle
		}
	}
	m.runs[id] = st
	m.persist(id)
}

// Running 报告该 id 是否有 Actor。
func (m *Manager) Running(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.actors[id] != nil
}

// RunningIDs 返回当前有 Actor 的会话 id，顺序不定。
func (m *Manager) RunningIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.actors))
	for id := range m.actors {
		ids = append(ids, id)
	}
	return ids
}

// Acquire 为该 id 占一个 Actor，并把状态写成 running。已经有 Actor 时取消新建的 context 并返回 ErrBusy。
// 返回的释放函数会取消 context、删掉 Actor、把状态写成 idle，并清空 PendingApprovals。
// parent 传给 context.WithCancel。
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

// Interrupt 调用该会话 Actor 的 cancel。没有 Actor 或 cancel 为空时什么都不做。
// id 用来取 actors 里的 Actor。
func (m *Manager) Interrupt(id string) {
	m.mu.Lock()
	a := m.actors[id]
	m.mu.Unlock()
	if a != nil && a.cancel != nil {
		a.cancel()
	}
}

// Forget 先 Interrupt，再从内存删掉 Actor 和运行状态，删除 run 文件，并从索引删掉该 id 后 Flush。
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

// IndexBlob 把 blob 放进该 id 的搜索索引并 Flush。Index 为 nil 时直接返回。
func (m *Manager) IndexBlob(id, blob string) {
	if m.Index == nil {
		return
	}
	m.Index.Put(id, blob)
	_ = m.Index.Flush()
}
