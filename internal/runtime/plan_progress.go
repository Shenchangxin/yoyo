// plan_progress.go 给 runtime 之外的循环读计划进度。桌面 agentloop/desk 在回合正常结束时用它判断要不要补一张 plan_open 卡片。

package runtime

// PlanProgress 返回 t 当前计划里的清单步数和未完成步数。t 为 nil 或计划为空时都是 0。
// 计划文本在工具集的锁里读，和 update_plan 并发也安全；步数的认法与 PlanOpen 相同。
func PlanProgress(t *WorkspaceTools) (total, open int) {
	return planSteps(planTextOf(t))
}

// PlanOpenInfo 是计划仍有 open 步（共 total 步）未完成时的可重试卡片，种类 plan_open，与 runtime.Run 原来发出的一致。
func PlanOpenInfo(open, total int) ErrorInfo {
	return planOpenInfo(open, total)
}
