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
		return t.recall(str(args["id"]), intArg(args["offset"]), intArg(args["limit"]))
	}
	hostFns["tool_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.toolSearch(str(args["query"]))
	}
	hostFns["task"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		if t != nil {
			t.mu.Lock()
			t.taskProfile = str(args["profile"])
			t.mu.Unlock()
		}
		if prompts := anyStrings(args["prompts"]); len(prompts) > 1 {
			return t.taskFanout(prompts, boolArg(args["isolate"]))
		}
		return t.task(str(args["prompt"]), boolArg(args["isolate"]))
	}
	hostFns["read_thread"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.readThread(str(args["session_id"]), str(args["query"]))
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
	hostFns["office_create"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeCreate(args)
	}
	hostFns["office_edit"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeEdit(str(args["path"]), str(args["old_str"]), str(args["new_str"]))
	}
	hostFns["office_query"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeQuery(str(args["path"]))
	}
	hostFns["office_render"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeRender(str(args["path"]))
	}
	hostFns["cite_sources"] = func(t *WorkspaceTools, args map[string]any, raw string) ToolResult {
		return t.citeSources(str(args["path"]), raw)
	}
	hostFns["memory_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.memorySearch(str(args["query"]), str(args["kind"]))
	}
	hostFns["memory_write"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.memoryWrite(str(args["kind"]), str(args["text"]), str(args["project"]))
	}
	hostFns["memory_forget"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.memoryForget(str(args["id"]))
	}
	hostFns["schedule_create"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.scheduleCreate(str(args["kind"]), str(args["spec"]), str(args["prompt"]))
	}
	hostFns["schedule_list"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.scheduleList()
	}
	hostFns["schedule_cancel"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.scheduleCancel(str(args["id"]))
	}
	hostFns["browser_open"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserOpen(str(args["url"]))
	}
	hostFns["browser_snapshot"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.browserSnapshot()
	}
	hostFns["browser_click"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserClick(str(args["selector"]))
	}
	hostFns["browser_type"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserType(str(args["selector"]), str(args["text"]))
	}
	hostFns["browser_fill"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserType(str(args["selector"]), str(args["text"]))
	}
	hostFns["browser_download"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserDownload(str(args["url"]), str(args["path"]))
	}
	hostFns["clipboard_read"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.clipboardRead()
	}
	hostFns["clipboard_write"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.clipboardWrite(str(args["text"]))
	}
	hostFns["screenshot_region"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.screenshot(str(args["path"]))
	}
	hostFns["fs_batch"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.fsBatch(str(args["from"]), str(args["to"]))
	}
	hostFns["connector_read"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.connectorRead(str(args["account"]), str(args["query"]))
	}
	hostFns["connector_draft"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.connectorDraft(str(args["account"]), str(args["to"]), str(args["subject"]), str(args["body"]))
	}
	hostFns["connector_send"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.connectorSend(str(args["draft_id"]))
	}
	hostFns["computer_act"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.computerAct(str(args["app"]), str(args["op"]), str(args["detail"]))
	}
	hostFns["project_list"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.projectList()
	}
	hostFns["notify_actionable"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.notifyActionable(str(args["title"]), str(args["body"]))
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
