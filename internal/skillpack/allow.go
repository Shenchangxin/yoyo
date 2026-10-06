package skillpack

import (
	"path"
	"strings"
	"unicode"
)

func FileAllowed(rel string) bool {
	rel = strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return false
	}
	base := path.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	ext := path.Ext(rel)
	switch ext {
	case ".exe", ".dll", ".so", ".dylib", ".bin", ".zip", ".tar", ".gz", ".tgz", ".7z", ".rar",
		".bat", ".cmd", ".msi", ".scr", ".com", ".wasm":
		return false
	}
	if ext == "" {
		if base == "license" || base == "notice" || base == "readme" || base == "makefile" || strings.EqualFold(base, "skill.md") {
			return true
		}
		return strings.Contains(rel, "/scripts/") || strings.HasPrefix(rel, "scripts/")
	}
	switch ext {
	case ".md", ".txt", ".py", ".js", ".ts", ".mjs", ".cjs", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".htm", ".css",
		".sh", ".ps1", ".rb", ".pl", ".r", ".sql", ".csv", ".tsv",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
		".jinja", ".j2", ".template", ".dot":
		return true
	default:
		return false
	}
}

func SanitizeRelPart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/\\")
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\") {
		return ""
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return ""
	}
	if s[0] == '.' {
		return ""
	}
	return s
}

func validRel(rel string) bool {
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if SanitizeRelPart(part) == "" {
			return false
		}
	}
	return FileAllowed(rel)
}

func isScriptRel(rel string) bool {
	rel = strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	if strings.Contains(rel, "/scripts/") || strings.HasPrefix(rel, "scripts/") {
		return true
	}
	switch path.Ext(rel) {
	case ".sh", ".py", ".ps1", ".rb", ".js", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}
