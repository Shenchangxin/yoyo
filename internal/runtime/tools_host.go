// 把宿主工具名登记到 hostFns。Call 按名字调用这里。git_commit 先 git add -A 再提交。具体实现分散在 tools.go、tools_codex.go、tools_personal.go 和相邻文件。

package runtime

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/tool"
)

// hostFn 是一个宿主工具实现，收到工具、解析后的参数和原始 JSON。
type hostFn func(t *WorkspaceTools, args map[string]any, raw string) ToolResult

// hostFns 是工具名到实现的表。init 填它，Call 按名查找。
var hostFns = map[string]hostFn{}

// init 登记全部内置宿主工具。未登记的名字若在 Extra 里，由 Call 交给扩展工具。
func init() {
	// read_file 按 path、offset、limit 调用 readFile。成功后再 absorbPathRules。
	hostFns["read_file"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		res := t.readFile(str(args["path"]), intArg(args["offset"]), intArg(args["limit"]))
		if res.Err == nil {
			t.absorbPathRules(str(args["path"]))
		}
		return res
	}
	// write_file 在 path 为空时拒绝。content 缺失、带 _elided，或内容像上下文占位时，不把占位符落盘，改走 resolveElidedWrite。mode 为空或 overwrite 调用 writeFile，create 调用 createFile，append 调用 appendFile。单次正文超过 writeFilePartMax（32 * 1024 字节）由 writeFile 和 appendFile 里的 refuseHugeWrite 拒绝。成功后 absorbPathRules。
	hostFns["write_file"] = func(t *WorkspaceTools, args map[string]any, raw string) ToolResult {
		path := str(args["path"])
		if path == "" {
			return ToolResult{Err: missingWriteArg("path", raw)}
		}
		content := str(args["content"])
		if _, ok := args["_elided"]; ok || looksLikeContextStub(content) || looksLikeContextStub(raw) {
			recall := str(args["_recall"])
			if recall == "" {
				recall = str(args["recall"])
			}
			return t.resolveElidedWrite(path, "content", recall)
		}
		if _, ok := args["content"]; !ok {
			return ToolResult{Err: fmt.Errorf("refusing write_file %s: argument \"content\" is missing. Send the full body; for a long file write it in parts with mode=\"append\"", path)}
		}
		var res ToolResult
		switch str(args["mode"]) {
		case "", "overwrite":
			res = t.writeFile(path, content)
		case "create":
			res = t.createFile(path, content)
		case "append":
			res = t.appendFile(path, content)
		default:
			return ToolResult{Err: fmt.Errorf("write_file mode %q is not supported; use overwrite, create, or append", str(args["mode"]))}
		}
		if res.Err == nil {
			t.absorbPathRules(path)
		}
		return res
	}
	// str_replace 先 checkEditArgs，要求 path、old_str、new_str 都在参数里，再调用 replace。成功后 absorbPathRules。
	hostFns["str_replace"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		if err := checkEditArgs("str_replace", args, "path", "old_str", "new_str"); err != nil {
			return ToolResult{Err: err}
		}
		res := t.replace(str(args["path"]), str(args["old_str"]), str(args["new_str"]), boolArg(args["replace_all"]))
		if res.Err == nil {
			t.absorbPathRules(str(args["path"]))
		}
		return res
	}
	// list_dir 在 path 为空时用 .，再调用 listDir。
	hostFns["list_dir"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		p := str(args["path"])
		if p == "" {
			p = "."
		}
		return t.listDir(p)
	}
	// glob 把 pattern 交给 glob。
	hostFns["glob"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.glob(str(args["pattern"]))
	}
	// grep 把 pattern、glob 和 path 交给 grep。
	hostFns["grep"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.grep(str(args["pattern"]), str(args["glob"]), str(args["path"]))
	}
	// shell 把 command 和 timeout_sec 交给 shell。
	hostFns["shell"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.shell(str(args["command"]), intArg(args["timeout_sec"]))
	}
	// load_skill 按 name 调用 loadSkill。
	hostFns["load_skill"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.loadSkill(str(args["name"]))
	}
	// apply_patch 在补丁被省略或像上下文占位时走 resolveElidedWrite，字段名是 patch。否则调用 applyPatch。
	hostFns["apply_patch"] = func(t *WorkspaceTools, args map[string]any, raw string) ToolResult {
		patch := str(args["patch"])
		if _, ok := args["_elided"]; ok || looksLikeContextStub(patch) || looksLikeContextStub(raw) {
			path := str(args["path"])
			if path == "" {
				path = salvageJSONStringField(raw, "path")
			}
			recall := str(args["_recall"])
			if recall == "" {
				recall = str(args["recall"])
			}
			return t.resolveElidedWrite(path, "patch", recall)
		}
		if strings.TrimSpace(patch) == "" {
			return ToolResult{Err: missingWriteArg("patch", raw)}
		}
		return t.applyPatch(patch)
	}
	// git_status 以只读执行 git status --short --branch。能力级别是 ReadWorkspace。
	hostFns["git_status"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.git([]string{"status", "--short", "--branch"}, false)
	}
	// git_diff 在 staged 为真时执行 git diff --cached，否则 git diff。两者都是只读，能力级别是 ReadWorkspace。
	hostFns["git_diff"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		if boolArg(args["staged"]) {
			return t.git([]string{"diff", "--cached"}, false)
		}
		return t.git([]string{"diff"}, false)
	}
	// git_commit 要求 message 非空。先以写权限执行 git add -A，失败直接返回；成功后再 git commit -m。add -A 会把工作区里已有的脏文件一起暂存并提交。write 为真时能力级别是 WriteWorkspace，动作名是 git。
	hostFns["git_commit"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		msg := str(args["message"])
		if msg == "" {
			return ToolResult{Err: fmt.Errorf("empty commit message")}
		}
		// add -A 暂存工作区里全部改动，包括这次任务没碰过的脏文件。失败就不再 commit。
		if res := t.git([]string{"add", "-A"}, true); res.Err != nil {
			return res
		}
		return t.git([]string{"commit", "-m", msg}, true)
	}
	// recall_context 按 id、offset、limit 调用 recall。
	hostFns["recall_context"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.recall(str(args["id"]), intArg(args["offset"]), intArg(args["limit"]))
	}
	// tool_search 按 query 调用 toolSearch。
	hostFns["tool_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.toolSearch(str(args["query"]))
	}
	// task 在 prompts 多于一条时调用 taskFanout，否则调用 task。isolate 和 profile 原样传下去。
	hostFns["task"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		profile := str(args["profile"])
		if prompts := anyStrings(args["prompts"]); len(prompts) > 1 {
			return t.taskFanout(prompts, boolArg(args["isolate"]), profile)
		}
		return t.task(str(args["prompt"]), boolArg(args["isolate"]), profile)
	}
	// read_thread 把 session_id、query、message 交给 readThread。
	hostFns["read_thread"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.readThread(str(args["session_id"]), str(args["query"]), str(args["message"]))
	}
	// update_plan 把原始 JSON 交给 updatePlan。
	hostFns["update_plan"] = func(t *WorkspaceTools, _ map[string]any, raw string) ToolResult {
		return t.updatePlan(raw)
	}
	// wait 把 seconds 交给 wait。
	hostFns["wait"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.wait(intArg(args["seconds"]))
	}
	// list_skills 调用 listSkills。
	hostFns["list_skills"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.listSkills()
	}
	// view_image 按 path 调用 viewImage。
	hostFns["view_image"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.viewImage(str(args["path"]))
	}
	// web_fetch 按 url 调用 webFetch。
	hostFns["web_fetch"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.webFetch(str(args["url"]))
	}
	// web_search 按 query 调用 webSearch。
	hostFns["web_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.webSearch(str(args["query"]))
	}
	// reach_status 调用 reachStatus。
	hostFns["reach_status"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.reachStatus()
	}
	// ask_user 把参数交给 askUserArgs。
	hostFns["ask_user"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.askUserArgs(args)
	}
	// run_skill_script 把 skill、script、args 交给 runSkillScript。
	hostFns["run_skill_script"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.runSkillScript(str(args["skill"]), str(args["script"]), str(args["args"]))
	}
	// read_skill_file 把 skill、path、offset、limit 交给 readSkillFile。
	hostFns["read_skill_file"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.readSkillFile(str(args["skill"]), str(args["path"]), intArg(args["offset"]), intArg(args["limit"]))
	}
	// office_create 把参数交给 officeCreate。
	hostFns["office_create"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeCreate(args)
	}
	// office_edit 先 checkEditArgs，要求 path、old_str、new_str，再调用 officeEdit。
	hostFns["office_edit"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		if err := checkEditArgs("office_edit", args, "path", "old_str", "new_str"); err != nil {
			return ToolResult{Err: err}
		}
		return t.officeEdit(str(args["path"]), str(args["old_str"]), str(args["new_str"]))
	}
	// office_query 按 path 调用 officeQuery。
	hostFns["office_query"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeQuery(str(args["path"]))
	}
	// office_render 按 path 调用 officeRender。
	hostFns["office_render"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.officeRender(str(args["path"]))
	}
	// cite_sources 把 path 和原始 JSON 交给 citeSources。
	hostFns["cite_sources"] = func(t *WorkspaceTools, args map[string]any, raw string) ToolResult {
		return t.citeSources(str(args["path"]), raw)
	}
	// memory_search 按 query 和 kind 调用 memorySearch。
	hostFns["memory_search"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.memorySearch(str(args["query"]), str(args["kind"]))
	}
	// memory_write 按 kind、text、project 调用 memoryWrite。
	hostFns["memory_write"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.memoryWrite(str(args["kind"]), str(args["text"]), str(args["project"]))
	}
	// memory_forget 按 id 调用 memoryForget。
	hostFns["memory_forget"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.memoryForget(str(args["id"]))
	}
	// schedule_create 按 kind、spec、prompt 调用 scheduleCreate。
	hostFns["schedule_create"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.scheduleCreate(str(args["kind"]), str(args["spec"]), str(args["prompt"]))
	}
	// schedule_list 调用 scheduleList。
	hostFns["schedule_list"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.scheduleList()
	}
	// schedule_cancel 按 id 调用 scheduleCancel。
	hostFns["schedule_cancel"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.scheduleCancel(str(args["id"]))
	}
	// browser_open 优先用 path 当本地页。path 非空时不用 url 覆盖它，模型经常两个都填，而 url 为 http://localhost/ 并不是服务器。lane 和 profile 会追加到参数串上，再交给 browserOpen。
	hostFns["browser_open"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		// 工作区路径才是本地页。path 非空时不用 url 覆盖它。模型经常两个都填，而 url 为 http://localhost/ 并不是一台服务器。
		raw := strings.TrimSpace(str(args["path"]))
		if raw == "" {
			raw = strings.TrimSpace(str(args["url"]))
		}
		if lane := str(args["lane"]); lane != "" {
			raw = strings.TrimSpace(raw + " lane=" + lane)
		}
		if profile := str(args["profile"]); profile != "" {
			raw = strings.TrimSpace(raw + " profile=" + profile)
		}
		return t.browserOpen(raw)
	}
	// browser_snapshot 按 level 调用 browserSnapshot。
	hostFns["browser_snapshot"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserSnapshot(str(args["level"]))
	}
	// browser_click 用 browserHandle 取目标，再调用 browserClick。
	hostFns["browser_click"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserClick(browserHandle(args))
	}
	// browser_type 把目标和 text 交给 browserType。
	hostFns["browser_type"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserType(browserHandle(args), str(args["text"]))
	}
	// browser_fill 把目标和 text 交给 browserFill。
	hostFns["browser_fill"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserFill(browserHandle(args), str(args["text"]))
	}
	// browser_scroll 把 direction、amount 和目标交给 browserScroll。
	hostFns["browser_scroll"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserScroll(str(args["direction"]), intArg(args["amount"]), browserHandle(args))
	}
	// browser_hover 把目标交给 browserHover。
	hostFns["browser_hover"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserHover(browserHandle(args))
	}
	// browser_select 用 value，空则用 text，再交给 browserSelect。
	hostFns["browser_select"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		val := str(args["value"])
		if val == "" {
			val = str(args["text"])
		}
		return t.browserSelect(browserHandle(args), val)
	}
	// browser_press 按 key 调用 browserPress。
	hostFns["browser_press"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserPress(str(args["key"]))
	}
	// browser_tabs 把 op、tab_id、url 交给 browserTabs。
	hostFns["browser_tabs"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserTabs(str(args["op"]), str(args["tab_id"]), str(args["url"]))
	}
	// browser_profile 把 op 和 name 交给 browserProfile。
	hostFns["browser_profile"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserProfile(str(args["op"]), str(args["name"]))
	}
	// browser_record 把 op 和 name 交给 browserRecord。
	hostFns["browser_record"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserRecord(str(args["op"]), str(args["name"]))
	}
	// browser_download 把 url 和 path 交给 browserDownload。
	hostFns["browser_download"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserDownload(str(args["url"]), str(args["path"]))
	}
	// browser_screenshot 按 path 调用 browserScreenshot。
	hostFns["browser_screenshot"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserScreenshot(str(args["path"]))
	}
	// browser_act 把参数交给 browserAct。
	hostFns["browser_act"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserAct(args)
	}
	// browser_verify 把参数交给 browserVerify。
	hostFns["browser_verify"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserVerify(args)
	}
	// browser_takeover 按 question 调用 browserTakeover。
	hostFns["browser_takeover"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserTakeover(str(args["question"]))
	}
	// browser_cookies 按 path 调用 browserCookies。
	hostFns["browser_cookies"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.browserCookies(str(args["path"]))
	}
	// clipboard_read 调用 clipboardRead。
	hostFns["clipboard_read"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.clipboardRead()
	}
	// clipboard_write 按 text 调用 clipboardWrite。
	hostFns["clipboard_write"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.clipboardWrite(str(args["text"]))
	}
	// screenshot_region 按 path 调用 screenshot。
	hostFns["screenshot_region"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.screenshot(str(args["path"]))
	}
	// fs_batch 把 from 和 to 交给 fsBatch。
	hostFns["fs_batch"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.fsBatch(str(args["from"]), str(args["to"]))
	}
	// connector_read 把 account 和 query 交给 connectorRead。
	hostFns["connector_read"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.connectorRead(str(args["account"]), str(args["query"]))
	}
	// connector_draft 把参数交给 connectorDraft。
	hostFns["connector_draft"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.connectorDraft(args)
	}
	// connector_send 按 draft_id 调用 connectorSend。
	hostFns["connector_send"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.connectorSend(str(args["draft_id"]))
	}
	// computer_act 把 app、op、detail 交给 computerAct。
	hostFns["computer_act"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.computerAct(str(args["app"]), str(args["op"]), str(args["detail"]))
	}
	// project_list 调用 projectList。
	hostFns["project_list"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.projectList()
	}
	// notify_actionable 把 title 和 body 交给 notifyActionable。
	hostFns["notify_actionable"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.notifyActionable(str(args["title"]), str(args["body"]))
	}
	// delegate_work 把参数交给 delegateWork。
	hostFns["delegate_work"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.delegateWork(args)
	}
	// complete_work 把 id 和 result 交给 completeWork。
	hostFns["complete_work"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.completeWork(str(args["id"]), str(args["result"]))
	}
	// answer_work 把 id 和 text 交给 answerWork。
	hostFns["answer_work"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.answerWork(str(args["id"]), str(args["text"]))
	}
	// create_goal 把 title、description 和 milestones 交给 createGoal。
	hostFns["create_goal"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.createGoal(str(args["title"]), str(args["description"]), anyStrings(args["milestones"]))
	}
	// watch_page 把参数交给 watchPage。
	hostFns["watch_page"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.watchPage(args)
	}
	// present_choices 把 title 和 options 交给 presentChoices。
	hostFns["present_choices"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.presentChoices(str(args["title"]), args["options"])
	}
	// personal_status 调用 personalStatus。
	hostFns["personal_status"] = func(t *WorkspaceTools, _ map[string]any, _ string) ToolResult {
		return t.personalStatus()
	}
	// remember_fact 按 text 调用 rememberFact。
	hostFns["remember_fact"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.rememberFact(str(args["text"]))
	}
	// read_page 把 id 和 space_id 交给 readPage。
	hostFns["read_page"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.readPage(str(args["id"]), str(args["space_id"]))
	}
	// list_pages 把 query 和 space_id 交给 listPages。
	hostFns["list_pages"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.listPages(str(args["query"]), str(args["space_id"]))
	}
	// review_page 把参数交给 reviewPage。
	hostFns["review_page"] = func(t *WorkspaceTools, args map[string]any, _ string) ToolResult {
		return t.reviewPage(args)
	}
}

// specJSON 用工具规格的名字、说明和参数构造 ToolJSON。
func specJSON(s artifact.ToolSpec) ToolJSON {
	return fn(s.Name, s.Description, s.Parameters)
}

// HostAnn 从 HostSpecMap 取该名字的注解。没有则返回空注解。
// name 用来查 HostSpecMap；没有则返回空注解。
func HostAnn(name string) tool.Annotations {
	if s, ok := tool.HostSpecMap()[name]; ok {
		return tool.AnnFromSpec(s)
	}
	return tool.Annotations{}
}
