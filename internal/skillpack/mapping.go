package skillpack

import "strings"

// ToolMapping translates pack action vocabulary into Yoyo host tools.
// Skills themselves never name tools; this string is the whole adapter.
func ToolMapping(id, goos string) string {
	id = SanitizeID(id)
	var body string
	switch id {
	case SuperpowersID:
		body = superpowersMapping()
	case NovelToGameID:
		body = novelToGameMapping()
	default:
		body = genericMapping()
	}
	return body + windowsScriptNote(goos)
}

func hostActionTable() string {
	return `**Tool Mapping for Yoyo:**
Skills speak in actions. On Yoyo those resolve to the host tools below. Never invent Claude/Codex names (` + "`Skill`" + `, ` + "`Bash`" + `, ` + "`TodoWrite`" + `, ` + "`Task`" + `).

| Action | Yoyo tool |
|---|---|
| Invoke a skill | ` + "`load_skill`" + ` with ` + "`name`" + ` from the Skills catalog |
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
| Numbered operator choice | ` + "`present_choices`" + ` |
| Dispatch a subagent | ` + "`task`" + ` |
`
}

func superpowersMapping() string {
	return hostActionTable() + `
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
`
}

func novelToGameMapping() string {
	return hostActionTable() + `
**This pack (novel-to-game):** domain workflow, not a methodology bootstrap. Nothing is preloaded at session start. ` + "`load_skill`" + ` ` + "`novel-to-game`" + ` when the operator wants a novel turned into a playable game (or they ` + "`@skill:novel-to-game`" + `).

**Stage skills** — ` + "`load_skill`" + ` the sibling **before** that stage's work:
` + "`novel-to-game`" + ` (orchestrator) · ` + "`novel-game-analyze`" + ` · ` + "`game-concept`" + ` · ` + "`game-world-design`" + ` · ` + "`game-art-direction`" + ` · ` + "`game-build`" + ` · ` + "`game-qa`" + `.

**Contracts** live in each skill's ` + "`references/`" + `. Read them with ` + "`read_skill_file`" + ` (` + "`skill`" + ` = the loaded skill, ` + "`path`" + ` = ` + "`references/…`" + `). They are not in the workspace. This adapter is ` + "`references/yoyo-tools.md`" + ` on ` + "`novel-to-game`" + `.

**Workspace artifacts** go under ` + "`game-adaptations/<project>/`" + `: ` + "`PRODUCT_BRIEF.md`" + `, ` + "`analysis/SOURCE_BIBLE.md`" + `, ` + "`concepts/CONCEPT.md`" + `, ` + "`design/GAME_DESIGN.md`" + `, ` + "`design/ART_DIRECTION.md`" + `, ` + "`build/`" + `, ` + "`qa/verification.json`" + `, ` + "`_progress.md`" + `. Do not write these into Pages.

**Modes:** ` + "`quick`" + ` / ` + "`director`" + ` / ` + "`resume`" + ` follow the orchestrator. High-risk intake (rights, rating, platform) uses ` + "`ask_user`" + `. An unresolved director concept choice uses ` + "`present_choices`" + `. Resume reads ` + "`_progress.md`" + ` and on-disk artifacts — not a chat summary. ` + "`qa/verification.json`" + ` is the only QA fact source.

**Subagents (` + "`task`" + `):** ` + "`profile=explore`" + ` for non-overlapping chapter batches that return fact tables only. Children do not inherit pack bootstrap and return a short ` + "`SUBAGENT_SUMMARY`" + ` — do not send the whole novel or expect a child to finish the pipeline. Depth is capped. Do not invent a ` + "`Task`" + ` tool.

**Playable evidence:** games that load relative JS/CSS/assets cannot run in the inspector HTML preview (srcdoc). Start a loopback static server with ` + "`shell`" + ` (` + "`python -m http.server`" + ` or the brief's start command) and ` + "`browser_open`" + ` the ` + "`http://127.0.0.1:…`" + ` URL so isolated Chrome can drive input and ` + "`browser_screenshot`" + ` can capture evidence. Do not treat inspector srcdoc as play evidence. Do not auto-fallback to a web target when the approved toolchain is missing — record ` + "`targetRuntime`" + ` vs ` + "`testedRuntime`" + `. The six QA checks do not prove fun, balance, rights, or ship quality.

**Skill loading:** ` + "`load_skill`" + ` a stage before acting on it. After compaction, load again if this session has not loaded it. Pack scripts (if any) use ` + "`run_skill_script`" + `. Public HTTP: ` + "`web_fetch`" + ` / ` + "`web_search`" + `. The operator sees the skill name on the live process rail as soon as you ` + "`load_skill`" + `.

**User instructions win:** YOYO.md, AGENTS.md, CLAUDE.md, and a direct skip request override skills.
`
}

func genericMapping() string {
	return hostActionTable() + `
**Skill loading:**
- Every skill: ` + "`load_skill`" + ` **before** acting on it, including clarifying questions.
- After compaction, loaded skill bodies are re-injected. Still ` + "`load_skill`" + ` a skill you have not loaded this session.
- Pack contracts: ` + "`read_skill_file`" + ` ` + "`path=references/yoyo-tools.md`" + ` when that file is listed.
- Pack scripts live in the pack, not ` + "`workspace/scripts/`" + `. Do not glob the workspace for them.
- The operator sees the skill name on the live process rail and companion pulse as soon as you ` + "`load_skill`" + `.

**Subagents (` + "`task`" + `):** ` + "`profile`" + ` is ` + "`explore`" + ` / ` + "`implement`" + ` / ` + "`qa`" + `. The child transcript is not absorbed; you only get ` + "`SUBAGENT_SUMMARY`" + `. Depth is capped. Do not invent a ` + "`Task`" + ` tool.

**User instructions win:** YOYO.md, AGENTS.md, CLAUDE.md, and a direct "skip the skill" request override skills.
`
}

func windowsScriptNote(goos string) string {
	if !strings.EqualFold(goos, "windows") {
		return ""
	}
	return `
**Windows:** Pack ` + "`.sh`" + ` and extensionless scripts run under Git Bash, not WSL ` + "`bash.exe`" + `. If Git Bash is missing, do not call ` + "`run_skill_script`" + ` on those files — follow the text path or use ` + "`web_fetch`" + ` for public HTTP.
`
}

// MappingFile is the install-time generated reference so read_skill_file
// can load the host adapter on demand. It is not vendored upstream.
func MappingFile(id, goos string) string {
	return "# Yoyo Tool Mapping\n\n" + ToolMapping(id, goos)
}
