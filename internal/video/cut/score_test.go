package cut

import "testing"

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

func TestSpeechRatio(t *testing.T) {
	r := speechRatio(0, 10, []Silence{{Start: 0, End: 5}})
	if r < 0.45 || r > 0.55 {
		t.Fatalf("got %v", r)
	}
}
