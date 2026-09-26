package runtime

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	reFeedEntry   = regexp.MustCompile(`(?is)<entry\b[^>]*>(.*?)</entry>`)
	reFeedItem    = regexp.MustCompile(`(?is)<item\b[^>]*>(.*?)</item>`)
	reFeedTotal   = regexp.MustCompile(`(?is)<(?:[a-z0-9]+:)?totalResults\b[^>]*>(\d+)`)
	reFeedPDFHref = regexp.MustCompile(`(?is)<link[^>]*title=["']pdf["'][^>]*href=["']([^"']+)["']|<link[^>]*href=["']([^"']+)["'][^>]*title=["']pdf["']`)
	reFeedAbsHref = regexp.MustCompile(`(?is)<link[^>]*rel=["']alternate["'][^>]*href=["']([^"']+)["']`)
	reFeedName    = regexp.MustCompile(`(?is)<(?:[a-z0-9]+:)?name\b[^>]*>(.*?)</(?:[a-z0-9]+:)?name>`)
	reInnerTags   = regexp.MustCompile(`<[^>]+>`)
)

func retryHTTPOn406(u *url.URL, status int) *url.URL {
	if u == nil || status != 406 || !strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	alt := *u
	alt.Scheme = "http"
	return &alt
}

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
	var b strings.Builder
	fmt.Fprintf(&b, "feed entries: %d", len(entries))
	if total != "" {
		fmt.Fprintf(&b, " (totalResults=%s)", total)
	}
	b.WriteByte('\n')
	limit := 25
	if len(entries) < limit {
		limit = len(entries)
	}
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
		summary := clipRunes(feedField(inner, "summary"), 360)
		if summary == "" {
			summary = clipRunes(feedField(inner, "description"), 360)
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
	return strings.TrimSpace(b.String()), true
}

func feedField(inner, tag string) string {
	re := regexp.MustCompile(`(?is)<(?:[a-z0-9]+:)?` + regexp.QuoteMeta(tag) + `\b[^>]*>(.*?)</(?:[a-z0-9]+:)?` + regexp.QuoteMeta(tag) + `>`)
	m := re.FindStringSubmatch(inner)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(reInnerTags.ReplaceAllString(m[1], " ")))
}

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

func clipRunes(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n])) + "…"
}
