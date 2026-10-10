package personal

import (
	"context"
	"time"
)

func (e *Engine) Run(ctx context.Context) {
	e.mu.Lock()
	e.running = true
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()
	poll := e.Poll
	if poll <= 0 {
		poll = time.Second
	}
	tick := time.NewTicker(poll)
	defer tick.Stop()
	e.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			e.mu.Lock()
			for _, c := range e.abort {
				c()
			}
			e.mu.Unlock()
			return
		case <-tick.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if e.Paused != nil && e.Paused() {
		return
	}
	now := e.now()
	e.mu.Lock()
	e.lastTick = now
	e.mu.Unlock()
	e.expireProposals()
	e.recoverIdeas()
	if now.Sub(e.store.lastIdeasAt()) >= 15*time.Minute {
		_ = e.RefreshIdeas()
	}
	var due []Task
	e.mu.Lock()
	inflight := len(e.inFlight)
	e.mu.Unlock()
	for _, t := range e.store.tasks() {
		e.mu.Lock()
		_, busy := e.inFlight[t.ID]
		e.mu.Unlock()
		if busy {
			continue
		}
		if !e.isDue(t, now) {
			continue
		}
		if t.Status == StatusWaitingApproval {
			if t.ActionID == "" {
				continue
			}
			p, ok := e.store.proposal(t.ActionID)
			if !ok {
				continue
			}
			if p.Status == ProposalAwaiting || p.Status == ProposalExecuting || p.Status == ProposalOutcomeUnknown {
				continue
			}
		}
		if t.Kind == KindAgent {
			continue
		}
		due = append(due, t)
		if inflight+len(due) >= 3 {
			break
		}
	}
	for i := range due {
		t := due[i]
		e.mu.Lock()
		e.inFlight[t.ID] = struct{}{}
		e.mu.Unlock()
		go func() {
			defer func() {
				e.mu.Lock()
				delete(e.inFlight, t.ID)
				e.mu.Unlock()
			}()
			e.runOne(ctx, t)
		}()
	}
}

func (e *Engine) isDue(t Task, now time.Time) bool {
	switch t.Status {
	case StatusQueued:
		return true
	case StatusScheduled:
		return t.NextRunAt != nil && !t.NextRunAt.After(now)
	case StatusRunning:
		return t.LeaseUntil != nil && !t.LeaseUntil.After(now)
	case StatusWaitingApproval:
		return true
	default:
		return false
	}
}

func (e *Engine) runOne(ctx context.Context, previous Task) {
	if ctx.Err() != nil {
		return
	}
	leaseID := e.newID("lease")
	leaseUntil := e.now().Add(e.Lease)
	expected := string(previous.Status)
	if previous.Status == StatusRunning {
		expected = string(StatusRunning)
	}
	claimed, ok := e.store.casTask(previous.ID, expected, previous.LeaseID, func(cur *Task) bool {
		if previous.Status == StatusRunning && cur.LeaseID != previous.LeaseID {
			return false
		}
		cur.Status = StatusRunning
		cur.LeaseID = leaseID
		cur.LeaseUntil = &leaseUntil
		cur.Attempts++
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.abort[claimed.ID] = cancel
	e.mu.Unlock()
	defer func() {
		cancel()
		e.mu.Lock()
		delete(e.abort, claimed.ID)
		e.mu.Unlock()
	}()
	beat := e.Lease / 3
	if beat < time.Millisecond {
		beat = time.Millisecond
	}
	hb := time.NewTicker(beat)
	defer hb.Stop()
	go func() {
		for {
			select {
			case <-runCtx.Done():
				return
			case <-hb.C:
				until := e.now().Add(e.Lease)
				_, ok := e.store.casTask(claimed.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
					cur.LeaseUntil = &until
					return true
				})
				if !ok {
					cancel()
					return
				}
			}
		}
	}()
	_, err := e.execute(runCtx, claimed, leaseID)
	if isLostLease(err) || runCtx.Err() != nil {
		e.store.casTask(claimed.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusQueued
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		return
	}
	if err != nil {
		e.store.casTask(claimed.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusFailed
			cur.Error = err.Error()
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
	}
}
