package desktop

import "reflect"

// chromeStatus 用 IsPaused、agentWorking 和待审批数量合成托盘状态。
type chromeStatus int

const (
	chromeIdle chromeStatus = iota
	chromeWorking
	chromePaused
	chromeApproval
)

// chromeStatusOf 按暂停、待审批、正在工作、空闲的顺序取托盘状态。暂停优先于审批，审批优先于工作。
// paused 为真时返回 chromePaused。
// working 只在 paused 为假且 approvals 不大于 0 时，为真则返回 chromeWorking。
// approvals 大于 0 且 paused 为假时返回 chromeApproval。
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

// sliceLen 返回切片或数组的长度。nil 和其他类型返回 0。
// v 为 nil 时返回 0。是切片或数组时返回长度，其他类型返回 0。
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

// pendingApprovalCount 在 App 和 RPC 都空时返回 0。否则返回 PendingApprovals 的长度。
func (s *Service) pendingApprovalCount() int {
	if s.App == nil && s.RPC == nil {
		return 0
	}
	return sliceLen(s.PendingApprovals())
}

// agentWorking 在 agentBusy 为真时返回真。App 和 RPC 都空时返回假。否则看 RunningIDs 是否非空。
func (s *Service) agentWorking() bool {
	if s.agentBusy.Load() {
		return true
	}
	if s.App == nil && s.RPC == nil {
		return false
	}
	return len(s.RunningIDs()) > 0
}

// chromeStatus 用 IsPaused、agentWorking 和待审批数量合成托盘状态。
func (s *Service) chromeStatus() chromeStatus {
	return chromeStatusOf(s.IsPaused(), s.agentWorking(), s.pendingApprovalCount())
}
