// 锁住提示缓存键、计划模式下的网络技能过滤，以及中文的 token 下界。
// FilterSkills 和 PromptCacheKey 的实现不在这个文件；缓存键的稳定性见 prompt_assembly_test.go。

package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestFilterSkillsNetworkInPlanMode 确认计划模式滤掉 Compatibility 为 network 的技能，留下 local。
func TestFilterSkillsNetworkInPlanMode(t *testing.T) {
	in := []artifact.Skill{
		{Name: "local", Description: "x", Body: "y"},
		{Name: "web", Description: "x", Body: "y", Compatibility: "network"},
	}
	got := FilterSkills(in, true)
	if len(got) != 1 || got[0].Name != "local" {
		t.Fatalf("%+v", got)
	}
}

// TestEstimateTokensCJK 确认「你好」至少计 2 个 token：汉字不能按英文单词那样合并计数。
func TestEstimateTokensCJK(t *testing.T) {
	if n := estimateTokens("你好"); n < 2 {
		t.Fatal(n)
	}
}
