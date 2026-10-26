// 锁住搜索条数、线程正文上限、日历必须给出时间范围，以及查询长度上限。

package connector

import "testing"

// TestBoundViewSearchCapsAndSource 确认结果被截到上限，并且 source 为 untrusted。
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

// TestBoundViewThreadAllowsLongerBody 确认 thread: 查询可以留下更长的正文。
func TestBoundViewThreadAllowsLongerBody(t *testing.T) {
	items := []map[string]string{{"id": "t1", "body": stringsRepeat("a", 13000)}}
	out, err := BoundView(KindMail, "thread:t1", items)
	if err != nil || len(out[0]["body"]) != 12000 {
		t.Fatalf("%+v %v", out, err)
	}
}

// TestBoundViewCalendarRequiresRange 确认没有时间范围的日历查询失败，合法范围可以返回。
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

// TestBoundViewQueryCap 确认超长查询被拒绝。
func TestBoundViewQueryCap(t *testing.T) {
	if _, err := BoundView(KindMail, stringsRepeat("q", 501), nil); err == nil {
		t.Fatal("query")
	}
}

// stringsRepeat 把字符串重复 n 次。
func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
