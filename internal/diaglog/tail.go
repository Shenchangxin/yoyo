// 本文件从日志文件尾部取出最近的 JSON 行。
// Tail 在 n 小于等于 0 时读 80 行。扫描缓冲的初始容量是 64*1024，单行上限是 1<<20。LastError 向前看 200 行。

package diaglog

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// Record 是从一行 JSON 里解出的字段。Raw 保留原文，不进入 JSON。
type Record struct {
	TS        string `json:"ts,omitempty"`
	Level     string `json:"level,omitempty"`
	Msg       string `json:"msg,omitempty"`
	Component string `json:"component,omitempty"`
	Cat       string `json:"cat,omitempty"`
	Process   string `json:"process,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	TurnID    string `json:"turn_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	SpanID    string `json:"span_id,omitempty"`
	Err       string `json:"err,omitempty"`
	CrashPath string `json:"crash_path,omitempty"`
	Raw       string `json:"-"`
}

// Filter 是 Tail 的筛选条件。空字段表示不限制。SessionID 区分大小写，其余三项不区分。
type Filter struct {
	Level     string
	Component string
	SessionID string
	Cat       string
}

// Tail 从当前日志文件取最后 n 条。l 为 nil 时返回 nil, nil。n 小于等于 0 时用 80。
func (l *Logger) Tail(n int, f Filter) ([]Record, error) {
	if l == nil {
		return nil, nil
	}
	if n <= 0 {
		n = 80
	}
	return TailFile(l.Path(), n, f)
}

// TailFile 扫描整个文件，只保留匹配的行，最后再截到 n 条。文件不存在时返回 nil, nil。
// 扫描器初始缓冲 64*1024，单行最大 1<<20。缓冲长度超过 n*4 时先收成最后 n 条。扫描出错时仍返回已经收下的行。
func TailFile(path string, n int, f Filter) ([]Record, error) {
	raw, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer raw.Close()
	var buf []Record
	sc := bufio.NewScanner(raw)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		rec := parseRecord(line)
		if !matchFilter(rec, f) {
			continue
		}
		buf = append(buf, rec)
		if len(buf) > n*4 {
			buf = buf[len(buf)-n:]
		}
	}
	if err := sc.Err(); err != nil {
		return buf, err
	}
	if len(buf) > n {
		buf = buf[len(buf)-n:]
	}
	return buf, nil
}

// parseRecord 把一行解成 Record，Raw 始终是原文。err 为空时再从对象里的 err 或 error 字符串补上。
func parseRecord(line string) Record {
	var rec Record
	_ = json.Unmarshal([]byte(line), &rec)
	rec.Raw = line
	if rec.Err == "" {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			if v, ok := m["err"].(string); ok {
				rec.Err = v
			} else if v, ok := m["error"].(string); ok {
				rec.Err = v
			}
		}
	}
	return rec
}

// matchFilter 按级别、组件、会话和分类筛选。级别、组件、分类不区分大小写。会话必须完全相等。空条件跳过。
func matchFilter(rec Record, f Filter) bool {
	if f.Level != "" && !strings.EqualFold(rec.Level, f.Level) {
		return false
	}
	if f.Component != "" && !strings.EqualFold(rec.Component, f.Component) {
		return false
	}
	if f.SessionID != "" && rec.SessionID != f.SessionID {
		return false
	}
	if f.Cat != "" && !strings.EqualFold(rec.Cat, f.Cat) {
		return false
	}
	return true
}

// LastError 从最后 200 行里找最新的一条 error，返回去掉首尾空白的消息和 Err，中间有一个空格。没有时返回空字符串。
func (l *Logger) LastError() string {
	recs, _ := l.Tail(200, Filter{})
	for i := len(recs) - 1; i >= 0; i-- {
		if strings.EqualFold(recs[i].Level, "error") {
			return strings.TrimSpace(recs[i].Msg + " " + recs[i].Err)
		}
	}
	return ""
}

// FormatRecords 把记录拼成多行文本。有 Raw 时用 Raw，否则依次写 TS、级别、组件和消息，中间用空格。
func FormatRecords(recs []Record) string {
	var b strings.Builder
	for i, r := range recs {
		if i > 0 {
			b.WriteByte('\n')
		}
		if r.Raw != "" {
			b.WriteString(r.Raw)
			continue
		}
		b.WriteString(r.TS)
		b.WriteByte(' ')
		b.WriteString(r.Level)
		b.WriteByte(' ')
		b.WriteString(r.Component)
		b.WriteByte(' ')
		b.WriteString(r.Msg)
	}
	return b.String()
}
