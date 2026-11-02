// executor.go 用 dsh.Agent 执行派工。它不决定接单。
// 每个工单一个代理。前台命令超过预算后转入后台，完成时 Steer 回同一代理。

package superagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// ErrBusy 表示没有空闲工人槽。它包装 ErrLater，效果队列不计这次失败。
var ErrBusy = fmt.Errorf("no worker slot is free: %w", ErrLater)

// Slots 记录正在启动或正在跑的工单。max 是普通槽位数。by 为 person 时可以多用一个。
type Slots struct {
	mu       sync.Mutex
	max      int
	starting map[string]string
	running  map[string]bool
}

// SetMax 改普通槽位数。n 小于 1 时当作 1。已经占着的槽继续跑完。
func (s *Slots) SetMax(n int) {
	if s == nil {
		return
	}
	if n < 1 {
		n = 1
	}
	s.mu.Lock()
	s.max = n
	s.mu.Unlock()
}

// NewSlots 返回 max 个普通槽。max 小于 1 时当作 1。
func NewSlots(max int) *Slots {
	if max < 1 {
		max = 1
	}
	return &Slots{max: max, starting: map[string]string{}, running: map[string]bool{}}
}

// Reserve 为 id 占一个启动槽。by 取 launch、wake 或 person。person 可以比 max 多用一个。
// id 已经占着时 fresh 为假。没有空位时 err 是 ErrBusy。
func (s *Slots) Reserve(id, by string) (fresh bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running[id] || s.starting[id] != "" {
		return false, nil
	}
	limit := s.max
	if by == "person" {
		limit++
	}
	if s.used() >= limit {
		return false, ErrBusy
	}
	s.starting[id] = by
	return true, nil
}

// Running 把 id 从启动槽改成正在跑。槽位数不变。
func (s *Slots) Running(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.starting, id)
	s.running[id] = true
}

// Free 释放 id 的启动槽和运行槽。
func (s *Slots) Free(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.starting, id)
	delete(s.running, id)
}

// Used 返回当前占用的工单数。同一个工单只算一个。
func (s *Slots) Used() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.used()
}

// used 返回启动中和正在跑的工单数。同一个工单只算一次。调用时要已持有锁。
func (s *Slots) used() int {
	n := len(s.running)
	for id := range s.starting {
		if !s.running[id] {
			n++
		}
	}
	return n
}

// Executor 把工单派工跑成 dsh 回合。Shell 在 dir 里执行命令，dir 为空时由实现自己选目录。
// Root 非空时，每张工单使用 Root/tickets/<id>，两张工单不共用目录。Foreground 小于等于 0 时用 dsh.DefaultForeground。
type Executor struct {
	Engine     *Engine
	Slots      *Slots
	Model      dsh.Completer
	Root       string
	PrepareCmd string
	Shell      func(ctx context.Context, dir, command string) (string, error)
	// RunPrepare 在 root 里执行准备命令。env 只有目录和账号说明，没有钥匙。
	RunPrepare func(ctx context.Context, root, command string, env map[string]string) (string, error)
	// Mount 把工作区工具登记到工人代理上。返回还在跑的后台命令数，以及最多等待多久。
	Mount func(reg *dsh.Registry, agent *dsh.Agent, dir, ticketID string) (func() int, func(time.Duration))
	// Env 提供这张工单的执行变量。值里没有钥匙。
	Env func(dir, ticketID string) map[string]string
	// OnReport 在需要人看的报告写下之后调用。progress 不调用。
	OnReport func(ticketID, state, summary, question string)
	// LogLoad 读出这张工单上次的会话日志。没有时返回 nil。
	LogLoad func(ticketID string) []byte
	// LogSave 记下这张工单的会话日志。重启后续到同一个代理上。
	LogSave    func(ticketID string, body []byte)
	Foreground time.Duration

	mu     sync.Mutex
	agents map[string]*dsh.Agent
	live   map[string]bool
	bgMu   sync.Mutex
	bg     map[string]int
	bgNote map[string]chan struct{}
}

// NewExecutor 返回执行器。slots 为 nil 时用 3 个槽，与原型 maxWorkers 的默认值相同。
func NewExecutor(engine *Engine, slots *Slots, model dsh.Completer) *Executor {
	if slots == nil {
		slots = NewSlots(3)
	}
	return &Executor{
		Engine:     engine,
		Slots:      slots,
		Model:      model,
		Foreground: dsh.DefaultForeground,
		agents:     map[string]*dsh.Agent{},
		live:       map[string]bool{},
	}
}

