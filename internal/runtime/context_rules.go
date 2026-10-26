// 工作区规则的读取、索引和路径规则。buildBill 取第一份命中；读文件和改文件时 absorbPathRules 把嵌套规则收进动态尾。
// LoadWorkspaceRules 不把多份规则的全文叠在一起。往上叠 AGENTS.md 的是 LoadStackedAgents，只在 StackAgentsMD 为真时由 buildBill 换上。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// LoadWorkspaceRules 按顺序读第一份非空文件：YOYO.md、.yoyo/YOYO.md、AGENTS.md、CLAUDE.md、.cursor/rules/yoyo.mdc。
// 命中后立刻返回，不把其余几份的全文叠上去。maxRunes 小于等于 0 时上限按 1500 个 rune，否则用传入值。根目录为空时返回空。
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

// WriteRulesIndex 把根上的规则文件和扫到的嵌套规则写进 .yoyo/rules/INDEX.md，只列路径和目录前缀，不把正文注入提示词。
// 跳过 node_modules、.git、vendor、dist、.next。超过 400 个文件或相对深度大于 4 就不再收录。
func WriteRulesIndex(root string) {
	if root == "" {
		return
	}
	dir := filepath.Join(root, ".yoyo", "rules")
	_ = os.MkdirAll(dir, 0o755)
	var b strings.Builder
	b.WriteString("# Rules index (DCD)\n\n")
	b.WriteString("Pins contain the first-hit YOYO.md/AGENTS.md/CLAUDE.md only. Nested files are listed here and are injected when a read or edit touches their directory. You can also read_file them.\n\n")
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

// LoadStackedAgents 从 root 往上走到 git 根，把沿途每一份 AGENTS.md 叠成一份。先读到的在后面，最上面的祖先在最前。
// maxBytes 小于等于 0 时按 32KiB 截断，截断按字节，并附上省略标记。没有 git 根时就停在 root。
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

// findGitRoot 返回 start 自身或祖先里第一个含 .git 的目录。找不到返回空串。
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

// pathRulesCapRunes 是路径规则动态尾的 rune 上限。
const pathRulesCapRunes = 2_000

// ruleBlock 是一份嵌套规则的相对路径和截断后的正文。
type ruleBlock struct {
	id   string
	text string
}

// absorbPathRules 记住适用于 rel 的嵌套规则，并跳过已经收过的路径。总长达到 pathRulesCapRunes 后不再追加。
// 工作区根上那份第一命中不在这里重复。
func (t *WorkspaceTools) absorbPathRules(rel string) {
	if t == nil || t.Workspace == "" {
		return
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || rel == "." {
		return
	}
	blocks := nestedRuleBlocks(t.Workspace, rel)
	if len(blocks) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pathRuleSeen == nil {
		t.pathRuleSeen = map[string]bool{}
	}
	var b strings.Builder
	if t.pathRules != "" {
		b.WriteString(t.pathRules)
	}
	for _, blk := range blocks {
		if t.pathRuleSeen[blk.id] {
			continue
		}
		if utf8.RuneCountInString(b.String()) >= pathRulesCapRunes {
			break
		}
		t.pathRuleSeen[blk.id] = true
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(blk.text)
	}
	t.pathRules = capRunes(b.String(), pathRulesCapRunes)
}

// PathRules 返回已经吸收的路径规则文本。
func (t *WorkspaceTools) PathRules() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pathRules
}

// nestedRuleBlocks 从 rel 的父目录往上找 AGENTS.md、CLAUDE.md 和 YOYO.md，直到根。每份截到 800 个 rune。
// LoadWorkspaceRules 命中的那条相对路径会跳过，避免和稳定前缀里的第一份重复。
func nestedRuleBlocks(root, rel string) []ruleBlock {
	_, pinned := LoadWorkspaceRules(root, 1)
	skip := map[string]bool{}
	if pinned != "" {
		skip[filepath.ToSlash(pinned)] = true
	}
	dir := filepath.ToSlash(filepath.Dir(rel))
	var out []ruleBlock
	for dir != "." && dir != "" && dir != "/" {
		for _, name := range []string{"AGENTS.md", "CLAUDE.md", "YOYO.md"} {
			relRule := dir + "/" + name
			if skip[relRule] {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relRule)))
			if err != nil {
				continue
			}
			text := strings.TrimSpace(string(raw))
			if text == "" {
				continue
			}
			out = append(out, ruleBlock{
				id:   relRule,
				text: "## Path rule (" + relRule + ")\n" + capRunes(text, 800),
			})
		}
		parent := filepath.ToSlash(filepath.Dir(dir))
		if parent == dir {
			break
		}
		dir = parent
	}
	return out
}

// capRunes 超过 max 个 rune 时截断并加上省略号。max 小于等于 0 时原样返回。
func capRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "…"
}
