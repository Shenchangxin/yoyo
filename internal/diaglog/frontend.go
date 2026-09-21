package diaglog

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

type FrontendEvent struct {
	TS        string `json:"ts"`
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Stack     string `json:"stack,omitempty"`
	Source    string `json:"source,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

var rendererMu sync.Mutex

func (l *Logger) AppendFrontend(evs []FrontendEvent) error {
	if l == nil || len(evs) == 0 {
		return nil
	}
	path := l.RendererPath()
	rendererMu.Lock()
	defer rendererMu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, ev := range evs {
		if ev.TS == "" {
			ev.TS = time.Now().UTC().Format(time.RFC3339Nano)
		}
		ev.Level = strings.ToLower(ev.Level)
		if ev.Level == "" {
			ev.Level = "error"
		}
		ev.Msg = Redact(ev.Msg)
		ev.Stack = Redact(ev.Stack)
		rec := map[string]any{
			"ts": ev.TS, "level": ev.Level, "msg": ev.Msg,
			"component": "frontend", "process": string(l.opt.Process),
		}
		if ev.Stack != "" {
			rec["stack"] = ev.Stack
		}
		if ev.Source != "" {
			rec["source"] = ev.Source
		}
		if ev.SessionID != "" {
			rec["session_id"] = ev.SessionID
		}
		if err := enc.Encode(rec); err != nil {
			return err
		}
		lvl := "error"
		if ev.Level == "warn" || ev.Level == "warning" {
			lvl = "warn"
		}
		args := []any{"component", "frontend", "source", ev.Source}
		if ev.SessionID != "" {
			args = append(args, "session_id", ev.SessionID)
		}
		if lvl == "warn" {
			l.slogger.Warn(ev.Msg, args...)
		} else {
			l.slogger.Error(ev.Msg, args...)
		}
	}
	return nil
}
