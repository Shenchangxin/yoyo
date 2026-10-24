package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/kernel"
)

// PackSession is one enabled methodology/content pack for this turn.
type PackSession struct {
	ID             string
	BootstrapSkill string
	Mapping        string
	Methodology    bool
}

const bootstrapAlready = "IMPORTANT: The using-superpowers (or pack bootstrap) skill content is included below. It is ALREADY LOADED — you are currently following it. Do NOT call load_skill for this bootstrap skill again."

const bootstrapMarker = "EXTREMELY_IMPORTANT"

// ApplyPackBootstrap records that a methodology pack is on, and leaves a
// one-line pin. The SKILL.md body stays out of the opening prompt; the
// model load_skill's it. Child sessions (Depth>0) are skipped structurally.
func ApplyPackBootstrap(t *WorkspaceTools) {
	if t == nil || t.Depth > 0 || len(t.Packs) == 0 {
		return
	}
	for _, p := range t.Packs {
		if p.Methodology {
			t.Methodology = true
		}
	}
	t.BootstrapPin = MethodologyPin(t.Packs)
}

func WrapBootstrap(name, body, mapping string) string {
	body = strings.TrimSpace(body)
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(bootstrapMarker)
	b.WriteString(">\nYou have superpowers.\n\n")
	b.WriteString("**")
	b.WriteString(bootstrapAlready)
	b.WriteString("**\n\n")
	if body != "" {
		b.WriteString(body)
		b.WriteByte('\n')
	} else {
		b.WriteString("Bootstrap skill ")
		b.WriteString(name)
		b.WriteString(" was enabled but its SKILL.md body was empty.\n")
	}
	if strings.TrimSpace(mapping) != "" {
		b.WriteByte('\n')
		b.WriteString(strings.TrimSpace(mapping))
		b.WriteByte('\n')
	}
	b.WriteString("</")
	b.WriteString(bootstrapMarker)
	b.WriteString(">\n")
	return b.String()
}

func StripPackBootstrap(t *WorkspaceTools) {
	if t == nil {
		return
	}
	drop := map[string]bool{}
	for _, p := range t.Packs {
		if n := strings.TrimSpace(p.BootstrapSkill); n != "" {
			drop[n] = true
		}
	}
	if len(drop) > 0 && len(t.Loaded) > 0 {
		var keep []string
		for _, n := range t.Loaded {
			if !drop[n] {
				keep = append(keep, n)
			}
		}
		t.Loaded = keep
	}
	t.Packs = nil
	t.BootstrapBodies = nil
	t.BootstrapPin = ""
	t.Methodology = false
}

func HasMethodologyPack(packs []PackSession) bool {
	for _, p := range packs {
		if p.Methodology {
			return true
		}
	}
	return false
}

func IsNovelGameSkill(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "novel-to-game", "novel-game-analyze", "game-concept", "game-world-design",
		"game-art-direction", "game-build", "game-qa":
		return true
	default:
		return false
	}
}

func WithoutNovelGamePack(packs []PackSession) []PackSession {
	if len(packs) == 0 {
		return packs
	}
	out := make([]PackSession, 0, len(packs))
	for _, p := range packs {
		if IsNovelGameSkill(p.ID) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func WithoutNovelGameSkills(names []string) []string {
	if len(names) == 0 {
		return names
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if IsNovelGameSkill(n) {
			continue
		}
		out = append(out, n)
	}
	return out
}

func DomainOrchestrators(packs []PackSession) map[string]bool {
	out := map[string]bool{}
	for _, p := range packs {
		if p.Methodology || strings.TrimSpace(p.BootstrapSkill) != "" {
			continue
		}
		if id := strings.TrimSpace(p.ID); id != "" {
			out[id] = true
		}
	}
	return out
}

// MethodologyPin tells the model a pack is enabled without inlining SKILL.md.
func MethodologyPin(packs []PackSession) string {
	var lines []string
	for _, p := range packs {
		name := strings.TrimSpace(p.BootstrapSkill)
		if name == "" && !p.Methodology {
			continue
		}
		if name == "" {
			name = strings.TrimSpace(p.ID)
		}
		if name == "" {
			continue
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = name
		}
		lines = append(lines, "- "+id+": enabled. Before writing product code, load_skill "+name+". The body is not loaded yet.")
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n## Methodology\n" + strings.Join(lines, "\n") + "\n"
}

// DomainRoutingPin is a short, cache-stable hint for enabled domain packs.
// It does not preload SKILL.md or the full ToolMapping adapter.
func DomainRoutingPin(packs []PackSession) string {
	var lines []string
	for _, p := range packs {
		if p.Methodology || strings.TrimSpace(p.BootstrapSkill) != "" {
			continue
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		lines = append(lines, domainRoutingLine(id))
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n## Enabled domain packs\n" + strings.Join(lines, "\n") + "\n"
}

func domainRoutingLine(id string) string {
	switch id {
	case "novel-to-game":
		return "- novel-to-game: load_skill novel-to-game when adapting a novel into a playable game. Stage siblings: novel-game-analyze, game-concept, game-world-design, game-art-direction, game-build, game-qa. @skill is an override, not a prerequisite."
	default:
		return "- " + id + ": load_skill " + id + " when the task matches this pack's catalog descriptions. @skill is an override, not a prerequisite."
	}
}

type SessionStart struct {
	SessionID         string
	Workspace         string
	Depth             int
	AdditionalContext string
}

func applySessionStart(bus *kernel.EventBus, hook SessionStart) SessionStart {
	if bus == nil {
		return hook
	}
	out, err := bus.Waterfall(HookSessionStart, hook)
	if err != nil || out == nil {
		return hook
	}
	if h, ok := out.(SessionStart); ok {
		return h
	}
	return hook
}
