package skillpack

// SuperpowersID is the obra/superpowers methodology pack.
const SuperpowersID = "superpowers"

// Catalog is the built-in pack registry. Adding a pack here does not vendor
// its skills; install still copies skills/ through the pack installer.
func Catalog() []Known {
	return []Known{SuperpowersKnown()}
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

func Lookup(id string) (Known, bool) {
	for _, k := range Catalog() {
		if k.ID == id {
			return k, true
		}
	}
	return Known{}, false
}
