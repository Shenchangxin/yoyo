// 按技能名读包内相对路径。没有目录的内置技能用 noSkillDirError 说明它只有说明、没有脚本。脚本执行在 tools_extra.go。

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// noSkillDirError 在 Skills 或 BootstrapBodies 里已有正文时，说明该技能是内置说明、不带脚本或文件。否则说明没有磁盘目录。
func (t *WorkspaceTools) noSkillDirError(skill string) error {
	if t != nil && (strings.TrimSpace(t.Skills[skill]) != "" || strings.TrimSpace(t.BootstrapBodies[skill]) != "") {
		return fmt.Errorf("skill %s is instruction-only: it is built in and ships no scripts or files. Its instructions are already in your context; check the work with read_file, glob or grep", skill)
	}
	return fmt.Errorf("skill %s has no on-disk directory", skill)
}

// readSkillFile 要求 skill 和 path。路径去掉 ./，拒绝空路径、含 .. 或以 / 开头的路径。WithinWorkspace 限制在技能目录内，ForbiddenDiagPath 拒绝诊断面。缺失时最多列出 24 个包内文件。指定 offset 或 limit 时按行号输出。超过 maxInlineBytes 时只给前 80 行并注明截断。
func (t *WorkspaceTools) readSkillFile(skill, rel string, offset, limit int) ToolResult {
	if t == nil {
		return ToolResult{Err: fmt.Errorf("no tools")}
	}
	skill = strings.TrimSpace(skill)
	rel = strings.TrimSpace(rel)
	if skill == "" || rel == "" {
		return ToolResult{Err: fmt.Errorf("read_skill_file: skill and path required")}
	}
	dir := t.skillDir(skill)
	if dir == "" {
		return ToolResult{Err: t.noSkillDirError(skill)}
	}
	relSlash := strings.ReplaceAll(rel, "\\", "/")
	relSlash = strings.TrimPrefix(relSlash, "./")
	if relSlash == "" || strings.Contains(relSlash, "..") || strings.HasPrefix(relSlash, "/") {
		return ToolResult{Err: fmt.Errorf("invalid skill path")}
	}
	p := filepath.Clean(filepath.Join(dir, filepath.FromSlash(relSlash)))
	if !capability.WithinWorkspace(dir, p) {
		return ToolResult{Err: fmt.Errorf("path escapes skill directory")}
	}
	if capability.ForbiddenDiagPath(t.Home, p) {
		return ToolResult{Err: fmt.Errorf("path denied: diagnostic plane")}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		avail := SkillPackFiles(dir)
		msg := fmt.Sprintf("missing %s", relSlash)
		if len(avail) > 0 {
			n := len(avail)
			if n > 24 {
				avail = avail[:24]
			}
			msg += "; pack files: " + strings.Join(avail, ", ")
			if n > 24 {
				msg += ", …"
			}
		}
		return ToolResult{Err: fmt.Errorf("%s", msg)}
	}
	if offset > 0 || limit > 0 {
		return ToolResult{Content: numberLines(string(b), offset, limit)}
	}
	if len(b) > maxInlineBytes {
		head := numberLines(string(b), 1, 80)
		return ToolResult{Content: fmt.Sprintf("%s\n[truncated %d bytes; re-read with offset/limit]\n", head, len(b))}
	}
	return ToolResult{Content: numberLines(string(b), 0, 0)}
}

// skillDir 先 resolveSkillName，再按原名或不区分大小写查找 SkillDirs。
func (t *WorkspaceTools) skillDir(skill string) string {
	skill = t.resolveSkillName(skill)
	if t == nil || t.SkillDirs == nil {
		return ""
	}
	if dir := t.SkillDirs[skill]; dir != "" {
		return dir
	}
	for name, d := range t.SkillDirs {
		if strings.EqualFold(name, skill) {
			return d
		}
	}
	return ""
}

// resolveSkillName 把别名收成规范名。先看 SkillMeta 的 Name、DisplayName 和目录基名，再看 Skills 和 SkillDirs 的键。都对不上则返回去掉空白后的原名。
func (t *WorkspaceTools) resolveSkillName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || t == nil {
		return name
	}
	if t.SkillMeta != nil {
		if sk, ok := t.SkillMeta[name]; ok && strings.TrimSpace(sk.Name) != "" {
			return sk.Name
		}
		for _, sk := range t.SkillMeta {
			if strings.EqualFold(sk.Name, name) && sk.Name != "" {
				return sk.Name
			}
			if sk.DisplayName != "" && strings.EqualFold(sk.DisplayName, name) {
				return sk.Name
			}
			if sk.Dir != "" && strings.EqualFold(filepath.Base(sk.Dir), name) {
				return sk.Name
			}
		}
	}
	if t.Skills != nil {
		if _, ok := t.Skills[name]; ok {
			return name
		}
		for k := range t.Skills {
			if strings.EqualFold(k, name) {
				return k
			}
		}
	}
	if t.SkillDirs != nil {
		if _, ok := t.SkillDirs[name]; ok {
			return name
		}
		for k, d := range t.SkillDirs {
			if strings.EqualFold(k, name) || (d != "" && strings.EqualFold(filepath.Base(d), name)) {
				return k
			}
		}
	}
	return name
}
