// 锁住计划清单的解析：PlanOpen 和 planSteps 怎样数总步数和未完成步数。plan_open 卡片和停止判断在 agentloop/desk。

package runtime

import (
	"testing"
)

// TestPlanOpen 确认空文本和已完成清单不是开着的计划，含 pending 或 in_progress 的清单才是。done 和 finished 也算完成。
func TestPlanOpen(t *testing.T) {
	if PlanOpen("") || PlanOpen("just a paragraph") {
		t.Fatal("empty")
	}
	open := "勘察后列出步骤\n\n1. [in_progress] 勘察现状\n2. [pending] 前端重写\n"
	if !PlanOpen(open) {
		t.Fatal("open plan")
	}
	done := "1. [complete] a\n2. [completed] b\n"
	if PlanOpen(done) {
		t.Fatal("done plan")
	}
	if PlanOpen("1. [done] a\n2. [finished] b\n") {
		t.Fatal("done aliases")
	}
}

// TestPlanSteps 确认三条清单里两条未完成；没有清单时总数和未完成数都是 0。
func TestPlanSteps(t *testing.T) {
	total, open := planSteps("1. [in_progress] a\n2. [complete] b\n3. [pending] c\n")
	if total != 3 || open != 2 {
		t.Fatalf("total=%d open=%d", total, open)
	}
	if total, open := planSteps("no checklist here"); total != 0 || open != 0 {
		t.Fatalf("total=%d open=%d", total, open)
	}
}
