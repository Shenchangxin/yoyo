// Package observe 把跨度追加写到调用方目录里的 spans.jsonl。
// Open 从 YOYO_OTEL_ENDPOINT 读取导出地址。环境变量未设置时端点是空字符串，End 不调用 exportOTLP，OTLP 关闭。
// 内存里的跨度超过 256 条时只留最后 128 条。
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

// ctxKey 是放进 context 的跨度键，避免和别的包撞名。
type ctxKey struct{}

// Span 是一条跨度。End 在结束前是零值。Status 由 Tracer.End 写成 ok 或 error。
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

// Tracer 把结束的跨度追加到 spans.jsonl，并在内存里留一段最近记录。
// endpoint 来自 Open 读到的 YOYO_OTEL_ENDPOINT，或后来的 SetEndpoint。空字符串表示不导出。
type Tracer struct {
	mu       sync.Mutex
	path     string
	buf      []Span
	endpoint string
}

// Open 创建 dir，权限 0o755。路径是 dir/spans.jsonl。创建目录的错误被忽略，文件要到 End 才打开。
// endpoint 设为 YOYO_OTEL_ENDPOINT 的当前值。变量未设置时是空字符串，OTLP 不会发出。
func Open(dir string) *Tracer {
	_ = os.MkdirAll(dir, 0o755)
	t := &Tracer{path: filepath.Join(dir, "spans.jsonl"), endpoint: os.Getenv("YOYO_OTEL_ENDPOINT")}
	return t
}

// SetEndpoint 改导出地址。url 为空时之后的 End 不再导出。t 为 nil 时直接返回。
func (t *Tracer) SetEndpoint(url string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.endpoint = url
	t.mu.Unlock()
}

// ContextWithSpan 把 sp 放进 ctx。ctx 为 nil 时用 Background。
func ContextWithSpan(ctx context.Context, sp Span) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, sp)
}

// SpanFrom 取出 ctx 里的跨度。ctx 为 nil 或没有跨度时返回零值。
func SpanFrom(ctx context.Context) Span {
	if ctx == nil {
		return Span{}
	}
	sp, _ := ctx.Value(ctxKey{}).(Span)
	return sp
}

// Start 开始一条没有父级的跨度，会话 id 用 sessionID。
func (t *Tracer) Start(name, sessionID string, attrs map[string]any) Span {
	return t.StartChild(Span{}, name, sessionID, attrs)
}

// StartContext 以 ctx 里的跨度为父级。会话和回合先取 diaglog.From，父级上的非空值再盖过它们。
// 返回的新 ctx 带上这条子跨度。回合 id 在 StartChild 之后若仍非空会写上，这样只有日志字段里有回合时也能留下。
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

// StartChild 生成子跨度，不写盘。TraceID 沿用父级，父级没有时用 diaglog.NewTraceID。SpanID 总是新的 diaglog.NewSpanID。
// ParentSpanID 是父级的 SpanID。属性先经 redactAttrs。sessionID 为空时改用父级的会话。TurnID 先抄父级。
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

// End 记下结束时间。err 为 nil 时 Status 是 ok。否则 Status 是 error，并把脱敏后的错误文本放进属性 error。
// 属性再脱敏一次。内存超过 256 条时只留最后 128 条。然后向 spans.jsonl 追加一行，权限 0o644。打开文件失败就返回，此时导出也不发生。
// endpoint 非空时在新的 goroutine 里调用 exportOTLP。endpoint 为空则不导出。t 为 nil 时直接返回。
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

// Recent 返回内存里最后 n 条，不读文件。n 小于等于 0 或大于缓冲长度时返回全部。t 为 nil 时返回 nil。
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

// Path 返回 spans.jsonl 的路径。t 为 nil 时返回空字符串。
func (t *Tracer) Path() string {
	if t == nil {
		return ""
	}
	return t.path
}

// redactAttrs 复制属性。键被 diaglog.LooksSecretKey 判为秘密时，值改成 [REDACTED]。字符串值再走 diaglog.Redact。其他类型原样放入。
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
