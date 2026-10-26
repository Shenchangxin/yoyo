// 给对话循环收紧连接器读取结果：条数、正文长度、日历时间范围，并标上 untrusted。
// Broker.Read 本身不截断。

package connector

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// searchCap 是普通查询最多返回的条数，值为 20。
	searchCap = 20
	// searchSnippet 是普通查询里正文字段的字节上限，值为 240。
	searchSnippet = 240
	// threadCap 是 thread: 或 id: 查询最多返回的条数，值为 20。
	threadCap = 20
	// threadBody 是线程查询里正文字段的字节上限，值为 12000。
	threadBody = 12000
	// titleCap 是 subject、title、location 的字节上限，值为 500。
	titleCap = 500
	// descCap 是 description 的字节上限，值为 2000。
	descCap = 2000
	// queryCap 是查询字符串的字节上限，值为 500。
	queryCap = 500
	// calendarSpan 是日历查询允许的最长时间跨度，为 366 天。
	calendarSpan = 366 * 24 * time.Hour
)

// rfc3339Tok 匹配查询文本里的 RFC3339 时间。
var rfc3339Tok = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`)

// BoundView 截断条数和字段，并把每条的 source 设为 untrusted。日历必须带两个时间且跨度不超过 366 天。查询超过 500 字节返回错误。
// kind 等于 KindCalendar 时先 ParseTimeRange，失败或跨度不符合就返回错误，再用 filterCalendar 过滤 items。
// query 的字节长度大于 queryCap 时返回错误。也传给 ParseTimeRange 和 threadQuery。
// items 在日历分支里传给 filterCalendar。条数大于 searchCap 或 threadCap 时截断。每条经 copyItem 后裁剪字段。
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

// threadQuery 在查询去掉空白并转成小写后，以 thread: 或 id: 开头时返回真。
// query 去掉空白并转成小写。以 thread: 或 id: 开头时返回真。
func threadQuery(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(q, "thread:") || strings.HasPrefix(q, "id:")
}

// ParseTimeRange 从查询里取出前两个能解析的 RFC3339 时间，按先后返回。不足两个则返回假。
// query 去掉空白后为空则返回假。否则用 rfc3339Tok 最多取 4 处，解析出至少两个时间才返回真。
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

// filterCalendar 保留开始时间落在 [min, max) 的条目。没有时间或解析失败的条目也保留。
// items 里每条用 start、date 或 time_min 解析时间。没有时间或解析失败的保留。时间等于 min 或晚于 min、且早于 max 的保留。
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

// parseLooseTime 按 RFC3339、带纳秒、无时区的日期时间或纯日期解析。
// raw 去掉空白后按 RFC3339、RFC3339Nano、2006-01-02T15:04:05 或 2006-01-02 解析。成功则返回 UTC 时间。
func parseLooseTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// clipField 在字段长于 n 个字节时截断并返回真。n 不大于 0 或未超长时返回假。
// row 取出 key 对应的值，超长时把截断后的值写回 row。
// key 用来读写 row 里的字段。
func clipField(row map[string]string, key string, n int) bool {
	v := row[key]
	if n <= 0 || len(v) <= n {
		return false
	}
	row[key] = v[:n]
	return true
}

// copyItem 复制一条字符串映射。
// it 的每个键值复制到新映射里。
func copyItem(it map[string]string) map[string]string {
	out := make(map[string]string, len(it)+1)
	for k, v := range it {
		out[k] = v
	}
	return out
}

// firstNonEmpty 返回第一段去掉空白后仍非空的字符串。
// ss 从前往后找，返回第一个去掉空白后非空的字符串。都没有时返回空字符串。
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
