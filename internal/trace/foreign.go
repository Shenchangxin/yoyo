// foreign.go 让一个进程读到别的进程追加进同一份会话文件的事件。
// 桌面和 yoyo daemon 打开同一个 home 时，拿着后台租约的那个往主会话写短讯、简报和例行结果，另一个用它把这些事件补发给自己的界面。
// 本进程写的行记着起始位置，读的时候跳过，不发两遍。

package trace

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// localKeep 是每个会话记住的本进程写入位置的条数上限。
const localKeep = 4096

func (s *Store) noteLocalLocked(sessionID string, off int64) {
	if s.local == nil {
		s.local = map[string][]int64{}
	}
	offs := append(s.local[sessionID], off)
	if len(offs) > localKeep {
		offs = append([]int64(nil), offs[len(offs)-localKeep:]...)
	}
	s.local[sessionID] = offs
}

func (s *Store) forgetLocalLocked(sessionID string) {
	if s.local != nil {
		delete(s.local, sessionID)
	}
}

// pruneLocalLocked 丢掉 pos 之前的位置。那些行已经读过了。
func (s *Store) pruneLocalLocked(sessionID string, pos int64) {
	offs := s.local[sessionID]
	i := 0
	for i < len(offs) && offs[i] < pos {
		i++
	}
	if i == 0 {
		return
	}
	if i == len(offs) {
		delete(s.local, sessionID)
		return
	}
	s.local[sessionID] = append([]int64(nil), offs[i:]...)
}

// Size 返回会话文件现在的长度。文件不存在时为 0。
func (s *Store) Size(sessionID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := os.Stat(s.path(sessionID))
	if err != nil {
		return 0
	}
	return st.Size()
}

// ForeignSince 读 from 之后别的进程写进来的完整行，返回这些事件和读到的位置。
// 本进程追加的行跳过。最后一行还没写完整时停在它前面，下次再读。文件变短（被整份重写）时不补发，位置挪到文件尾。
func (s *Store) ForeignSince(sessionID string, from int64) ([]Event, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, from, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, from, err
	}
	size := st.Size()
	if size < from {
		s.pruneLocalLocked(sessionID, size)
		return nil, size, nil
	}
	if size == from {
		return nil, from, nil
	}
	if _, err := f.Seek(from, io.SeekStart); err != nil {
		return nil, from, err
	}
	local := map[int64]bool{}
	for _, off := range s.local[sessionID] {
		if off >= from {
			local[off] = true
		}
	}
	r := bufio.NewReaderSize(io.LimitReader(f, size-from), 64*1024)
	pos := from
	var out []Event
	for {
		line, _ := r.ReadBytes('\n')
		if len(line) == 0 || line[len(line)-1] != '\n' {
			break
		}
		start := pos
		pos += int64(len(line))
		if local[start] {
			continue
		}
		body := bytes.TrimSpace(line)
		if len(body) == 0 {
			continue
		}
		var ev Event
		if json.Unmarshal(body, &ev) != nil {
			continue
		}
		if ev.SessionID == "" {
			ev.SessionID = sessionID
		}
		out = append(out, ev)
	}
	s.pruneLocalLocked(sessionID, pos)
	return out, pos, nil
}
