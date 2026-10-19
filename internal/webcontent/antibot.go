package webcontent

import (
	"strings"
	"unicode/utf8"
)

// LooksAntibot reports high-confidence challenge pages (Cloudflare / Jina captcha).
func LooksAntibot(body string) bool {
	sample := body
	if utf8.RuneCountInString(sample) > 4000 {
		sample = string([]rune(sample)[:4000])
	}
	lower := strings.ToLower(sample)
	if strings.Contains(lower, "warning:") && strings.Contains(lower, "requiring captcha") {
		return true
	}
	if strings.Contains(lower, "just a moment") && (strings.Contains(lower, "cloudflare") || strings.Contains(lower, "cf-") || strings.Contains(lower, "challenge")) {
		return true
	}
	if strings.Contains(lower, "attention required! | cloudflare") {
		return true
	}
	if strings.Contains(lower, "/cdn-cgi/challenge-platform/") {
		return true
	}
	if strings.Contains(lower, "enable javascript and cookies to continue") {
		return true
	}
	return false
}

// ThinExtract reports that stripped HTML is not enough article text.
func ThinExtract(stripped, raw string) bool {
	if LooksAntibot(raw) {
		return true
	}
	s := strings.TrimSpace(stripped)
	if s == "" {
		return true
	}
	if utf8.RuneCountInString(s) < 200 {
		return true
	}
	if len(raw) > 4000 && utf8.RuneCountInString(s) < 400 {
		return true
	}
	return false
}
