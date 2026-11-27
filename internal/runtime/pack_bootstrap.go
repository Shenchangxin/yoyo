// 本回合已启用的方法论包和领域包。ApplyPackBootstrap 在 Depth 大于 0 时直接返回，不写 MethodologyPin。深度为 0 时只把 BootstrapPin 设成 MethodologyPin 的一行提示，不把 SKILL.md 正文写进开场提示。

package runtime

import (
	"strings"
)

// PackSession 是本回合启用的一个方法论或内容包。
type PackSession struct {
	ID             string
	BootstrapSkill string
	Mapping        string
	Methodology    bool
}

// bootstrapMarker 是 EXTREMELY_IMPORTANT，引导技能正文用 <EXTREMELY_IMPORTANT> 标签包住。
// isBootstrapBody 靠它认出引导正文，capLoadedSkills 压缩已加载技能时不丢这一段。
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
// packs 逐项看 Methodology，有一项为真就返回真。
func HasMethodologyPack(packs []PackSession) bool {
	for _, p := range packs {
		if p.Methodology {
			return true
		}
	}
	return false
}

// IsNovelGameSkill 只认 novel-to-game、novel-game-analyze、game-concept、game-world-design、game-art-direction、game-build、game-qa。比较前去掉空白并转成小写。
// name 去掉空白并转成小写后再和上述名单比较。
func IsNovelGameSkill(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "novel-to-game", "novel-game-analyze", "game-concept", "game-world-design",
		"game-art-direction", "game-build", "game-qa":
		return true
	default:
		return false
	}
}

// DomainOrchestrators 收集既不是方法论、又没有 BootstrapSkill 的包 ID。
// packs 里 Methodology 为真，或 BootstrapSkill 去掉空白后非空的项跳过；其余 ID 去掉空白后非空才放进表。
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
// packs 逐项取 BootstrapSkill，空白且 Methodology 为假则跳过；该名字仍空白则改用 ID，两者去掉空白都为空也跳过。
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
// packs 里 Methodology 为真或 BootstrapSkill 去掉空白后非空的项跳过，其余 ID 去掉空白后非空才交给 domainRoutingLine。
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
// id 等于 novel-to-game 时返回阶段技能那一行，否则写进按目录描述 load_skill 的那一行。
func domainRoutingLine(id string) string {
	switch id {
	case "novel-to-game":
		return "- novel-to-game: load_skill novel-to-game when adapting a novel into a playable game. Stage siblings: novel-game-analyze, game-concept, game-world-design, game-art-direction, game-build, game-qa. @skill is an override, not a prerequisite."
	default:
		return "- " + id + ": load_skill " + id + " when the task matches this pack's catalog descriptions. @skill is an override, not a prerequisite."
	}
}
