// worker.go 拥有执行器和工人之间的缝：Worker 接口、工人能力、开工参数、外部工人的打开函数，以及包住 dsh.Agent 的 dshWorker。
// 不拥有回合循环和催促规则（executor.go），也不拥有外部进程（acpworker）。
// 执行器在 runAssignment 里选工人；app 通过 Runtime.SetOpener 接上外部 agent。

package superagent

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// YoyoAgent 是 yoyo 自带工人的 agent 编号。空串和它都走 dsh。
const YoyoAgent = "yoyo"

// Worker 是一张工单上的一个工人。dsh 代理和外部 agent 都实现它，执行器只看这些方法。
type Worker interface {
	// Followup 排在当前一轮之后；空闲时立刻开一轮。
	Followup(msg dsh.UserMessage)
	// Steer 在一步中间插话。不支持时返回假，调用方改用 Followup。
	Steer(msg dsh.UserMessage) bool
	// Cancel 停下这一轮。keepInbox 为假时清掉还没读的输入。
	Cancel(cause dsh.CancelCause, keepInbox bool)
	// WhenIdle 阻塞到这一轮和排队的输入都跑完。
	WhenIdle()
	// Status 返回 running 或 idle。
	Status() string
	// Session 返回归一化的会话日志。任务页和 worker_log 只读它。
	Session() *dsh.Session
	// Restore 把已存的日志灌回去，回合号接在 turn 后面。
	Restore(events []dsh.Event, turn int)
	// Caps 返回这个工人的能力。
	Caps() WorkerCaps
	// Close 释放外部进程。dsh 工人什么都不做。
	Close() error
}

// WorkerCaps 是执行器按工人区分的规则。dsh 为 {4, true, true, false, 0}；外部工人 Nudges 为 1，PlanGate 为假。
type WorkerCaps struct {
	// Nudges 是没写完成报告时最多催几次。
	Nudges int
	// PlanGate 为真时，ready 要先检查计划文件里没有未完成步骤。
	PlanGate bool
	// Steer 为真表示能在一步中间插话。
	Steer bool
	// External 为真表示背后是外部进程。工单结束后执行器关掉它。
	External bool
	// TurnMinutes 是一轮最长几分钟，超时的一轮以 turn_timeout 结束。0 表示不限。
	TurnMinutes int
}

// WorkerSpec 是打开一个外部工人需要的全部输入。
type WorkerSpec struct {
	TicketID, AssignmentID, Agent string
	// Pinned 为真表示人用 @ 点名了这个 agent。
	Pinned bool
	// Dir 是工单目录或工单记下的 Directory。
	Dir string
	// Roots 是可读写的其他目录：命名目录和任务组共享目录。
	Roots []string
	// ReadRoots 是只读目录，例如前置任务的目录。
	ReadRoots []string
	// System 是工人说明。
	System string
	// Tools 是这张工单的 orchestrator_* 工具，外部工人经 MCP 调用它们。
	Tools []dsh.Tool
	// Pointer 是上次的外部会话编号，用来续接。
	Pointer string
	// OnPointer 在拿到新的外部会话编号时调用。
	OnPointer func(string)
	// OnUsage 在外部 agent 报用量时调用。
	OnUsage func(Usage)
	// OnAttention 在等批准时写一句，空串清掉。
	OnAttention func(text string)
}

// Opener 为 spec 打开一个外部工人。agent 不在线时返回包着 ErrAgentOffline 的错误。
type Opener func(ctx context.Context, spec WorkerSpec) (Worker, error)

// ErrAgentOffline 表示指定的 agent 现在不能接单。
var ErrAgentOffline = errors.New("agent offline")

// 外部工人一轮结束的两个特殊原因。执行器按它们改报 waiting 或 blocked，不当成失败重试。
const (
	// ReasonTurnTimeout 是一轮超过 TurnMinutes 被停下，TurnEnd.Kind 为 aborted。
	ReasonTurnTimeout = "turn_timeout"
	// ReasonAuthRequired 是 agent 要先登录，TurnEnd.Kind 为 error，原因以它开头。
	ReasonAuthRequired = "auth_required"
)

