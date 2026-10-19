package webcontent

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var (
	reResultA       = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	reResultSnippet = regexp.MustCompile(`(?is)<(?:a|span)[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</(?:a|span)>`)
	reLiteLink      = regexp.MustCompile(`(?is)<a[^>]*rel="nofollow"[^>]*href="(https?://[^"]+)"[^>]*>(.*?)</a>`)
	reHref          = regexp.MustCompile(`(?i)href="([^"]+)"`)
)

// ParseDuckDuckGoHTML extracts organic results from html.duckduckgo.com or lite.
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

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
