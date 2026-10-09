package connector

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	searchCap     = 20
	searchSnippet = 240
	threadCap     = 20
	threadBody    = 12000
	titleCap      = 500
	descCap       = 2000
	queryCap      = 500
	calendarSpan  = 366 * 24 * time.Hour
)

var rfc3339Tok = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`)

// BoundView shapes connector_read for the chat loop: bounded search/thread,
// calendar range, and an untrusted source label. Broker.Read stays unbounded
// for workers that already cap their own scans.
func BoundView(kind Kind, query string, items []map[string]string) ([]map[string]string, error) {
	if len(query) > queryCap {
		return nil, fmt.Errorf("connector: query must be at most %d characters", queryCap)
	}
	if kind == KindCalendar {
		min, max, ok := ParseTimeRange(query)
		if !ok {
			return nil, fmt.Errorf("connector: calendar read needs RFC3339 timeMin and timeMax spanning at most 366 days")
		}
		if !max.After(min) {
			return nil, fmt.Errorf("connector: calendar timeMax must be after timeMin")
		}
		if max.Sub(min) > calendarSpan {
			return nil, fmt.Errorf("connector: calendar range must be at most 366 days")
		}
		items = filterCalendar(items, min, max)
	}
	thread := threadQuery(query)
	limit := searchCap
	bodyLimit := searchSnippet
	if thread {
		limit = threadCap
		bodyLimit = threadBody
	}
	truncated := false
	if len(items) > limit {
		items = items[:limit]
		truncated = true
	}
	out := make([]map[string]string, 0, len(items))
	for _, it := range items {
		row := copyItem(it)
		row["source"] = "untrusted"
		clipField(row, "subject", titleCap)
		clipField(row, "title", titleCap)
		clipField(row, "location", titleCap)
		if clipField(row, "description", descCap) {
			row["truncated"] = "true"
		}
		bodyKey := "body"
		if strings.TrimSpace(row["body"]) == "" && row["snippet"] != "" {
			bodyKey = "snippet"
		}
		if clipField(row, bodyKey, bodyLimit) {
			row["truncated"] = "true"
		}
		if truncated {
			row["truncated"] = "true"
		}
		out = append(out, row)
	}
	return out, nil
}

func threadQuery(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(q, "thread:") || strings.HasPrefix(q, "id:")
}

func ParseTimeRange(query string) (time.Time, time.Time, bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return time.Time{}, time.Time{}, false
	}
	var stamps []time.Time
	for _, raw := range rfc3339Tok.FindAllString(q, 4) {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, raw)
		}
		if err == nil {
			stamps = append(stamps, t.UTC())
		}
	}
	if len(stamps) < 2 {
		return time.Time{}, time.Time{}, false
	}
	min, max := stamps[0], stamps[1]
	if max.Before(min) {
		min, max = max, min
	}
	return min, max, true
}

func filterCalendar(items []map[string]string, min, max time.Time) []map[string]string {
	var out []map[string]string
	for _, it := range items {
		raw := firstNonEmpty(it["start"], it["date"], it["time_min"])
		if raw == "" {
			out = append(out, it)
			continue
		}
		t, ok := parseLooseTime(raw)
		if !ok {
			out = append(out, it)
			continue
		}
		if (t.Equal(min) || t.After(min)) && t.Before(max) {
			out = append(out, it)
		}
	}
	return out
}

func parseLooseTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func clipField(row map[string]string, key string, n int) bool {
	v := row[key]
	if n <= 0 || len(v) <= n {
		return false
	}
	row[key] = v[:n]
	return true
}

func copyItem(it map[string]string) map[string]string {
	out := make(map[string]string, len(it)+1)
	for k, v := range it {
		out[k] = v
	}
	return out
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
