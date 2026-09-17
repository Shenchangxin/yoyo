package runtime

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/tool"
)

type hostFn func(t *WorkspaceTools, args map[string]any, raw string) ToolResult

var hostFns = map[string]hostFn{}

func init() {
	hostFns["read_file"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.readFile(str(args["path"]), intArg(args["offset"]), intArg(args["limit"]))
	}
	hostFns["write_file"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.writeFile(str(args["path"]), str(args["content"]))
	}
	hostFns["str_replace"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.replace(str(args["path"]), str(args["old_str"]), str(args["new_str"]), boolArg(args["replace_all"]))
	}
	hostFns["list_dir"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		p := str(args["path"])
		if p == "" {
			p = "."
		}
		return t.listDir(p)
	}
	hostFns["glob"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.glob(str(args["pattern"]))
	}
	hostFns["grep"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.grep(str(args["pattern"]), str(args["glob"]), str(args["path"]))
	}
	hostFns["shell"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.shell(str(args["command"]), intArg(args["timeout_sec"]))
	}
	hostFns["load_skill"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.loadSkill(str(args["name"]))
	}
	hostFns["apply_patch"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.applyPatch(str(args["patch"]))
	}
	hostFns["git_status"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.git([]string{"status", "--short", "--branch"}, false)
	}
	hostFns["git_diff"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		if boolArg(args["staged"]) {
			return t.git([]string{"diff", "--cached"}, false)
		}
		return t.git([]string{"diff"}, false)
	}
	hostFns["git_commit"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		msg := str(args["message"])
		if msg == "" {
			return ToolResult{Err: fmt.Errorf("empty commit message")}
		}
		if res := t.git([]string{"add", "-A"}, true); res.Err != nil {
			return res
		}
		return t.git([]string{"commit", "-m", msg}, true)
	}
	hostFns["recall_context"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.recall(str(args["id"]))
	}
	hostFns["tool_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.toolSearch(str(args["query"]))
	}
	hostFns["task"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.task(str(args["prompt"]), boolArg(args["isolate"]))
	}
	hostFns["update_plan"] = func(t *WorkspaceTools, _ map[string]any, raw string) ToolResult {
		return t.updatePlan(raw)
	}
	hostFns["wait"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.wait(intArg(args["seconds"]))
	}
	hostFns["list_skills"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.listSkills()
	}
	hostFns["view_image"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.viewImage(str(args["path"]))
	}
	hostFns["web_fetch"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.webFetch(str(args["url"]))
	}
	hostFns["web_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.webSearch(str(args["query"]))
	}
	hostFns["ask_user"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.askUser(str(args["question"]))
	}
	hostFns["run_skill_script"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.runSkillScript(str(args["skill"]), str(args["script"]), str(args["args"]))
	}
}

func specJSON(s artifact.ToolSpec) ToolJSON {
	return fn(s.Name, s.Description, s.Parameters)
}

func HostAnn(name string) tool.Annotations {
	if s, ok := tool.HostSpecMap()[name]; ok {
		return tool.AnnFromSpec(s)
	}
	return tool.Annotations{}
}
