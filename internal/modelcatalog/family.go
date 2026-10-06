package modelcatalog

import (
	"regexp"
	"strings"
)

var (
	dateSuffix = regexp.MustCompile(`-\d{8}$`)
	versionTok = regexp.MustCompile(`^(?:v)?\d+(?:\.\d+)*[a-z]?$`)
	oSeries    = regexp.MustCompile(`^o\d+[a-z]?$`)
)

// FamilyOf groups dated/versioned model ids the way the bundled snapshot does
// (claude-sonnet-4-6 → claude-sonnet, gpt-5.2-pro → gpt-pro). Used when
// models.dev omits family.
func FamilyOf(id string) string {
	s := strings.ToLower(strings.TrimSpace(id))
	if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
		s = s[i+1:]
	}
	s = dateSuffix.ReplaceAllString(s, "")
	s = strings.TrimSuffix(s, "-latest")
	s = strings.TrimSuffix(s, "-preview")
	if strings.HasPrefix(s, "text-embedding") {
		return "text-embedding"
	}
	if strings.Contains(s, "gpt-image") || strings.HasPrefix(s, "dall-e") || strings.HasPrefix(s, "chatgpt-image") {
		return "gpt-image"
	}
	var parts []string
	for _, p := range strings.Split(s, "-") {
		if p == "" {
			continue
		}
		if oSeries.MatchString(p) {
			parts = append(parts, "o")
			continue
		}
		if versionTok.MatchString(p) {
			continue
		}
		if p == "reasoner" {
			p = "thinking"
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		if s == "" {
			return id
		}
		return s
	}
	return strings.Join(parts, "-")
}
