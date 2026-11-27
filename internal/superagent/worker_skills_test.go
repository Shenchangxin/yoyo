package superagent

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestExternalPromptListsSkills 锁住外部工人的说明列出装好的技能和 SKILL.md 路径，跳过没目录的条目，也不提 load_skill。
func TestExternalPromptListsSkills(t *testing.T) {
	dir := filepath.Join("C:\\", "home", "skills", "pdf")
	text := externalWorkerPrompt(nil, WorkerSpec{Agent: "codex", TicketID: "tkt-9", Skills: []SkillRef{
		{Name: "pdf", Description: "读写 PDF\n表单", Dir: dir},
		{Name: "cas-only", Description: "没有目录"},
	}})
	for _, want := range []string{"pdf：读写 PDF 表单", filepath.Join(dir, "SKILL.md"), "@skill:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "cas-only") || strings.Contains(text, "load_skill") {
		t.Fatalf("prompt = %s", text)
	}
	if bare := externalWorkerPrompt(nil, WorkerSpec{Agent: "codex", TicketID: "tkt-9"}); strings.Contains(bare, "装好的技能") {
		t.Fatalf("no skills should write no catalog:\n%s", bare)
	}
}

// TestSkillCatalogCapsLength 锁住说明截断和条数上限。
func TestSkillCatalogCapsLength(t *testing.T) {
	var skills []SkillRef
	for i := 0; i < skillCatalogMax+5; i++ {
		skills = append(skills, SkillRef{Name: "s" + strings.Repeat("x", i), Description: strings.Repeat("长", 300), Dir: "/skills/s"})
	}
	var b strings.Builder
	writeSkillCatalog(&b, skills)
	lines := strings.Count(b.String(), "\n- ")
	if lines != skillCatalogMax {
		t.Fatalf("rows = %d", lines)
	}
	if strings.Contains(b.String(), strings.Repeat("长", skillCatalogRunes+1)) {
		t.Fatal("description not clipped")
	}
}
