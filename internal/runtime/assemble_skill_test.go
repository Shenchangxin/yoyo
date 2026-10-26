// 技能目录在 Harbor 与聊天叠层上的差别，以及动态段里的建议标题。

package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestAssemblePinsHarborKeepsLegacyHeader 确认 Harbor 目录保留 load_skill 说明，且不含 Trigger:。
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

// TestAssemblePinsChatCatalogTriggerRulesAndBudget 确认聊天目录含 Trigger、预算省略和 Enabled domain packs，且不含 Tool Mapping for Yoyo。
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

// TestAssembleDynamicHintSection 确认动态段含 Suggested skills this turn。
func TestAssembleDynamicHintSection(t *testing.T) {
	got := AssembleDynamicHint("", nil, "", "", "", FormatSkillHints([]SkillSuggestion{
		{Name: "weekly-report", Action: SkillActionHint, Reason: "description overlap"},
	}))
	if !strings.Contains(got, "## Suggested skills this turn") || !strings.Contains(got, "weekly-report") {
		t.Fatal(got)
	}
}
