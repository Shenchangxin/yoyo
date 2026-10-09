package app

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/personal"
)

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

func (a *App) PersonalAnswer(id, text string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Answer(id, text)
}

func (a *App) PersonalDecide(id, hash string, approve bool) (personal.Proposal, error) {
	if a.Personal == nil {
		return personal.Proposal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Decide(id, hash, approve)
}

func (a *App) PersonalIdea(id, action string) (personal.Idea, error) {
	if a.Personal == nil {
		return personal.Idea{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.DecideIdea(id, action)
}

func (a *App) PersonalGoal(title, description string, milestones []string) (personal.Goal, error) {
	if a.Personal == nil {
		return personal.Goal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.CreateGoal(title, description, "", milestones)
}

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

func (a *App) PersonalCancel(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Cancel(id)
}

func (a *App) PersonalChoice(id, option string) (personal.Choice, error) {
	if a.Personal == nil {
		return personal.Choice{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.SelectChoice(id, option)
}

func (a *App) PersonalRefreshIdeas() error {
	if a.Personal == nil {
		return fmt.Errorf("no personal engine")
	}
	return a.Personal.RefreshIdeas()
}

func (a *App) PersonalPause(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Pause(id)
}

func (a *App) PersonalResume(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Resume(id)
}

func (a *App) PersonalRetry(id string) (personal.Task, error) {
	if a.Personal == nil {
		return personal.Task{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.Retry(id)
}

func (a *App) PersonalGoalStatus(id, status string) (personal.Goal, error) {
	if a.Personal == nil {
		return personal.Goal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.SetGoalStatus(id, status)
}

func (a *App) PersonalMilestone(goalID, milestoneID string, done bool) (personal.Goal, error) {
	if a.Personal == nil {
		return personal.Goal{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.ToggleMilestone(goalID, milestoneID, done)
}

func (a *App) PersonalMonitorStatus(id, status string) (personal.Monitor, error) {
	if a.Personal == nil {
		return personal.Monitor{}, fmt.Errorf("no personal engine")
	}
	return a.Personal.SetMonitorStatus(id, status)
}
