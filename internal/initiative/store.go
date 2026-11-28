package initiative

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/lease"
)

// Store 把开口的记忆写在 dir/initiative.json。判断、送达和反馈都经过它的锁。
// 桌面和 yoyo daemon 可以同时打开同一份：每次改之前拿 initiative.json.lock 并重读磁盘，读的时候文件变了就重读，
// 一边点的「别提醒」，另一边下一轮就看得到，送达的计数也不会被对方盖掉。
type Store struct {
	mu    sync.Mutex
	path  string
	book  Book
	stamp fileStamp
}

// fileStamp 是上次读到或写下时文件的修改时间和长度。
type fileStamp struct {
	mod  time.Time
	size int64
}

// Open 打开 dir 里的记录。目录不存在时创建。文件还不存在时从空记录开始。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "initiative.json")}
	s.reloadLocked(true)
	return s, nil
}

// reloadLocked 在文件变了（或 force）时重读。读不到或解不开时保留内存里的。调用方持有 s.mu。
func (s *Store) reloadLocked(force bool) {
	st, err := os.Stat(s.path)
	if err != nil {
		return
	}
	stamp := fileStamp{mod: st.ModTime(), size: st.Size()}
	if !force && stamp == s.stamp {
		return
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var book Book
	if json.Unmarshal(b, &book) != nil {
		return
	}
	s.book = book
	s.stamp = stamp
}

// edit 拿到跨进程的文件锁、重读磁盘、改、写回。调用方持有 s.mu。
func (s *Store) edit(fn func()) {
	unlock, _ := lease.Lock(s.path + ".lock")
	defer unlock()
	s.reloadLocked(true)
	fn()
	_ = s.flush()
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
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	if st, err := os.Stat(s.path); err == nil {
		s.stamp = fileStamp{mod: st.ModTime(), size: st.Size()}
	}
	return nil
}

// Seen 复制已经见过的签名。公开检索用它跳过已经说过的链接。
func (s *Store) Seen() map[string]string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadLocked(false)
	out := make(map[string]string, len(s.book.Seen))
	for id, sig := range s.book.Seen {
		out[id] = sig
	}
	return out
}

// Assess 用这一轮材料判断。新见到的基线和当天的承诺写回磁盘，这一轮不算已经开口。
func (s *Store) Assess(in Input) Briefing {
	if s == nil {
		briefing, _ := Assess(in)
		return briefing
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var briefing Briefing
	s.edit(func() {
		in.Book = s.book
		var book Book
		briefing, book = Assess(in)
		s.book = book
	})
	return briefing
}

// Deliver 记下 signals 里已经送到的话，并写回磁盘。
func (s *Store) Deliver(signals []Signal, now time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.edit(func() {
		s.book = Deliver(s.book, signals, now)
	})
}

// Feedback 记下反应。wrong 时返回要忘掉的记忆编号。
func (s *Store) Feedback(className, memoryID, kind, day string) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	forget := ""
	s.edit(func() {
		var book Book
		book, forget = Feedback(s.book, className, memoryID, kind, day)
		s.book = book
	})
	return forget
}

// Quiet 复制 day 这一天仍被按住的类别。
func (s *Store) Quiet(day string) map[string]Mute {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadLocked(false)
	return Quiet(s.book, day)
}

// ReachSent 是 day 这一天已经写进会话的资讯条数。换了一天则为 0。
func (s *Store) ReachSent(day string) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadLocked(false)
	if s.book.Reach.Day != day {
		return 0
	}
	return s.book.Reach.N
}

// Recent 复制最近 n 天的打断和反应计数，从新到旧。
func (s *Store) Recent(n int) []DayTally {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reloadLocked(false)
	return Recent(s.book, n)
}

// NoteSchedule 记下一条到期的日程。
func (s *Store) NoteSchedule(id, title, day string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.edit(func() {
		s.book = NoteSchedule(s.book, id, title, day)
	})
}
