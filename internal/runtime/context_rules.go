package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// LoadWorkspaceRules reads Cursor/Claude/Codex-style always-on files.
// Only the first hit is used so we do not stack three copies of the same policy.
func LoadWorkspaceRules(root string, maxRunes int) (text, source string) {
	if root == "" {
		return "", ""
	}
	if maxRunes <= 0 {
		maxRunes = 1500
	}
	candidates := []string{
		"YOYO.md",
		filepath.Join(".yoyo", "YOYO.md"),
		"AGENTS.md",
		"CLAUDE.md",
		filepath.Join(".cursor", "rules", "yoyo.mdc"),
	}
	for _, rel := range candidates {
		p := filepath.Join(root, rel)
		b, err := os.ReadFile(p)
		if err != nil || len(b) == 0 {
			continue
		}
		s := strings.TrimSpace(string(b))
		if s == "" {
			continue
		}
		return capRunes(s, maxRunes), rel
	}
	return "", ""
}

func capRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "\n…[YOYO.md truncated]"
}
