package initiative

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store 把开口的记忆写在 dir/initiative.json。
type Store struct {
	mu   sync.Mutex
	path string
	book Book
}

// Open 打开 dir 里的记录。目录不存在时创建。文件还不存在时从空记录开始。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "initiative.json")}
	_ = s.load()
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.book)
}

func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.book, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Assess 用这一轮材料判断，并把更新后的记录写回磁盘。
func (s *Store) Assess(in Input) Briefing {
	if s == nil {
		briefing, _ := Assess(in)
		return briefing
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	in.Book = s.book
	briefing, book := Assess(in)
	s.book = book
	_ = s.flush()
	return briefing
}

// Feedback 记下反应。wrong 时返回要忘掉的记忆编号。
func (s *Store) Feedback(className, memoryID, kind, day string) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	book, forget := Feedback(s.book, className, memoryID, kind, day)
	s.book = book
	_ = s.flush()
	return forget
}

// NoteSchedule 记下一条到期的日程。
func (s *Store) NoteSchedule(id, title, day string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.book = NoteSchedule(s.book, id, title, day)
	_ = s.flush()
}
