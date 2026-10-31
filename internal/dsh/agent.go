// agent.go 是默认的回合驱动器。
// 它拥有一个代理的一生：收件箱、回合和步骤、取消，以及前台预算后的后台作业。
// 模型调用和工具执行期间不持有锁。

package dsh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	phaseIdle = iota
	phaseRunning
	phaseMaintenance
)

// CancelCause 是取消原因。Kind 为 disposed 时不再拉起后续回合。
type CancelCause struct {
	Kind   string
	Reason string
}

// Error 返回 Kind 和 Reason。
func (c CancelCause) Error() string {
	if c.Reason == "" {
		return c.Kind
	}
	return c.Kind + ": " + c.Reason
}

// Request 是一次模型请求。Messages 已经去掉空的系统节点。
type Request struct {
	Provider  string
	Model     string
	Messages  []Message
	Tools     []ToolSchema
	MaxTokens int
}

// Completer 跑一次模型请求。ctx 取消时返回已说出的前缀和错误。
type Completer interface {
	Complete(ctx context.Context, req Request) (Assistant, error)
}

// Options 是创建代理时的路线和提示。ReplacePrompt 为真时提示变化改写头部，否则追加。
// MaxParallel 小于等于 0 时用 DefaultMaxParallel。PreStep 返回 enter 为假时本回合记为 blocked，不调用模型。
type Options struct {
	Provider      string
	Model         string
	Prompt        string
	ReplacePrompt bool
	MaxParallel   int
	MaxTokens     int
	PreStep       func(ctx context.Context, claimed []UserMessage) (enter bool, messages []UserMessage, err error)
}

// TurnEnd 是 turn/end 的载荷。Kind 取 completed、blocked、aborted、error、max-tokens。
type TurnEnd struct {
	Kind   string
	Reason string
}

// Agent 是一个会话上的驱动器。Status 在维护任务期间仍是 idle。
type Agent struct {
	id    string
	model Completer
	tools *Registry
	opts  Options

	session *Session
	inbox   Inbox

	mu          sync.Mutex
	phaseKind   int
	turn        int
	step        int
	lastTurn    int
	wakeReq     bool
	phaseCancel context.CancelCauseFunc
	phaseCtx    context.Context
	done        chan struct{}
	life        context.Context
	lifeCancel  context.CancelCauseFunc
	mint        int
	headerLog   bool
	toolSig     string
}

// New 创建空闲代理。id 是会话标识。model 为 nil 时第一步会以错误结束回合。tools 为 nil 时当作空注册表。
func New(id string, model Completer, tools *Registry, opts Options) *Agent {
	if tools == nil {
		tools = NewRegistry()
	}
	if opts.MaxParallel < 1 {
		opts.MaxParallel = DefaultMaxParallel
	}
	life, cancel := context.WithCancelCause(context.Background())
	return &Agent{
		id: id, model: model, tools: tools, opts: opts,
		session: NewSession(), life: life, lifeCancel: cancel,
	}
}

// ID 返回会话标识。
func (a *Agent) ID() string { return a.id }

// Session 返回这份代理的日志。
func (a *Agent) Session() *Session { return a.session }

// Status 返回 idle 或 running。维护任务对外仍是 idle。
func (a *Agent) Status() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.phaseKind == phaseRunning {
		return "running"
	}
	return "idle"
}

// Followup 把 message 放进下一回合并唤醒。message.ID 为空时由代理编号。
func (a *Agent) Followup(message UserMessage) {
	a.send(message, NextTurn, true)
}

// Steer 把 message 放进下一步并唤醒。空闲时这会开一个回合。
func (a *Agent) Steer(message UserMessage) {
	if message.Source == "" {
		message.Source = "steer"
	}
	a.send(message, NextStep, true)
}

// Inject 把 message 放进下一步，不唤醒。空闲代理一直等到 Followup 或 Steer。
func (a *Agent) Inject(message UserMessage) {
	if message.Source == "" {
		message.Source = "inject"
	}
	a.send(message, NextStep, false)
}

