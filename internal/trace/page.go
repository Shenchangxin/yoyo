package trace

const (
	// DefaultPageTurns is the renderer hot window: live turn plus recent
	// history. Older turns load on scroll, never on send. A long coding
	// thread's full JSONL must never cross Wails in one shot.
	DefaultPageTurns = 8
	// DefaultPageBytes is a hard marshal budget for one TrajectoryPage
	// after UI slimming. WebView2 parses this on the UI thread.
	DefaultPageBytes = 256_000
)

// Page is a store-seq window of a session JSONL. Seq is the sidecar index
// cursor, independent of the live Hub seq.
type Page struct {
	Events  []Event
	HeadSeq int64
	TailSeq int64
	Older   bool
}

// PageTurns returns the last `turns` operator/user events (and everything
// between them) ending at beforeSeq, or at EOF when beforeSeq is 0.
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

// ReadFromCheckpoint returns events from the last checkpoint (inclusive) to
// EOF so MessagesFromEvents can rebuild without scanning the cold prefix.
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
