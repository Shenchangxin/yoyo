package trace

import (
	"bufio"
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
}

type Store struct {
	Dir string
	mu  sync.Mutex
}

func NewStore(dir string) *Store {
	return &Store{Dir: dir}
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
	if ev.ItemKind == "" {
		ev.ItemKind = ItemKindOf(ev.Type)
	}
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path(ev.SessionID), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(ev)
}

func (s *Store) Read(sessionID string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
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

func (s *Store) Remove(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