// Cancel 中止当前活动。keepInbox 为真时保留收件箱。没有活动时不给以后的工作武装取消。
func (a *Agent) Cancel(cause CancelCause, keepInbox bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !keepInbox {
		a.inbox.Clear()
		if a.phaseKind != phaseIdle {
			a.wakeReq = false
		}
	}
	if cause.Kind == "disposed" && a.lifeCancel != nil {
		a.lifeCancel(cause)
	}
	if a.phaseCancel != nil {
		a.phaseCancel(cause)
	}
}

// WhenIdle 等到驱动器和维护任务都停。新的唤醒会继续等。
func (a *Agent) WhenIdle() {
	for {
		a.mu.Lock()
		if a.phaseKind == phaseIdle {
			a.mu.Unlock()
			return
		}
		ch := a.done
		a.mu.Unlock()
		if ch != nil {
			<-ch
		}
	}
}

// RunMaintenance 在真正空闲时跑 fn。已有驱动器时返回错误。fn 的 ctx 会在 Cancel 时取消。
// 对外 Status 保持 idle。fn 返回后，若取消原因不是 disposed 且有挂起的唤醒，再开驱动器。
func (a *Agent) RunMaintenance(fn func(ctx context.Context) error) error {
	a.mu.Lock()
	if a.phaseKind != phaseIdle {
		a.mu.Unlock()
		return fmt.Errorf("agent %s already has active work", a.id)
	}
	ctx, cancel := context.WithCancelCause(a.life)
	a.phaseKind = phaseMaintenance
	a.phaseCancel = cancel
	a.phaseCtx = ctx
	a.wakeReq = false
	a.done = make(chan struct{})
	done := a.done
	a.mu.Unlock()
	err := fn(ctx)
	a.mu.Lock()
	wake := a.wakeReq && a.inbox.HasPending()
	var cause CancelCause
	if c, ok := context.Cause(ctx).(CancelCause); ok {
		cause = c
	}
	a.phaseKind = phaseIdle
	a.phaseCancel = nil
	a.phaseCtx = nil
	if wake && cause.Kind != "disposed" {
		a.wakeLocked()
	}
	close(done)
	a.mu.Unlock()
	return err
}

// RunJob 在 budget 内等待 fn。budget 小于等于 0 时用 DefaultForeground。
// 预算内结束则把输出交给调用方。否则转入后台，完成后 Steer 一条来源为 job 的消息。
func (a *Agent) RunJob(id string, budget time.Duration, fn JobFunc) JobView {
	return RunForeground(a.life, id, budget, fn, func(output string, err error) {
		text := output
		if err != nil && text == "" {
			text = err.Error()
		}
		a.Steer(UserMessage{Text: "job " + id + " finished\n" + text, Source: "job"})
	})
}

// send 把 message 放进 target。wakeup 为真时唤醒。ID 为空时由代理编号。
// 正在取消的回合上，唤醒的消息改放到下一回合。
func (a *Agent) send(message UserMessage, target InboxTarget, wakeup bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if message.ID == "" {
		a.mint++
		message.ID = fmt.Sprintf("%s-m-%d", a.id, a.mint)
	}
	afterAbort := wakeup && a.phaseKind != phaseIdle && a.phaseCtx != nil && a.phaseCtx.Err() != nil
	resolved := target
	if afterAbort {
		resolved = NextTurn
	}
	if err := a.inbox.Append(resolved, message); err != nil {
		return
	}
	a.session.Append(EventInbox, a.turn, a.step, message)
	if wakeup {
		a.wakeLockedAfter(afterAbort)
	}
}

// wakeLockedAfter 在空闲时启动驱动器。afterAbort 为真且当前活动已取消时，记下结束后还要再醒。调用时要已持有锁。
func (a *Agent) wakeLockedAfter(afterAbort bool) {
	if a.phaseKind != phaseIdle {
		var cause CancelCause
		if a.phaseCtx != nil {
			if c, ok := context.Cause(a.phaseCtx).(CancelCause); ok {
				cause = c
			}
		}
		aborted := a.phaseCtx != nil && a.phaseCtx.Err() != nil
		if cause.Kind != "disposed" && aborted && (a.phaseKind == phaseMaintenance || afterAbort) {
			a.wakeReq = true
		}
		return
	}
	a.wakeLocked()
}

