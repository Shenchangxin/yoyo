package skillpack

import "strings"

// ToolMapping translates Superpowers action vocabulary into Yoyo host tools.
// Skills themselves never name tools; this string is the whole adapter.
func ToolMapping(id, goos string) string {
	id = SanitizeID(id)
	if id != SuperpowersID && id != "" {
		return genericMapping(goos)
	}
	var b strings.Builder
	b.WriteString(`**Tool Mapping for Yoyo:**
Skills speak in actions. On Yoyo those resolve to the host tools below. Never invent Claude/Codex names (` + "`Skill`" + `, ` + "`Bash`" + `, ` + "`TodoWrite`" + `, ` + "`Task`" + `).

| Action | Yoyo tool |
|---|---|
| Invoke a skill | ` + "`load_skill`" + ` with ` + "`name`" + ` from the Skills catalog (example: ` + "`brainstorming`" + `) |
| List skills | The Skills catalog is already pinned. ` + "`list_skills`" + ` if you need names only |
| Read a workspace file | ` + "`read_file`" + ` |
| Read a file inside a skill pack (` + "`references/`" + `, templates, prompts) | ` + "`read_skill_file`" + ` with ` + "`skill`" + ` + ` + "`path`" + ` (pack dirs are outside the workspace jail) |
| Create / overwrite a workspace file | ` + "`write_file`" + ` |
| Targeted edit | ` + "`str_replace`" + ` (preferred) or ` + "`apply_patch`" + ` |
| List / find / search files | ` + "`list_dir`" + `, ` + "`glob`" + `, ` + "`grep`" + ` |
| Run a shell command | ` + "`shell`" + `. Prefer grep/glob/read_file for reading source |
| Run a pack helper | ` + "`run_skill_script`" + ` (` + "`skill`" + `, ` + "`script`" + ` under ` + "`scripts/`" + `). Invoke through the interpreter even when the file has no extension |
| Fetch a URL / search the web | ` + "`web_fetch`" + ` / ` + "`web_search`" + ` — not shell curl |
| Create or update todos | ` + "`update_plan`" + ` (statuses: pending, in_progress, complete; one in_progress) |
| Ask the human | ` + "`ask_user`" + ` |
| Dispatch a subagent | ` + "`task`" + ` |

**Subagents (` + "`task`" + `):**
- ` + "`profile`" + `: ` + "`explore`" + ` (read-only), ` + "`implement`" + `, ` + "`qa`" + ` (opt-in).
- ` + "`isolate=true`" + ` copies/worktrees the workspace — prefer this over ` + "`git worktree add`" + ` for ` + "`using-git-worktrees`" + `.
- ` + "`prompts[]`" + ` fans out independent tasks (max_parallel, default 4).
- The child transcript is **not** absorbed; you only get ` + "`SUBAGENT_SUMMARY`" + `.
- There is no mailbox ` + "`followup_task`" + ` / ` + "`wait_agent`" + `. To continue a child, dispatch ` + "`task`" + ` again with the next instruction, or do the work in this session.
- Depth is capped. Do not invent a ` + "`Task`" + ` tool.

**Skill loading:**
- ` + "`using-superpowers`" + ` is already in context when this mapping is present. Do **not** ` + "`load_skill`" + ` it again.
- Every other skill: ` + "`load_skill`" + ` **before** any other action, including clarifying questions and repo exploration.
- After compaction, loaded skill bodies are re-injected. Still ` + "`load_skill`" + ` a skill you have not loaded this session.
- Pack scripts live in the pack, not ` + "`workspace/scripts/`" + `. Do not glob the workspace for them.
- The operator sees the skill name on the live process rail and companion pulse as soon as you ` + "`load_skill`" + `.

**Brainstorming visual companion:** optional. The Node ` + "`start-server.sh`" + ` helper is not required on the Yoyo desktop. Follow the text path in the skill (classify, write back understanding, get approval) unless the operator asked for the visual companion.

**User instructions win:** YOYO.md, AGENTS.md, CLAUDE.md, and a direct "skip the skill" request override skills. Default Yoyo "start writing this turn" does **not** override a matching process skill.
`)
	if strings.EqualFold(goos, "windows") {
		b.WriteString(`
**Windows:** Pack ` + "`.sh`" + ` and extensionless scripts run under Git Bash, not WSL ` + "`bash.exe`" + `. If Git Bash is missing, do not call ` + "`run_skill_script`" + ` on those files — follow the text path or use ` + "`web_fetch`" + ` for public HTTP.
`)
	}
	return b.String()
}

func genericMapping(goos string) string {
	return ToolMapping(SuperpowersID, goos)
}

// MappingFile is the install-time generated reference dropped next to
// using-superpowers so read_skill_file can load it on demand.
func MappingFile(id, goos string) string {
	return "# Yoyo Tool Mapping\n\n" + ToolMapping(id, goos)
}
