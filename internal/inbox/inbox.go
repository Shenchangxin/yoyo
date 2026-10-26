// Package inbox 把待办写在目录里的 inbox.json。
// Push 按 Key 去重，新条目未读为真。条数超过 500 时只留最后 500 条。同 Key 已存在时返回旧条目，不改它。
package inbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Kind 是一条收件的种类。
type Kind string

const (
	// KindApproval 是待审批，值为 approval。
	KindApproval Kind = "approval"
	// KindSchedule 是日程，值为 schedule。
	KindSchedule Kind = "schedule"
	// KindAsk 是向用户提问，值为 ask_user。
	KindAsk Kind = "ask_user"
	// KindHeartbeat 是心跳，值为 heartbeat。
	KindHeartbeat Kind = "heartbeat"
	// KindArtifact 是产物，值为 artifact。
	KindArtifact Kind = "artifact"
	// KindWork 是工作项，值为 work。
	KindWork Kind = "work"
	// KindProposal 是提议，值为 proposal。
	KindProposal Kind = "proposal"
	// KindWatch 是监视，值为 watch。
	KindWatch Kind = "watch"
)

// Item 是收件箱里的一条。
type Item struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	SessionID string    `json:"session_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Key       string    `json:"key,omitempty"`
	Unread    bool      `json:"unread"`
	CreatedAt time.Time `json:"created_at"`
}

// Store 把条目保存在内存，并刷到 inbox.json。
type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
}

// Open 创建 dir，权限 0o755，路径是 dir/inbox.json。文件还不存在时照样返回 Store。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "inbox.json")}
	_ = s.load()
	return s, nil
}

// Push 追加一条。Key 非空且已有同 Key 时返回那条旧的。
// ID 为空时用 UTC 时间，格式 in-20060102T150405.000000000。CreatedAt 为零时用当前 UTC。Unread 设为真。
// 追加后若超过 500 条，只留最后 500 条，然后刷盘。
// it 的 Key 非空且已有同 Key 时返回那条旧的。ID 为空时写成当前 UTC 的 in-20060102T150405.000000000。CreatedAt 为零时写成当前 UTC。Unread 设为 true 后追加，超过 500 条时只留最后 500 条。
func (s *Store) Push(it Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it.Key != "" {
		for _, existing := range s.items {
			if existing.Key == it.Key {
				return existing
			}
		}
	}
	if it.ID == "" {
		it.ID = time.Now().UTC().Format("in-20060102T150405.000000000")
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = time.Now().UTC()
	}
	it.Unread = true
	s.items = append(s.items, it)
	if len(s.items) > 500 {
		s.items = s.items[len(s.items)-500:]
	}
	_ = s.flush()
	return it
}

// List 返回按 CreatedAt 从新到旧排序的副本。
func (s *Store) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Item(nil), s.items...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Unread 返回 Unread 为真的条数。
func (s *Store) Unread() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, it := range s.items {
		if it.Unread {
			n++
		}
	}
	return n
}

// MarkRead 把指定 ID 的 Unread 设为假并刷盘。没有这条时也刷盘。
// id 与每条的 ID 比较，相等则把 Unread 设为 false。
func (s *Store) MarkRead(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Unread = false
		}
	}
	_ = s.flush()
}

// Dismiss 删掉指定 ID。找到才刷盘并返回真，找不到返回假。
// id 与每条的 ID 比较，相等的条目删掉。找到才刷盘并返回 true。
func (s *Store) Dismiss(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	found := false
	for _, it := range s.items {
		if it.ID == id {
			found = true
			continue
		}
		out = append(out, it)
	}
	if !found {
		return false
	}
	s.items = out
	_ = s.flush()
	return true
}

// load 从 inbox.json 解出条目。文件不存在时返回错误，Open 会忽略。
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.items)
}

// flush 先把 JSON 写到路径加 .tmp，权限 0o644，再改成 inbox.json。
func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
