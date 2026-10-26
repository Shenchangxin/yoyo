// 把任务、目标、监视、想法和提案写进目录下的 workspace.json。
// flush 先写 path.tmp，权限 0o600，再改名替换。

package personal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store 把 doc 读写到 path 指向的 workspace.json。
type Store struct {
	mu   sync.Mutex
	path string
	doc  doc
}

// openStore 按 0o755 创建目录。文件读失败时仍返回存储。SamplePages 为 nil 时改成空映射。LastIdeasAt 为零时写成当前 UTC。
func openStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "workspace.json")}
	_ = s.load()
	if s.doc.SamplePages == nil {
		s.doc.SamplePages = map[string]string{}
	}
	if s.doc.LastIdeasAt.IsZero() {
		s.doc.LastIdeasAt = time.Now().UTC()
	}
	return s, nil
}

// load 读取 path 并把 JSON 解进 doc。
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.doc)
}

// flush 把 doc 写成缩进 JSON，先写 path.tmp，权限 0o600，再改名替换 path。
func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// snapshot 在锁内复制任务、目标、监视、想法、产物、提案和选择。Worker.Running 留给 Engine.Snapshot 填写。
func (s *Store) snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{
		Tasks:     append([]Task(nil), s.doc.Tasks...),
		Goals:     append([]Goal(nil), s.doc.Goals...),
		Monitors:  append([]Monitor(nil), s.doc.Monitors...),
		Ideas:     append([]Idea(nil), s.doc.Ideas...),
		Artifacts: append([]Artifact(nil), s.doc.Artifacts...),
		Proposals: append([]Proposal(nil), s.doc.Proposals...),
		Choices:   append([]Choice(nil), s.doc.Choices...),
		LastIdeas: s.doc.LastIdeasAt,
		Worker:    WorkerStatus{LastTickAt: s.doc.LastTickAt},
	}
}

// cloneMap 复制映射的顶层键。nil 得到空映射。
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// putTask 按 ID 替换任务，没有则追加，然后 flush。
func (s *Store) putTask(t Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Tasks {
		if s.doc.Tasks[i].ID == t.ID {
			s.doc.Tasks[i] = t
			_ = s.flush()
			return
		}
	}
	s.doc.Tasks = append(s.doc.Tasks, t)
	_ = s.flush()
}

// task 按 ID 查找，并复制 Input 和 State 后返回。
func (s *Store) task(id string) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.doc.Tasks {
		if t.ID == id {
			t.Input = cloneMap(t.Input)
			t.State = cloneMap(t.State)
			return t, true
		}
	}
	return Task{}, false
}

// tasks 返回任务切片的副本。
func (s *Store) tasks() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Task, len(s.doc.Tasks))
	copy(out, s.doc.Tasks)
	return out
}

// casTask 在 expectStatus 或 leaseID 非空时要求与当前值一致。patch 返回 false 则不写盘。
func (s *Store) casTask(id, expectStatus, leaseID string, patch func(*Task) bool) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Tasks {
		t := &s.doc.Tasks[i]
		if t.ID != id {
			continue
		}
		if expectStatus != "" && string(t.Status) != expectStatus {
			return Task{}, false
		}
		if leaseID != "" && t.LeaseID != leaseID {
			return Task{}, false
		}
		if !patch(t) {
			return Task{}, false
		}
		out := *t
		out.Input = cloneMap(t.Input)
		out.State = cloneMap(t.State)
		_ = s.flush()
		return out, true
	}
	return Task{}, false
}

// putGoal 按 ID 替换目标，没有则追加，然后 flush。
func (s *Store) putGoal(g Goal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Goals {
		if s.doc.Goals[i].ID == g.ID {
			s.doc.Goals[i] = g
			_ = s.flush()
			return
		}
	}
	s.doc.Goals = append(s.doc.Goals, g)
	_ = s.flush()
}

// goal 按 ID 查找目标。
func (s *Store) goal(id string) (Goal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.doc.Goals {
		if g.ID == id {
			return g, true
		}
	}
	return Goal{}, false
}

// goals 返回目标切片的副本。
func (s *Store) goals() []Goal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Goal(nil), s.doc.Goals...)
}

// casGoal 在 expect 非空时要求状态一致。patch 返回 false 则不写盘。返回值带上里程碑副本。
func (s *Store) casGoal(id, expect string, patch func(*Goal) bool) (Goal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Goals {
		g := &s.doc.Goals[i]
		if g.ID != id {
			continue
		}
		if expect != "" && g.Status != expect {
			return Goal{}, false
		}
		if !patch(g) {
			return Goal{}, false
		}
		out := *g
		out.Milestones = append([]Milestone(nil), g.Milestones...)
		_ = s.flush()
		return out, true
	}
	return Goal{}, false
}

// putMonitor 按 ID 替换监视，没有则追加，然后 flush。
func (s *Store) putMonitor(m Monitor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Monitors {
		if s.doc.Monitors[i].ID == m.ID {
			s.doc.Monitors[i] = m
			_ = s.flush()
			return
		}
	}
	s.doc.Monitors = append(s.doc.Monitors, m)
	_ = s.flush()
}

