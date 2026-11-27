// 内置技能包目录。写在这里不会把技能文件打进二进制，安装时仍要复制 skills/。

package skillpack

// SuperpowersID 是 obra/superpowers 方法论包的 id。
const SuperpowersID = "superpowers"

// NovelToGameID 是 zenstory-ai/novel-to-game 领域包的 id。它不是方法论包，没有会话开始时的引导技能。
const NovelToGameID = "novel-to-game"

// Catalog 返回内置目录，顺序是 Superpowers 然后 NovelToGame。这个顺序就是已知包在列表里的顺序。
func Catalog() []Known {
	return []Known{SuperpowersKnown(), NovelToGameKnown()}
}

// SuperpowersKnown 描述 obra/superpowers：默认分支 main，技能目录 skills，引导技能 using-superpowers，Methodology 为真。
func SuperpowersKnown() Known {
	return Known{
		ID:             SuperpowersID,
		Name:           "Superpowers",
		Description:    "Software-development skills that work together: brainstorm, spec, plan, TDD, review, and finish. When on, the agent loads the matching skill before writing code.",
		License:        "MIT",
		DefaultRepo:    "obra/superpowers",
		DefaultRef:     "main",
		SkillsRel:      "skills",
		BootstrapSkill: "using-superpowers",
		Methodology:    true,
	}
}

// NovelToGameKnown 描述 zenstory-ai/novel-to-game：默认分支 main，技能目录 skills。不设引导技能，Methodology 为假。
func NovelToGameKnown() Known {
	return Known{
		ID:          NovelToGameID,
		Name:        "NovelToGame",
		Description: "Turn a novel into a playable game: deconstruct the story, pick a playable concept, design world and art, build it, and QA with recorded play evidence. Just describe what you want; the agent picks the right skill.",
		License:     "MIT",
		DefaultRepo: "zenstory-ai/novel-to-game",
		DefaultRef:  "main",
		SkillsRel:   "skills",
	}
}

// Lookup 按净化后的 id 在 Catalog 里查找。没有则返回零值和 false。
func Lookup(id string) (Known, bool) {
	id = SanitizeID(id)
	for _, k := range Catalog() {
		if k.ID == id {
			return k, true
		}
	}
	return Known{}, false
}