// Launch 启动 ticketID 上待运行的派工。没有空槽时返回 ErrBusy。派工不是 pending 时释放刚占的槽并返回 not_pending。
func (e *Executor) Launch(ticketID string) error {
	fresh, err := e.Slots.Reserve(ticketID, "launch")
	if err != nil {
		return err
	}
	if !fresh {
		return errCode("already")
	}
	release := true
	defer func() {
		if release {
			e.Slots.Free(ticketID)
		}
	}()
	ticket, ok := e.Engine.Ticket(ticketID)
	if !ok {
		return errCode("unknownTicket")
	}
	if len(ticket.Assignments) == 0 || ticket.Assignments[len(ticket.Assignments)-1].State != "pending" {
		return errCode("not_pending")
	}
	asgID := ticket.Assignments[len(ticket.Assignments)-1].ID
	release = false
	go e.runAssignment(ticketID, asgID, ticket)
	return nil
}

// runAssignment 准备目录，再跑这一轮。准备失败就结束工单，不自动重试。
func (e *Executor) runAssignment(ticketID, asgID string, ticket Ticket) {
	defer e.Slots.Free(ticketID)
	defer func() {
		e.mu.Lock()
		delete(e.live, ticketID)
		e.mu.Unlock()
	}()
	dir, err := e.ensureWorkDir(ticketID)
	if err != nil {
		_ = e.Engine.FailStart(ticketID, err.Error())
		return
	}
	if err := e.Engine.AssignmentStarted(ticketID, asgID); err != nil {
		return
	}
	reg := dsh.NewRegistry()
	agent := dsh.New("ticket-"+ticketID, e.Model, reg, dsh.Options{
		Prompt: workerPrompt(e, dir, ticketID),
		Model:  "yoyo-dsh",
	})
	reg.Add(e.reportTool(ticketID))
	reg.Add(e.replyTool(ticketID))
	reg.Add(e.handbackTool(ticketID))
	reg.Add(e.followTool(ticketID))
	reg.Add(e.unrelatedTool(ticketID))
	var jobs func() int
	var wait func(time.Duration)
	if e.Mount != nil && dir != "" {
		jobs, wait = e.Mount(reg, agent, dir, ticketID)
	} else {
		reg.Add(e.shellTool(agent, ticketID))
	}
	e.Slots.Running(ticketID)
	e.mu.Lock()
	e.agents[ticketID] = agent
	e.live[ticketID] = true
	e.mu.Unlock()
	e.restoreAgent(agent, ticketID)
	e.ackSnapshot(ticketID, ticket)
	agent.Followup(dsh.UserMessage{Text: assignmentText(ticket, dir), Source: "assignment"})
	var deadline time.Time
	tries := 0
	for {
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			agent.Cancel(dsh.CancelCause{Kind: "limit", Reason: "background"}, false)
			_ = e.Engine.Report(ticketID, Report{State: "waiting", Summary: "后台命令超过三十分钟，已停掉"})
			e.storeAgent(agent, ticketID)
			return
		}
		agent.WhenIdle()
		e.storeAgent(agent, ticketID)
		if stop, again := e.afterTurn(agent, ticketID, &tries); stop {
			return
		} else if again {
			continue
		}
		if e.busyLeft(ticketID, jobs) == 0 {
			e.settle(ticketID)
			return
		}
		if deadline.IsZero() {
			deadline = time.Now().Add(30 * time.Minute)
		}
		remain := time.Until(deadline)
		if remain <= 0 {
			agent.Cancel(dsh.CancelCause{Kind: "limit", Reason: "background"}, false)
			_ = e.Engine.Report(ticketID, Report{State: "waiting", Summary: "后台命令超过三十分钟，已停掉"})
			e.storeAgent(agent, ticketID)
			return
		}
		if wait != nil {
			wait(remain)
			continue
		}
		e.waitBackground(ticketID, remain)
	}
}

// busyLeft 是工人后台命令和执行器自己记下的后台命令之和。
func (e *Executor) busyLeft(ticketID string, jobs func() int) int {
	n := e.backgroundLeft(ticketID)
	if jobs != nil {
		n += jobs()
	}
	return n
}

