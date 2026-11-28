package memory

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Shenchangxin/yoyo/internal/lease"
)

// fileStamp 是上次读到或写下时 memory.json 的修改时间和长度。
type fileStamp struct {
	mod  time.Time
	size int64
}

// begin 拿到 memory.json 的跨进程锁并重读文件，返回放开锁的函数。调用方持有 s.mu。
// 桌面和 yoyo daemon 同时开着一个 home 时，两边都会记下记忆：先重读再改，谁也不盖掉对方刚记下的。
func (s *Store) begin() func() {
	unlock, _ := lease.Lock(s.path + ".lock")
	s.reloadLocked(true)
	return unlock
}

// syncLocked 在文件被别的进程改过时重读。读的路径用它，不拿文件锁：文件是整份换掉的，读到的总是完整的一份。
func (s *Store) syncLocked() {
	s.reloadLocked(false)
}

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
	var items []Item
	if json.Unmarshal(b, &items) != nil {
		return
	}
	s.items = items
	s.stamp = stamp
}

// markLocked 记下刚写下的文件的样子，自己写的不再当成别人改的去重读。
func (s *Store) markLocked() {
	if st, err := os.Stat(s.path); err == nil {
		s.stamp = fileStamp{mod: st.ModTime(), size: st.Size()}
	}
}
