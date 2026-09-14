package trace

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
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
)

type Event struct {
	TS               time.Time      `json:"ts"`
	Type             EventType      `json:"type"`
	Source           string         `json:"source"`
	SessionID        string         `json:"session_id"`
	HarnessSnapshot  string         `json:"harness_snapshot,omitempty"`
	ModelFingerprint string         `json:"model_fingerprint,omitempty"`
	TaskID           string         `json:"task_id,omitempty"`
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
	return filepath.Join(s.Dir, sessionID+".jsonl")
}

func (s *Store) Append(ev Event) error {
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
