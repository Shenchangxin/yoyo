package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

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
		return ToolResult{Err: fmt.Errorf("skill %s has no on-disk directory", skill)}
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

func (t *WorkspaceTools) skillDir(skill string) string {
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
