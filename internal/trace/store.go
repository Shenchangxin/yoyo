package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type EventType string

const (
	TypeSystem     EventType = "system"
	TypeUser       EventType = "user"
	TypeAssistant  EventType = "assistant"
	TypeReasoning  EventType = "reasoning"
	TypeToolCall   EventType = "tool_call"
	TypeToolResult EventType = "tool_result"
	TypeInject     EventType = "context_injection"
	TypeSubagent   EventType = "subagent"
	TypeCompact    EventType = "compaction"
	TypeError      EventType = "error"
	TypeEval       EventType = "eval"
	TypeEvolve     EventType = "evolve"
	TypeTurnEnd    EventType = "turn_end"
	TypeApproval   EventType = "approval"
	TypePlan       EventType = "plan"
	TypeFileChange EventType = "file_change"
	TypeAsk        EventType = "ask_user"
	TypePersonal   EventType = "personal"
)

func ItemKindOf(t EventType) string {
	switch t {
	case TypeUser:
		return "user"
	case TypeAssistant:
		return "assistant"
	case TypeReasoning:
		return "reasoning"
	case TypeToolCall:
		return "tool_call"
	case TypeToolResult:
		return "tool_result"
	case TypeInject:
		return "inject"
	case TypeSubagent:
		return "subagent"
	case TypeCompact:
		return "compact"
	case TypeTurnEnd:
		return "turn_end"
	case TypeApproval:
		return "approval"
	case TypePlan:
		return "plan"
	case TypeFileChange:
		return "file_change"
	case TypeAsk:
		return "ask_user"
	default:
		return string(t)
	}
}

type Event struct {
	TS               time.Time      `json:"ts"`
	Type             EventType      `json:"type"`
	Source           string         `json:"source"`
	SessionID        string         `json:"session_id"`
	HarnessSnapshot  string         `json:"harness_snapshot,omitempty"`
	ModelFingerprint string         `json:"model_fingerprint,omitempty"`
	TaskID           string         `json:"task_id,omitempty"`
	TurnID           string         `json:"turn_id,omitempty"`
	ItemKind         string         `json:"item_kind,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
	// Seq is an ephemeral live-bus or page cursor. It is never written to JSONL.
	Seq int64 `json:"seq,omitempty"`
}

type Store struct {
	Dir string
	mu  sync.Mutex
	mem map[string]*sessionIndex
}

func NewStore(dir string) *Store {
	return &Store{Dir: dir, mem: map[string]*sessionIndex{}}
}

func (s *Store) path(sessionID string) string {
	clean := filepath.FromSlash(sessionID)
	if strings.ContainsRune(clean, os.PathSeparator) {
		dir := filepath.Join(s.Dir, filepath.Dir(clean))
		_ = os.MkdirAll(dir, 0o755)
		return filepath.Join(s.Dir, clean+".jsonl")
	}
	return filepath.Join(s.Dir, sessionID+".jsonl")
}

func (s *Store) Append(ev Event) error {
	return s.append(ev, ev.Type == TypeUser)
}

// AppendDurable fsyncs the user row before returning. Crash resume depends on it.
func (s *Store) AppendDurable(ev Event) error {
	return s.append(ev, true)
}

func (s *Store) append(ev Event, durable bool) error {
	if ev.ItemKind == "" {
		ev.ItemKind = ItemKindOf(ev.Type)
	}
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	ev.Seq = 0
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	p := s.path(ev.SessionID)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	off := st.Size()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(ev); err != nil {
		return err
	}
	n, err := f.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if durable {
		if err := f.Sync(); err != nil {
			return err
		}
	}
	s.recordAppendLocked(ev.SessionID, ev, off, n)
	return nil
}

func (s *Store) Read(sessionID string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readRangeLocked(sessionID, 0, 0)
}

func (s *Store) Remove(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropIndexLocked(sessionID)
	err := os.Remove(s.path(sessionID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Store) Replace(sessionID string, evs []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path(sessionID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := range evs {
		ev := evs[i]
		ev.Seq = 0
		if ev.ItemKind == "" {
			ev.ItemKind = ItemKindOf(ev.Type)
		}
		if ev.TS.IsZero() {
			ev.TS = time.Now().UTC()
		}
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
	s.dropIndexLocked(sessionID)
	s.mem[sessionID] = s.rebuildIndexLocked(sessionID)
	return nil
}

func (s *Store) ListSessions() ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".jsonl" {
			ids = append(ids, name[:len(name)-len(".jsonl")])
		}
	}
	return ids, nil
}
