package video

import (
	"regexp"
	"strings"
)

var (
	parenPair = regexp.MustCompile(`[（(][^（()）]*[）)]`)
	openParen = regexp.MustCompile(`[（(].*$`)
	spaces    = regexp.MustCompile(`[\s　]+`)
)

// NormalizeName strips role aliases in parentheses and whitespace so
// 「林小雨（主角）」and「林小雨」collapse to the same key.
func NormalizeName(name string) string {
	s := strings.TrimSpace(name)
	s = parenPair.ReplaceAllString(s, "")
	s = openParen.ReplaceAllString(s, "")
	s = spaces.ReplaceAllString(s, "")
	return strings.ToLower(strings.TrimSpace(s))
}

func NormalizeSceneKey(location, timeOfDay string) string {
	loc := spaces.ReplaceAllString(strings.TrimSpace(location), "")
	t := strings.TrimSpace(timeOfDay)
	return strings.ToLower(loc) + "|" + strings.ToLower(t)
}

func NormalizeVideoResolution(raw, provider string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "p", "")
	switch s {
	case "480":
		s = "480p"
	case "720":
		s = "720p"
	case "1080":
		s = "1080p"
	case "2k":
		s = "2k"
	default:
		if s == "" {
			s = "720p"
		} else {
			s = strings.ToLower(raw)
		}
	}
	switch strings.ToLower(provider) {
	case "volcengine":
		if s == "1080p" || s == "2k" {
			return "720p"
		}
		if s != "480p" {
			return "720p"
		}
		return s
	case "minimax":
		if s == "1080p" || s == "2k" {
			return "2k"
		}
		return "768p"
	case "aliyun":
		switch s {
		case "480p":
			return "480P"
		case "1080p":
			return "1080P"
		default:
			return "720P"
		}
	}
	return s
}