// settle 在一轮做完且没有后台命令时，补一条 waiting。人已经接过的报告保持原样。
func (e *Executor) settle(ticketID string) {
	ticket, ok := e.Engine.Ticket(ticketID)
	if !ok || ticket.Ended != nil || len(ticket.Assignments) == 0 {
		return
	}
	asg := ticket.Assignments[len(ticket.Assignments)-1]
	if len(asg.Reports) > 0 {
		switch asg.Reports[len(asg.Reports)-1].State {
		case "waiting", "ready", "blocked", "failed", "dropped":
			return
		}
	}
	_ = e.Engine.Report(ticketID, Report{State: "waiting", Summary: "这一轮做完了，在等下一条消息"})
}

// beginBackground 记 ticketID 上一个还没结束的后台命令。
func (e *Executor) beginBackground(ticketID string) {
	e.bgMu.Lock()
	defer e.bgMu.Unlock()
	if e.bg == nil {
		e.bg = map[string]int{}
	}
	if e.bgNote == nil {
		e.bgNote = map[string]chan struct{}{}
	}
	if e.bgNote[ticketID] == nil {
		e.bgNote[ticketID] = make(chan struct{}, 1)
	}
	e.bg[ticketID]++
}

// endBackground 结束 ticketID 上一个后台命令，并在归零时叫醒等待槽位的循环。
func (e *Executor) endBackground(ticketID string) {
	e.bgMu.Lock()
	if e.bg[ticketID] > 0 {
		e.bg[ticketID]--
	}
	left := e.bg[ticketID]
	ch := e.bgNote[ticketID]
	e.bgMu.Unlock()
	if left == 0 && ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// backgroundLeft 返回 ticketID 上还没结束的后台命令数。
func (e *Executor) backgroundLeft(ticketID string) int {
	e.bgMu.Lock()
	defer e.bgMu.Unlock()
	return e.bg[ticketID]
}

// waitBackground 等到 ticketID 的后台命令数发生变化，或 d 用完。没有通道时直接返回。
func (e *Executor) waitBackground(ticketID string, d time.Duration) {
	e.bgMu.Lock()
	ch := e.bgNote[ticketID]
	e.bgMu.Unlock()
	if ch == nil || d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ch:
	case <-timer.C:
	}
}

// ackSnapshot 把这次派工已经写进提示的输入标成已投递。之后新到的才再送一次。
func (e *Executor) ackSnapshot(ticketID string, ticket Ticket) {
	if e == nil || e.Engine == nil {
		return
	}
	var ids []string
	for _, in := range ticket.Inputs {
		if in.ID != "" {
			ids = append(ids, in.ID)
		}
	}
	e.Engine.AckInputs(ticketID, ids)
}

// afterTurn 看上一回合。暂时失败时再开一次并返回 again。其他失败记下 failed 并返回 stop。
func (e *Executor) afterTurn(agent *dsh.Agent, ticketID string, tries *int) (stop, again bool) {
	kind, reason := lastTurnEnd(agent)
	if kind != "error" {
		return false, false
	}
	if tries != nil && *tries < 3 && passingError(reason) {
		*tries++
		agent.Followup(dsh.UserMessage{
			Text:   "上一回合因为暂时的原因中断了：" + clip(reason, 200) + "。从停下的地方继续，不要重复已经做成的外部动作。",
			Source: "retry",
		})
		return false, true
	}
	if e.Engine != nil {
		summary := strings.TrimSpace(reason)
		if summary == "" {
			summary = "这一回合失败了"
		}
		_ = e.Engine.Report(ticketID, Report{State: "failed", Summary: clip(summary, 200)})
	}
	return true, false
}

// HasAgent 在 ticketID 的代理已经创建时为真。
func (e *Executor) HasAgent(ticketID string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.agents[ticketID] != nil
}

// DeliverPending 把还没投递的输入送进 ticketID 的代理。代理还没创建时返回 ErrLater。
func (e *Executor) DeliverPending(ticketID string) error {
	if e == nil || e.Engine == nil {
		return nil
	}
	texts, ids := e.Engine.PendingInputs(ticketID)
	if len(texts) == 0 {
		return nil
	}
	if !e.HasAgent(ticketID) {
		return ErrLater
	}
	for _, text := range texts {
		e.Deliver(ticketID, text)
	}
	e.Engine.AckInputs(ticketID, ids)
	return nil
}

// Deliver 把 text 交给 ticketID 的代理，作为人的下一步消息。代理还在跑时直接送入。
// 这一轮已经结束时，用人的额外槽再开一轮。代理还没启动时什么都不做。
func (e *Executor) Deliver(ticketID, text string) {
	e.mu.Lock()
	agent := e.agents[ticketID]
	running := e.live[ticketID]
	e.mu.Unlock()
	if agent == nil || text == "" {
		return
	}
	if running {
		agent.Steer(dsh.UserMessage{Text: text, Source: "human"})
		return
	}
	go e.wake(ticketID, agent, text)
}

