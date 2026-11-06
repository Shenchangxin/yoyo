package app

import (
	"testing"
	"time"
)

func at(day int, hh, mm int) time.Time {
	// 2026-11-06 是周五，07 是周六。
	return time.Date(2026, 11, day, hh, mm, 0, 0, time.Local)
}

// TestDeskQuietCrossesMidnight 锁住 23:00–08:00 这种跨午夜的安静时段：开始那一分钟算，结束那一分钟不算。
func TestDeskQuietCrossesMidnight(t *testing.T) {
	d := DeskConfig{QuietStart: "23:00", QuietEnd: "08:00"}
	cases := []struct {
		t    time.Time
		want bool
	}{
		{at(6, 22, 59), false},
		{at(6, 23, 0), true},
		{at(6, 2, 30), true},
		{at(6, 7, 59), true},
		{at(6, 8, 0), false},
		{at(6, 12, 0), false},
	}
	for _, c := range cases {
		if got := d.Quiet(c.t); got != c.want {
			t.Fatalf("Quiet(%s)=%v want %v", c.t.Format("15:04"), got, c.want)
		}
	}
}

// TestDeskQuietSameDayAndUnset 锁住白天的区间，以及没填、填错、首尾相同都表示没有安静时段。
func TestDeskQuietSameDayAndUnset(t *testing.T) {
	if !(DeskConfig{QuietStart: "12:00", QuietEnd: "13:30"}).Quiet(at(6, 12, 45)) {
		t.Fatal("12:45 should be quiet inside 12:00-13:30")
	}
	for _, d := range []DeskConfig{{}, {QuietStart: "23:00"}, {QuietStart: "9:00", QuietEnd: "9:00"}, {QuietStart: "25:00", QuietEnd: "08:00"}} {
		if d.Quiet(at(6, 9, 0)) {
			t.Fatalf("%+v should never be quiet", d)
		}
	}
}

// TestDeskHoldToastWeekend 锁住周末只放审批和出错，平日不受影响。
func TestDeskHoldToastWeekend(t *testing.T) {
	d := DeskConfig{QuietWeekend: true}
	if d.HoldToast("done", at(6, 15, 0)) {
		t.Fatal("Friday done toast should pass")
	}
	if !d.HoldToast("done", at(7, 15, 0)) {
		t.Fatal("Saturday done toast should be held")
	}
	if d.HoldToast("approval", at(7, 15, 0)) || d.HoldToast("error", at(7, 15, 0)) {
		t.Fatal("approvals and errors still reach the person on weekends")
	}
	if !(DeskConfig{QuietStart: "23:00", QuietEnd: "08:00"}).HoldToast("approval", at(6, 23, 30)) {
		t.Fatal("quiet hours hold every toast")
	}
}
