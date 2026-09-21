package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

func (t *WorkspaceTools) skillDirRoots() []string {
	if t == nil || len(t.SkillDirs) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, dir := range t.SkillDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		out = append(out, dir)
	}
	return out
}

func (t *WorkspaceTools) decorateSkillBody(name, body string) string {
	if t == nil || t.SkillDirs == nil {
		return body
	}
	dir := strings.TrimSpace(t.SkillDirs[name])
	if dir == "" {
		return body
	}
	files := SkillPackFiles(dir)
	var b strings.Builder
	b.WriteString(body)
	b.WriteString("\n\n---\nSkill pack directory: ")
	b.WriteString(dir)
	if len(files) > 0 {
		b.WriteString("\nPack files:\n")
		for _, f := range files {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteByte('\n')
		}
	}
	b.WriteString("Run helpers with run_skill_script (skill=")
	b.WriteString(name)
	b.WriteString(", script=<file under scripts/>) or the absolute path above. Workspace-relative scripts/ will not find this pack.\n")
	return b.String()
}

func (t *WorkspaceTools) expandSkillScripts(command string) string {
	if t == nil || strings.TrimSpace(command) == "" {
		return command
	}
	argv := SplitShellArgv(command)
	if len(argv) == 0 {
		return command
	}
	changed := false
	for i, arg := range argv {
		resolved, ok := t.resolveSkillScriptArg(arg)
		if !ok {
			continue
		}
		argv[i] = resolved
		changed = true
	}
	if !changed {
		return command
	}
	if wrap := scriptInterpreter(argv[0]); wrap != "" {
		argv = append([]string{wrap}, argv...)
	}
	return joinShellArgv(argv)
}

func (t *WorkspaceTools) resolveSkillScriptArg(arg string) (string, bool) {
	arg = strings.TrimSpace(strings.Trim(arg, `"'`))
	if arg == "" {
		return "", false
	}
	rel := strings.ReplaceAll(arg, "\\", "/")
	rel = strings.TrimPrefix(rel, "./")
	if !strings.HasPrefix(rel, "scripts/") {
		return "", false
	}
	if t.Workspace != "" {
		ws := filepath.Join(t.Workspace, filepath.FromSlash(rel))
		if _, err := os.Stat(ws); err == nil {
			return "", false
		}
	}
	abs, ok := t.lookupSkillScript(rel)
	if !ok {
		return "", false
	}
	return abs, true
}

func (t *WorkspaceTools) lookupSkillScript(rel string) (string, bool) {
	if t == nil || t.SkillDirs == nil {
		return "", false
	}
	names := append([]string{}, t.Loaded...)
	if len(names) == 0 {
		for name := range t.SkillDirs {
			names = append(names, name)
		}
	}
	var hits []string
	for _, name := range names {
		dir := t.SkillDirs[name]
		if dir == "" {
			continue
		}
		cand := filepath.Join(dir, filepath.FromSlash(rel))
		cand = filepath.Clean(cand)
		if !capability.WithinWorkspace(dir, cand) {
			continue
		}
		if _, err := os.Stat(cand); err == nil {
			hits = append(hits, cand)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	return "", false
}

func normalizeSkillScript(script string) (string, error) {
	script = strings.TrimSpace(strings.ReplaceAll(script, "\\", "/"))
	script = strings.TrimPrefix(script, "./")
	if script == "" || strings.Contains(script, "..") || strings.HasPrefix(script, "/") {
		return "", fmt.Errorf("invalid script path")
	}
	if !strings.HasPrefix(script, "scripts/") {
		script = "scripts/" + script
	}
	return script, nil
}

func requireScriptInterpreter(path string) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh":
		if runtime.GOOS == "windows" && scriptInterpreter(path) == "" {
			return fmt.Errorf("missing bash to run %s (install Git Bash)", filepath.Base(path))
		}
	case ".py":
		if scriptInterpreter(path) == "" {
			return fmt.Errorf("missing python to run %s", filepath.Base(path))
		}
	}
	return nil
}

func scriptInterpreter(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh":
		if lookPath("bash") != "" {
			return "bash"
		}
		if lookPath("sh") != "" {
			return "sh"
		}
		return ""
	case ".py":
		if runtime.GOOS == "windows" && lookPath("py") != "" {
			return "py"
		}
		if lookPath("python3") != "" {
			return "python3"
		}
		if lookPath("python") != "" {
			return "python"
		}
		return ""
	case ".js", ".mjs":
		if lookPath("node") != "" {
			return "node"
		}
		return ""
	case ".ps1":
		if lookPath("pwsh") != "" {
			return "pwsh"
		}
		if lookPath("powershell") != "" {
			return "powershell"
		}
		return ""
	default:
		return ""
	}
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func joinShellArgv(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, a := range argv {
		parts = append(parts, quoteShellArg(a))
	}
	return strings.Join(parts, " ")
}

func quoteShellArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"'\\") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
