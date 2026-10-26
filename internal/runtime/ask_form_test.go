// ask_form 的解析与空闲回答。缺 recommended 时采用第一项。没有选项时不编造选择。

package runtime

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseAskFormFillsRecommended 确认缺 recommended 时采用第一项 ink，AskIdleAnswer 带上方向和 askIdleNote，AskPrompt 用标题。
func TestParseAskFormFillsRecommended(t *testing.T) {
	form := parseAskForm(map[string]any{
		"title": "方向",
		"questions": []any{
			map[string]any{
				"id":     "direction",
				"prompt": "按这个方向改吗？",
				"options": []any{
					map[string]any{"id": "yes", "label": "按这个方向"},
					map[string]any{"id": "no", "label": "先别改操作"},
				},
				"recommended": "yes",
			},
			map[string]any{
				"prompt": "画面？",
				"options": []any{
					map[string]any{"id": "ink", "label": "暗黑水墨"},
					map[string]any{"id": "bright", "label": "更亮"},
				},
			},
		},
	})
	if len(form.Questions) != 2 {
		t.Fatalf("questions %+v", form.Questions)
	}
	if form.Questions[1].ID != "q2" || form.Questions[1].Recommended != "ink" {
		t.Fatalf("default recommended %+v", form.Questions[1])
	}
	payload := mustAskJSON(t, form)
	got := AskIdleAnswer(payload)
	if !strings.Contains(got, "direction: yes — 按这个方向") || !strings.Contains(got, "q2: ink — 暗黑水墨") || !strings.Contains(got, askIdleNote) {
		t.Fatalf("idle %q", got)
	}
	if AskPrompt(payload) != "方向" {
		t.Fatalf("prompt %q", AskPrompt(payload))
	}
}

// TestAskIdleWithoutOptionsDoesNotInventAChoice 确认没有选项时返回 askNoOperator，不编造选择。
func TestAskIdleWithoutOptionsDoesNotInventAChoice(t *testing.T) {
	form := parseAskForm(map[string]any{"question": "登录完成了吗？"})
	got := AskIdleAnswer(mustAskJSON(t, form))
	if got != askNoOperator {
		t.Fatalf("%q", got)
	}
}

// TestWaitsForNodMatchesDesignHold 确认设计稿里的等待确认用语命中，已经做完的句子不命中。
func TestWaitsForNodMatchesDesignHold(t *testing.T) {
	letter := "请确认两件事：\n1. 这个方向对吗？\n你点头我就开工。"
	if !waitsForNod(letter) {
		t.Fatal("expected the design hold")
	}
	if waitsForNod("难度档已全部落码并通过验证。") {
		t.Fatal("a finished status is not a hold")
	}
	if !waitsForNod("Please confirm the direction before I write the files.") {
		t.Fatal("english hold")
	}
}

// TestChosenOptionReadsCardLine 确认从卡片行读出 ink，并跳过 note: 行。
func TestChosenOptionReadsCardLine(t *testing.T) {
	ans := "choice: b — Beta\nnote: operator answered"
	if got := chosenOption(ans, []string{"a", "b"}); got != "b" {
		t.Fatalf("%q", got)
	}
}

// TestAskNudgeIsControlText 确认继续轻推算控制文本，普通句子不算。
func TestAskNudgeIsControlText(t *testing.T) {
	if !isControlUser(askContinueNudge) || !isControlUser(askContinueNudgeZH) {
		t.Fatal("nudge must not become the operator objective")
	}
}

// mustAskJSON 把表单编成 JSON，失败则终止测试。
func mustAskJSON(t *testing.T, form AskForm) string {
	t.Helper()
	normalizeAskForm(&form)
	b, err := json.Marshal(form)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
