// 判断技能包相对路径能否落盘，并认出脚本。

package skillpack

import (
	"path"
	"strings"
	"unicode"
)

// FileAllowed 判断相对路径能否放进技能包。拒绝空路径、含 ..、绝对路径和点开头的文件名。拒绝 exe、dll、so、dylib、bin、压缩包、bat、cmd、msi、scr、com、wasm。无扩展名时只允许 license、notice、readme、makefile、skill.md，或位于 scripts/ 下。其余扩展名只允许文档、脚本、图片和模板这一组。
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

// SanitizeRelPart 只接受一段路径：字母、数字、连字符、下划线和点，且不能以点开头。空、.、.. 或含斜杠时返回空串。
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

// validRel 要求每一段都能通过 SanitizeRelPart，并且整条路径通过 FileAllowed。
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

// isScriptRel 在路径位于 scripts/ 下，或扩展名是 sh、py、ps1、rb、js、mjs、cjs 时返回 true。
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
