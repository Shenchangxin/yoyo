package runtime

import "testing"

func TestPinVisibleLanguageDropsEnglishOpener(t *testing.T) {
	msg := Message{Content: "I'll start by loading the skill.", ToolCalls: []ToolCall{{Name: "load_skill"}}}
	nudge := pinVisibleLanguage(&msg, "把这本小说做成游戏")
	if nudge == "" || msg.Content != "" {
		t.Fatalf("content=%q nudge=%q", msg.Content, nudge)
	}
	if !englishOpener("Let me read the file") || englishOpener("先读取技能合同。") {
		t.Fatal("opener detector")
	}
}

func TestPinVisibleLanguageSkipsEnglishOperator(t *testing.T) {
	msg := Message{Content: "I'll start by loading the skill."}
	if pinVisibleLanguage(&msg, "turn this novel into a game") != "" {
		t.Fatal("english operator must keep the letter")
	}
	if msg.Content == "" {
		t.Fatal("stripped english operator letter")
	}
}
