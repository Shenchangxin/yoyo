// 把开场上下文拆成可计量的层。系统、规则、目录和记忆在稳定前缀里；技能正文、今日和笔记是动态尾，由调用方接到历史后面。
// 真实对话从历史消息计，不从这份账单计。playbook 写进 pins，不写进动态尾。

package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// contextBill 是开场上下文的分层文本。systemWire 和 pinsWire 才是放进消息的两段；System 额外带上 playbook 和 ladder，只给 IdentityTokens 计数。
type contextBill struct {
	System  string
	Rules   string
	Catalog string
	Memory  string
	Bodies  string
	Today   string
	Notes   string

	systemWire string
	pinsWire   string
}

// prefix 返回系统消息和 pins 接在一起的稳定前缀。
func (b contextBill) prefix() string { return b.systemWire + b.pinsWire }

// dynamic 返回技能正文、今日和笔记，不含 playbook。
func (b contextBill) dynamic() string { return b.Bodies + b.Today + b.Notes }

// write 把各层的估算 token 写进账本，并重写前缀和动态尾的 token。
// rep 为 nil 时直接返回，否则写入各层估算 token。
func (b contextBill) write(rep *ShapeReport) {
	if rep == nil {
		return
	}
	rep.IdentityTokens = estimateTokens(b.System)
	rep.RulesTokens = estimateTokens(b.Rules)
	rep.CatalogTokens = estimateTokens(b.Catalog)
	rep.SkillBodyTokens = estimateTokens(b.Bodies)
	rep.MemoryTokens = estimateTokens(b.Memory)
	rep.TodayTokens = estimateTokens(b.Today)
	rep.NotesTokens = estimateTokens(b.Notes)
	rep.PrefixTokens = estimateTokens(b.prefix())
	rep.DynamicTokens = estimateTokens(b.dynamic())
}

// billInput 是 buildBill 的材料。Checkpoint 和 Suggested 只进入笔记段。
type billInput struct {
	Loop      artifact.LoopPreset
	Fragments []artifact.PromptFragment
	Playbook  artifact.Playbook
	Skills    []artifact.Skill
	Workspace string
	Tools     *WorkspaceTools
	// Page 是当前页面文本，只在 Chat 为真时接进系统提示。
	Page       string
	Profile    string
	Plan       string
	Notes      string
	Checkpoint string
	Suggested  string
	// Chat 为真时只留下桌面真正发出的系统提示：身份、档案和页面。不放 playbook、规则、技能目录和笔记。
	Chat bool
}

// buildBill 组装身份、环境和 pins。规则用 LoadWorkspaceRules，上限是 Loop.RulesTokens。
// StackAgentsMD 为真且叠到了 AGENTS.md 时，用那份替换第一命中，字节上限是 RulesTokens 的 4 倍，再按 RulesTokens 个 rune 截断，来源记成 AGENTS.md。
// playbook 放进 pinsWire。动态三层留给调用方接到历史后面。
func buildBill(in billInput) contextBill {
	var b contextBill
	b.systemWire = AssembleIdentity(in.Loop, in.Fragments)
	if in.Chat {
		// 与 desk.PromptText 同一段：身份、档案、页面。环境、playbook、规则和目录不在这条提示里。
		b.systemWire = strings.TrimSpace(b.systemWire)
		for _, text := range []string{in.Profile, in.Page} {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if b.systemWire != "" {
				b.systemWire += "\n\n"
			}
			b.systemWire += text
		}
		b.System = b.systemWire
		return b
	}
	if env := AssembleEnvironment(in.Workspace); env != "" {
		b.systemWire += "\n## Environment\n" + env
	}
	pinOpts := pinsOptsFromTools(in.Tools)
	rules, src := LoadWorkspaceRules(in.Workspace, in.Loop.RulesTokens)
	if in.Loop.StackAgentsMD {
		if stacked := LoadStackedAgents(in.Workspace, in.Loop.RulesTokens*4); stacked != "" {
			rules = capRunes(stacked, in.Loop.RulesTokens)
			src = "AGENTS.md"
		}
	}
	b.Rules = rulesSection(rules, src)
	b.Catalog = catalogSection(in.Loop, in.Skills, pinOpts)
	b.Memory = memorySection(in.Profile, ProjectMemoryPin(in.Workspace))
	playbook := playbookSection(in.Loop, in.Playbook)
	b.System = b.systemWire + playbook
	b.pinsWire = playbook + b.Catalog + b.Rules + b.Memory
	b.Bodies = skillBodiesText(in.Tools)
	b.Today = AssembleToday(in.Tools)
	b.Notes = notesSection(in.Plan, in.Notes, in.Checkpoint, in.Suggested, pathRulesOf(in.Tools))
	return b
}

// HarborPrompt 拼评测 Harbor 一题的系统提示：身份和环境、手册、技能目录、工作区规则和记忆，后面接上已加载技能正文、今日和计划。
// 内容和 Harbor 以前发给模型的 system、developer 两段相同，只是合成一段，交给 agentloop/desk 的 Turn.Prompt。
// 桌面聊天的提示只有身份、档案和页面（见 desk.PromptText），Harbor 不能用那一版：evolve 改的手册和技能目录要进模型，打分才有意义。
// tools 为 nil 时环境里没有工作区路径，也没有目录预算和技能正文。
func HarborPrompt(loop artifact.LoopPreset, fragments []artifact.PromptFragment, playbook artifact.Playbook, skills []artifact.Skill, tools *WorkspaceTools) string {
	workspace := ""
	if tools != nil {
		workspace = tools.Workspace
	}
	bill := buildBill(billInput{
		Loop:      loop,
		Fragments: fragments,
		Playbook:  playbook,
		Skills:    skills,
		Workspace: workspace,
		Tools:     tools,
		Plan:      planTextOf(tools),
	})
	return strings.TrimRight(bill.prefix()+bill.dynamic(), "\n")
}

