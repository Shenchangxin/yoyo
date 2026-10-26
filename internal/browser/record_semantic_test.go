// 本文件锁住点击加输入并成 fill、空白点击并成导航、连续滚动只留一次，以及技能草稿会提到 ref。
// 由 go test 调用。

package browser

import (
	"strings"
	"testing"
)

// TestCollapseClickInputBecomesFill 锁住点进邮箱框再输入会收成带 {{email}} 的 fill。
func TestCollapseClickInputBecomesFill(t *testing.T) {
	doc := collapseRecordEvents([]RecordEvent{
		{Kind: "click", URL: "https://ex.test/login", Target: RecordTarget{Tag: "input", ID: "email", Name: "Email", NameAttr: "email"}},
		{Kind: "input", URL: "https://ex.test/login", Value: "a@b.com", Target: RecordTarget{Tag: "input", ID: "email", Name: "Email", NameAttr: "email"}},
	}, "https://ex.test/login")
	found := false
	for _, st := range doc.Steps {
		if st.Kind == "fill" && strings.Contains(st.Value, "{{email}}") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parameterized fill, steps=%+v inputs=%+v", doc.Steps, doc.Inputs)
	}
}

// TestCollapseBlankClickBecomesNavigate 锁住 about:blank 上点 body 再导航会收成 navigate。
func TestCollapseBlankClickBecomesNavigate(t *testing.T) {
	doc := collapseRecordEvents([]RecordEvent{
		{Kind: "click", URL: "about:blank", Target: RecordTarget{Tag: "body"}},
		{Kind: "navigation", URL: "https://ex.test/done"},
	}, "about:blank")
	found := false
	for _, st := range doc.Steps {
		if st.Kind == "navigate" && st.URL == "https://ex.test/done" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%+v", doc.Steps)
	}
}

// TestCollapseLastScrollWins 锁住两次滚动再点击时只留下一步 scroll。
func TestCollapseLastScrollWins(t *testing.T) {
	doc := collapseRecordEvents([]RecordEvent{
		{Kind: "scroll", DeltaY: 10},
		{Kind: "scroll", DeltaY: 80},
		{Kind: "click", Target: RecordTarget{Tag: "button", Name: "More"}},
	}, "")
	scrolls := 0
	for _, st := range doc.Steps {
		if st.Kind == "scroll" {
			scrolls++
		}
	}
	if scrolls != 1 {
		t.Fatalf("scrolls=%d steps=%+v", scrolls, doc.Steps)
	}
}

// TestGenerateRecordedSkillMentionsRefs 锁住草稿里有 browser_snapshot、ref= 和输入名 email。
func TestGenerateRecordedSkillMentionsRefs(t *testing.T) {
	body := generateRecordedSkill(StepsDocument{
		StartURL:  "https://ex.test",
		SkillName: "recorded-ex-test",
		Steps:     []SemanticStep{{Kind: "click", Target: RecordTarget{Name: "Go"}}},
		Inputs:    []SkillInput{{Name: "email", Kind: "email"}},
	})
	if !strings.Contains(body, "browser_snapshot") || !strings.Contains(body, "ref=") || !strings.Contains(body, "`email`") {
		t.Fatal(body)
	}
}

// TestRedactRecordValueAndURL 锁住密码字段变成 {{sensitive_value}}，地址里的 token 被打码。
func TestRedactRecordValueAndURL(t *testing.T) {
	if got := redactRecordValue(RecordTarget{Type: "password"}, "hunter2"); got != "{{sensitive_value}}" {
		t.Fatalf("password field: %s", got)
	}
	if got := redactRecordURL("https://ex.test/?token=abc&q=1"); !strings.Contains(got, "token=%2A%2A%2A") && !strings.Contains(got, "token=***") {
		t.Fatalf("url: %s", got)
	}
}