// wakeLocked 从空闲状态启动驱动器。调用时要已持有锁。
func (a *Agent) wakeLocked() {
	ctx, cancel := context.WithCancelCause(a.life)
	a.phaseKind = phaseRunning
	a.phaseCancel = cancel
	a.phaseCtx = ctx
	a.turn = a.lastTurn
	a.step = 0
	a.wakeReq = false
	a.done = make(chan struct{})
	done := a.done
	go func() {
		a.drive(ctx)
		a.mu.Lock()
		last := a.turn
		wake := a.wakeReq && a.inbox.HasPending()
		a.phaseKind = phaseIdle
		a.lastTurn = last
		a.phaseCancel = nil
		a.phaseCtx = nil
		a.step = 0
		if wake {
			var cause CancelCause
			if c, ok := context.Cause(ctx).(CancelCause); ok {
				cause = c
			}
			if cause.Kind != "disposed" {
				a.wakeLocked()
			}
		}
		close(done)
		a.mu.Unlock()
	}()
}

// drive 在 ctx 上连续跑回合，直到这一轮不再继续或出错。
func (a *Agent) drive(ctx context.Context) {
	for {
		cont, err := a.runTurn(ctx)
		if err != nil || !cont {
			return
		}
	}
}

// runTurn 在 ctx 上跑一个回合。cont 为真表示收件箱里还有该进下一回合的消息。
func (a *Agent) runTurn(ctx context.Context) (cont bool, err error) {
	a.mu.Lock()
	turn := a.turn + 1
	a.turn = turn
	a.mu.Unlock()
	a.session.Append(EventTurnStart, turn, 0, nil)
	ended := ""
	defer func() {
		if ended == "" {
			if ctx.Err() != nil {
				ended = "aborted"
			} else if err != nil {
				ended = "error"
			} else {
				ended = "completed"
			}
		}
		reason := ""
		if ctx.Err() != nil {
			if c, ok := context.Cause(ctx).(CancelCause); ok {
				reason = c.Kind
			}
		}
		a.session.Append(EventTurnEnd, turn, 0, TurnEnd{Kind: ended, Reason: reason})
	}()

	target := NextTurn
	turnEnds := ""
	opened := 0
	for {
		if ctx.Err() != nil {
			ended = "aborted"
			return false, ctx.Err()
		}
		a.mu.Lock()
		step := a.step + 1
		claimed := a.inbox.Claim(target, turn)
		a.mu.Unlock()
		if a.opts.PreStep != nil {
			enter, msgs, preErr := a.opts.PreStep(ctx, claimed)
			if preErr != nil {
				ended = "error"
				return false, preErr
			}
			if !enter {
				ended = "blocked"
				return false, nil
			}
			if msgs != nil {
				claimed = msgs
			}
		}
		if turnEnds != "" && len(claimed) == 0 {
			break
		}
		if opened == 0 && len(claimed) == 0 {
			ended = "completed"
			return false, nil
		}
		end, stepErr := a.runStep(ctx, turn, step, claimed)
		opened++
		if stepErr != nil {
			a.recover(turn, step)
			if ctx.Err() != nil {
				ended = "aborted"
				return false, stepErr
			}
			ended = "error"
			return false, stepErr
		}
		if turnEnds != "max-tokens" {
			turnEnds = end
		}
		a.mu.Lock()
		pendingStep := len(a.inbox.nextStep) > 0
		a.mu.Unlock()
		if turnEnds != "" && !pendingStep {
			break
		}
		target = NextStep
	}
	if turnEnds != "" {
		ended = turnEnds
	}
	a.mu.Lock()
	pending := a.inbox.HasPending()
	a.mu.Unlock()
	if !pending {
		return false, nil
	}
	return true, nil
}