// casMonitor 在 expectStatus 非空或 expectChecks 大于等于 0 时核对这些字段。patch 返回 false 则不写盘。
func (s *Store) casMonitor(id, expectStatus string, expectChecks int, patch func(*Monitor) bool) (Monitor, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Monitors {
		m := &s.doc.Monitors[i]
		if m.ID != id {
			continue
		}
		if expectStatus != "" && m.Status != expectStatus {
			return Monitor{}, false
		}
		if expectChecks >= 0 && m.Checks != expectChecks {
			return Monitor{}, false
		}
		if !patch(m) {
			return Monitor{}, false
		}
		out := *m
		_ = s.flush()
		return out, true
	}
	return Monitor{}, false
}

// monitor 按 ID 查找监视。
func (s *Store) monitor(id string) (Monitor, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.doc.Monitors {
		if m.ID == id {
			return m, true
		}
	}
	return Monitor{}, false
}

// putPage 按 ID 替换监视页快照，没有则追加，然后 flush。这是 workspace.json 里的抓取快照，不是 pages 包的页面库。
func (s *Store) putPage(p MonitorPage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Pages {
		if s.doc.Pages[i].ID == p.ID {
			s.doc.Pages[i] = p
			_ = s.flush()
			return
		}
	}
	s.doc.Pages = append(s.doc.Pages, p)
	_ = s.flush()
}

// page 按 ID 查找监视页快照。
func (s *Store) page(id string) (MonitorPage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.doc.Pages {
		if p.ID == id {
			return p, true
		}
	}
	return MonitorPage{}, false
}

// insertIdea 在 ID 已存在时返回 false，否则追加并 flush。
func (s *Store) insertIdea(it Idea) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.doc.Ideas {
		if e.ID == it.ID {
			return false
		}
	}
	s.doc.Ideas = append(s.doc.Ideas, it)
	_ = s.flush()
	return true
}

// idea 按 ID 查找想法。
func (s *Store) idea(id string) (Idea, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.doc.Ideas {
		if it.ID == id {
			return it, true
		}
	}
	return Idea{}, false
}

// ideas 返回想法切片的副本。
func (s *Store) ideas() []Idea {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Idea(nil), s.doc.Ideas...)
}

// casIdea 在 expect 非空时要求状态一致。patch 返回 false 则不写盘。
func (s *Store) casIdea(id, expect string, patch func(*Idea) bool) (Idea, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Ideas {
		it := &s.doc.Ideas[i]
		if it.ID != id {
			continue
		}
		if expect != "" && it.Status != expect {
			return Idea{}, false
		}
		if !patch(it) {
			return Idea{}, false
		}
		out := *it
		_ = s.flush()
		return out, true
	}
	return Idea{}, false
}

// putArtifact 追加一条产物并 flush。相同 ID 也会再追加一条。
func (s *Store) putArtifact(a Artifact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.Artifacts = append(s.doc.Artifacts, a)
	_ = s.flush()
}

// putProposal 按 ID 替换提案，没有则追加，然后 flush。
func (s *Store) putProposal(p Proposal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Proposals {
		if s.doc.Proposals[i].ID == p.ID {
			s.doc.Proposals[i] = p
			_ = s.flush()
			return
		}
	}
	s.doc.Proposals = append(s.doc.Proposals, p)
	_ = s.flush()
}

// proposal 按 ID 查找提案。
func (s *Store) proposal(id string) (Proposal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.doc.Proposals {
		if p.ID == id {
			return p, true
		}
	}
	return Proposal{}, false
}

// casProposal 在 expect 或 hash 非空时要求状态和哈希一致。patch 返回 false 则不写盘。
func (s *Store) casProposal(id, expect, hash string, patch func(*Proposal) bool) (Proposal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Proposals {
		p := &s.doc.Proposals[i]
		if p.ID != id {
			continue
		}
		if expect != "" && string(p.Status) != expect {
			return Proposal{}, false
		}
		if hash != "" && p.Hash != hash {
			return Proposal{}, false
		}
		if !patch(p) {
			return Proposal{}, false
		}
		out := *p
		_ = s.flush()
		return out, true
	}
	return Proposal{}, false
}

// putChoice 按 ID 替换选择，没有则追加，然后 flush。
func (s *Store) putChoice(c Choice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Choices {
		if s.doc.Choices[i].ID == c.ID {
			s.doc.Choices[i] = c
			_ = s.flush()
			return
		}
	}
	s.doc.Choices = append(s.doc.Choices, c)
	_ = s.flush()
}

// choice 按 ID 查找选择。
func (s *Store) choice(id string) (Choice, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.doc.Choices {
		if c.ID == id {
			return c, true
		}
	}
	return Choice{}, false
}

// setTick 写下 LastTickAt 并 flush。
func (s *Store) setTick(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.LastTickAt = at
	_ = s.flush()
}

// setIdeasAt 写下 LastIdeasAt 并 flush。
func (s *Store) setIdeasAt(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.LastIdeasAt = at
	_ = s.flush()
}

// sample 把样例页正文写入 SamplePages 并 flush。
func (s *Store) sample(key, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc.SamplePages == nil {
		s.doc.SamplePages = map[string]string{}
	}
	s.doc.SamplePages[key] = text
	_ = s.flush()
}

// sampleGet 返回 SamplePages 里该键的正文，没有则为空串。
func (s *Store) sampleGet(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doc.SamplePages[key]
}

// lastIdeasAt 返回上次刷新想法的时间。
func (s *Store) lastIdeasAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doc.LastIdeasAt
}

// errf 给格式化错误加上 personal: 前缀。
func errf(format string, args ...any) error {
	return fmt.Errorf("personal: "+format, args...)
}
