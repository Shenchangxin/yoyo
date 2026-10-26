// 个人事务桌面入口：快照、回答、审批、目标和监视。引擎由 openPersonal 打开。引擎为空时这些方法返回 no personal engine。

package app

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/personal"
)

// PersonalSnapshot 返回任务、目标、监视、想法、产物、提案和选择。引擎为空时 worker.running 为 false，lid_close_stops 和 awake_required 为 true。有记忆库时附上最多 32 条画像记忆。
func (a *App) PersonalSnapshot() map[string]any {
	out := map[string]any{
		"tasks": nil, "goals": nil, "monitors": nil, "ideas": nil,
		"artifacts": nil, "proposals": nil, "choices": nil,
		"worker":          map[string]any{"running": false},
		"lid_close_stops": true,
		"awake_required":  true,
	}
	if a.Personal != nil {
		s := a.Personal.Snapshot()
		out["tasks"] = s.Tasks
		out["goals"] = s.Goals
		out["monitors"] = s.Monitors
		out["ideas"] = s.Ideas
		out["artifacts"] = s.Artifacts
		out["proposals"] = s.Proposals
		out["choices"] = s.Choices
		out["worker"] = s.Worker
	}
	if a.Memory != nil {
		out["memory"] = a.Memory.Search("", memory.KindProfile, 32)
	}
	return out
}

// PersonalAnswer 把文本交给个人引擎的 Answer。
// id 传给 Personal.Answer。
// text 传给 Personal.Answer。
func (a *App) PersonalAnswer(id, text string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Answer(id, text)
}

// PersonalDecide 按 hash 批准或拒绝一条提案。
// approve 传给 Personal.Decide。
func (a *App) PersonalDecide(id, hash string, approve bool) (personal.Proposal, error) {
	if a.Personal == nil {
		return personal.Proposal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Decide(id, hash, approve)
}

// PersonalIdea 按 action 处理一条想法。
// id 传给 Personal.DecideIdea。
func (a *App) PersonalIdea(id, action string) (personal.Idea, error) {
	if a.Personal == nil {
		return personal.Idea{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.DecideIdea(id, action)
}

// PersonalGoal 用标题、描述和里程碑创建目标。
// title 传给 Personal.CreateGoal。
// description 传给 Personal.CreateGoal。
// milestones 传给 Personal.CreateGoal。
func (a *App) PersonalGoal(title, description string, milestones []string) (personal.Goal, error) {
	if a.Personal == nil {
		return personal.Goal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.CreateGoal(title, description, "", milestones)
}

// PersonalWatch 创建监视，并返回 monitor 和 task。
// title 传给 Personal.Watch。
// url 传给 Personal.Watch。
// condition 传给 Personal.Watch。
// value 传给 Personal.Watch。
// interval 传给 Personal.Watch。
func (a *App) PersonalWatch(title, url, condition, value string, interval int) (map[string]any, error) {
	if a.Personal == nil {
		return nil, fmt.Errorf("no personal engine")
	}
	m, task, err := a.Personal.Watch(title, url, condition, value, interval)
	if err != nil {
		return nil, err
	}
	return map[string]any{"monitor": m, "task": task}, nil
}

// PersonalCancel 取消一条个人任务。
// id 传给 Personal.Cancel。
func (a *App) PersonalCancel(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Cancel(id)
}

// PersonalChoice 为选择记录选定 option。
// id 传给 Personal.SelectChoice。
func (a *App) PersonalChoice(id, option string) (personal.Choice, error) {
	if a.Personal == nil {
		return personal.Choice{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.SelectChoice(id, option)
}

// PersonalRefreshIdeas 让个人引擎刷新想法。
func (a *App) PersonalRefreshIdeas() error {
	if a.Personal == nil {
		return fmt.Errorf("no personal engine")
	}
	return a.Personal.RefreshIdeas()
}

// PersonalPause 暂停一条个人任务。
// id 传给 Personal.Pause。
func (a *App) PersonalPause(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Pause(id)
}

// PersonalResume 恢复一条个人任务。
// id 传给 Personal.Resume。
func (a *App) PersonalResume(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Resume(id)
}

// PersonalRetry 重试一条个人任务。
// id 传给 Personal.Retry。
func (a *App) PersonalRetry(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Retry(id)
}

// PersonalGoalStatus 设置目标状态。
// id 传给 Personal.SetGoalStatus。
// status 传给 Personal.SetGoalStatus。
func (a *App) PersonalGoalStatus(id, status string) (personal.Goal, error) {
	if a.Personal == nil {
		return personal.Goal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.SetGoalStatus(id, status)
}

// PersonalMilestone 按 done 切换目标上的一个里程碑。
// goalID 传给 Personal.ToggleMilestone。
// milestoneID 传给 Personal.ToggleMilestone。
func (a *App) PersonalMilestone(goalID, milestoneID string, done bool) (personal.Goal, error) {
	if a.Personal == nil {
		return personal.Goal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.ToggleMilestone(goalID, milestoneID, done)
}

// PersonalMonitorStatus 设置监视状态。
// id 传给 Personal.SetMonitorStatus。
// status 传给 Personal.SetMonitorStatus。
func (a *App) PersonalMonitorStatus(id, status string) (personal.Monitor, error) {
	if a.Personal == nil {
		return personal.Monitor{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.SetMonitorStatus(id, status)
}