// runStep 跑 turn 的第 step 步。claimed 是这一步取走的用户消息。ctx 取消时停。
// 返回的 end 为空表示工具还没结束回合，同回合还要再叫一次模型。
func (a *Agent) runStep(ctx context.Context, turn, step int, claimed []UserMessage) (end string, err error) {
	a.mu.Lock()
	a.step = step
	a.mu.Unlock()
	a.session.Append(EventStepStart, turn, step, nil)
	defer a.session.Append(EventStepEnd, turn, step, nil)

	startsSeries := !a.headerLog
	nodes := a.session.SystemTexts()
	for _, c := range ProjectSystem(nodes, a.opts.Prompt, !a.opts.ReplacePrompt, startsSeries) {
		a.session.Append(EventSystem, turn, step, c)
	}
	for _, m := range claimed {
		a.session.Append(EventUser, turn, step, m)
	}
	schemas := a.tools.Schemas()
	sig := schemaSig(schemas)
	if !a.headerLog || sig != a.toolSig {
		a.session.Append(EventRequestHeader, turn, step, sig)
		a.headerLog = true
		a.toolSig = sig
	}
	if a.model == nil {
		return "", fmt.Errorf("agent %s has no model", a.id)
	}
	req := Request{
		Provider: a.opts.Provider, Model: a.opts.Model,
		Messages: a.session.Derive(), Tools: schemas, MaxTokens: a.opts.MaxTokens,
	}
	out, callErr := a.model.Complete(ctx, req)
	if callErr != nil {
		if ctx.Err() != nil && (out.Text != "" || len(out.ToolCalls) > 0) {
			out.Interrupted = true
			a.session.Append(EventAssistant, turn, step, out)
		} else if ctx.Err() != nil {
			a.session.Append(EventAssistantAttempt, turn, step, nil)
		}
		return "", callErr
	}
	if out.Finish == "" {
		if len(out.ToolCalls) > 0 {
			out.Finish = "tool-calls"
		} else {
			out.Finish = "stop"
		}
	}
	a.session.Append(EventAssistant, turn, step, out)
	if out.Finish == "max-tokens" {
		return "max-tokens", nil
	}
	if len(out.ToolCalls) == 0 {
		return "completed", nil
	}
	concluded, extra, batchErr := ExecuteBatch(ctx, a.session, turn, step, out.ToolCalls, a.tools, a.opts.MaxParallel)
	a.mu.Lock()
	for _, m := range extra {
		if m.Source == "" {
			m.Source = "tool"
		}
		_ = a.inbox.Append(NextStep, m)
	}
	a.mu.Unlock()
	if batchErr != nil {
		return "", batchErr
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if concluded {
		return "completed", nil
	}
	return "", nil
}

// recover 给 turn 的 step 里还没有结果的工具调用补上错误结果。
func (a *Agent) recover(turn, step int) {
	for _, rec := range RecoveryResults(a.session.UnansweredToolCalls(turn, step)) {
		if !rec.Call.Started {
			a.session.Append(EventToolCall, turn, step, ToolCall{ID: rec.Call.ID, Name: rec.Call.Name, Arguments: rec.Call.Arguments})
		}
		a.session.Append(EventToolResult, turn, step, ToolResultEvent{
			ID: rec.Call.ID, Name: rec.Call.Name, Text: rec.Text, IsError: true, Code: rec.Code,
		})
	}
}

// SetPrompt 换成下一步要投影的系统提示。text 原样写入。空串会在下一步把已有系统节点清空。
func (a *Agent) SetPrompt(text string) {
	a.mu.Lock()
	a.opts.Prompt = text
	a.mu.Unlock()
}

// SetRoute 换成模型名和输出上限。model 为空时保留原来的模型。maxTokens 小于等于 0 时保留原来的上限。
func (a *Agent) SetRoute(model string, maxTokens int) {
	a.mu.Lock()
	if model != "" {
		a.opts.Model = model
	}
	if maxTokens > 0 {
		a.opts.MaxTokens = maxTokens
	}
	a.mu.Unlock()
}

// schemaSig 把 tools 的名字、说明和参数模式拼成稳定签名。清单没变时不再写请求头。
func schemaSig(tools []ToolSchema) string {
	var b strings.Builder
	for _, t := range tools {
		b.WriteString(t.Name)
		b.WriteByte(0)
		b.WriteString(t.Description)
		b.WriteByte(0)
		if t.Parameters != nil {
			raw, err := json.Marshal(t.Parameters)
			if err == nil {
				b.Write(raw)
			}
		}
		b.WriteByte(0)
	}
	return b.String()
}
