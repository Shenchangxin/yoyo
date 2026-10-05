package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// OutlineTurn is one operator prompt in the session index. The renderer
// builds a jump list from this, not from mounted DOM — ChatGPT's native TOC
// broke on virtualized long threads for that reason.
type OutlineTurn struct {
	Seq   int64  `json:"seq"`
	Title string `json:"title"`
	ID    string `json:"id,omitempty"`
}

var mentionTok = regexp.MustCompile(`@(?:file|folder|skill):\S+|@harness\b`)

const outlineTitleRunes = 42

// OutlineTitle is the one-line jump label: mentions stripped, whitespace
// collapsed, 42 runes — the same length as session titles.
func OutlineTitle(text string) string {
	s := mentionTok.ReplaceAllString(text, " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) > outlineTitleRunes {
		return string(r[:outlineTitleRunes]) + "…"
	}
	return s
}

func isSteerUser(ev Event) bool {
	if ev.Type != TypeUser {
		return false
	}
	if ev.Source == "steer" {
		return true
	}
	name := payloadString(ev, "name")
	kind := payloadString(ev, "kind")
	return name == "steer" || kind == "steer"
}

func payloadString(ev Event, keys ...string) string {
	if ev.Payload == nil {
		return ""
	}
	for _, k := range keys {
		if s, ok := ev.Payload[k].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// OutlineTurns walks the sidecar index and reads only TypeUser lines.
// Steer injects stay out of the directory. Callers never dump the JSONL.
func (s *Store) OutlineTurns(sessionID string) ([]OutlineTurn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(sessionID)
	p := s.path(sessionID)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []OutlineTurn{}, nil
		}
		return nil, err
	}
	defer f.Close()
	out := make([]OutlineTurn, 0, 16)
	buf := make([]byte, 0, 4096)
	for _, e := range idx.entries {
		if e.Type != TypeUser || e.N <= 0 {
			continue
		}
		if cap(buf) < e.N {
			buf = make([]byte, e.N)
		} else {
			buf = buf[:e.N]
		}
		n, err := f.ReadAt(buf, e.Off)
		if n == 0 {
			if err != nil {
				continue
			}
			continue
		}
		var ev Event
		if json.Unmarshal(bytes.TrimSpace(buf[:n]), &ev) != nil {
			continue
		}
		if isSteerUser(ev) {
			continue
		}
		out = append(out, OutlineTurn{
			Seq:   e.Seq,
			Title: OutlineTitle(payloadString(ev, "text")),
			ID:    payloadString(ev, "id"),
		})
	}
	return out, nil
}

// PageAroundUser returns a renderer window of `turns` operator turns that
// contains userSeq. The target sits at the top of the window when enough
// later turns follow; near EOF the window pads backward so a jump is never
// a one-row stub.
func (s *Store) PageAroundUser(sessionID string, userSeq int64, turns int) (Page, error) {
	if turns <= 0 {
		turns = DefaultPageTurns
	}
	if userSeq <= 0 {
		return s.PageTurns(sessionID, 0, turns)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(sessionID)
	if len(idx.entries) == 0 {
		return Page{}, nil
	}
	users := make([]int, 0, 16)
	for i, e := range idx.entries {
		if e.Type == TypeUser {
			users = append(users, i)
		}
	}
	pos := -1
	for i, ui := range users {
		if idx.entries[ui].Seq == userSeq {
			pos = i
			break
		}
	}
	if pos < 0 {
		return Page{}, fmt.Errorf("turn %d not in session", userSeq)
	}
	start := pos
	end := start + turns
	if end > len(users) {
		end = len(users)
	}
	if end-start < turns {
		start = end - turns
		if start < 0 {
			start = 0
		}
	}
	startEntry := users[start]
	var endOff int64
	if end < len(users) {
		endOff = idx.entries[users[end]].Off
	} else {
		endOff = idx.size
	}
	off := idx.entries[startEntry].Off
	evs, err := s.readRangeLocked(sessionID, off, endOff)
	if err != nil {
		return Page{}, err
	}
	endEntry := users[end-1]
	last := idx.entries[endEntry]
	if end < len(users) {
		last = idx.entries[users[end]-1]
	} else {
		last = idx.entries[len(idx.entries)-1]
	}
	span := 0
	for i := startEntry; i < len(idx.entries); i++ {
		if idx.entries[i].Off+int64(idx.entries[i].N) > endOff {
			break
		}
		span++
	}
	if len(evs) == span {
		for i := range evs {
			evs[i].Seq = idx.entries[startEntry+i].Seq
		}
	}
	return Page{
		Events:  evs,
		HeadSeq: idx.entries[startEntry].Seq,
		TailSeq: last.Seq,
		Older:   startEntry > 0,
	}, nil
}
