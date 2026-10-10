package skillpack

// SuperpowersID is the obra/superpowers methodology pack.
const SuperpowersID = "superpowers"

// NovelToGameID is the zenstory-ai/novel-to-game domain pack.
// It is not a methodology pack: no session-start bootstrap.
const NovelToGameID = "novel-to-game"

// Catalog is the built-in pack registry. Adding a pack here does not vendor
// its skills; install still copies skills/ through the pack installer.
// Order is the Skills → Packs list order for known entries.
func Catalog() []Known {
	return []Known{SuperpowersKnown(), NovelToGameKnown()}
}

func SuperpowersKnown() Known {
	return Known{
		ID:             SuperpowersID,
		Name:           "Superpowers",
		Description:    "Composable software-development methodology: brainstorm, spec, plan, TDD/SDD, review, and finish. Session-start bootstrap requires invoking matching skills before acting.",
		License:        "MIT",
		DefaultRepo:    "obra/superpowers",
		DefaultRef:     "main",
		SkillsRel:      "skills",
		BootstrapSkill: "using-superpowers",
		Methodology:    true,
	}
}

func NovelToGameKnown() Known {
	return Known{
		ID:          NovelToGameID,
		Name:        "NovelToGame",
		Description: "Source-grounded novel-to-game workflow: deconstruct a novel, pick a playable concept, design world and art, build for the approved runtime, and QA with recorded play evidence. Domain pack — not a session-start methodology. Enable per workspace, then @skill:novel-to-game when the operator asks to turn a book into a game.",
		License:     "MIT",
		DefaultRepo: "zenstory-ai/novel-to-game",
		DefaultRef:  "main",
		SkillsRel:   "skills",
	}
}

func Lookup(id string) (Known, bool) {
	id = SanitizeID(id)
	for _, k := range Catalog() {
		if k.ID == id {
			return k, true
		}
	}
	return Known{}, false
}
