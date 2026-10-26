// 锁住 Meter 的超额停止和零上限：费率 1e6 时两枚输入 token 超过 0.5 美元，Check(0) 返回 nil。

package runtime

import (
	"errors"
	"testing"
)

// TestMeterBudgetStop 确认 Add(2, 0) 之后 Check(0.5) 得到 ErrBudget 且 Used 不小于 1，Check(0) 返回 nil。
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
