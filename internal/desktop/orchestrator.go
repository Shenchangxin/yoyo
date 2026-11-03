// orchestrator.go 把编排台方法挂到 Wails。隔离进程时改走 JSON-RPC。

package desktop

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/app"
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

// OrchestratorDeliver 把还没交给编排台的评估标成已交付。
func (s *Service) OrchestratorDeliver() error {
	if s.RPC != nil {
		_, err := s.call("orchestrator.deliver", nil)
		return err
	}
	if s.App == nil {
		return fmt.Errorf("app is not running")
	}
	s.App.OrchestratorDeliver()
	return nil
}

// OrchestratorTranscript 返回工单会话。ticketID 是工单编号。
func (s *Service) OrchestratorTranscript(ticketID string) ([]superagent.TranscriptLine, error) {
	if s.RPC != nil {
		return decode[[]superagent.TranscriptLine](s.call("orchestrator.transcript", map[string]any{"ticketId": ticketID}))
	}
	if s.App == nil {
		return nil, fmt.Errorf("app is not running")
	}
	return s.App.WorkerTranscript(ticketID), nil
}

// ComposeToday 按画像、记忆和最近对话写今天的简报。
func (s *Service) ComposeToday() (app.TodayBrief, error) {
	if s.RPC != nil {
		return decode[app.TodayBrief](s.call("today.compose", nil))
	}
	if s.App == nil {
		return app.TodayBrief{}, fmt.Errorf("app is not running")
	}
	return s.App.ComposeToday(), nil
}

// OrchestratorAct 执行编排台动作。kind、ticketID、triggerID、text 和 reason 原样交给应用。
func (s *Service) OrchestratorAct(kind, ticketID, triggerID, text, reason string) error {
	if s.RPC != nil {
		_, err := s.call("orchestrator.act", map[string]any{
			"kind": kind, "ticketId": ticketID, "triggerId": triggerID, "text": text, "reason": reason,
		})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("app is not running")
	}
	return s.App.OrchestratorAct(kind, ticketID, triggerID, text, reason)
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
