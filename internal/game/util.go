package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func writeJSON(root, rel string, v any) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			return s[i : j+1]
		}
	}
	return s
}

func hasWord(s string, words ...string) bool {
	low := strings.ToLower(s)
	for _, w := range words {
		if strings.Contains(low, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

func genericWorldTitle(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "game world", "relic run", "新建世界", "new world", "new game":
		return true
	default:
		return false
	}
}

func preferWorldTitle(cur, planned, intent string) string {
	if !genericWorldTitle(cur) {
		return strings.TrimSpace(cur)
	}
	if !genericWorldTitle(planned) {
		return strings.TrimSpace(planned)
	}
	return titleFromIntent(intent, firstNonEmpty(cur, planned, "Game world"))
}

func titleFromIntent(intent, fallback string) string {
	t := strings.Join(strings.Fields(strings.TrimSpace(intent)), " ")
	if t == "" {
		return fallback
	}
	r := []rune(t)
	if len(r) > 16 {
		return string(r[:16])
	}
	return t
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
