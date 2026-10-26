// 本文件从 DuckDuckGo 的 HTML 或 lite 页面抽出有机结果。最多留下 maxHits 条。

package webcontent

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var (
	// reResultA 匹配 class 含 result__a 的链接，抓 href 和内部 HTML。
	reResultA = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	// reResultSnippet 匹配 class 含 result__snippet 的 a 或 span。
	reResultSnippet = regexp.MustCompile(`(?is)<(?:a|span)[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</(?:a|span)>`)
	// reLiteLink 匹配 rel="nofollow" 且 href 为 http(s) 的链接。
	reLiteLink = regexp.MustCompile(`(?is)<a[^>]*rel="nofollow"[^>]*href="(https?://[^"]+)"[^>]*>(.*?)</a>`)
	// reHref 匹配 href 属性。本文件目前没有使用它。
	reHref = regexp.MustCompile(`(?i)href="([^"]+)"`)
)

// ParseDuckDuckGoHTML 从 html.duckduckgo.com 或 lite 页面抽出结果。空白页面返回 nil。result__a 没有命中时改用 lite 链接。
func ParseDuckDuckGoHTML(page string) []Hit {
	if strings.TrimSpace(page) == "" {
		return nil
	}
	hits := parseResultA(page)
	if len(hits) == 0 {
		hits = parseLiteLinks(page)
	}
	return hits
}

// parseResultA 用 result__a 抽链接，并用同序号的 result__snippet 当摘要。正则最多取 maxHits*2 处，结果停在 maxHits 条。skipSERPLink 为真的跳过。
func parseResultA(page string) []Hit {
	anchors := reResultA.FindAllStringSubmatch(page, maxHits*2)
	snips := reResultSnippet.FindAllStringSubmatch(page, maxHits*2)
	var hits []Hit
	for i, m := range anchors {
		if len(m) < 3 {
			continue
		}
		link := decodeDDGHref(m[1])
		title := collapseSpace(stripTags(m[2]))
		if skipSERPLink(link, title) {
			continue
		}
		h := Hit{Title: title, URL: link}
		if i < len(snips) && len(snips[i]) > 1 {
			h.Snippet = collapseSpace(stripTags(snips[i][1]))
		}
		hits = append(hits, h)
		if len(hits) >= maxHits {
			break
		}
	}
	return hits
}

// parseLiteLinks 从 rel=nofollow 的链接抽结果，最多看 40 处匹配，留下 maxHits 条。同一链接只留一次。
func parseLiteLinks(page string) []Hit {
	seen := map[string]bool{}
	var hits []Hit
	for _, m := range reLiteLink.FindAllStringSubmatch(page, 40) {
		if len(m) < 3 {
			continue
		}
		link := decodeDDGHref(m[1])
		title := collapseSpace(stripTags(m[2]))
		if skipSERPLink(link, title) || seen[link] {
			continue
		}
		seen[link] = true
		hits = append(hits, Hit{Title: title, URL: link})
		if len(hits) >= maxHits {
			break
		}
	}
	return hits
}

// decodeDDGHref 解开 HTML 转义。以 // 开头时补上 https:。查询参数 uddg 非空时返回解码后的值。
// 主机名含 duckduckgo.com 的 http(s) 地址返回空。其他 http(s) 返回原 URL。其余返回空。
func decodeDDGHref(raw string) string {
	raw = html.UnescapeString(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if q := u.Query().Get("uddg"); q != "" {
		if decoded, err := url.QueryUnescape(q); err == nil && decoded != "" {
			return decoded
		}
		return q
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		if strings.Contains(strings.ToLower(u.Host), "duckduckgo.com") {
			return ""
		}
		return u.String()
	}
	return ""
}

// skipSERPLink 在链接和标题都空、链接含 duckduckgo.com，或标题是 next、previous、more results、feedback、privacy、all、images、videos、news、maps 时返回 true。
func skipSERPLink(link, title string) bool {
	if link == "" && title == "" {
		return true
	}
	host := strings.ToLower(link)
	if strings.Contains(host, "duckduckgo.com") {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "next", "previous", "more results", "feedback", "privacy", "all", "images", "videos", "news", "maps":
		return true
	}
	return false
}

// stripTags 去掉 HTML 标签并把标签边界换成空格，同时做 HTML 反转义。
func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range html.UnescapeString(s) {
		switch r {
		case '<':
			in = true
		case '>':
			in = false
			b.WriteByte(' ')
		default:
			if !in {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// collapseSpace 把连续空白收成单个空格并去掉两端空白。
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
