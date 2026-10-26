// 锁住提示缓存键、计划模式下的网络技能过滤，以及中文的 token 下界。
// FilterSkills 和 AssemblePrefix 的实现不在这个文件。

package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestPromptCacheKeyStableAndBusts 确认同一 harness、前缀和工具列表得到同一个非空键，harness 从 h1 换成 h2 后键必须改变。
func TestPromptCacheKeyStableAndBusts(t *testing.T) {
	prefix := AssemblePrefix(DefaultLoop(), nil, artifact.Playbook{}, nil, "tabs", "YOYO.md")
	tools := BuiltinToolJSON()
	a := PromptCacheKey("h1", prefix, tools)
	b := PromptCacheKey("h1", prefix, tools)
	if a != b || a == "" {
		t.Fatalf("%s %s", a, b)
	}
	c := PromptCacheKey("h2", prefix, tools)
	if c == a {
		t.Fatal("harness change must bust cache")
	}
}

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

// TestCountTokensCJK 确认「你好」至少计 2 个 token，且 TokenizerName 对 gpt-4.1 不返回空串。
func TestCountTokensCJK(t *testing.T) {
	if CountTokens("你好") < 2 {
		t.Fatal(CountTokens("你好"))
	}
	if TokenizerName("gpt-4.1") == "" {
		t.Fatal("tokenizer")
	}
}
