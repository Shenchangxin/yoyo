// snapshot.go 把引擎收成一份可落盘的快照，并按这份快照还原。
// 回调 OnLaunch 和 OnInput 不在快照里。还原不改它们。

package superagent

import "encoding/json"

// Snapshot 是引擎里要留住的记录。Seq 和 N 继续编号，避免还原后撞上旧编号。
type Snapshot struct {
	Policy    Policy
	Triggers  []Trigger
	Threads   []Thread
	Tickets   []Ticket
	Decisions []Decision
	Outbox    []Outbound
	Seq       int
	N         int
}

// Export 返回一份深拷贝。编码失败时返回错误。
func (e *Engine) Export() (Snapshot, error) {
	if e == nil {
		return Snapshot{}, errCode("empty")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	raw, err := json.Marshal(e.snapshotLocked())
	if err != nil {
		return Snapshot{}, err
	}
	var out Snapshot
	if err := json.Unmarshal(raw, &out); err != nil {
		return Snapshot{}, err
	}
	return out, nil
}

// Import 用 snap 换掉记录。snap 的 Mode 为空时当作 observe，Version 小于 1 时当作 1。
func (e *Engine) Import(snap Snapshot) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if snap.Policy.Mode == "" {
		snap.Policy.Mode = "observe"
	}
	if snap.Policy.Version < 1 {
		snap.Policy.Version = 1
	}
	e.policy = snap.Policy
	e.seq = snap.Seq
	e.n = snap.N
	e.triggers = map[string]*Trigger{}
	e.threads = map[string]*Thread{}
	e.tickets = map[string]*Ticket{}
	e.decisions = append([]Decision(nil), snap.Decisions...)
	e.outbox = append([]Outbound(nil), snap.Outbox...)
	for i := range snap.Triggers {
		cp := snap.Triggers[i]
		if cp.ID == "" {
			continue
		}
		e.triggers[cp.ID] = &cp
	}
	for i := range snap.Threads {
		cp := snap.Threads[i]
		if cp.Key == "" {
			continue
		}
		e.threads[cp.Key] = &cp
	}
	for i := range snap.Tickets {
		cp := snap.Tickets[i]
		if cp.ID == "" {
			continue
		}
		e.tickets[cp.ID] = &cp
	}
}

// snapshotLocked 收集当前记录。调用时要已持有锁。切片与活数据共享，调用方要在解锁前把它编码走。
func (e *Engine) snapshotLocked() Snapshot {
	snap := Snapshot{Policy: e.policy, Seq: e.seq, N: e.n, Decisions: append([]Decision(nil), e.decisions...), Outbox: append([]Outbound(nil), e.outbox...)}
	for _, tr := range e.triggers {
		if tr != nil {
			snap.Triggers = append(snap.Triggers, *tr)
		}
	}
	for _, th := range e.threads {
		if th != nil {
			snap.Threads = append(snap.Threads, *th)
		}
	}
	for _, ticket := range e.tickets {
		if ticket != nil {
			snap.Tickets = append(snap.Tickets, *ticket)
		}
	}
	return snap
}
