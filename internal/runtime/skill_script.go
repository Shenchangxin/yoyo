// 给技能正文加上包路径，并把 shell 里的 scripts/ 相对路径改写成包内绝对路径。工作区里已存在的同名相对路径不改。解释器选择也在这里。

package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/pwsh"
)

// skillDirRoots 返回去重后的 SkillDirs 值。工具或映射为空时返回 nil。
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

// decorateSkillBody 用该技能的 SkillDirs 目录调用 DecorateSkillBody。
func (t *WorkspaceTools) decorateSkillBody(name, body string) string {
	dir := ""
	if t != nil && t.SkillDirs != nil {
		dir = t.SkillDirs[name]
	}
	return DecorateSkillBody(name, body, dir)
}

// skillLoadHint 只给 novel-game-analyze 追加一句：长原文要用 task profile=explore 分章，不要凭 @file 摘录重写小说。其他名字返回空串。
func skillLoadHint(name string) string {
	switch strings.TrimSpace(name) {
	case "novel-game-analyze":
		return "\n\nLong source: after this body, dispatch task profile=explore for non-overlapping chapter batches that return fact tables only. Do not reconstruct the novel from an @file excerpt; read_file the path or send chapter ranges to children."
	default:
		return ""
	}
}

// DecorateSkillBody 在有目录时把包路径、文件清单，以及 run_skill_script 和 read_skill_file 的用法写在正文前面，这样 @ 注入和 load_skill 在正文被截断前都能看到。有 references/yoyo-tools.md 时点名先读它。Windows 上没有可用 POSIX shell 时说明不要跑 .sh。目录为空时只在正文后加 skillLoadHint。
func DecorateSkillBody(name, body, dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return body + skillLoadHint(name)
	}
	files := SkillPackFiles(dir)
	var b strings.Builder
	b.WriteString("Skill pack directory: ")
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
	b.WriteString(", script=<file under scripts/>). Read pack files with read_skill_file (skill=")
	b.WriteString(name)
	b.WriteString(", path=relative).")
	if packHasRel(files, "references/yoyo-tools.md") {
		b.WriteString(" Host adapter: read_skill_file path=references/yoyo-tools.md before following pack procedures.")
	}
	b.WriteString(" Workspace-relative scripts/ is not this pack — do not glob the workspace for them. Public HTTP: prefer web_fetch (if https returns 406, retry the http URL). Do not write probe scripts or wait-loop a hung process.")
	if runtime.GOOS == "windows" {
		if windowsPosixShell() == "" {
			b.WriteString(" Windows WSL bash.exe cannot run pack .sh — do not call run_skill_script on .sh; use web_fetch for public HTTP.")
		} else {
			b.WriteString(" Pack .sh runs under Git Bash, not WSL bash.exe.")
		}
	}
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString(skillLoadHint(name))
	return b.String()
}

// packHasRel 按小写斜杠路径判断清单里是否有 want。
func packHasRel(files []string, want string) bool {
	want = strings.ToLower(strings.ReplaceAll(want, "\\", "/"))
	for _, f := range files {
		if strings.ToLower(strings.ReplaceAll(f, "\\", "/")) == want {
			return true
		}
	}
	return false
}

// expandSkillScripts 把命令参数里的 scripts/ 相对路径换成包内绝对路径。工作区里已经存在该相对路径时不改。改写后若脚本需要解释器，把解释器插到参数前面。
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

// resolveSkillScriptArg 只处理 scripts/ 开头的参数。工作区里 Stat 成功则返回假，避免盖住工作区文件。否则 lookupSkillScript。
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

// lookupSkillScript 在已加载技能里找，没有已加载名单则扫全部 SkillDirs。路径必须在该技能目录内。恰好一个文件存在才返回真。
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

// normalizeSkillScript 把反斜杠改成斜杠，去掉 ./。空路径、含 .. 或以 / 开头则返回错误。没有 scripts/ 前缀时补上。
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

// requireScriptInterpreter 对 .sh 和无扩展名要求 posix 解释器，对 .py 要求 Python。Windows 上没有可用 shell 时说明 WSL 的 bash.exe 不能用。其他扩展名不检查。
func requireScriptInterpreter(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".sh", "":
		if scriptInterpreter(path) == "" {
			if runtime.GOOS == "windows" {
				return fmt.Errorf("cannot run %s: Windows WSL bash.exe is not a usable POSIX shell. Install Git Bash, or use web_fetch for public HTTP (retry http if https returns 406)", filepath.Base(path))
			}
			return fmt.Errorf("missing bash to run %s", filepath.Base(path))
		}
	case ".py":
		if scriptInterpreter(path) == "" {
			return fmt.Errorf("missing python to run %s", filepath.Base(path))
		}
	}
	return nil
}

// posixShellBin 在 Windows 上返回 windowsPosixShell。其他系统依次找 bash 和 sh。
func posixShellBin() string {
	if runtime.GOOS == "windows" {
		return windowsPosixShell()
	}
	if lookPath("bash") != "" {
		return "bash"
	}
	if lookPath("sh") != "" {
		return "sh"
	}
	return ""
}

// scriptInterpreter 按扩展名选择解释器。无扩展名和 .sh 用 posixShellBin。Windows 上 .py 优先 py，再 python3、python。.js 和 .mjs 用 node。.ps1 用 pwsh.Look，Windows 上再退回 pwsh.Fallback。
func scriptInterpreter(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return posixShellBin()
	}
	switch ext {
	case ".sh":
		return posixShellBin()
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
		if p := pwsh.Look(); p != "" {
			return p
		}
		if runtime.GOOS == "windows" {
			return pwsh.Fallback()
		}
		if lookPath("pwsh") != "" {
			return "pwsh"
		}
		return ""
	default:
		return ""
	}
}

// lookPath 把 exec.LookPath 的失败收成空串。
func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

// joinShellArgv 给每个参数加引号后用空格连接。
func joinShellArgv(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, a := range argv {
		parts = append(parts, quoteShellArg(a))
	}
	return strings.Join(parts, " ")
}

// quoteShellArg 在空串或含空白、引号、反斜杠时加双引号，并把内部双引号转义。
func quoteShellArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"'\\") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
