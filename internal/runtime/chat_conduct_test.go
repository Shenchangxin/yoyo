// 操作者说中文时清掉英文开场。英文操作者保留原句。

package runtime

import "testing"

// TestPinVisibleLanguageDropsEnglishOpener 确认汉字操作者下英文开场被清空，并返回重写轻推。
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

// TestPinVisibleLanguageSkipsEnglishOperator 确认英文操作者保留原句。
func TestPinVisibleLanguageSkipsEnglishOperator(t *testing.T) {
	msg := Message{Content: "I'll start by loading the skill."}
	if pinVisibleLanguage(&msg, "turn this novel into a game") != "" {
		t.Fatal("english operator must keep the letter")
	}
	if msg.Content == "" {
		t.Fatal("stripped english operator letter")
	}
}
