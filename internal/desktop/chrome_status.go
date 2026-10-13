package desktop

import "reflect"

type chromeStatus int

const (
	chromeIdle chromeStatus = iota
	chromeWorking
	chromePaused
	chromeApproval
)

func chromeStatusOf(paused, working bool, approvals int) chromeStatus {
	if paused {
		return chromePaused
	}
	if approvals > 0 {
		return chromeApproval
	}
	if working {
		return chromeWorking
	}
	return chromeIdle
}

func sliceLen(v any) int {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len()
	default:
		return 0
	}
}

func (s *Service) pendingApprovalCount() int {
	if s.App == nil && s.RPC == nil {
		return 0
	}
	return sliceLen(s.PendingApprovals())
}

func (s *Service) agentWorking() bool {
	if s.agentBusy.Load() {
		return true
	}
	if s.App == nil && s.RPC == nil {
		return false
	}
	return len(s.RunningIDs()) > 0
}

func (s *Service) chromeStatus() chromeStatus {
	return chromeStatusOf(s.IsPaused(), s.agentWorking(), s.pendingApprovalCount())
}
