package runtime

import (
	"errors"
	"testing"
)

func TestMeterBudgetStop(t *testing.T) {
	m := &Meter{USDPerMTok: 1e6}
	m.Add(2, 0)
	err := m.Check(0.5)
	var b ErrBudget
	if !errors.As(err, &b) {
		t.Fatalf("want ErrBudget, got %v", err)
	}
	if b.Used < 1 {
		t.Fatalf("used=%v", b.Used)
	}
	if m.Check(0) != nil {
		t.Fatal("zero cap is unlimited")
	}
}