// wake 用人的额外槽把 text 送进已经停过一轮的代理。槽满时这次不启动。
func (e *Executor) wake(ticketID string, agent *dsh.Agent, text string) {
	fresh, err := e.Slots.Reserve(ticketID, "person")
	if err != nil || !fresh {
		if err == nil && agent != nil {
			agent.Steer(dsh.UserMessage{Text: text, Source: "human"})
		}
		return
	}
	e.mu.Lock()
	e.live[ticketID] = true
	e.mu.Unlock()
	e.Slots.Running(ticketID)
	defer e.Slots.Free(ticketID)
	defer func() {
		e.mu.Lock()
		delete(e.live, ticketID)
		e.mu.Unlock()
	}()
	agent.Steer(dsh.UserMessage{Text: text, Source: "human"})
	agent.WhenIdle()
	e.settle(ticketID)
}

// reportTool 返回 ticketID 使用的 orchestrator_report。progress 不结束回合，其余成功的报告会结束回合。
func (e *Executor) reportTool(ticketID string) dsh.Tool {
	return dsh.Tool{
		Name:        "orchestrator_report",
		Description: "向工单报告状态。state 取 progress、waiting、ready、blocked、failed 或 dropped。summary 是给人看的一句。question 是要人回答的问题。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				State    string `json:"state"`
				Summary  string `json:"summary"`
				Question string `json:"question"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			err := e.Engine.Report(ticketID, Report{State: body.State, Summary: body.Summary, Question: body.Question})
			if err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			if e.OnReport != nil && (body.Question != "" || body.State == "ready" || body.State == "blocked" || body.State == "failed") {
				e.OnReport(ticketID, body.State, body.Summary, body.Question)
			}
			return dsh.Result{Text: "recorded " + body.State, Concludes: body.State != "progress"}, nil
		},
	}
}

// shellTool 返回 agent 上的 shell。ticketID 用来在后台命令结束前占住工人槽。命令在前台预算内结束就把输出交回，否则转入后台。
func (e *Executor) shellTool(agent *dsh.Agent, ticketID string) dsh.Tool {
	return dsh.Tool{
		Name:        "shell",
		Description: "运行一条命令。超过前台预算后转入后台，完成时再投递输出。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(ctx context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			budget := e.Foreground
			dir, dirErr := e.workDir(ticketID)
			if dirErr != nil {
				return dsh.Result{Text: dirErr.Error(), IsError: true}, nil
			}
			view := dsh.RunForeground(ctx, call.ID, budget, func(ctx context.Context) (string, error) {
				if e.Shell == nil {
					return "", errCode("no_shell")
				}
				return e.Shell(ctx, dir, body.Command)
			}, func(output string, err error) {
				text := output
				if err != nil && text == "" {
					text = err.Error()
				}
				if agent != nil {
					agent.Steer(dsh.UserMessage{Text: "job " + call.ID + " finished\n" + text, Source: "job"})
				}
				e.endBackground(ticketID)
			})
			if view.Err != nil && !view.Background {
				return dsh.Result{Text: view.Err.Error(), IsError: true}, nil
			}
			if view.Background {
				e.beginBackground(ticketID)
				return dsh.Result{Text: "background job " + view.ID}, nil
			}
			return dsh.Result{Text: view.Output}, nil
		},
	}
}

// ticketDir 返回 ticketID 自己的目录，并创建它。Root 为空时返回空串，调用方沿用原来的目录。
// 编号里带路径分隔或 .. 时拒绝，避免写出执行根之外。
func (e *Executor) ticketDir(ticketID string) (string, error) {
	if e == nil || strings.TrimSpace(e.Root) == "" {
		return "", nil
	}
	id := strings.TrimSpace(ticketID)
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return "", errCode("bad_ticket_dir")
	}
	dir := filepath.Join(e.Root, "tickets", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// assignmentText 把 ticket 的编号、目录和输入拼成派工正文。dir 为空时不写目录。
func assignmentText(ticket Ticket, dir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "工单 %s\n", ticket.ID)
	if dir != "" {
		fmt.Fprintf(&b, "工作目录 %s\n", dir)
	}
	for _, in := range ticket.Inputs {
		if in.Text == "" {
			continue
		}
		b.WriteString(in.Text)
		b.WriteByte('\n')
	}
	b.WriteString("完成后调用 orchestrator_report。")
	return b.String()
}
