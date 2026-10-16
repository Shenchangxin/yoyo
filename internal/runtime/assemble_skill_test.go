package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

func TestAssemblePinsHarborKeepsLegacyHeader(t *testing.T) {
	got := AssemblePins(DefaultLoop(), artifact.Playbook{}, []artifact.Skill{
		{Name: "plan-first", Description: "Keep a visible task plan."},
	}, "", "")
	if !strings.Contains(got, "load with load_skill when relevant") {
		t.Fatal(got)
	}
	if strings.Contains(got, "Trigger:") {
		t.Fatal("harbor catalog must not gain chat trigger rules")
	}
}

func TestAssemblePinsChatCatalogTriggerRulesAndBudget(t *testing.T) {
	var skills []artifact.Skill
	for i := 0; i < 40; i++ {
		skills = append(skills, artifact.Skill{
			Name:        string(rune('a'+i%26)) + strings.Repeat("x", 8) + string(rune('0'+i%10)),
			Description: strings.Repeat("word ", 80),
		})
	}
	skills = append(skills, artifact.Skill{Name: "novel-to-game", Description: "Turn a novel into a playable game."})
	got := AssemblePinsOpts(DefaultLoop(), artifact.Playbook{}, skills, "", "", PinsOpts{
		CatalogBudget: 400,
		ChatCatalog:   true,
		Routing:       DomainRoutingPin([]PackSession{{ID: "novel-to-game"}}),
	})
	if !strings.Contains(got, "Trigger:") || !strings.Contains(got, "load_skill") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "omitted to fit the catalog budget") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "## Enabled domain packs") || !strings.Contains(got, "novel-to-game") {
		t.Fatal(got)
	}
	if strings.Contains(got, "Tool Mapping for Yoyo") {
		t.Fatal("routing pin must not dump full mapping")
	}
}

func TestAssembleDynamicHintSection(t *testing.T) {
	got := AssembleDynamicHint("", nil, "", "", "", FormatSkillHints([]SkillSuggestion{
		{Name: "weekly-report", Action: SkillActionHint, Reason: "description overlap"},
	}))
	if !strings.Contains(got, "## Suggested skills this turn") || !strings.Contains(got, "weekly-report") {
		t.Fatal(got)
	}
}
