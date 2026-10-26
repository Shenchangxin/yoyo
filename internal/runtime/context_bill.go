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
	Loop       artifact.LoopPreset
	Fragments  []artifact.PromptFragment
	Playbook   artifact.Playbook
	Skills     []artifact.Skill
	Workspace  string
	Tools      *WorkspaceTools
	Soft       bool
	Page       string
	Profile    string
	Plan       string
	Notes      string
	Checkpoint string
	Suggested  string
}

// buildBill 组装身份、环境和 pins。规则用 LoadWorkspaceRules，上限是 Loop.RulesTokens。
// StackAgentsMD 为真且叠到了 AGENTS.md 时，用那份替换第一命中，字节上限是 RulesTokens 的 4 倍，再按 RulesTokens 个 rune 截断，来源记成 AGENTS.md。
// playbook 放进 pinsWire。Soft 为真时再加上行动阶梯、个人和页面钉子。动态三层留给调用方接到历史后面。
func buildBill(in billInput) contextBill {
	var b contextBill
	b.systemWire = AssembleIdentity(in.Loop, in.Fragments)
	if env := AssembleEnvironment(in.Workspace); env != "" {
		b.systemWire += "\n## Environment\n" + env
	}
	pinOpts := pinsOptsFromTools(in.Tools, in.Soft)
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
	ladder := ""
	if in.Soft {
		ladder = ActionLadderPin + PersonalPin + PagesPin
	}
	playbook := playbookSection(in.Loop, in.Playbook)
	b.System = b.systemWire + playbook + ladder
	b.pinsWire = playbook + b.Catalog + b.Rules + b.Memory + ladder
	b.Bodies = skillBodiesText(in.Tools)
	b.Today = AssembleToday(in.Tools)
	b.Notes = notesSection(in.Plan, in.Notes, in.Checkpoint, in.Suggested, pathRulesOf(in.Tools), in.Page, in.Soft)
	return b
}

// playbookSection 按 PlaybookTokens 渲染，小于等于 0 时用 2000。渲染结果为空则返回空串。这段进入 pins，不进入动态尾。
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
func pathRulesOf(tools *WorkspaceTools) string {
	if tools == nil {
		return ""
	}
	return tools.PathRules()
}

// notesSection 拼建议、计划、笔记、检查点、路径规则，以及 Soft 时的当前页面。笔记截到 2000 个 rune，页面截到 3200 个 rune。
func notesSection(plan, notes, checkpoint, suggested, pathRules, page string, soft bool) string {
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
	if soft && strings.TrimSpace(page) != "" {
		b.WriteString("\n## Current page (untrusted)\n")
		b.WriteString(capRunes(page, 3200))
		if !strings.HasSuffix(page, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// historyTokens 只数用户、助手和工具。系统、developer、memory，以及以 dynMarker 开头的动态块不计。
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
