package diaglog

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type ctxKey struct{}

type Fields struct {
	Component  string
	Cat        string
	SessionID  string
	TurnID     string
	TaskID     string
	RequestID  string
	TraceID    string
	SpanID     string
	Harness    string
	Model      string
}

func With(ctx context.Context, f Fields) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	prev := From(ctx)
	return context.WithValue(ctx, ctxKey{}, prev.merge(f))
}

func From(ctx context.Context) Fields {
	if ctx == nil {
		return Fields{}
	}
	f, _ := ctx.Value(ctxKey{}).(Fields)
	return f
}

func (a Fields) merge(b Fields) Fields {
	if b.Component != "" {
		a.Component = b.Component
	}
	if b.Cat != "" {
		a.Cat = b.Cat
	}
	if b.SessionID != "" {
		a.SessionID = b.SessionID
	}
	if b.TurnID != "" {
		a.TurnID = b.TurnID
	}
	if b.TaskID != "" {
		a.TaskID = b.TaskID
	}
	if b.RequestID != "" {
		a.RequestID = b.RequestID
	}
	if b.TraceID != "" {
		a.TraceID = b.TraceID
	}
	if b.SpanID != "" {
		a.SpanID = b.SpanID
	}
	if b.Harness != "" {
		a.Harness = b.Harness
	}
	if b.Model != "" {
		a.Model = b.Model
	}
	return a
}

type ctxHandler struct {
	next slog.Handler
	log  *Logger
}

func (h *ctxHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level >= slog.LevelInfo {
		return h.next.Enabled(ctx, level)
	}
	if h.log != nil {
		if h.log.debugOK(From(ctx).Cat) {
			return true
		}
		if m, _ := h.log.cats.Load().(map[string]bool); len(m) > 0 {
			return true
		}
	}
	return h.next.Enabled(ctx, level)
}

func (h *ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	f := From(ctx)
	cat := f.Cat
	seen := map[string]bool{}
	r.Attrs(func(a slog.Attr) bool {
		seen[a.Key] = true
		if a.Key == "cat" && a.Value.Kind() == slog.KindString {
			cat = a.Value.String()
		}
		return true
	})
	if r.Level < slog.LevelInfo && h.log != nil && !h.log.debugOK(cat) {
		return nil
	}
	add := func(k, v string) {
		if v == "" || seen[k] {
			return
		}
		r.AddAttrs(slog.String(k, v))
		seen[k] = true
	}
	add("component", f.Component)
	add("cat", f.Cat)
	add("session_id", f.SessionID)
	add("turn_id", f.TurnID)
	add("task_id", f.TaskID)
	add("request_id", f.RequestID)
	add("trace_id", f.TraceID)
	add("span_id", f.SpanID)
	add("harness", f.Harness)
	add("model", f.Model)
	return h.next.Handle(ctx, r)
}

func (h *ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ctxHandler{next: h.next.WithAttrs(attrs), log: h.log}
}

func (h *ctxHandler) WithGroup(name string) slog.Handler {
	return &ctxHandler{next: h.next.WithGroup(name), log: h.log}
}

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		if t, ok := a.Value.Any().(time.Time); ok {
			return slog.String("ts", t.UTC().Format(time.RFC3339Nano))
		}
	case slog.MessageKey:
		return slog.String("msg", Redact(a.Value.String()))
	case slog.LevelKey:
		return slog.String("level", strings.ToLower(a.Value.String()))
	}
	if LooksSecretKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(Redact(a.Value.String()))
	}
	return a
}

func replaceConsoleAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.MessageKey && a.Value.Kind() == slog.KindString {
		return slog.String(slog.MessageKey, Redact(a.Value.String()))
	}
	if LooksSecretKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(Redact(a.Value.String()))
	}
	return a
}
