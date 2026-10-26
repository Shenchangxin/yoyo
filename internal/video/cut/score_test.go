// score_test.go 检查高光窗口是否落在有语音的区间且互不重叠。

package cut

import "testing"

// TestProposePrefersSpeechOverSilence 确认选出的片段落在静音之外，并且带有 energy 分数。
func TestProposePrefersSpeechOverSilence(t *testing.T) {
	dur := 30.0
	silences := []Silence{{Start: 0, End: 8}, {Start: 22, End: 30}}
	scenes := []float64{8, 15, 22}
	got := Propose(dur, scenes, silences, 2, 8)
	if len(got) == 0 {
		t.Fatal("no clips")
	}
	mid := 0
	for _, c := range got {
		if c.Start >= 7 && c.End <= 23 {
			mid++
		}
		if c.Scores["energy"] < 1 {
			t.Fatalf("missing energy score: %+v", c)
		}
	}
	if mid == 0 {
		t.Fatalf("expected a clip in the speech region, got %+v", got)
	}
}

// TestProposeKeepsNonOverlap 确认三条片段相邻起点不早于上一条结束减去 0.6 秒。
func TestProposeKeepsNonOverlap(t *testing.T) {
	got := Propose(60, []float64{10, 20, 30, 40, 50}, nil, 3, 12)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].Start < got[i-1].End-0.6 {
			t.Fatalf("overlap %+v %+v", got[i-1], got[i])
		}
	}
}

// TestSpeechRatio 确认前一半静音时语音占比落在 0.45 到 0.55。
func TestSpeechRatio(t *testing.T) {
	r := speechRatio(0, 10, []Silence{{Start: 0, End: 5}})
	if r < 0.45 || r > 0.55 {
		t.Fatalf("got %v", r)
	}
}
