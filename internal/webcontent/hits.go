package webcontent

import (
	"fmt"
	"strings"
)

const maxHits = 8

// Hit is one public-web search result.
type Hit struct {
	Title   string
	URL     string
	Snippet string
}

// FormatHits renders numbered title/url/snippet lines. Empty if nothing usable.
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
			// already in the heading
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

// Useful reports whether search output has real results, not chrome or empty HTML.
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
