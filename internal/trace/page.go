// 本文件按操作者回合切一页 JSONL。字节预算常量给调用方用，这里不按它截断。

package trace

const (
	// DefaultPageTurns 是默认往回取的操作者回合数，值是 8。PageTurns 和 PageAroundUser 在 turns 小于等于 0 时用它。
	DefaultPageTurns = 8
	// DefaultPageBytes 是单页轨迹的封送字节预算，值是 256_000。本包的分页函数不读取它。
	DefaultPageBytes = 256_000
)

// Page 是会话 JSONL 上的一段。Seq 来自旁边的索引，不是实时 Hub 的序号。
type Page struct {
	// Events 是这一段里的事件。条数和索引一致时，Seq 会填上索引光标。
	Events []Event
	// HeadSeq 是这一段第一条的索引光标。
	HeadSeq int64
	// TailSeq 是这一段最后一条的索引光标。
	TailSeq int64
	// Older 表示这段前面还有事件。
	Older bool
}

// PageTurns 返回截止到 beforeSeq 之前、含最后 turns 个操作者用户事件及其间全部事件的一页。beforeSeq 为 0 时截止到文件末尾。
// turns 小于等于 0 时用 DefaultPageTurns。操作者用户事件不含 source 或 kind 为 steer 的行。没有条目或 end 小于等于 0 时返回空 Page。
func (s *Store) PageTurns(sessionID string, beforeSeq int64, turns int) (Page, error) {
	if turns <= 0 {
		turns = DefaultPageTurns
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(sessionID)
	if len(idx.entries) == 0 {
		return Page{}, nil
	}
	end := len(idx.entries)
	if beforeSeq > 0 {
		end = 0
		for i, e := range idx.entries {
			if e.Seq < beforeSeq {
				end = i + 1
			} else {
				break
			}
		}
	}
	if end <= 0 {
		return Page{}, nil
	}
	start := 0
	users := 0
	for i := end - 1; i >= 0; i-- {
		start = i
		if idx.entries[i].operatorUser() {
			users++
			if users >= turns {
				break
			}
		}
	}
	off := idx.entries[start].Off
	endOff := idx.size
	if end < len(idx.entries) {
		last := idx.entries[end-1]
		endOff = last.Off + int64(last.N)
	}
	evs, err := s.readRangeLocked(sessionID, off, endOff)
	if err != nil {
		return Page{}, err
	}
	if len(evs) == end-start {
		for i := range evs {
			evs[i].Seq = idx.entries[start+i].Seq
		}
	}
	return Page{
		Events:  evs,
		HeadSeq: idx.entries[start].Seq,
		TailSeq: idx.entries[end-1].Seq,
		Older:   start > 0,
	}, nil
}

// ReadFromCheckpoint 从最后一条类型为 compaction 且 kind 为 checkpoint 的行（含该行）读到文件末尾。没有这样的行就从头读。没有索引条目时读整个文件。
func (s *Store) ReadFromCheckpoint(sessionID string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(sessionID)
	if len(idx.entries) == 0 {
		return s.readRangeLocked(sessionID, 0, 0)
	}
	off := int64(0)
	for _, e := range idx.entries {
		if e.Type == TypeCompact && e.Kind == "checkpoint" {
			off = e.Off
		}
	}
	return s.readRangeLocked(sessionID, off, idx.size)
}
