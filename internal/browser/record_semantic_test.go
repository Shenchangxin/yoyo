package browser

import (
	"strings"
	"testing"
)

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

func TestRedactRecordValueAndURL(t *testing.T) {
	if got := redactRecordValue(RecordTarget{Type: "password"}, "hunter2"); got != "{{sensitive_value}}" {
		t.Fatalf("password field: %s", got)
	}
	if got := redactRecordURL("https://ex.test/?token=abc&q=1"); !strings.Contains(got, "token=%2A%2A%2A") && !strings.Contains(got, "token=***") {
		t.Fatalf("url: %s", got)
	}
}
