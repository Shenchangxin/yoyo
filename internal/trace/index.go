package trace

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// idxEntry is one JSONL line's location. Seq is a durable cursor for UI paging
// (not the live Hub seq). Kind is set for compaction so checkpointed reads
// can seek without unmarshalling the prefix.
type idxEntry struct {
	Seq  int64     `json:"seq"`
	Off  int64     `json:"off"`
	N    int       `json:"n"`
	Type EventType `json:"t"`
	Kind string    `json:"k,omitempty"`
}

type sessionIndex struct {
	size    int64
	entries []idxEntry
}

func (s *Store) idxPath(sessionID string) string {
	return s.path(sessionID) + ".idx"
}

func payloadKind(ev Event) string {
	if ev.Payload == nil {
		return ""
	}
	k, _ := ev.Payload["kind"].(string)
	return k
}

func (s *Store) indexLocked(sessionID string) *sessionIndex {
	if s.mem == nil {
		s.mem = map[string]*sessionIndex{}
	}
	if idx := s.mem[sessionID]; idx != nil && s.indexFreshLocked(sessionID, idx) {
		return idx
	}
	idx := s.loadOrRebuildLocked(sessionID)
	s.mem[sessionID] = idx
	return idx
}

func (s *Store) indexFreshLocked(sessionID string, idx *sessionIndex) bool {
	st, err := os.Stat(s.path(sessionID))
	if err != nil {
		return os.IsNotExist(err) && idx.size == 0 && len(idx.entries) == 0
	}
	return st.Size() == idx.size
}

func (s *Store) loadOrRebuildLocked(sessionID string) *sessionIndex {
	p := s.path(sessionID)
	st, err := os.Stat(p)
	if err != nil {
		return &sessionIndex{}
	}
	idx := loadIdxFile(s.idxPath(sessionID))
	if idx != nil && lastCover(idx, st.Size()) {
		idx.size = st.Size()
		return idx
	}
	return s.rebuildIndexLocked(sessionID)
}

func lastCover(idx *sessionIndex, size int64) bool {
	if len(idx.entries) == 0 {
		return size == 0
	}
	last := idx.entries[len(idx.entries)-1]
	return last.Off+int64(last.N) == size
}

func loadIdxFile(path string) *sessionIndex {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	idx := &sessionIndex{}
	line := 0
	for sc.Scan() {
		b := sc.Bytes()
		if len(b) == 0 {
			continue
		}
		line++
		if line == 1 {
			var head struct {
				V    int   `json:"v"`
				Size int64 `json:"size"`
			}
			if json.Unmarshal(b, &head) != nil || head.V != 1 {
				return nil
			}
			idx.size = head.Size
			continue
		}
		var e idxEntry
		if json.Unmarshal(b, &e) != nil || e.Seq == 0 {
			return nil
		}
		idx.entries = append(idx.entries, e)
	}
	if sc.Err() != nil {
		return nil
	}
	return idx
}

func (s *Store) rebuildIndexLocked(sessionID string) *sessionIndex {
	p := s.path(sessionID)
	f, err := os.Open(p)
	if err != nil {
		return &sessionIndex{}
	}
	defer f.Close()
	idx := &sessionIndex{}
	r := bufio.NewReaderSize(f, 64*1024)
	var off int64
	var seq int64
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			seq++
			n := len(line)
			var peek struct {
				Type    EventType      `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			_ = json.Unmarshal(line, &peek)
			kind := ""
			if peek.Payload != nil {
				kind, _ = peek.Payload["kind"].(string)
			}
			idx.entries = append(idx.entries, idxEntry{
				Seq: seq, Off: off, N: n, Type: peek.Type, Kind: kind,
			})
			off += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return &sessionIndex{}
		}
	}
	idx.size = off
	_ = writeIdxFile(s.idxPath(sessionID), idx)
	return idx
}

func writeIdxFile(path string, idx *sessionIndex) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(map[string]any{"v": 1, "size": idx.size}); err != nil {
		return err
	}
	for i := range idx.entries {
		if err := enc.Encode(idx.entries[i]); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) recordAppendLocked(sessionID string, ev Event, off int64, n int) {
	if s.mem == nil {
		s.mem = map[string]*sessionIndex{}
	}
	idx := s.mem[sessionID]
	if idx == nil || idx.size != off {
		s.mem[sessionID] = s.rebuildIndexLocked(sessionID)
		return
	}
	e := idxEntry{
		Seq:  int64(len(idx.entries)) + 1,
		Off:  off,
		N:    n,
		Type: ev.Type,
		Kind: payloadKind(ev),
	}
	if len(idx.entries) > 0 {
		e.Seq = idx.entries[len(idx.entries)-1].Seq + 1
	}
	idx.entries = append(idx.entries, e)
	idx.size = off + int64(n)
	_ = appendIdxLine(s.idxPath(sessionID), idx.size, e)
}

func appendIdxLine(path string, size int64, e idxEntry) error {
	// Cheap path: append the entry. Header size is stale until the next
	// rebuild; freshness is lastCover() against jsonl size.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	st, _ := f.Stat()
	enc := json.NewEncoder(f)
	if st.Size() == 0 {
		if err := enc.Encode(map[string]any{"v": 1, "size": size}); err != nil {
			return err
		}
	}
	return enc.Encode(e)
}

func (s *Store) dropIndexLocked(sessionID string) {
	if s.mem != nil {
		delete(s.mem, sessionID)
	}
	_ = os.Remove(s.idxPath(sessionID))
}

func (s *Store) readRangeLocked(sessionID string, off, end int64) ([]Event, error) {
	p := s.path(sessionID)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	if off > 0 {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return nil, err
		}
	}
	var r io.Reader = f
	if end > off {
		r = io.LimitReader(f, end-off)
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var out []Event
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, sc.Err()
}
