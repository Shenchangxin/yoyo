package observe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Span is a local OpenTelemetry-shaped record. Export is file JSONL; set
// YOYO_OTEL_ENDPOINT later without changing callers.
type Span struct {
	TraceID   string         `json:"trace_id"`
	SpanID    string         `json:"span_id"`
	Name      string         `json:"name"`
	Start     time.Time      `json:"start"`
	End       time.Time      `json:"end,omitempty"`
	Status    string         `json:"status,omitempty"`
	Attrs     map[string]any `json:"attrs,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
}

type Tracer struct {
	mu   sync.Mutex
	path string
	buf  []Span
}

func Open(dir string) *Tracer {
	_ = os.MkdirAll(dir, 0o755)
	return &Tracer{path: filepath.Join(dir, "spans.jsonl")}
}

func (t *Tracer) Start(name, sessionID string, attrs map[string]any) Span {
	now := time.Now().UTC()
	return Span{
		TraceID:   now.Format("20060102T150405.000000000"),
		SpanID:    now.Format("150405.000000000"),
		Name:      name,
		Start:     now,
		Attrs:     attrs,
		SessionID: sessionID,
	}
}

func (t *Tracer) End(s Span, err error) {
	if t == nil {
		return
	}
	s.End = time.Now().UTC()
	s.Status = "ok"
	if err != nil {
		s.Status = "error"
		if s.Attrs == nil {
			s.Attrs = map[string]any{}
		}
		s.Attrs["error"] = err.Error()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, s)
	if len(t.buf) > 64 {
		t.flushLocked()
	}
	f, e := os.OpenFile(t.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if e != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	_ = enc.Encode(s)
}

func (t *Tracer) flushLocked() {
	if len(t.buf) > 256 {
		t.buf = t.buf[len(t.buf)-128:]
	}
}

func (t *Tracer) Recent(n int) []Span {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if n <= 0 || n > len(t.buf) {
		n = len(t.buf)
	}
	out := make([]Span, n)
	copy(out, t.buf[len(t.buf)-n:])
	return out
}

func (t *Tracer) Path() string {
	if t == nil {
		return ""
	}
	return t.path
}
