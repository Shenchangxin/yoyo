// 本回合已启用的方法论包和领域包。ApplyPackBootstrap 在 Depth 大于 0 时直接返回，不写 MethodologyPin。深度为 0 时只把 BootstrapPin 设成 MethodologyPin 的一行提示，不把 SKILL.md 正文写进开场提示。

package runtime

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/kernel"
)

// PackSession 是本回合启用的一个方法论或内容包。
type PackSession struct {
	ID             string
	BootstrapSkill string
	Mapping        string
	Methodology    bool
}

// bootstrapAlready 是 WrapBootstrap 塞进正文的一句说明，声明引导技能已经加载、不要再 load_skill。
const bootstrapAlready = "IMPORTANT: The using-superpowers (or pack bootstrap) skill content is included below. It is ALREADY LOADED — you are currently following it. Do NOT call load_skill for this bootstrap skill again."

// bootstrapMarker 是 EXTREMELY_IMPORTANT，用作 WrapBootstrap 的包裹标记。
const bootstrapMarker = "EXTREMELY_IMPORTANT"

// ApplyPackBootstrap 在 t 为 nil、Depth 大于 0 或 Packs 为空时直接返回。否则只要有 Methodology 为真的包就把 t.Methodology 设为真，并把 BootstrapPin 设为 MethodologyPin。不在这里写入 Loaded 或 BootstrapBodies。
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

// WrapBootstrap 用 bootstrapMarker 包住说明、正文和 mapping。正文为空时写明该技能已启用但 SKILL.md 为空。ApplyPackBootstrap 不调用本函数。
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

// StripPackBootstrap 从 Loaded 去掉各包的 BootstrapSkill，并清空 Packs、BootstrapBodies、BootstrapPin，把 Methodology 设为假。
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

// HasMethodologyPack 在任一包的 Methodology 为真时返回真。
func HasMethodologyPack(packs []PackSession) bool {
	for _, p := range packs {
		if p.Methodology {
			return true
		}
	}
	return false
}

// IsNovelGameSkill 只认 novel-to-game、novel-game-analyze、game-concept、game-world-design、game-art-direction、game-build、game-qa。比较前去掉空白并转成小写。
func IsNovelGameSkill(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "novel-to-game", "novel-game-analyze", "game-concept", "game-world-design",
		"game-art-direction", "game-build", "game-qa":
		return true
	default:
		return false
	}
}

// WithoutNovelGamePack 去掉 ID 属于 IsNovelGameSkill 的包。空切片原样返回。
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

// WithoutNovelGameSkills 去掉属于 IsNovelGameSkill 的名字。空切片原样返回。
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

// DomainOrchestrators 收集既不是方法论、又没有 BootstrapSkill 的包 ID。
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

// MethodologyPin 为每个有引导技能或 Methodology 的包写一行，让模型在写产品代码前 load_skill。正文不内联。没有可写的行则返回空串。
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

// DomainRoutingPin 给已启用的领域包一行缓存稳定的提示。跳过方法论包和带 BootstrapSkill 的包。不预载 SKILL.md，也不写出完整工具映射。
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

// domainRoutingLine 对 novel-to-game 写出阶段技能名单。其余 ID 写成按目录描述 load_skill。两句都说明 @skill 是覆盖，不是前提。
func domainRoutingLine(id string) string {
	switch id {
	case "novel-to-game":
		return "- novel-to-game: load_skill novel-to-game when adapting a novel into a playable game. Stage siblings: novel-game-analyze, game-concept, game-world-design, game-art-direction, game-build, game-qa. @skill is an override, not a prerequisite."
	default:
		return "- " + id + ": load_skill " + id + " when the task matches this pack's catalog descriptions. @skill is an override, not a prerequisite."
	}
}

// SessionStart 是会话开始钩子的载荷，含会话 ID、工作区、深度和附加上下文。
type SessionStart struct {
	SessionID         string
	Workspace         string
	Depth             int
	AdditionalContext string
}

// applySessionStart 在总线为空、Waterfall 出错或结果不是 SessionStart 时返回原 hook。
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
