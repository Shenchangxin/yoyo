package desktop

import "testing"

func TestPlaceIdolRejectsBlank(t *testing.T) {
	s := &Service{}
	if err := s.PlaceIdol("  "); err == nil {
		t.Fatal("blank id")
	}
}

func TestIdolDeskEvent(t *testing.T) {
	if !idolDeskEvent("pin") || !idolDeskEvent("save_idol") || idolDeskEvent("turn") {
		t.Fatal("desk events")
	}
}
