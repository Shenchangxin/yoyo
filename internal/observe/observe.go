package observe

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

type ctxKey struct{}

type Span struct {
	TraceID      string         `json:"trace_id"`
	SpanID       string         `json:"span_id"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	Name         string         `json:"name"`
	Start        time.Time      `json:"start"`
	End          time.Time      `json:"end,omitempty"`
	Status       string         `json:"status,omitempty"`
	Attrs        map[string]any `json:"attrs,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
	TurnID       string         `json:"turn_id,omitempty"`
}

type Tracer struct {
	mu       sync.Mutex
	path     string
	buf      []Span
	endpoint string
}

func Open(dir string) *Tracer {
	_ = os.MkdirAll(dir, 0o755)
	t := &Tracer{path: filepath.Join(dir, "spans.jsonl"), endpoint: os.Getenv("YOYO_OTEL_ENDPOINT")}
	return t
}

func (t *Tracer) SetEndpoint(url string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.endpoint = url
	t.mu.Unlock()
}

func ContextWithSpan(ctx context.Context, sp Span) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, sp)
}

func SpanFrom(ctx context.Context) Span {
	if ctx == nil {
		return Span{}
	}
	sp, _ := ctx.Value(ctxKey{}).(Span)
	return sp
}

func (t *Tracer) Start(name, sessionID string, attrs map[string]any) Span {
	return t.StartChild(Span{}, name, sessionID, attrs)
}

func (t *Tracer) StartContext(ctx context.Context, name string, attrs map[string]any) (Span, context.Context) {
	parent := SpanFrom(ctx)
	sessionID := ""
	turnID := ""
	f := diaglog.From(ctx)
	if f.SessionID != "" {
		sessionID = f.SessionID
	}
	if f.TurnID != "" {
		turnID = f.TurnID
	}
	if parent.SessionID != "" {
		sessionID = parent.SessionID
	}
	if parent.TurnID != "" {
		turnID = parent.TurnID
	}
	sp := t.StartChild(parent, name, sessionID, attrs)
	if turnID != "" {
		sp.TurnID = turnID
	}
	return sp, ContextWithSpan(ctx, sp)
}

func (t *Tracer) StartChild(parent Span, name, sessionID string, attrs map[string]any) Span {
	now := time.Now().UTC()
	traceID := parent.TraceID
	if traceID == "" {
		traceID = diaglog.NewTraceID()
	}
	sp := Span{
		TraceID:      traceID,
		SpanID:       diaglog.NewSpanID(),
		ParentSpanID: parent.SpanID,
		Name:         name,
		Start:        now,
		Attrs:        redactAttrs(attrs),
		SessionID:    sessionID,
		TurnID:       parent.TurnID,
	}
	if sessionID == "" {
		sp.SessionID = parent.SessionID
	}
	return sp
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
		s.Attrs["error"] = diaglog.Redact(err.Error())
	}
	s.Attrs = redactAttrs(s.Attrs)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, s)
	if len(t.buf) > 256 {
		t.buf = t.buf[len(t.buf)-128:]
	}
	f, e := os.OpenFile(t.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if e != nil {
		return
	}
	enc := json.NewEncoder(f)
	_ = enc.Encode(s)
	_ = f.Close()
	if t.endpoint != "" {
		go exportOTLP(t.endpoint, s)
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

func redactAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		if diaglog.LooksSecretKey(k) {
			out[k] = "[REDACTED]"
			continue
		}
		switch s := v.(type) {
		case string:
			out[k] = diaglog.Redact(s)
		default:
			out[k] = v
		}
	}
	return out
}
