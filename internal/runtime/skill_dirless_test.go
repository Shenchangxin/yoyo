package runtime

import (
	"strings"
	"testing"
)

// verify-artifact is built in: it has instructions and no directory. The
// agent asked to run a script from it and got "has no on-disk directory",
// which read as a broken install. The error now says what the skill is.
func TestRunSkillScriptExplainsInstructionOnlySkill(t *testing.T) {
	tools := &WorkspaceTools{
		Skills: map[string]string{"verify-artifact": DefaultSkillMD()},
	}
	res := tools.Call("run_skill_script", `{"skill":"verify-artifact","script":"scripts/verify.py"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "instruction-only") {
		t.Fatalf("expected an instruction-only explanation, got %+v", res)
	}
	if strings.Contains(res.Err.Error(), "on-disk directory") {
		t.Fatalf("message must not read as a broken install: %v", res.Err)
	}
}

func TestReadSkillFileExplainsInstructionOnlySkill(t *testing.T) {
	tools := &WorkspaceTools{
		Skills: map[string]string{"verify-artifact": DefaultSkillMD()},
	}
	res := tools.Call("read_skill_file", `{"skill":"verify-artifact","path":"SKILL.md"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "instruction-only") {
		t.Fatalf("expected an instruction-only explanation, got %+v", res)
	}
}

func TestUnknownSkillKeepsDirectoryError(t *testing.T) {
	tools := &WorkspaceTools{}
	res := tools.Call("run_skill_script", `{"skill":"no-such-skill","script":"scripts/x.py"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "has no on-disk directory") {
		t.Fatalf("unknown skill must keep the directory error, got %+v", res)
	}
}
