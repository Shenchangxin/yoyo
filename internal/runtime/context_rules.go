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

func WriteRulesIndex(root string) {
	if root == "" {
		return
	}
	dir := filepath.Join(root, ".yoyo", "rules")
	_ = os.MkdirAll(dir, 0o755)
	var b strings.Builder
	b.WriteString("# Rules index (DCD)\n\n")
	b.WriteString("Pins contain the first-hit YOYO.md/AGENTS.md/CLAUDE.md, then stacked AGENTS.md from git root to cwd. Nested files are listed here — load with read_file when the path matches the directory that contains the rule.\n\n")
	seen := map[string]bool{}
	add := func(rel string) {
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			return
		}
		seen[rel] = true
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			dir = "/"
		} else {
			dir = dir + "/"
		}
		b.WriteString("- ")
		b.WriteString(rel)
		b.WriteString(" — when path matches ")
		b.WriteString(dir)
		b.WriteString("*\n")
	}
	for _, name := range []string{"YOYO.md", "AGENTS.md", "CLAUDE.md", filepath.Join(".yoyo", "YOYO.md"), filepath.Join(".cursor", "rules", "yoyo.mdc")} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			add(name)
		}
	}
	n := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == "dist" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}
		n++
		if n > 400 {
			return filepath.SkipAll
		}
		base := d.Name()
		if base != "CLAUDE.md" && base != "AGENTS.md" && !strings.HasSuffix(base, ".mdc") && !(strings.Contains(path, string(filepath.Separator)+".yoyo"+string(filepath.Separator)+"rules") && strings.HasSuffix(base, ".md")) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(filepath.ToSlash(rel), "/")
		if depth > 4 {
			return nil
		}
		add(rel)
		return nil
	})
	_ = os.WriteFile(filepath.Join(dir, "INDEX.md"), []byte(b.String()), 0o644)
}

func LoadStackedAgents(root string, maxBytes int) string {
	if root == "" {
		return ""
	}
	if maxBytes <= 0 {
		maxBytes = 32 * 1024
	}
	gitRoot := findGitRoot(root)
	if gitRoot == "" {
		gitRoot = root
	}
	var parts []string
	cur := root
	for {
		p := filepath.Join(cur, "AGENTS.md")
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			parts = append([]string{strings.TrimSpace(string(b))}, parts...)
		}
		if samePath(cur, gitRoot) {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	if len(parts) == 0 {
		return ""
	}
	s := strings.Join(parts, "\n\n")
	if len(s) > maxBytes {
		return s[:maxBytes] + "\n…[AGENTS.md truncated]"
	}
	return s
}

func findGitRoot(start string) string {
	cur := start
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return ""
		}
		cur = parent
	}
}

func capRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "\n…[YOYO.md truncated]"
}
