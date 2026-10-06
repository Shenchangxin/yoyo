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

// ApplyPackBootstrap preloads pack bootstrap bodies into working memory.
// Child sessions (Depth>0) are skipped structurally — model-level SUBAGENT-STOP is not enough.
func ApplyPackBootstrap(t *WorkspaceTools) {
	if t == nil || t.Depth > 0 || len(t.Packs) == 0 {
		return
	}
	if t.BootstrapBodies == nil {
		t.BootstrapBodies = map[string]string{}
	}
	for _, p := range t.Packs {
		name := strings.TrimSpace(p.BootstrapSkill)
		if name == "" {
			continue
		}
		body := ""
		if t.Skills != nil {
			body = t.Skills[name]
		}
		t.BootstrapBodies[name] = WrapBootstrap(name, body, p.Mapping)
		t.MarkSkillsLoaded(name)
		if p.Methodology {
			t.Methodology = true
		}
	}
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
