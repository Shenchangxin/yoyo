// 锁住 FSRS-6 的数值和官方 Go 实现一致：初始状态、当天复习、遗忘、目标记住率换算，以及从 SM-2 迁移。

package learn

import (
	"math"
	"testing"
	"time"
)

type fsrsStep struct {
	days   float64
	rating Rating
	s, d   float64
}

// TestFSRSMatchesReference 逐步对照 go-fsrs v4.0.0 在默认参数下算出的稳定度和难度。
func TestFSRSMatchesReference(t *testing.T) {
	seqs := [][]fsrsStep{
		{
			{0, RatingGood, 2.3065000000, 2.1181039705},
			{2, RatingGood, 10.9643323358, 2.1112142358},
			{6, RatingGood, 33.2190589551, 2.1043313908},
			{20, RatingAgain, 2.4319265846, 7.3899757880},
			{0, RatingGood, 2.4319265846, 7.3778141815},
			{3, RatingHard, 5.2812571467, 8.2444992291},
			{9, RatingEasy, 21.1823315842, 7.6431210471},
		},
		{
			{0, RatingAgain, 0.2120000000, 6.4133000000},
			{0, RatingGood, 0.2466891878, 6.4021150693},
			{1, RatingGood, 2.0214772516, 6.3909413235},
			{4, RatingGood, 9.0093617736, 6.3797787515},
		},
		{
			{0, RatingEasy, 8.2956000000, 1.0000000000},
			{10, RatingGood, 44.0969424425, 1.0000000000},
			{40, RatingGood, 161.9232670096, 1.0000000000},
		},
		{
			{0, RatingHard, 1.2931000000, 5.1121707056},
			{1, RatingAgain, 0.3753949288, 8.3786321175},
			{1, RatingHard, 1.0718498796, 8.9088888131},
			{3, RatingGood, 3.3467400320, 8.8952082936},
		},
	}
	for i, seq := range seqs {
		var cur Memory
		for j, st := range seq {
			cur = NextMemory(cur, st.days, st.rating)
			if math.Abs(cur.Stability-st.s) > 1e-8 || math.Abs(cur.Difficulty-st.d) > 1e-8 {
				t.Fatalf("seq %d step %d: got S=%.10f D=%.10f want S=%.10f D=%.10f", i, j, cur.Stability, cur.Difficulty, st.s, st.d)
			}
		}
	}
}

// TestFSRSCurveAndInterval 确认间隔等于稳定度时记住率是九成，降低目标记住率会拉长间隔。
func TestFSRSCurveAndInterval(t *testing.T) {
	if got := Retrievability(10, 10); math.Abs(got-0.9) > 1e-12 {
		t.Fatalf("R(S,S) = %v", got)
	}
	if got := Retrievability(1, 2.3065); math.Abs(got-0.9468474994) > 1e-9 {
		t.Fatalf("R(1,2.3065) = %v", got)
	}
	if got := Retrievability(30, 10); math.Abs(got-0.8093881036) > 1e-9 {
		t.Fatalf("R(30,10) = %v", got)
	}
	if got := FSRSInterval(7.5, 0.9); math.Abs(got-7.5) > 1e-9 {
		t.Fatalf("interval at 0.9 = %v", got)
	}
	next := NextMemory(Memory{Stability: 10, Difficulty: 5}, 10, RatingGood)
	if math.Abs(next.Stability-32.0267294820) > 1e-8 {
		t.Fatalf("S = %v", next.Stability)
	}
	if got := FSRSInterval(next.Stability, 0.8); math.Abs(got-106.1993470468) > 1e-6 {
		t.Fatalf("interval at 0.8 = %v", got)
	}
	if ClampRetention(0) != DefaultRetention || ClampRetention(0.2) != MinRetention || ClampRetention(0.999) != MaxRetention {
		t.Fatal("retention clamp drifted")
	}
}

// TestFSRSSameDayNeverShrinksOnGood 确认当天复习评为吃力以上时稳定度不会变小，重来可以变小。
func TestFSRSSameDayNeverShrinksOnGood(t *testing.T) {
	cur := Memory{Stability: 50, Difficulty: 5}
	for _, r := range []Rating{RatingHard, RatingGood, RatingEasy} {
		if next := NextMemory(cur, 0, r); next.Stability < cur.Stability {
			t.Fatalf("rating %d shrank stability to %v", r, next.Stability)
		}
	}
	if next := NextMemory(cur, 0, RatingAgain); next.Stability >= cur.Stability {
		t.Fatalf("again kept stability %v", next.Stability)
	}
}

// TestMemoryFromSM2 确认旧易度和间隔折出的状态和官方迁移公式一致，无效输入当成没复习过。
func TestMemoryFromSM2(t *testing.T) {
	m := MemoryFromSM2(2.5, 10)
	if math.Abs(m.Stability-10) > 1e-9 || math.Abs(m.Difficulty-6.914055235771551) > 1e-8 {
		t.Fatalf("%+v", m)
	}
	if m := MemoryFromSM2(1.3, 1); m.Stability != 1 || m.Difficulty != 10 {
		t.Fatalf("%+v", m)
	}
	if MemoryFromSM2(2.5, 0).Seen() || MemoryFromSM2(1, 5).Seen() {
		t.Fatal("invalid sm2 state must stay unseen")
	}
}

// TestDueAfterAndElapsed 确认到期时刻受最短间隔约束，整天数向下取整。
func TestDueAfterAndElapsed(t *testing.T) {
	at := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	due := dueAfter(at, Memory{Stability: 0.2, Difficulty: 5}, 0.9, 1)
	if got := due.Sub(at); got != 24*time.Hour {
		t.Fatalf("min interval = %v", got)
	}
	due = dueAfter(at, Memory{Stability: 4, Difficulty: 5}, 0.9, 1)
	if got := due.Sub(at).Hours(); math.Abs(got-96) > 1e-6 {
		t.Fatalf("interval hours = %v", got)
	}
	if elapsedWholeDays(at, at.Add(47*time.Hour)) != 1 || elapsedWholeDays(at, at.Add(-time.Hour)) != 0 || elapsedWholeDays(time.Time{}, at) != 0 {
		t.Fatal("elapsed days drifted")
	}
	if r, ok := ParseRating("hard"); !ok || r != RatingHard || r.String() != "hard" {
		t.Fatal("rating names drifted")
	}
	if _, ok := ParseRating("fine"); ok {
		t.Fatal("unknown rating accepted")
	}
}
