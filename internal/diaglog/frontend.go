// 本文件把前端事件追加到 renderer.jsonl，并在进程日志里镜像一条 warn 或 error。
// 文件权限 0o600。级别为空时按 error 写。消息和堆栈先脱敏。

package diaglog

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

// FrontendEvent 是前端送来的一条事件。
type FrontendEvent struct {
	TS        string `json:"ts"`
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	Stack     string `json:"stack,omitempty"`
	Source    string `json:"source,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// rendererMu 串行化对 renderer.jsonl 的追加。
var rendererMu sync.Mutex

// AppendFrontend 把事件追加到 RendererPath。l 为 nil 或没有事件时返回 nil。
// TS 为空时用当前 UTC 的 RFC3339Nano。级别转成小写，空级别写成 error。消息和堆栈经 Redact。
// 每条都写入 component=frontend。级别是 warn 或 warning 时镜像成 Warn，其余镜像成 Error。
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
