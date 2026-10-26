// 锁住界面缩放只落在 1、1.25、1.5，空值和中间值的吸附结果。

package app

import "testing"

// TestSnapUIScale 锁住 0 和 0.85、1.1 吸到 1，1.2 吸到 1.25，1.4 吸到 1.5。
func TestSnapUIScale(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{1, 1},
		{1.25, 1.25},
		{1.5, 1.5},
		{0, 1},
		{0.85, 1},
		{1.1, 1},
		{1.2, 1.25},
		{1.4, 1.5},
	}
	for _, c := range cases {
		if got := snapUIScale(c.in); got != c.want {
			t.Fatalf("snapUIScale(%v)=%v want %v", c.in, got, c.want)
		}
	}
}
