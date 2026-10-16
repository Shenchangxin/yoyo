package runtime

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

func (t *WorkspaceTools) webSearch(query string) ToolResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return ToolResult{Err: fmt.Errorf("web_search: empty query")}
	}
	if t != nil && t.SearchAPI != nil {
		out, err := t.SearchAPI(query)
		if err == nil && strings.TrimSpace(out) != "" {
			return ToolResult{Content: "search: " + query + "\n\n" + out}
		}
	}
	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	res := t.webFetch(u)
	if res.Err != nil {
		return res
	}
	return ToolResult{Content: "search (fallback scrape): " + query + "\n\n" + res.Content}
}

func (t *WorkspaceTools) askUser(question string) ToolResult {
	question = strings.TrimSpace(question)
	if question == "" {
		return ToolResult{Err: fmt.Errorf("ask_user: empty question")}
	}
	return t.askForm(AskForm{Questions: []AskQuestion{{ID: "q1", Prompt: question}}})
}

func (t *WorkspaceTools) askUserArgs(args map[string]any) ToolResult {
	return t.askForm(parseAskForm(args))
}

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

func (t *WorkspaceTools) applyAllowedFromSkills() {
}
