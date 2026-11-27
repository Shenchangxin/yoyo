// ask_user、run_skill_script 的入口。脚本路径收在技能目录里，再交给 shell。不解析 SKILL.md，那在 skill_file.go 和 skill_script.go。

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// askUserArgs 用 parseAskForm 解析参数后交给 askForm。
// args 交给 parseAskForm，结果再交给 askForm。
func (t *WorkspaceTools) askUserArgs(args map[string]any) ToolResult {
	return t.askForm(parseAskForm(args))
}

// runSkillScript 在技能目录里找脚本并执行。路径经 normalizeSkillScript，再用 WithinWorkspace 限制在该目录内。找不到目录时用 noSkillDirError。解释器缺失则拒绝。超时 30 秒，交给 shell。
// skill 去掉空白后交给 resolveSkillName，空串则报缺参，否则用来取技能目录。
// script 去掉空白，空串则报缺参；否则交给 normalizeSkillScript，再接到技能目录上。
// args 非空时交给 SplitShellArgv，接在脚本参数后面。
func (t *WorkspaceTools) runSkillScript(skill, script, args string) ToolResult {
	if t == nil {
		return ToolResult{Err: fmt.Errorf("no tools")}
	}
	skill = t.resolveSkillName(strings.TrimSpace(skill))
	script = strings.TrimSpace(script)
	if skill == "" || script == "" {
		return ToolResult{Err: fmt.Errorf("run_skill_script: skill and script required")}
	}
	dir := t.skillDir(skill)
	if dir == "" {
		return ToolResult{Err: t.noSkillDirError(skill)}
	}
	relSlash, err := normalizeSkillScript(script)
	if err != nil {
		return ToolResult{Err: err}
	}
	rel := filepath.FromSlash(relSlash)
	p := filepath.Clean(filepath.Join(dir, rel))
	if !capability.WithinWorkspace(dir, p) {
		return ToolResult{Err: fmt.Errorf("script escapes skill directory")}
	}
	if _, err := os.Stat(p); err != nil {
		avail := SkillPackFiles(filepath.Join(dir, "scripts"))
		msg := fmt.Sprintf("missing script %s", relSlash)
		if len(avail) > 0 {
			msg += "; available: " + strings.Join(avail, ", ")
		}
		return ToolResult{Err: fmt.Errorf("%s", msg)}
	}
	if err := requireScriptInterpreter(p); err != nil {
		return ToolResult{Err: err}
	}
	argv := []string{p}
	if wrap := scriptInterpreter(p); wrap != "" {
		argv = []string{wrap, p}
	}
	if args != "" {
		argv = append(argv, SplitShellArgv(args)...)
	}
	return t.shell(joinShellArgv(argv), 30)
}
