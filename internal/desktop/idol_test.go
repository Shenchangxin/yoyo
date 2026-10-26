package desktop

import "testing"

// TestPlaceIdolRejectsBlank 锁住空白 id 会被 PlaceIdol 拒绝。
func TestPlaceIdolRejectsBlank(t *testing.T) {
	s := &Service{}
	if err := s.PlaceIdol("  "); err == nil {
		t.Fatal("blank id")
	}
}

// TestIdolDeskEvent 锁住 pin 和 save_idol 会通知桌面，turn 不会。
func TestIdolDeskEvent(t *testing.T) {
	if !idolDeskEvent("pin") || !idolDeskEvent("save_idol") || idolDeskEvent("turn") {
		t.Fatal("desk events")
	}
}