// externalAgent 在 agent 不是 yoyo 自带工人时为真。
func externalAgent(agent string) bool {
	agent = strings.TrimSpace(agent)
	return agent != "" && agent != YoyoAgent
}

// agentKey 把空 agent 记成 yoyo，计数和视图用同一个名字。
func agentKey(agent string) string {
	if strings.TrimSpace(agent) == "" {
		return YoyoAgent
	}
	return strings.TrimSpace(agent)
}

// dshWorker 把 *dsh.Agent 包成 Worker。行为与执行器原来直接调用代理相同。
type dshWorker struct {
	agent *dsh.Agent
}

// Followup 转给代理。
func (w *dshWorker) Followup(msg dsh.UserMessage) { w.agent.Followup(msg) }

// Steer 转给代理，dsh 总能插话，所以返回真。
func (w *dshWorker) Steer(msg dsh.UserMessage) bool {
	w.agent.Steer(msg)
	return true
}

// Cancel 转给代理。
func (w *dshWorker) Cancel(cause dsh.CancelCause, keepInbox bool) { w.agent.Cancel(cause, keepInbox) }

// WhenIdle 转给代理。
func (w *dshWorker) WhenIdle() { w.agent.WhenIdle() }

// Status 转给代理。
func (w *dshWorker) Status() string { return w.agent.Status() }

// Session 返回代理自己的会话。
func (w *dshWorker) Session() *dsh.Session { return w.agent.Session() }

// Restore 把 events 灌进代理会话。turn 大于 0 时回合号接在它后面。
func (w *dshWorker) Restore(events []dsh.Event, turn int) {
	w.agent.Session().Restore(events)
	if turn > 0 {
		w.agent.ResumeTurn(turn)
	}
}

// Caps 返回 dsh 工人的能力：催四次、检查计划、能插话。
func (w *dshWorker) Caps() WorkerCaps {
	return WorkerCaps{Nudges: maxWorkerNudges, PlanGate: true, Steer: true}
}

// Close 什么都不做。dsh 代理没有外部进程。
func (w *dshWorker) Close() error { return nil }

// newDshWorker 为 ticketID 建 yoyo 自带的代理，登记五个 orchestrator 工具，再挂工作区工具或 shell。
// dir 是工单目录；为空或没有 Mount 时只挂 shell。返回的 jobs 和 wait 来自 Mount，没有时为 nil。
func newDshWorker(e *Executor, ticketID, dir string) (*dshWorker, func() int, func(time.Duration)) {
	reg := dsh.NewRegistry()
	agent := dsh.New("ticket-"+ticketID, e.Model, reg, dsh.Options{
		Prompt: workerPrompt(e, dir, ticketID),
		Model:  "yoyo-dsh",
	})
	for _, tool := range e.orchestratorTools(ticketID, true) {
		reg.Add(tool)
	}
	var jobs func() int
	var wait func(time.Duration)
	if e.Mount != nil && dir != "" {
		jobs, wait = e.Mount(reg, agent, dir, ticketID)
	} else {
		reg.Add(e.shellTool(agent, ticketID))
	}
	return &dshWorker{agent: agent}, jobs, wait
}

// orchestratorTools 返回 ticketID 的五个 orchestrator 工具。planGate 为假时 ready 不检查计划文件。
func (e *Executor) orchestratorTools(ticketID string, planGate bool) []dsh.Tool {
	return []dsh.Tool{
		e.reportToolWith(ticketID, planGate),
		e.replyTool(ticketID),
		e.handbackTool(ticketID),
		e.followTool(ticketID),
		e.unrelatedTool(ticketID),
	}
}

// steerOrFollow 先插话，工人不支持时改成排队。
func steerOrFollow(w Worker, msg dsh.UserMessage) {
	if !w.Steer(msg) {
		w.Followup(msg)
	}
}
