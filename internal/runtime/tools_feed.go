// 把 Atom 或 RSS 收成短列表，供 web_fetch 在去掉标签之前返回。正文上限是 feedInlineBudget 个 rune。406 时把 https 改成 http 的重试也在这里。

package runtime

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// reFeedEntry、reFeedItem、reFeedTotal、reFeedPDFHref、reFeedAbsHref、reFeedName、reInnerTags 是解析源条目的正则。
var (
	// reFeedEntry 匹配 entry 元素及其内部文本。
	reFeedEntry = regexp.MustCompile(`(?is)<entry\b[^>]*>(.*?)</entry>`)
	// reFeedItem 匹配 item 元素及其内部文本。
	reFeedItem = regexp.MustCompile(`(?is)<item\b[^>]*>(.*?)</item>`)
	// reFeedTotal 匹配 totalResults 里的数字。
	reFeedTotal = regexp.MustCompile(`(?is)<(?:[a-z0-9]+:)?totalResults\b[^>]*>(\d+)`)
	// reFeedPDFHref 匹配 title 为 pdf 的 link 的 href，两种属性顺序都认。
	reFeedPDFHref = regexp.MustCompile(`(?is)<link[^>]*title=["']pdf["'][^>]*href=["']([^"']+)["']|<link[^>]*href=["']([^"']+)["'][^>]*title=["']pdf["']`)
	// reFeedAbsHref 匹配 rel 为 alternate 的 link 的 href。
	reFeedAbsHref = regexp.MustCompile(`(?is)<link[^>]*rel=["']alternate["'][^>]*href=["']([^"']+)["']`)
	// reFeedName 匹配 name 元素的内部文本。
	reFeedName = regexp.MustCompile(`(?is)<(?:[a-z0-9]+:)?name\b[^>]*>(.*?)</(?:[a-z0-9]+:)?name>`)
	// reInnerTags 匹配一对尖括号标签。
	reInnerTags = regexp.MustCompile(`<[^>]+>`)
)

// retryHTTPOn406 仅在状态为 406 且方案是 https 时返回同一 URL 的 http 副本。其余情况返回 nil。
func retryHTTPOn406(u *url.URL, status int) *url.URL {
	if u == nil || status != 406 || !strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	alt := *u
	alt.Scheme = "http"
	return &alt
}

// isXMLFeed 在内容类型含 json 时返回假。含 xml、atom 或 rss 时返回真。否则看正文是否以 <?xml、<feed 或 <rss 开头。
func isXMLFeed(contentType, body string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "json") {
		return false
	}
	if strings.Contains(ct, "xml") || strings.Contains(ct, "atom") || strings.Contains(ct, "rss") {
		return true
	}
	trim := strings.TrimSpace(body)
	return strings.HasPrefix(trim, "<?xml") || strings.HasPrefix(trim, "<feed") || strings.HasPrefix(trim, "<rss")
}

// feedInlineBudget 是 7000。压缩后的源按 rune 数停在这个数以内，避免再被收成 recall 摘要。
const feedInlineBudget = 7000

// formatFeed 先取 entry，没有再取 item，最多 40 条。没有条目则返回假。按 20/160、16/80、12/0、8/0 这四档缩短，直到 rune 数不超过 feedInlineBudget。仍超出则 clipRunes 截到该预算。
func formatFeed(body string) (string, bool) {
	entries := reFeedEntry.FindAllStringSubmatch(body, 40)
	if len(entries) == 0 {
		entries = reFeedItem.FindAllStringSubmatch(body, 40)
	}
	if len(entries) == 0 {
		return "", false
	}
	total := ""
	if m := reFeedTotal.FindStringSubmatch(body); len(m) > 1 {
		total = m[1]
	}
	cfgs := []struct{ limit, summary int }{
		{20, 160},
		{16, 80},
		{12, 0},
		{8, 0},
	}
	var last string
	for _, cfg := range cfgs {
		last = renderFeed(entries, total, cfg.limit, cfg.summary)
		if utf8.RuneCountInString(last) <= feedInlineBudget {
			return last, true
		}
	}
	return clipRunes(last, feedInlineBudget), true
}

