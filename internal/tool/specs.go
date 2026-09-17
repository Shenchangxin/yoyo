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
		spec("recall_context", "Retrieve a previously elided tool result by id from the context spill store.", "", obj(map[string]any{"id": strP()}, "id"), read),
		spec("tool_search", "Look up extra (MCP/WASM) tool schemas by substring. Use when extra tools were deferred.", "", obj(map[string]any{"query": strP()}, "query"), read),
		spec("task", "Run a subagent on a prompt. Returns a summary only (child transcript is isolated). Set isolate=true to copy/worktree the workspace.", "", obj(map[string]any{"prompt": strP(), "isolate": boolP()}, "prompt"), Annotations{}),
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
		spec("run_skill_script", "Run a script from a loaded skill's scripts/ directory inside the workspace jail.", "shell", obj(map[string]any{"skill": strP(), "script": strP(), "args": strP()}, "skill", "script"), shell),
	}
}

func HostSpecMap() map[string]artifact.ToolSpec {
	m := map[string]artifact.ToolSpec{}
	for _, s := range HostSpecs() {
		m[s.Name] = s
	}
	return m
}
