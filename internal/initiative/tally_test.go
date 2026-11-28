package initiative

import (
	"testing"
	"time"
)

func TestDeliverAndFeedbackCountTheDay(t *testing.T) {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.Local)
	day := now.Format("2006-01-02")
	due := Fact{ID: "cal:1", Code: "calendar", Class: "calendar", Name: "评审", State: "due", Rank: 1, Eligible: true}
	gone := Fact{ID: "channel:feishu", Code: "channel", Class: "source", Name: "飞书", State: "error", Rank: 3, Eligible: true}
	brief, book := Assess(Input{Now: now, Facts: []Fact{due, gone}})
	book = Deliver(book, brief.Signals, now)
	if got := book.Tally[day].Spoken; got != 2 {
		t.Fatalf("spoken = %d, want 2", got)
	}
	book, _ = Feedback(book, "calendar", "", "used", day)
	book, _ = Feedback(book, "source", "", "ignore", day)
	days := Recent(book, 7)
	if len(days) != 1 || days[0].Day != day || days[0].Used != 1 || days[0].Ignored != 1 {
		t.Fatalf("recent = %+v", days)
	}
	rate, ok := Accepted(days)
	if !ok || rate != 0.5 {
		t.Fatalf("accepted = %v %v", rate, ok)
	}
}

func TestTallyKeepsThirtyDays(t *testing.T) {
	book := cloneBook(Book{})
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local)
	for i := 0; i < 40; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		book = bump(book, day, func(t *Tally) { t.Spoken++ })
	}
	if len(book.Tally) != tallyDays {
		t.Fatalf("kept %d days", len(book.Tally))
	}
	if _, ok := book.Tally["2026-10-01"]; ok {
		t.Fatal("the oldest day should be gone")
	}
	if _, ok := Accepted(nil); ok {
		t.Fatal("no spoken means no rate")
	}
}

func TestSignalCarriesReasonAndRefs(t *testing.T) {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.Local)
	done := Fact{
		ID: "routine:job-1@20261127T0900", Code: "routine", Class: "routine:job-1", Name: "每日汇总", State: "done", Rank: 2, Eligible: true,
		Reason: "你设的例行跑完了",
		Refs:   []Ref{{Kind: "session", Label: "这次运行", Target: "s-1"}, {Kind: "file", Label: "api_key.txt"}, {Kind: "file", Label: " "}},
	}
	brief, _ := Assess(Input{Now: now, Facts: []Fact{done}})
	if len(brief.Signals) != 1 || !brief.Signals[0].Speak {
		t.Fatalf("routine result should speak on first sight, %#v", brief.Signals)
	}
	s := brief.Signals[0]
	if s.Reason != "你设的例行跑完了" || len(s.Refs) != 1 || s.Refs[0].Target != "s-1" {
		t.Fatalf("reason or refs = %#v", s)
	}
}

func TestRoutineEachRunSpeaksOnce(t *testing.T) {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.Local)
	first := Fact{ID: "routine:job-1@a", Code: "routine", Class: "routine:job-1", Name: "巡检", State: "done", Rank: 2, Eligible: true}
	brief, book := Assess(Input{Now: now, Facts: []Fact{first}})
	book = Deliver(book, brief.Signals, now)
	again, _ := Assess(Input{Now: now.Add(time.Minute), Facts: []Fact{first}, Book: book})
	if again.Speak {
		t.Fatalf("a delivered run should stay quiet, %#v", again.Signals)
	}
	second := first
	second.ID = "routine:job-1@b"
	later, _ := Assess(Input{Now: now.Add(time.Hour), Facts: []Fact{second}, Book: book})
	if !later.Speak {
		t.Fatalf("a new run of the same routine should speak, %#v", later.Signals)
	}
	book, _ = Feedback(book, "routine:job-1", "", "mute", now.Format("2006-01-02"))
	muted, _ := Assess(Input{Now: now.Add(2 * time.Hour), Facts: []Fact{second}, Book: book})
	if muted.Speak || len(muted.Signals) != 1 || !muted.Signals[0].Held {
		t.Fatalf("a muted routine is held, %#v", muted.Signals)
	}
}
