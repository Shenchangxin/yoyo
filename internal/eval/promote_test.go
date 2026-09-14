package eval

import "testing"

func TestShouldPromote(t *testing.T) {
	if !ShouldPromote(4, 3, 5, 3) {
		t.Fatal("held-in improve, held-out flat should accept")
	}
	if ShouldPromote(4, 3, 5, 2) {
		t.Fatal("held-out drop should reject")
	}
	if ShouldPromote(4, 3, 4, 3) {
		t.Fatal("no improvement should reject")
	}
	ok, reason := Promote(Metrics{HeldInPass: 1, HeldOutPass: 1}, Metrics{HeldInPass: 2, HeldOutPass: 1, SafetyFail: 1})
	if ok {
		t.Fatal(reason)
	}
}
