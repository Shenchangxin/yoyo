package game

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode"
)

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title, id string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	s = slugRe.ReplaceAllString(b.String(), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "world"
	}
	if len(s) > 24 {
		s = s[:24]
	}
	if len(id) >= 8 {
		s += "-" + id[:8]
	}
	return s
}