// playbookSection 按 PlaybookTokens 渲染，小于等于 0 时用 2000。渲染结果为空则返回空串。这段进入 pins，不进入动态尾。
// loop 的 PlaybookTokens 小于等于 0 时用 2000，再传给 RenderBudget。
func playbookSection(loop artifact.LoopPreset, playbook artifact.Playbook) string {
	pbBudget := loop.PlaybookTokens
	if pbBudget <= 0 {
		pbBudget = 2000
	}
	rendered := playbook.RenderBudget(pbBudget)
	if strings.TrimSpace(rendered) == "" {
		return ""
	}
	return "\n## Playbook\n" + rendered
}

// rulesSection 包上工作区规则标题。正文为空返回空串。来源为空时标题用 YOYO.md。
// src 为空时标题写成 YOYO.md。
func rulesSection(rules, src string) string {
	if strings.TrimSpace(rules) == "" {
		return ""
	}
	if src == "" {
		src = "YOYO.md"
	}
	return "\n## Workspace rules (" + src + ")\n" + rules + "\n"
}

// catalogSection 拼技能目录、路由和额外钉子。loop 参数不参与正文。
// skills 非空时交给 renderSkillCatalog。
// opts 的 Routing 和 Extra 去掉空白后非空则写入，并传给 renderSkillCatalog。
func catalogSection(loop artifact.LoopPreset, skills []artifact.Skill, opts PinsOpts) string {
	var b strings.Builder
	if len(skills) > 0 {
		b.WriteString(renderSkillCatalog(skills, opts))
	}
	if strings.TrimSpace(opts.Routing) != "" {
		b.WriteString(opts.Routing)
		if !strings.HasSuffix(opts.Routing, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(opts.Extra) != "" {
		b.WriteString(opts.Extra)
		if !strings.HasSuffix(opts.Extra, "\n") {
			b.WriteByte('\n')
		}
	}
	_ = loop
	return b.String()
}

// memorySection 内联档案记忆，并接上项目记忆钉子。项目记忆是路径提示，不是文件正文。
// profile 去掉空白后非空时写入 Profile memory。
// project 直接接在后面。
func memorySection(profile, project string) string {
	var b strings.Builder
	if strings.TrimSpace(profile) != "" {
		b.WriteString("\n## Profile memory\n")
		b.WriteString(profile)
		if !strings.HasSuffix(profile, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString(project)
	return b.String()
}

// skillBodiesText 把已加载技能正文放进动态尾，并先按 capLoadedSkills 截断。没有正文时返回空串。
// tools 传给 loadedFrom。
func skillBodiesText(tools *WorkspaceTools) string {
	loaded := capLoadedSkills(loadedFrom(tools))
	if len(loaded) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Loaded skills (re-injected after compaction)\n")
	for _, body := range loaded {
		b.WriteString(body)
		b.WriteByte('\n')
	}
	return b.String()
}

// pathRulesOf 返回工具上已经吸收的路径规则。工具为 nil 时返回空串。
// tools 为 nil 时返回空串，否则调用 PathRules。
func pathRulesOf(tools *WorkspaceTools) string {
	if tools == nil {
		return ""
	}
	return tools.PathRules()
}

// notesSection 拼建议、计划、笔记、检查点和路径规则。笔记截到 2000 个 rune。
// suggested 去掉空白后非空时写在最前。
// plan 去掉空白后非空时写入 Plan 段。
// notes 去掉空白后非空时写入 Session notes 段。
// checkpoint 去掉空白后非空时写入 Checkpoint 段。
// pathRules 去掉空白后非空时写入 Path rules 段。
func notesSection(plan, notes, checkpoint, suggested, pathRules string) string {
	var b strings.Builder
	if strings.TrimSpace(suggested) != "" {
		b.WriteString(suggested)
		if !strings.HasSuffix(suggested, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(plan) != "" {
		b.WriteString("\n## Plan\n")
		b.WriteString(plan)
		if !strings.HasSuffix(plan, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(notes) != "" {
		b.WriteString("\n## Session notes\n")
		b.WriteString(capRunes(notes, 2000))
		if !strings.HasSuffix(notes, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(checkpoint) != "" {
		b.WriteString("\n## Checkpoint\n")
		b.WriteString(checkpoint)
		if !strings.HasSuffix(checkpoint, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(pathRules) != "" {
		b.WriteString("\n## Path rules (untrusted until you re-read them)\n")
		b.WriteString(pathRules)
		if !strings.HasSuffix(pathRules, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// historyTokens 只数用户、助手和工具。系统、developer、memory，以及以 dynMarker 开头的动态块不计。
// msgs 逐条按 Role 跳过系统、developer 和 memory，再跳过正文以 dynMarker 开头的，其余交给 messageTokens。
func historyTokens(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem, RoleDeveloper, RoleMemory:
			continue
		}
		if strings.HasPrefix(m.Content, dynMarker) {
			continue
		}
		n += messageTokens(m)
	}
	return n
}
