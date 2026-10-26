// normalize.go 整理人名、场景键和各供应商的分辨率写法。

package video

import (
	"regexp"
	"strings"
)

// 这些正则用来去掉括号别名和各种空白。
var (
	// parenPair 匹配成对的中文或英文括号及其内部。
	parenPair = regexp.MustCompile(`[（(][^（()）]*[）)]`)
	// openParen 匹配未闭合的括号直到结尾。
	openParen = regexp.MustCompile(`[（(].*$`)
	// spaces 匹配连续的空白，含全角空格。
	spaces    = regexp.MustCompile(`[\s　]+`)
)

// NormalizeName 去掉成对括号里的别名和未闭合的括号后缀，再去掉空白并转成小写，使「林小雨（主角）」和「林小雨」落到同一键。
func NormalizeName(name string) string {
	s := strings.TrimSpace(name)
	s = parenPair.ReplaceAllString(s, "")
	s = openParen.ReplaceAllString(s, "")
	s = spaces.ReplaceAllString(s, "")
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeSceneKey 把地点和时段收成小写的「地点|时段」。地点里的空白会去掉。
func NormalizeSceneKey(location, timeOfDay string) string {
	loc := spaces.ReplaceAllString(strings.TrimSpace(location), "")
	t := strings.TrimSpace(timeOfDay)
	return strings.ToLower(loc) + "|" + strings.ToLower(t)
}

// NormalizeVideoResolution 把分辨率收成 480p、720p、1080p 或 2k，空值用 720p。volcengine 只留下 480p，其余收成 720p。minimax 把 1080p 和 2k 收成 2k，其余收成 768p。aliyun 把 480p、1080p 收成 480P、1080P，其余收成 720P。
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
