package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// contextBill is the opening context, split so the meter can show each
// layer. System, rules, catalog, and memory are the cache-stable prefix.
// Bodies, today, and notes sit in the dynamic tail. Real chat is counted
// from history messages, not from this bill.
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

func (b contextBill) prefix() string { return b.systemWire + b.pinsWire }

func (b contextBill) dynamic() string { return b.Bodies + b.Today + b.Notes }

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

func rulesSection(rules, src string) string {
	if strings.TrimSpace(rules) == "" {
		return ""
	}
	if src == "" {
		src = "YOYO.md"
	}
	return "\n## Workspace rules (" + src + ")\n" + rules + "\n"
}

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

func pathRulesOf(tools *WorkspaceTools) string {
	if tools == nil {
		return ""
	}
	return tools.PathRules()
}

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
