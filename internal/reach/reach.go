// Package reach 从画像里挑一句，准备一次只读的公开检索。
// 拥有：该搜哪一句、一条结果怎么变成开口材料、同一条链接不重复。
// 不拥有：发请求、开工单、改写检索里的数字。
package reach

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/initiative"
)

// Item 是一条公开检索结果。Line 是它要对上的那句画像。
type Item struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
	Line    string `json:"line,omitempty"`
}

// Log 记下下次检索时间和还没说出口的那一条。
type Log struct {
	Next    string `json:"next,omitempty"`
	Cursor  int    `json:"cursor,omitempty"`
	Pending *Item  `json:"pending,omitempty"`
}

// Lines 留下能拿去检索的短句。密钥、链接和过长的句子丢掉。
func Lines(texts []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, text := range texts {
		line := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
		if scrub(line) == "" {
			continue
		}
		n := utf8.RuneCountInString(line)
		if n < 2 || n > 40 {
			continue
		}
		if strings.Contains(strings.ToLower(line), "http") {
			continue
		}
		key := strings.ToLower(line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, line)
		if len(out) == 6 {
			break
		}
	}
	return out
}

// Due 在还没有下次时间，或时间已到时为真。
func Due(log Log, now time.Time) bool {
	if strings.TrimSpace(log.Next) == "" {
		return true
	}
	at, err := time.Parse(time.RFC3339, log.Next)
	if err != nil {
		return true
	}
	return !now.Before(at)
}

// Gap 是两次检索之间的间隔。三小时再加上当天里固定的几分钟，避免卡在整点。
func Gap(now time.Time) time.Duration {
	return 3*time.Hour + time.Duration(now.YearDay()%40)*time.Minute
}

// Plan 在没有待说的结果、画像非空、并且到点时，返回下一句。
func Plan(log Log, lines []string, now time.Time) (query string, ok bool) {
	if log.Pending != nil || len(lines) == 0 || !Due(log, now) {
		return "", false
	}
	i := log.Cursor % len(lines)
	if i < 0 {
		i = 0
	}
	return lines[i], true
}

// ID 用链接的摘要当稳定编号。
func ID(rawURL string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawURL)))
	return "reach:" + hex.EncodeToString(sum[:8])
}

// Emit 把还没见过的待说结果收成一条开口材料。
func Emit(item *Item, seen map[string]bool) (initiative.Fact, bool) {
	if item == nil || strings.TrimSpace(item.Title) == "" || !publicURL(item.URL) {
		return initiative.Fact{}, false
	}
	if seen[ID(item.URL)] {
		return initiative.Fact{}, false
	}
	return fact(*item), true
}

// Choose 取第一条有标题、有公开链接、并且还没见过的结果。
func Choose(items []Item, seen map[string]bool) *Item {
	for _, item := range items {
		if strings.TrimSpace(item.Title) == "" || !publicURL(item.URL) {
			continue
		}
		if seen[ID(item.URL)] {
			continue
		}
		copy := item
		copy.Title = clip(copy.Title, 80)
		copy.Snippet = clip(scrubKeep(copy.Snippet), 180)
		copy.URL = strings.TrimSpace(copy.URL)
		return &copy
	}
	return nil
}

// ParseEnvelope 读取 agent-reach 的一行 JSON。ok 不为真时返回空。
func ParseEnvelope(raw string) []Item {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	line := raw
	if i := strings.LastIndex(raw, "\n"); i >= 0 {
		line = strings.TrimSpace(raw[i+1:])
	}
	var env struct {
		OK    bool `json:"ok"`
		Items []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Text  string `json:"text"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(line), &env) != nil || !env.OK {
		return nil
	}
	var out []Item
	for _, item := range env.Items {
		out = append(out, Item{Title: item.Title, URL: item.URL, Snippet: item.Text})
	}
	return out
}

// ParseHost 读取宿主检索格式化后的标题、链接和摘要。
func ParseHost(text string) []Item {
	var out []Item
	var cur *Item
	flush := func() {
		if cur != nil && strings.TrimSpace(cur.Title) != "" {
			out = append(out, *cur)
		}
		cur = nil
	}
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if n, title, ok := numbered(trim); ok && n > 0 {
			flush()
			cur = &Item{Title: title}
			continue
		}
		if cur == nil {
			continue
		}
		if cur.URL == "" && publicURL(trim) {
			cur.URL = trim
			continue
		}
		if cur.Snippet == "" {
			cur.Snippet = trim
		}
	}
	flush()
	return out
}

func fact(item Item) initiative.Fact {
	var b strings.Builder
	if line := strings.TrimSpace(item.Line); line != "" {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString(strings.TrimSpace(item.URL))
	if snip := strings.TrimSpace(item.Snippet); snip != "" {
		b.WriteByte('\n')
		b.WriteString(snip)
	}
	return initiative.Fact{
		ID: ID(item.URL), Code: "reach", Class: "reach", Name: item.Title,
		Detail: b.String(), State: "match", Sig: strings.TrimSpace(item.URL), Rank: 2, Eligible: true,
	}
}

func publicURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "https://") {
		return false
	}
	if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "0.0.0.0") {
		return false
	}
	return true
}

func numbered(line string) (int, string, bool) {
	i := 0
	n := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		n = n*10 + int(line[i]-'0')
		i++
	}
	if i == 0 || i+2 > len(line) || line[i] != '.' || line[i+1] != ' ' {
		return 0, "", false
	}
	return n, strings.TrimSpace(line[i+2:]), true
}

func scrub(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	lower := strings.ToLower(text)
	for _, word := range []string{"secret", "token", "password", "cookie", "credential", "api_key", "apikey"} {
		if strings.Contains(lower, word) {
			return ""
		}
	}
	return strings.TrimSpace(text)
}

func scrubKeep(text string) string {
	if scrub(text) == "" && strings.TrimSpace(text) != "" {
		return ""
	}
	return strings.TrimSpace(text)
}

func clip(text string, n int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return string([]rune(text)[:n])
}
