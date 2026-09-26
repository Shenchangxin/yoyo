package tool

import "github.com/Shenchangxin/yoyo/internal/artifact"

func spec(name, desc, cap string, params map[string]any, ann Annotations) artifact.ToolSpec {
	return artifact.ToolSpec{
		Name:            name,
		Description:     desc,
		Parameters:      params,
		Capability:      cap,
		Impl:            artifact.ToolImplHost,
		ReadOnly:        ann.ReadOnly,
		ConcurrencySafe: ann.ConcurrencySafe,
		OpenWorld:       ann.OpenWorld,
		Destructive:     ann.Destructive,
		Exclusive:       ann.Exclusive,
	}
}

func obj(props map[string]any, required ...string) map[string]any {
	m := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		m["required"] = required
	}
	return m
}

func strP() map[string]any  { return map[string]any{"type": "string"} }
func intP() map[string]any  { return map[string]any{"type": "integer"} }
func boolP() map[string]any { return map[string]any{"type": "boolean"} }

// HostSpecs is the canonical CAS-versioned builtin catalog.
func HostSpecs() []artifact.ToolSpec {
	read := Annotations{ReadOnly: true, ConcurrencySafe: true}
	write := Annotations{}
	shell := Annotations{Exclusive: true, OpenWorld: true, Destructive: true}
	net := Annotations{OpenWorld: true, Exclusive: true}
	return []artifact.ToolSpec{
		spec("read_file", "Read a UTF-8 file under the workspace. Optional offset/limit are 1-based line numbers.", "read_workspace", obj(map[string]any{"path": strP(), "offset": intP(), "limit": intP()}, "path"), read),
		spec("write_file", "Write a UTF-8 file under the workspace.", "write_workspace", obj(map[string]any{"path": strP(), "content": strP()}, "path", "content"), write),
		spec("str_replace", "Replace old_str with new_str. Fails if old_str is not unique unless replace_all is true.", "write_workspace", obj(map[string]any{"path": strP(), "old_str": strP(), "new_str": strP(), "replace_all": boolP()}, "path", "old_str", "new_str"), write),
		spec("list_dir", "List a directory under the workspace.", "read_workspace", obj(map[string]any{"path": strP()}), read),
		spec("glob", "Find files by glob pattern relative to the workspace (e.g. **/*.go).", "read_workspace", obj(map[string]any{"pattern": strP()}, "pattern"), read),
		spec("grep", "Search file contents with a regex. Optional glob limits which files are scanned.", "read_workspace", obj(map[string]any{"pattern": strP(), "glob": strP(), "path": strP()}, "pattern"), read),
		spec("shell", "Run a shell command in the workspace directory. Prefer glob/grep/read_file when possible.", "shell", obj(map[string]any{"command": strP(), "timeout_sec": intP()}, "command"), shell),
		spec("load_skill", "Load a skill body by name into context.", "", obj(map[string]any{"name": strP()}, "name"), read),
		spec("apply_patch", "Apply a Codex-style patch (*** Begin Patch / *** Add File or *** Update File / *** End Patch).", "write_workspace", obj(map[string]any{"patch": strP()}, "patch"), write),
		spec("git_status", "Show git status of the workspace.", "read_workspace", obj(map[string]any{}), read),
		spec("git_diff", "Show git diff. Optional staged=true for --cached.", "read_workspace", obj(map[string]any{"staged": boolP()}), read),
		spec("git_commit", "Stage all and commit with a message. Prefer this over shell git.", "write_workspace", obj(map[string]any{"message": strP()}, "message"), write),
		spec("recall_context", "Retrieve a previously elided tool result by id. Always pass offset/limit (1-based lines, default 200). Full bytes stay in spill.", "", obj(map[string]any{"id": strP(), "offset": intP(), "limit": intP()}, "id"), read),
		spec("tool_search", "Look up extra (MCP/WASM) tool schemas by substring. Use when extra tools were deferred.", "", obj(map[string]any{"query": strP()}, "query"), read),
		spec("task", "Run a subagent on a prompt. Returns a summary only (child transcript is isolated). Set isolate=true to copy/worktree the workspace. prompts[] fans out up to max_parallel (L3, default 4). profile is explore | implement | qa (qa is opt-in).", "", obj(map[string]any{"prompt": strP(), "isolate": boolP(), "profile": strP(), "prompts": map[string]any{"type": "array", "items": strP()}}, "prompt"), Annotations{}),
		spec("update_plan", "Replace the current task plan with a list of steps and statuses (pending, in_progress, complete).", "", obj(map[string]any{
			"explanation": strP(),
			"plan": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"step":   strP(),
						"status": strP(),
					},
					"required": []string{"step", "status"},
				},
			},
		}, "plan"), read),
		spec("wait", "Pause up to 30 seconds before the next action. Use when polling a command or a file.", "", obj(map[string]any{"seconds": intP()}), read),
		spec("list_skills", "List skill names available via load_skill.", "", obj(map[string]any{}), read),
		spec("view_image", "Inspect an image file under the workspace (size and type). Prefer this over dumping binary via read_file.", "read_workspace", obj(map[string]any{"path": strP()}, "path"), read),
		spec("web_fetch", "HTTP GET a public https URL and return extracted text. Never used for localhost or private IPs. Requires network approval.", "network", obj(map[string]any{"url": strP()}, "url"), net),
		spec("web_search", "Search the public web and return extracted result text. Requires network approval.", "network", obj(map[string]any{"query": strP()}, "query"), net),
		spec("ask_user", "Ask the operator a short question and wait for the answer.", "", obj(map[string]any{"question": strP()}, "question"), read),
		spec("run_skill_script", "Run a helper from a loaded skill pack's scripts/ directory. Pass skill name and script filename (optionally scripts/...). The pack is not inside the workspace.", "shell", obj(map[string]any{"skill": strP(), "script": strP(), "args": strP()}, "skill", "script"), shell),
		spec("office_create", "Create a real .docx, .xlsx, .pptx, or .pdf under the workspace. Spreadsheets accept formula cells starting with =.", "write_workspace", obj(map[string]any{"path": strP(), "kind": strP(), "title": strP(), "body": strP(), "headings": map[string]any{"type": "array", "items": strP()}, "rows": map[string]any{"type": "array"}, "slides": map[string]any{"type": "array", "items": strP()}}, "path"), write),
		spec("office_edit", "Replace text inside an office document without rewriting the whole package (keeps OOXML structure).", "write_workspace", obj(map[string]any{"path": strP(), "old_str": strP(), "new_str": strP()}, "path", "old_str", "new_str"), write),
		spec("office_query", "Extract text or cells from a workspace office document.", "read_workspace", obj(map[string]any{"path": strP()}, "path"), read),
		spec("office_render", "Render an office document to text for look-fix.", "read_workspace", obj(map[string]any{"path": strP()}, "path"), read),
		spec("cite_sources", "Write citations.json next to a report. Each source needs url, excerpt, and hash.", "write_workspace", obj(map[string]any{"path": strP(), "sources": map[string]any{"type": "array"}}, "path", "sources"), write),
		spec("memory_search", "Search long-term memory (profile, project, episodic). Staging items are included.", "", obj(map[string]any{"query": strP(), "kind": strP()}, "query"), read),
		spec("memory_write", "Stage a memory item. Promotion still requires Harbor/operator checkout.", "memory_write", obj(map[string]any{"kind": strP(), "text": strP(), "project": strP()}, "text"), write),
		spec("memory_forget", "Delete a memory item by id.", "memory_write", obj(map[string]any{"id": strP()}, "id"), write),
		spec("schedule_create", "Create a cron, once, heartbeat, or webhook job. Jobs run in an isolated worktree on this awake machine and never send_as_you without a later approval.", "schedule", obj(map[string]any{"kind": strP(), "spec": strP(), "prompt": strP()}, "prompt"), write),
		spec("schedule_list", "List scheduled jobs.", "schedule", obj(map[string]any{}), read),
		spec("schedule_cancel", "Cancel a scheduled job by id.", "schedule", obj(map[string]any{"id": strP()}, "id"), write),
		spec("browser_open", "Open a URL in the isolated browser profile (not the operator Chrome). Pass lane=attached to attach to a debug Chrome on port 9222.", "browser", obj(map[string]any{"url": strP(), "lane": strP()}, "url"), net),
		spec("browser_snapshot", "Return the last isolated-browser snapshot text.", "browser", obj(map[string]any{}), read),
		spec("browser_click", "Click a selector in the isolated browser. Requires CDP/Chrome.", "browser", obj(map[string]any{"selector": strP()}, "selector"), net),
		spec("browser_type", "Type into a selector in the isolated browser.", "browser", obj(map[string]any{"selector": strP(), "text": strP()}, "selector", "text"), net),
		spec("browser_fill", "Fill a selector in the isolated browser.", "browser", obj(map[string]any{"selector": strP(), "text": strP()}, "selector", "text"), net),
		spec("browser_download", "Download a URL into the workspace as raw bytes via the isolated browser.", "browser", obj(map[string]any{"url": strP(), "path": strP()}, "url", "path"), net),
		spec("browser_screenshot", "Capture the isolated browser page (not the operator desktop) into a workspace PNG.", "browser", obj(map[string]any{"path": strP()}), net),
		spec("browser_takeover", "Pause and show a headed Yoyo browser so the operator can finish login, MFA, or CAPTCHA, then resume.", "browser", obj(map[string]any{"question": strP()}), net),
		spec("browser_cookies", "Import a Netscape cookies.txt into the isolated browser profile. Does not touch the operator Chrome.", "browser", obj(map[string]any{"path": strP()}, "path"), net),
		spec("clipboard_read", "Read the OS clipboard into the turn. Contents are untrusted. Always asks unless this session already granted clipboard.", "clipboard", obj(map[string]any{}), read),
		spec("clipboard_write", "Write text to the OS clipboard.", "clipboard", obj(map[string]any{"text": strP()}, "text"), write),
		spec("screenshot_region", "Capture the virtual computer-use display into a workspace PNG. Never the operator primary screen.", "computer_use", obj(map[string]any{"path": strP()}), Annotations{Exclusive: true, OpenWorld: true, Destructive: true}),
		spec("fs_batch", "Rename or move files inside an authorized workspace directory.", "write_workspace", obj(map[string]any{"from": strP(), "to": strP()}, "from", "to"), write),
		spec("connector_read", "Read items from a connected mail/calendar/drive/IM account.", "read_connector", obj(map[string]any{"account": strP(), "query": strP()}, "account"), read),
		spec("connector_draft", "Draft a mail or calendar event on a connected account. Calendar drafts use subject as title, to as RFC3339 start or attendee. Sending is a separate approval.", "write_connector", obj(map[string]any{"account": strP(), "to": strP(), "subject": strP(), "body": strP()}, "account", "body"), write),
		spec("connector_send", "Send a previously drafted mail or calendar event as the operator.", "send_as_you", obj(map[string]any{"draft_id": strP()}, "draft_id"), Annotations{Exclusive: true, OpenWorld: true, Destructive: true}),
		spec("computer_act", "Issue a virtual-display action against an allowlisted app. Never the operator desktop.", "computer_use", obj(map[string]any{"app": strP(), "op": strP(), "detail": strP()}, "op"), Annotations{Exclusive: true, OpenWorld: true, Destructive: true}),
		spec("project_list", "List personal-OS projects (not git repos).", "", obj(map[string]any{}), read),
		spec("notify_actionable", "Push an Inbox item the operator can act on. Does not send mail.", "", obj(map[string]any{"title": strP(), "body": strP()}, "title"), write),
		spec("read_thread", "Extract snippets from another session, list sessions when session_id is empty, or pass message to steer/enqueue that session. Does not dump the full thread.", "", obj(map[string]any{"session_id": strP(), "query": strP(), "message": strP()}), read),
	}
}

func HostSpecMap() map[string]artifact.ToolSpec {
	m := map[string]artifact.ToolSpec{}
	for _, s := range HostSpecs() {
		m[s.Name] = s
	}
	return m
}
