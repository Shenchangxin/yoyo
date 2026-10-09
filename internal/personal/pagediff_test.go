package personal

import (
	"strings"
	"testing"
)

func TestPageLinesTrimDedupBound(t *testing.T) {
	got := pageLines("  Jobs \n\n Siemens   AI\r\nSiemens AI\n", 0)
	if len(got) != 2 || got[0] != "Jobs" || got[1] != "Siemens AI" {
		t.Fatalf("%q", got)
	}
	if len(pageLines("a\nb\nc", 2)) != 2 {
		t.Fatal("limit")
	}
	if len(pageLines(strings.Repeat("x", 500), 0)[0]) != 300 {
		t.Fatal("clip")
	}
}

func TestDiffPageNewUpdatedRemoved(t *testing.T) {
	before := []string{
		"AI jobs in Munich",
		"Siemens · Werkstudent AI · 2 openings · posted 1 day ago",
		"BCG · Intern",
	}
	after := []string{
		"AI jobs in Munich",
		"SAP · Working Student AI Engineer",
		"Siemens · Werkstudent AI · 3 openings · posted 2 days ago",
	}
	d := diffPage(before, after)
	if len(d.Added) != 1 || d.Added[0] != "SAP · Working Student AI Engineer" {
		t.Fatalf("added %+v", d)
	}
	if len(d.Updated) != 1 || d.Updated[0] != "Siemens · Werkstudent AI · 3 openings · posted 2 days ago" {
		t.Fatalf("updated %+v", d)
	}
	if len(d.Removed) != 1 || d.Removed[0] != "BCG · Intern" {
		t.Fatalf("removed %+v", d)
	}
	none := diffPage(before, before)
	if len(none.Added)+len(none.Updated)+len(none.Removed) != 0 {
		t.Fatalf("%+v", none)
	}
}

func TestDiffPageNumberUpdates(t *testing.T) {
	pairs := [][2]string{
		{"Price: $399.99", "Price: $279.99"},
		{"Only 3 left", "Only 0 left"},
		{"Tickets available: 12", "Tickets available: 0"},
		{"Latest release 1.2.3", "Latest release 2.0.0"},
		{"1 comment", "2 comments"},
	}
	for _, p := range pairs {
		d := diffPage([]string{"Shop", p[0]}, []string{"Shop", p[1]})
		if len(d.Added) != 0 || len(d.Removed) != 0 || len(d.Updated) != 1 || d.Updated[0] != p[1] {
			t.Fatalf("%s -> %+v", p[0], d)
		}
	}
}

func TestDiffPageRemovedNumberedLine(t *testing.T) {
	d := diffPage([]string{"Listing 101 · 2 bed", "Listing 102 · 2 bed"}, []string{"Listing 102 · 2 bed"})
	if len(d.Removed) != 1 || d.Removed[0] != "Listing 101 · 2 bed" || len(d.Added)+len(d.Updated) != 0 {
		t.Fatalf("%+v", d)
	}
	d = diffPage([]string{"1. Apple", "2. Banana", "3. Cherry"}, []string{"1. Apple", "2. Cherry"})
	if len(d.Updated) != 1 || d.Updated[0] != "2. Cherry" || len(d.Removed) != 1 || d.Removed[0] != "2. Banana" {
		t.Fatalf("%+v", d)
	}
}

func TestDescribeAndCountPageDiff(t *testing.T) {
	added := make([]string, 10)
	for i := range added {
		added[i] = "Role " + string(rune('0'+i))
	}
	diff := PageDiff{Added: added, Updated: []string{"Price $12"}}
	text := describePageDiff(diff)
	if !strings.HasPrefix(text, "New:\n• Role 0\n") || !strings.Contains(text, "+2 more\nUpdated:\n• Price $12") || strings.Contains(text, "Removed") {
		t.Fatalf("%q", text)
	}
	if countPageDiff(diff) != "11 lines changed (10 new, 1 updated)" {
		t.Fatal(countPageDiff(diff))
	}
	if countPageDiff(PageDiff{Added: []string{"a"}}) != "1 line changed (1 new)" {
		t.Fatal(countPageDiff(PageDiff{Added: []string{"a"}}))
	}
	if describePageDiff(PageDiff{}) != "" || countPageDiff(PageDiff{}) != "" {
		t.Fatal("empty")
	}
}

func TestRelativeTimeIsNotNews(t *testing.T) {
	before := []string{
		"3 points by ada 58 minutes ago | hide",
		"Vor 2 Stunden veröffentlicht",
		"Show HN: A new tool",
	}
	after := []string{
		"3 points by ada 1 hour ago | hide",
		"Vor einer Stunde veröffentlicht",
		"Show HN: A new tool",
	}
	d := diffPage(before, after)
	if len(d.Added)+len(d.Updated)+len(d.Removed) != 0 {
		t.Fatalf("%+v", d)
	}
	d = diffPage(before, append([]string{"4 points by ada 1 hour ago | hide"}, after[1:]...))
	if len(d.Updated) != 1 || d.Updated[0] != "4 points by ada 1 hour ago | hide" {
		t.Fatalf("%+v", d)
	}
}

func TestWithoutRelativeTimesKeepsPrice(t *testing.T) {
	page := func(time, price string) string { return "Posted " + time + " · Price " + price }
	if withoutRelativeTimes(page("58 minutes ago", "$399")) != withoutRelativeTimes(page("1 hour ago", "$399")) {
		t.Fatal("time")
	}
	if withoutRelativeTimes(page("1 hour ago", "$399")) == withoutRelativeTimes(page("1 hour ago", "$279")) {
		t.Fatal("price")
	}
}
