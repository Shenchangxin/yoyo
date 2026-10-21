package cut

import "testing"

func TestPunchSplitsMiddle(t *testing.T) {
	got := Punch([]Clip{{ID: "a", Start: 0, End: 10, Hash: "h"}}, 3, 5)
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].End != 3 || got[1].Start != 5 || got[0].Hash != "" || got[1].Hash != "" {
		t.Fatalf("%+v", got)
	}
}

func TestPunchDropsCoveredClip(t *testing.T) {
	got := Punch([]Clip{{Start: 2, End: 4}, {Start: 6, End: 8}}, 1, 5)
	if len(got) != 1 || got[0].Start != 6 {
		t.Fatalf("%+v", got)
	}
}

func TestInsertSpanOrders(t *testing.T) {
	got := InsertSpan([]Clip{{Start: 5, End: 8}}, 1, 2)
	if len(got) != 2 || got[0].Start != 1 || got[1].Start != 5 {
		t.Fatalf("%+v", got)
	}
}
