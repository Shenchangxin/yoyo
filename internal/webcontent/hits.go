// 本文件把搜索命中格式化成有限的几行。条数上限是 8，摘要上限是 280 个 rune。

package webcontent

import (
	"fmt"
	"strings"
)

// maxHits 是 FormatHits、parseResultA 和 parseLiteLinks 保留的命中条数，值是 8。
const maxHits = 8

// Hit 是一条公开网页搜索结果。
type Hit struct {
	// Title 是标题。
	Title string
	// URL 是链接。
	URL string
	// Snippet 是摘要。FormatHits 超过 280 个 rune 就截断并加 …。
	Snippet string
}

// FormatHits 把命中排成带编号的标题、链接和摘要。标题和链接都空的跳过。最多 maxHits 条。没有可用行时返回空字符串。
// hits 逐条取 Title、URL、Snippet。标题和链接去掉空白后都空则跳过。留下的行数达到 maxHits 就停止。
func FormatHits(hits []Hit) string {
	var lines []string
	for _, h := range hits {
		title := strings.TrimSpace(h.Title)
		link := strings.TrimSpace(h.URL)
		snip := strings.TrimSpace(h.Snippet)
		if title == "" && link == "" {
			continue
		}
		if len(lines) >= maxHits {
			break
		}
		n := len(lines) + 1
		line := fmt.Sprintf("%d. %s", n, title)
		if title == "" {
			line = fmt.Sprintf("%d. %s", n, link)
		}
		if link != "" && title != "" {
			line += "\n   " + link
		} else if link != "" && title == "" {
			// 链接已经写在标题那一行。
		}
		if snip != "" {
			if len([]rune(snip)) > 280 {
				snip = string([]rune(snip)[:280]) + "…"
			}
			line += "\n   " + snip
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// Useful 判断文本像不像真的搜索结果。空白返回 false。含 <html 且不含 http 返回 false。
// http:// 与 https:// 合计至少 1 次且长度至少 24 时返回 true。否则长度至少 80 才返回 true。
func Useful(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "<html") && !strings.Contains(s, "http") {
		return false
	}
	if strings.Count(lower, "http://")+strings.Count(lower, "https://") >= 1 && len(s) >= 24 {
		return true
	}
	return len(s) >= 80
}
