package initiative

import (
	"testing"
	"time"
)

// TestTwoStoresOnOneFileSeeEachOther 模拟桌面和 daemon 同时打开同一份记录：一边的送达和反馈，另一边看得到，也不会被盖掉。
func TestTwoStoresOnOneFileSeeEachOther(t *testing.T) {
	dir := t.TempDir()
	lead, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 28, 9, 0, 0, 0, time.Local)
	day := now.Format("2006-01-02")
	ui.Feedback("reach", "", "mute", day)
	if _, muted := lead.Quiet(day)["reach"]; !muted {
		t.Fatal("the leader did not see the mute from the other process")
	}
	lead.Deliver([]Signal{{ID: "cal:1", Code: "calendar", Class: "calendar", Speak: true, Sig: "due"}}, now)
	ui.Feedback("calendar", "", "used", day)
	days := lead.Recent(1)
	if len(days) != 1 || days[0].Spoken != 1 || days[0].Used != 1 {
		t.Fatalf("tally = %#v", days)
	}
	if _, muted := ui.Quiet(day)["reach"]; !muted {
		t.Fatal("the leader's delivery overwrote the mute")
	}
}
