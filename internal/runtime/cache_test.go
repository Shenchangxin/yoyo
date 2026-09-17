package runtime

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

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

func TestCountTokensCJK(t *testing.T) {
	if CountTokens("你好") < 2 {
		t.Fatal(CountTokens("你好"))
	}
	if TokenizerName("gpt-4.1") == "" {
		t.Fatal("tokenizer")
	}
}
