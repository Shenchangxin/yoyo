// 锁住没有目录的内置技能。verify-artifact 有说明、没有脚本，错误要说明它是 instruction-only，而不是像安装坏了。未知技能仍报没有磁盘目录。

package runtime

import (
	"strings"
	"testing"
)

// TestRunSkillScriptExplainsInstructionOnlySkill 锁住对 verify-artifact 跑脚本时，错误含 instruction-only，且不含 on-disk directory。
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

// TestReadSkillFileExplainsInstructionOnlySkill 锁住读取该技能的文件时同样说明它是 instruction-only。
func TestReadSkillFileExplainsInstructionOnlySkill(t *testing.T) {
	tools := &WorkspaceTools{
		Skills: map[string]string{"verify-artifact": DefaultSkillMD()},
	}
	res := tools.Call("read_skill_file", `{"skill":"verify-artifact","path":"SKILL.md"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "instruction-only") {
		t.Fatalf("expected an instruction-only explanation, got %+v", res)
	}
}

// TestUnknownSkillKeepsDirectoryError 锁住目录和正文都没有的技能仍报 has no on-disk directory。
func TestUnknownSkillKeepsDirectoryError(t *testing.T) {
	tools := &WorkspaceTools{}
	res := tools.Call("run_skill_script", `{"skill":"no-such-skill","script":"scripts/x.py"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "has no on-disk directory") {
		t.Fatalf("unknown skill must keep the directory error, got %+v", res)
	}
}
