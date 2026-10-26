// Package memory 把记忆写在目录里的 memory.json。
// Write 只让 profile 直接生效，其他种类先标成待审。ProfilePin 拼档案文本时跳过待审条目。
package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind 是一条记忆的种类。
type Kind string

const (
	// KindProfile 是档案，值为 profile。Write 对它把 Staging 设为假。
	KindProfile Kind = "profile"
	// KindProject 是项目，值为 project。Write 对它把 Staging 设为真。
	KindProject Kind = "project"
	// KindEpisodic 是情景，值为 episodic。Write 对它把 Staging 设为真。
	KindEpisodic Kind = "episodic"
	// KindSemantic 是语义，值为 semantic。Write 对它把 Staging 设为真。
	KindSemantic Kind = "semantic"
)

// Item 是一条记忆。
type Item struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Text      string    `json:"text"`
	Project   string    `json:"project,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Staging   bool      `json:"staging,omitempty"`
	Source    string    `json:"source,omitempty"`
}

// Store 把条目保存在内存，并刷到 memory.json。
type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
}

// Open 创建 dir，权限 0o755，路径是 dir/memory.json。文件还不存在时照样返回 Store。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "memory.json")}
	_ = s.load()
	return s, nil
}

// load 从 memory.json 解出条目。文件不存在时返回错误，Open 会忽略。
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.items)
}

// flush 先把 JSON 写到路径加 .tmp，权限 0o600，再改成 memory.json。
func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Write 追加一条并刷盘。ID 为空时用 UTC 时间，格式 20060102T150405.000000000。CreatedAt 为零时用当前 UTC。
// Kind 不是 profile 时 Staging 为真。Kind 是 profile 时 Staging 为假。
func (s *Store) Write(it Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it.ID == "" {
		it.ID = time.Now().UTC().Format("20060102T150405.000000000")
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = time.Now().UTC()
	}
	it.Staging = it.Kind != KindProfile
	if it.Kind == KindProfile {
		it.Staging = false
	}
	s.items = append(s.items, it)
	_ = s.flush()
	return it
}

// Forget 删掉指定 ID。无论是否找到都会刷盘。找到返回真。
func (s *Store) Forget(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	ok := false
	for _, it := range s.items {
		if it.ID == id {
			ok = true
			continue
		}
		out = append(out, it)
	}
	s.items = out
	_ = s.flush()
	return ok
}

// Search 按种类和子串筛选。q 和文本、项目都转成小写再包含比较。kind 为空则不限种类。
// 结果按 CreatedAt 从新到旧。limit 大于 0 且条数超过它时只留前面这么多条。limit 小于等于 0 则不截断。
func (s *Store) Search(q string, kind Kind, limit int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []Item
	for _, it := range s.items {
		if kind != "" && it.Kind != kind {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.Text), q) && !strings.Contains(strings.ToLower(it.Project), q) {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ProfilePin 拼出档案文本。先 Search 空查询、种类 profile、上限 32 条，再跳过 Staging 为真的条目。
// 每行是 "- " 加文本加换行。budget 大于 0 且再加这一行会超过 budget 时停止。budget 小于等于 0 则不按长度截断。返回值按字节累计。
func (s *Store) ProfilePin(budget int) string {
	items := s.Search("", KindProfile, 32)
	var b strings.Builder
	n := 0
	for _, it := range items {
		if it.Staging {
			continue
		}
		line := "- " + it.Text + "\n"
		if budget > 0 && n+len(line) > budget {
			break
		}
		b.WriteString(line)
		n += len(line)
	}
	return b.String()
}

// Promote 把指定 ID 的 Staging 设为假并刷盘，返回真。没有这条时返回假，不刷盘。
func (s *Store) Promote(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Staging = false
			_ = s.flush()
			return true
		}
	}
	return false
}

// All 返回条目切片的副本。
func (s *Store) All() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Item(nil), s.items...)
	return out
}

// Path 返回 memory.json 的路径，不加锁。
func (s *Store) Path() string { return s.path }
