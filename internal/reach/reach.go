// Package reach 准备一次只读的公开检索：画像相关的网页，以及新闻和社交热榜。
// 拥有：这一轮该问哪个渠道、一条结果怎么变成开口材料、同一条链接不重复。
// 不拥有：发请求、开工单、改写检索里的数字。
package reach

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/initiative"
)

// Item 是一条公开检索结果。Line 是它要对上的那句画像。Note 是写给人看的便条，有它时不再把标题和摘要原样倒出来。
type Item struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
	Line    string `json:"line,omitempty"`
	Note    string `json:"note,omitempty"`
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

// Probe 是一轮只读检索。Channel 和 Action 对应 agent-reach 的渠道动作。
// Query 只在 search 时使用。Line 是它对上的画像，热榜可以空着。
// Fallback 是 agent-reach 没有结果时交给其他公开检索的词。
type Probe struct {
	Channel  string
	Action   string
	Query    string
	Line     string
	Fallback string
}

// Next 到点且没有待说结果时，轮流查画像、新闻和热榜。没有画像时仍然查热榜和新闻。
func Next(log Log, lines []string, now time.Time) (Probe, bool) {
	if log.Pending != nil || !Due(log, now) {
		return Probe{}, false
	}
	slot := log.Cursor % 5
	if slot < 0 {
		slot = 0
	}
	line := ""
	if len(lines) > 0 {
		i := log.Cursor % len(lines)
		if i < 0 {
			i = 0
		}
		line = lines[i]
	}
	switch slot {
	case 1:
		return Probe{Channel: "bilibili", Action: "hot", Fallback: "B站 热门"}, true
	case 2:
		return Probe{Channel: "v2ex", Action: "hot", Fallback: "V2EX 今日热门"}, true
	case 4:
		q := line
		if q == "" {
			q = "热点"
		}
		return Probe{Channel: "twitter", Action: "search", Query: q, Line: line, Fallback: "社交媒体 热点 " + q}, true
	default:
		q := line
		if slot == 3 {
			if line != "" {
				q = line + " 新闻"
			} else {
				q = "今日新闻"
			}
		}
		if q == "" {
			q = "今日新闻"
		}
		return Probe{Channel: "web", Action: "search", Query: q, Line: line, Fallback: q}, true
	}
}

// Command 是 agent-reach 的参数。社交搜索限定公开网页，不打开浏览器。
func Command(p Probe) []string {
	args := []string{p.Channel, p.Action}
	switch p.Action {
	case "search":
		if q := strings.TrimSpace(p.Query); q != "" {
			args = append(args, q, "--limit", "5")
		}
	case "hot":
		args = append(args, "--limit", "5")
	}
	if p.Channel == "twitter" || p.Channel == "reddit" {
		args = append(args, "--backend", "public-web")
	}
	args = append(args, "--timeout", "20")
	return args
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

var exaHeader = regexp.MustCompile(`^(Title|URL|Published|Author):\s*(.*)$`)

// ParseExa 读取 Exa 的 Title / URL 分块。没有公开链接的块丢掉。
func ParseExa(text string) []Item {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var out []Item
	chunks := strings.Split(text, "\nTitle: ")
	for i, block := range chunks {
		if i > 0 {
			block = "Title: " + block
		}
		block = strings.TrimSpace(block)
		if !strings.HasPrefix(block, "Title:") {
			continue
		}
		fields := map[string]string{}
		var body []string
		header := true
		for _, line := range strings.Split(block, "\n") {
			if header {
				if m := exaHeader.FindStringSubmatch(line); m != nil {
					fields[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
					continue
				}
				header = false
			}
			trim := strings.TrimSpace(line)
			if trim == "" || trim == "Highlights:" || trim == "Text:" {
				continue
			}
			body = append(body, trim)
		}
		title := fields["title"]
		link := fields["url"]
		if title == "" || !publicURL(link) {
			continue
		}
		out = append(out, Item{Title: title, URL: link, Snippet: strings.Join(body, " ")})
	}
	return out
}

func fact(item Item) initiative.Fact {
	detail := strings.TrimSpace(item.Note)
	if detail == "" {
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
		detail = b.String()
	}
	return initiative.Fact{
		ID: ID(item.URL), Code: "reach", Class: "reach", Name: item.Title,
		Detail: detail, State: "match", Sig: strings.TrimSpace(item.URL), Rank: 2, Eligible: true,
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
