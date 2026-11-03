package initiative

import (
	"testing"
	"time"
)

func TestAssessBaselinesLinesThenSpeaksOnChange(t *testing.T) {
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.Local)
	line := Fact{ID: "line:yoyo", Code: "line", Class: "line", Name: "yoyo", State: "moving", Sig: "moving|main.go", Rank: 5, Eligible: true}
	first, book := Assess(Input{Now: now, Facts: []Fact{line}})
	if first.Speak {
		t.Fatal("first sight should stay quiet")
	}
	second, _ := Assess(Input{Now: now, Facts: []Fact{line}, Book: book})
	if second.Speak {
		t.Fatal("same signature should stay quiet")
	}
	line.State = "stalled"
	line.Sig = "stalled|blank.txt"
	third, book := Assess(Input{Now: now.Add(time.Hour), Facts: []Fact{line}, Book: book})
	if !third.Speak || !third.Signals[0].Speak || third.Signals[0].Why != "changed" {
		t.Fatalf("change = %#v", third)
	}
	line.Sig = "quiet|"
	line.State = "quiet"
	fourth, _ := Assess(Input{Now: now.Add(2 * time.Hour), Facts: []Fact{line}, Book: book})
	if fourth.Speak || !fourth.Signals[0].Held {
		t.Fatalf("same class same day should be held, %#v", fourth.Signals)
	}
}

func TestAssessOverdueSpeaksOnceAndMuteHolds(t *testing.T) {
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.Local)
	due := Fact{ID: "cal:1", Code: "calendar", Class: "calendar", Name: "评审", State: "overdue", Rank: 0, Eligible: true}
	brief, book := Assess(Input{Now: now, Facts: []Fact{due}})
	if !brief.Speak || brief.Signals[0].Why != "overdue" {
		t.Fatalf("%#v", brief)
	}
	book, _ = Feedback(book, "calendar", "", "ignore", "2026-11-03")
	book, _ = Feedback(book, "calendar", "", "ignore", "2026-11-04")
	book, forget := Feedback(book, "calendar", "mem-1", "ignore", "2026-11-05")
	if forget != "" {
		t.Fatal("ignore should not forget")
	}
	held, _ := Assess(Input{Now: now.AddDate(0, 0, 3), Facts: []Fact{due}, Book: book})
	if held.Speak || !held.Signals[0].Held {
		t.Fatalf("three ignores should hold, %#v", held.Signals)
	}
	book, forget = Feedback(book, "calendar", "mem-1", "wrong", "2026-11-06")
	if forget != "mem-1" {
		t.Fatalf("wrong = %q", forget)
	}
}

func TestAssessIdleHoldsGitNotOutage(t *testing.T) {
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.Local)
	git := Fact{ID: "git:yoyo", Code: "git", Class: "git", Name: "yoyo", State: "dirty", Sig: "main/dirty", Rank: 5, Eligible: true}
	_, book := Assess(Input{Now: now, Facts: []Fact{git}})
	git.Sig = "main/clean"
	git.State = "clean"
	held, _ := Assess(Input{Now: now.Add(time.Hour), Idle: 25 * time.Minute, Facts: []Fact{git}, Book: book})
	if held.Speak || !held.Signals[0].Held {
		t.Fatalf("idle git = %#v", held.Signals)
	}
	off := Fact{ID: "net", Code: "net", Class: "net", Name: "network", State: "offline", Rank: 4, Eligible: true}
	out, _ := Assess(Input{Now: now, Idle: time.Hour, Facts: []Fact{off}})
	if !out.Speak {
		t.Fatal("offline still speaks while idle")
	}
}

func TestAssessUncheckedDoesNotSpeak(t *testing.T) {
	now := time.Date(2026, 11, 3, 9, 0, 0, 0, time.Local)
	mail := Fact{ID: "mail", Code: "mail", Class: "mail", Name: "mail", State: "unchecked", Eligible: false}
	brief, book := Assess(Input{Now: now, Welcome: true, Facts: []Fact{mail}})
	if !brief.Speak {
		t.Fatal("welcome still speaks")
	}
	again, _ := Assess(Input{Now: now.Add(time.Hour), Welcome: true, Facts: []Fact{mail}, Book: book})
	if again.Speak {
		t.Fatal("welcome is once a day")
	}
}

func TestScheduleStaysVisibleAndOffStops(t *testing.T) {
	if !AllowSchedule("") || !AllowSchedule("observe") || AllowSchedule("off") {
		t.Fatal("mode")
	}
	book := NoteSchedule(Book{}, "job-1", "整理下载目录的名字", "2026-11-03")
	if len(book.Open) != 1 || book.Open[0].Class != "schedule" {
		t.Fatalf("%#v", book.Open)
	}
	book = NoteSchedule(book, "job-1", "token inside", "2026-11-03")
	if book.Open[0].Title != "job-1" {
		t.Fatalf("secret title leaked: %q", book.Open[0].Title)
	}
}
