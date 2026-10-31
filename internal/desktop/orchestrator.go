// orchestrator.go 把编排台方法挂到 Wails。隔离进程时改走 JSON-RPC。

package desktop

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// OrchestratorView 返回编排台快照。
func (s *Service) OrchestratorView() (superagent.View, error) {
	if s.RPC != nil {
		return decode[superagent.View](s.call("orchestrator.view", nil))
	}
	if s.App == nil {
		return superagent.View{}, fmt.Errorf("app is not running")
	}
	return s.App.OrchestratorView(), nil
}

// OrchestratorDecide 评估 triggerID。kind 与 reason 写进决定。
func (s *Service) OrchestratorDecide(triggerID, kind, reason string) error {
	if s.RPC != nil {
		_, err := s.call("orchestrator.decide", map[string]any{"triggerId": triggerID, "kind": kind, "reason": reason})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("app is not running")
	}
	return s.App.OrchestratorDecide(triggerID, kind, reason)
}

// OrchestratorMode 换成 mode。mode 只能是 off、observe 或 auto。
func (s *Service) OrchestratorMode(mode string) error {
	if s.RPC != nil {
		_, err := s.call("orchestrator.mode", map[string]any{"mode": mode})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("app is not running")
	}
	return s.App.OrchestratorMode(mode)
}

// OrchestratorPaused 设置暂停。paused 为真时两秒轮次不接受新触发。
func (s *Service) OrchestratorPaused(paused bool) error {
	if s.RPC != nil {
		_, err := s.call("orchestrator.paused", map[string]any{"paused": paused})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("app is not running")
	}
	return s.App.OrchestratorPaused(paused)
}

// OrchestratorAdmission 写下三格准入。listedUnaddressed、unlistedAddressed、unlistedUnaddressed 只能是空、explicit、inferred 或 ignore。
func (s *Service) OrchestratorAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed string) error {
	if s.RPC != nil {
		_, err := s.call("orchestrator.admission", map[string]any{
			"listedUnaddressed":   listedUnaddressed,
			"unlistedAddressed":   unlistedAddressed,
			"unlistedUnaddressed": unlistedUnaddressed,
		})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("app is not running")
	}
	return s.App.OrchestratorAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed)
}
