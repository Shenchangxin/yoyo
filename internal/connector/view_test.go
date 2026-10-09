package connector

import "testing"

func TestBoundViewSearchCapsAndSource(t *testing.T) {
	var items []map[string]string
	for i := 0; i < 25; i++ {
		items = append(items, map[string]string{
			"id":      "m",
			"subject": "hi",
			"body":    stringsRepeat("x", 400),
		})
	}
	out, err := BoundView(KindMail, "form", items)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 20 {
		t.Fatalf("cap %d", len(out))
	}
	if out[0]["source"] != "untrusted" || len(out[0]["body"]) != 240 || out[0]["truncated"] != "true" {
		t.Fatalf("%+v", out[0])
	}
}

func TestBoundViewThreadAllowsLongerBody(t *testing.T) {
	items := []map[string]string{{"id": "t1", "body": stringsRepeat("a", 13000)}}
	out, err := BoundView(KindMail, "thread:t1", items)
	if err != nil || len(out[0]["body"]) != 12000 {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestBoundViewCalendarRequiresRange(t *testing.T) {
	if _, err := BoundView(KindCalendar, "calendar", nil); err == nil {
		t.Fatal("range")
	}
	items := []map[string]string{
		{"subject": "Standup", "start": "2026-09-26T01:00:00Z"},
		{"subject": "Later", "start": "2027-01-01T00:00:00Z"},
	}
	out, err := BoundView(KindCalendar, "timeMin=2026-09-01T00:00:00Z timeMax=2026-10-01T00:00:00Z", items)
	if err != nil || len(out) != 1 || out[0]["subject"] != "Standup" {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestBoundViewQueryCap(t *testing.T) {
	if _, err := BoundView(KindMail, stringsRepeat("q", 501), nil); err == nil {
		t.Fatal("query")
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
