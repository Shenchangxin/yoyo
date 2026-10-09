package personal

func (e *Engine) abortTask(id string) {
	e.mu.Lock()
	if c, ok := e.abort[id]; ok {
		c()
	}
	e.mu.Unlock()
}

func (e *Engine) Pause(id string) (Task, error) {
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled || cur.Status == StatusPaused {
			return false
		}
		cur.Status = StatusPaused
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot pause task")
	}
	e.abortTask(id)
	if m, ok := e.taskMonitor(id); ok && m.Status == "active" {
		m.Status = "paused"
		e.store.putMonitor(m)
	}
	e.changed()
	return out, nil
}

func (e *Engine) Resume(id string) (Task, error) {
	t, ok := e.store.task(id)
	if !ok {
		return Task{}, errf("unknown task")
	}
	next := StatusQueued
	if t.Kind == KindMonitor {
		next = StatusScheduled
	}
	now := e.now()
	out, ok := e.store.casTask(id, string(StatusPaused), "", func(cur *Task) bool {
		cur.Status = next
		cur.Error = ""
		cur.UpdatedAt = now
		if next == StatusScheduled {
			cur.NextRunAt = &now
		}
		return true
	})
	if !ok {
		return t, errf("task is not paused")
	}
	if m, mok := e.taskMonitor(id); mok && m.Status == "paused" {
		m.Status = "active"
		m.NextCheckAt = now
		e.store.putMonitor(m)
	}
	e.changed()
	return out, nil
}

func (e *Engine) Retry(id string) (Task, error) {
	t, ok := e.store.task(id)
	if !ok {
		return Task{}, errf("unknown task")
	}
	if t.ActionID != "" {
		if p, pok := e.store.proposal(t.ActionID); pok && p.Status == ProposalOutcomeUnknown {
			return t, errf("send outcome is unknown; check the provider before retrying")
		}
	}
	if t.Status != StatusFailed && t.Status != StatusCancelled {
		return t, errf("only failed or cancelled work can be retried")
	}
	now := e.now()
	out, ok := e.store.casTask(id, string(t.Status), "", func(cur *Task) bool {
		cur.Status = StatusQueued
		cur.Error = ""
		cur.Question = ""
		cur.Result = ""
		cur.ActionID = ""
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = now
		return true
	})
	if !ok {
		return t, errf("task changed")
	}
	e.changed()
	return out, nil
}

func (e *Engine) SetGoalStatus(id, status string) (Goal, error) {
	status = normalizeGoalStatus(status)
	g, ok := e.store.casGoal(id, "", func(cur *Goal) bool {
		if status == "" {
			return false
		}
		cur.Status = status
		return true
	})
	if !ok {
		return Goal{}, errf("unknown goal")
	}
	if status == "paused" || status == "done" {
		for _, t := range e.store.tasks() {
			if t.GoalID == id && t.Status != StatusSucceeded && t.Status != StatusCancelled {
				_, _ = e.Pause(t.ID)
			}
		}
	}
	e.changed()
	return g, nil
}

func normalizeGoalStatus(status string) string {
	switch status {
	case "active", "paused", "done":
		return status
	default:
		return ""
	}
}

func (e *Engine) ToggleMilestone(goalID, milestoneID string, done bool) (Goal, error) {
	g, ok := e.store.casGoal(goalID, "", func(cur *Goal) bool {
		for i := range cur.Milestones {
			if cur.Milestones[i].ID == milestoneID {
				cur.Milestones[i].Done = done
				return true
			}
		}
		return false
	})
	if !ok {
		return Goal{}, errf("unknown milestone")
	}
	e.changed()
	return g, nil
}

func (e *Engine) SetMonitorStatus(id, status string) (Monitor, error) {
	if status != "active" && status != "paused" && status != "stopped" {
		return Monitor{}, errf("monitor status must be active, paused, or stopped")
	}
	m, ok := e.store.monitor(id)
	if !ok {
		return Monitor{}, errf("unknown monitor")
	}
	expect := m.Status
	out, ok := e.store.casMonitor(id, expect, -1, func(cur *Monitor) bool {
		cur.Status = status
		if status == "active" {
			cur.NextCheckAt = e.now()
			cur.Error = ""
		}
		return true
	})
	if !ok {
		return m, errf("monitor changed")
	}
	if t, tok := e.store.task(m.TaskID); tok {
		switch status {
		case "paused":
			_, _ = e.Pause(t.ID)
		case "stopped":
			_, _ = e.Cancel(t.ID)
		case "active":
			if t.Status == StatusPaused {
				_, _ = e.Resume(t.ID)
			}
		}
	}
	e.changed()
	return out, nil
}

func (e *Engine) taskMonitor(taskID string) (Monitor, bool) {
	for _, m := range e.store.snapshot().Monitors {
		if m.TaskID == taskID {
			return m, true
		}
	}
	return Monitor{}, false
}

func (e *Engine) InvalidateAccount(account string) {
	if account == "" {
		return
	}
	for _, p := range e.store.snapshot().Proposals {
		if p.Account != account || p.Status != ProposalAwaiting {
			continue
		}
		e.store.casProposal(p.ID, string(ProposalAwaiting), p.Hash, func(cur *Proposal) bool {
			cur.Status = ProposalExpired
			cur.Error = "account or connection changed"
			cur.Activity = append(cur.Activity, Activity{At: e.now(), Status: string(ProposalExpired), Detail: "account disconnected"})
			return true
		})
	}
	e.changed()
}
