package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/schedule"
)

func TestBriefSlotsDefaultsAndOff(t *testing.T) {
	slots := DeskConfig{}.BriefSlots()
	if len(slots) != 2 || slots[0].Minute != 8*60+30 || slots[1].Minute != 21*60+30 {
		t.Fatalf("defaults = %+v", slots)
	}
	slots = DeskConfig{BriefMorning: "off", BriefEvening: "22:15"}.BriefSlots()
	if len(slots) != 1 || slots[0].Slot != "evening" || slots[0].Minute != 22*60+15 {
		t.Fatalf("off and custom = %+v", slots)
	}
}

func TestDueBriefsWindow(t *testing.T) {
	d := DeskConfig{BriefMorning: "08:30", BriefEvening: "23:30"}
	at := func(day, h, m int) time.Time { return time.Date(2026, 11, day, h, m, 0, 0, time.Local) }
	if got := dueBriefs(d, at(27, 8, 29)); len(got) != 0 {
		t.Fatalf("before the slot = %+v", got)
	}
	got := dueBriefs(d, at(27, 9, 40))
	if len(got) != 1 || got[0].Slot != "morning" || got[0].Day != "2026-11-27" {
		t.Fatalf("inside the window = %+v", got)
	}
	if got := dueBriefs(d, at(27, 10, 30)); len(got) != 0 {
		t.Fatalf("after the window = %+v", got)
	}
	got = dueBriefs(d, at(28, 0, 40))
	if len(got) != 1 || got[0].Slot != "evening" || got[0].Day != "2026-11-27" {
		t.Fatalf("evening window past midnight = %+v", got)
	}
}

func TestClaimBriefBacksOff(t *testing.T) {
	a := &App{}
	now := time.Date(2026, 11, 27, 8, 31, 0, 0, time.Local)
	if !a.claimBrief("morning:2026-11-27", now) {
		t.Fatal("first claim")
	}
	if a.claimBrief("morning:2026-11-27", now.Add(time.Minute)) {
		t.Fatal("still writing")
	}
	a.releaseBrief("morning:2026-11-27")
	if a.claimBrief("morning:2026-11-27", now.Add(5*time.Minute)) {
		t.Fatal("should wait before retrying")
	}
	if !a.claimBrief("morning:2026-11-27", now.Add(briefRetry+time.Minute)) {
		t.Fatal("retry after the backoff")
	}
}

func TestRoutineResultsGoThroughTheGate(t *testing.T) {
	a := &App{}
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.Local)
	job := schedule.Job{ID: "job-1", Prompt: "看一下昨天的报错\n细节略"}
	a.noteRoutine(job, "s-1", "[quiet]", nil, now)
	a.noteRoutine(job, "s-2", "  ", nil, now)
	a.noteRoutine(job, "s-5", "[quiet]。", nil, now)
	if facts := a.routineFacts(now); len(facts) != 0 {
		t.Fatalf("quiet runs should not become facts, %+v", facts)
	}
	a.noteRoutine(job, "s-3", "昨天有 3 个新报错，集中在登录。", nil, now)
	a.noteRoutine(schedule.Job{ID: "job-2", Prompt: "周报"}, "s-4", "", errors.New("paused"), now)
	facts := a.routineFacts(now)
	if len(facts) != 2 {
		t.Fatalf("facts = %+v", facts)
	}
	if facts[0].Name != "看一下昨天的报错" || facts[0].Class != "routine:job-1" || len(facts[0].Refs) != 1 || facts[0].Refs[0].Target != "s-3" {
		t.Fatalf("done fact = %+v", facts[0])
	}
	if facts[1].State != "failed" || facts[1].Rank != 1 || !strings.Contains(facts[1].Reason, "没跑成") {
		t.Fatalf("failed fact = %+v", facts[1])
	}
	if stale := a.routineFacts(now.Add(routineKeep + time.Hour)); len(stale) != 0 {
		t.Fatalf("stale results should be dropped, %+v", stale)
	}
}
