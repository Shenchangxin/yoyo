package diaglog

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	reBearer    = regexp.MustCompile(`(?i)(authorization:\s*(?:bearer\s+)?)(\S+)`)
	reSK        = regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9_-]{8,}`)
	rePEM       = regexp.MustCompile(`-----BEGIN [A-Z ]+-----[\s\S]*?-----END [A-Z ]+-----`)
	reJSONSecret = regexp.MustCompile(`(?i)("(api[_-]?key|secret|password|token|search_key|authorization)"\s*:\s*")([^"]*)(")`)
	reAssign    = regexp.MustCompile(`(?i)\b(YOYO_API_KEY|OPENAI_API_KEY|ANTHROPIC_API_KEY|DEEPSEEK_API_KEY|API_KEY|SEARCH_KEY)=([^\s]+)`)
)

const redacted = "[REDACTED]"

func Redact(s string) string {
	if s == "" {
		return s
	}
	s = rePEM.ReplaceAllString(s, redacted+" PEM")
	s = reBearer.ReplaceAllString(s, "${1}"+redacted)
	s = reSK.ReplaceAllString(s, redacted+" KEY")
	s = reJSONSecret.ReplaceAllString(s, "${1}"+redacted+"${4}")
	s = reAssign.ReplaceAllString(s, "${1}="+redacted)
	return s
}

func RedactBytes(b []byte) []byte {
	return []byte(Redact(string(b)))
}

func PayloadPreview(s string, maxRunes int) string {
	s = Redact(s)
	s = strings.Join(strings.Fields(s), " ")
	if maxRunes <= 0 {
		maxRunes = 512
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes]) + "…"
}

func LooksSecretKey(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "api_key", "apikey", "key", "secret", "password", "token", "authorization", "search_key", "searchkey":
		return true
	default:
		return strings.Contains(n, "secret") || strings.Contains(n, "password") || strings.HasSuffix(n, "_key")
	}
}