// renderFeed 写出条数、totalResults、id、标题、时间和链接。summaryRunes 大于 0 才写摘要。超出 limit 的条目只报省略数。
func renderFeed(entries [][]string, total string, limit, summaryRunes int) string {
	if limit > len(entries) {
		limit = len(entries)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "feed entries: %d", len(entries))
	if total != "" {
		fmt.Fprintf(&b, " (totalResults=%s)", total)
	}
	b.WriteByte('\n')
	for i := 0; i < limit; i++ {
		inner := entries[i][1]
		id := feedField(inner, "id")
		if id == "" {
			id = feedField(inner, "guid")
		}
		title := feedField(inner, "title")
		published := feedField(inner, "published")
		if published == "" {
			published = feedField(inner, "updated")
		}
		if published == "" {
			published = feedField(inner, "pubDate")
		}
		summary := ""
		if summaryRunes > 0 {
			summary = clipRunes(feedField(inner, "summary"), summaryRunes)
			if summary == "" {
				summary = clipRunes(feedField(inner, "description"), summaryRunes)
			}
		}
		authors := feedAuthors(inner)
		pdf := ""
		if m := reFeedPDFHref.FindStringSubmatch(inner); len(m) > 0 {
			pdf = m[1]
			if pdf == "" {
				pdf = m[2]
			}
		}
		abs := ""
		if m := reFeedAbsHref.FindStringSubmatch(inner); len(m) > 1 {
			abs = m[1]
		}
		fmt.Fprintf(&b, "\n%d. %s\n", i+1, title)
		if id != "" {
			fmt.Fprintf(&b, "   id: %s\n", compactArxivID(id))
		}
		if published != "" {
			fmt.Fprintf(&b, "   published: %s\n", published)
		}
		if authors != "" {
			fmt.Fprintf(&b, "   authors: %s\n", authors)
		}
		if abs != "" {
			fmt.Fprintf(&b, "   url: %s\n", abs)
		}
		if pdf != "" {
			fmt.Fprintf(&b, "   pdf: %s\n", pdf)
		}
		if summary != "" {
			fmt.Fprintf(&b, "   summary: %s\n", summary)
		}
	}
	if len(entries) > limit {
		fmt.Fprintf(&b, "\n… %d more entries omitted\n", len(entries)-limit)
	}
	return strings.TrimSpace(b.String())
}

// feedField 取出指定标签的内部文本，去掉内层标签并反转义 HTML。
func feedField(inner, tag string) string {
	re := regexp.MustCompile(`(?is)<(?:[a-z0-9]+:)?` + regexp.QuoteMeta(tag) + `\b[^>]*>(.*?)</(?:[a-z0-9]+:)?` + regexp.QuoteMeta(tag) + `>`)
	m := re.FindStringSubmatch(inner)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(reInnerTags.ReplaceAllString(m[1], " ")))
}

// feedAuthors 最多取 12 个 name，用逗号连接。
func feedAuthors(inner string) string {
	names := reFeedName.FindAllStringSubmatch(inner, 12)
	var out []string
	for _, m := range names {
		n := strings.TrimSpace(html.UnescapeString(reInnerTags.ReplaceAllString(m[1], " ")))
		if n != "" {
			out = append(out, n)
		}
	}
	return strings.Join(out, ", ")
}

// compactArxivID 把含 /abs/ 的地址收成 arXiv id，并去掉末尾 .pdf。含 arxiv 的其它地址取最后一段。
func compactArxivID(id string) string {
	id = strings.TrimSpace(id)
	if i := strings.LastIndex(id, "/abs/"); i >= 0 {
		return strings.TrimSuffix(id[i+5:], ".pdf")
	}
	if i := strings.LastIndex(id, "/"); i >= 0 && strings.Contains(id, "arxiv") {
		return id[i+1:]
	}
	return id
}

// clipRunes 先把空白收成单空格。n 小于等于 0 或 rune 数不超过 n 时原样返回，否则截断并加省略号。
func clipRunes(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n])) + "…"
}
