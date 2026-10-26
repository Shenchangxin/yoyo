// 本文件把上下文里的字段补进 slog 记录，并在调试级别上按分类过滤。
// 低于 INFO 的记录在 Handle 里若 debugOK 为假会被丢掉。JSON 时间键改成 ts，级别转成小写。

package diaglog

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// ctxKey 是放进 context 的字段键。
type ctxKey struct{}

// Fields 是随 ctx 传递的日志字段。空字符串在 merge 时不覆盖已有值。
type Fields struct {
	Component string
	Cat       string
	SessionID string
	TurnID    string
	TaskID    string
	RequestID string
	TraceID   string
	SpanID    string
	Harness   string
	Model     string
}

// With 把 f 合并进 ctx 里已有的 Fields。ctx 为 nil 时用 Background。非空字段覆盖旧值。
func With(ctx context.Context, f Fields) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	prev := From(ctx)
	return context.WithValue(ctx, ctxKey{}, prev.merge(f))
}

// From 取出 ctx 里的 Fields。ctx 为 nil 或类型不对时返回零值。
func From(ctx context.Context) Fields {
	if ctx == nil {
		return Fields{}
	}
	f, _ := ctx.Value(ctxKey{}).(Fields)
	return f
}

// merge 用 b 里的非空字段覆盖 a，并返回副本。
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

// ctxHandler 在下一层处理器之外补上下文字段，并按分类决定调试行是否启用。
type ctxHandler struct {
	next slog.Handler
	log  *Logger
}

// Enabled 判断这条记录要不要进入 Handle。级别大于等于 INFO 时交给下一层。
// 更低的级别在 debugOK 命中当前分类时直接启用。分类表非空时也先启用，真正丢掉发生在 Handle。
// 都不是时仍交给下一层。
// ctx 在 level 大于等于 slog.LevelInfo 时传给 next.Enabled，更低时交给 From 取分类，最后仍可能传给 next.Enabled。
// level 大于等于 slog.LevelInfo 时交给 next.Enabled，否则看 debugOK 和分类表。
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

// Handle 把 ctx 里的字段补进记录，已有同名属性则不覆盖。记录自带的 cat 字符串会改写用于过滤的分类。
// 级别低于 INFO 且 debugOK 为假时直接返回 nil，不写下一层。
// r 先遍历属性记下已有键，字符串 cat 会改写分类。Level 低于 slog.LevelInfo 且 debugOK 为假时返回 nil，否则补上未见过的字段后传给 next.Handle。
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

// WithAttrs 返回带上这些属性的新处理器，仍使用同一个 Logger。
// attrs 传给 next.WithAttrs。
func (h *ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ctxHandler{next: h.next.WithAttrs(attrs), log: h.log}
}

// WithGroup 返回带上分组名的新处理器，仍使用同一个 Logger。
// name 传给 next.WithGroup。
func (h *ctxHandler) WithGroup(name string) slog.Handler {
	return &ctxHandler{next: h.next.WithGroup(name), log: h.log}
}

// replaceAttr 改写 JSON 属性。时间换成键 ts，值是 UTC 的 RFC3339Nano。消息键改成 msg 并脱敏。级别转成小写。
// LooksSecretKey 为真的键，值改成占位文本。字符串值再走 Redact。
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

// replaceConsoleAttr 只脱敏控制台文本，不改时间键和级别键的名字。
// 消息是字符串时替换成脱敏结果。秘密键的值改成占位文本。其他字符串值走 Redact。
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
