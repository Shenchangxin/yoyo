package diaglog

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

type Record struct {
	TS         string `json:"ts,omitempty"`
	Level      string `json:"level,omitempty"`
	Msg        string `json:"msg,omitempty"`
	Component  string `json:"component,omitempty"`
	Cat        string `json:"cat,omitempty"`
	Process    string `json:"process,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	TurnID     string `json:"turn_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	SpanID     string `json:"span_id,omitempty"`
	Err        string `json:"err,omitempty"`
	CrashPath  string `json:"crash_path,omitempty"`
	Raw        string `json:"-"`
}

type Filter struct {
	Level     string
	Component string
	SessionID string
	Cat       string
}

func (l *Logger) Tail(n int, f Filter) ([]Record, error) {
	if l == nil {
		return nil, nil
	}
	if n <= 0 {
		n = 80
	}
	return TailFile(l.Path(), n, f)
}

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

func (l *Logger) LastError() string {
	recs, _ := l.Tail(200, Filter{})
	for i := len(recs) - 1; i >= 0; i-- {
		if strings.EqualFold(recs[i].Level, "error") {
			return strings.TrimSpace(recs[i].Msg + " " + recs[i].Err)
		}
	}
	return ""
}

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
