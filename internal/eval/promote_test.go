// 锁住晋升门：held-in 上升且 held-out 持平可以通过；任一侧下降或两侧都不变则拒绝。SafetyFail 大于 0 时 Promote 拒绝，即使通过数上升。

package eval

import "testing"

// TestShouldPromote 覆盖持平上升、held-out 下降、两侧不变，以及 SafetyFail 大于 0 时 Promote 返回假。
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
