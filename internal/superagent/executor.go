// executor.go 用 Worker 执行派工：yoyo 自带的 dsh 代理，或经 Opener 打开的外部 agent。它不决定接单。
// 每个工单一个工人。前台命令超过预算后转入后台，完成时 Steer 回同一代理。

package superagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
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
	LogSave func(ticketID string, body []byte)
	// Profile 返回操作者已经留下的偏好和说过的话。空串表示没有。它进工人的系统提示，只读。
	Profile    func() string
	Foreground time.Duration
	// Opener 打开外部 agent 的工人。为 nil 时所有工单都走 yoyo 自带的 dsh 代理。
	Opener Opener
	// AgentLimit 返回 agent 同时最多跑几张工单。为 nil 或返回小于 1 时只受 Slots 限制。
	AgentLimit func(agent string) int
	// AgentName 返回 agent 给人看的名字。为 nil 或返回空串时用编号。
	AgentName func(agent string) string
	// Roots 返回这张工单除工作目录外可读写的目录，例如任务说明里点名的目录。外部工人的说明和目录限制都用它。
	Roots func(ticketID string) []string

	mu sync.Mutex
	// agents 是 yoyo 自带工人的代理，workers 是外部工人。同一张工单只在其中一张表里。
	agents  map[string]*dsh.Agent
	workers map[string]Worker
	// workerOf 记外部工人属于哪个 agent。
	workerOf map[string]string
	// agentRun 是每个 agent 正在跑的工单数，liveAgent 记每张正在跑的工单算在谁名下。
	agentRun  map[string]int
	liveAgent map[string]string
	live      map[string]bool
	held      map[string]bool
	// prefix 是最后一次交接之前的日志，只有 owner 那个会话写盘时拼在前面。新工人读不到它，任务页仍然完整。
	prefix map[string]logPrefix
	bgMu   sync.Mutex
	bg     map[string]int
	bgNote map[string]chan struct{}
}

// logPrefix 是一张工单交接之前的日志，以及接手之后在写日志的那个会话。
type logPrefix struct {
	owner  *dsh.Session
	events []dsh.Event
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

// workerLocked 返回 ticketID 的工人。外部工人优先；只有 dsh 代理时包一层。没有时为 nil。调用时要已持有 mu。
func (e *Executor) workerLocked(ticketID string) Worker {
	if w := e.workers[ticketID]; w != nil {
		return w
	}
	if agent := e.agents[ticketID]; agent != nil {
		return &dshWorker{agent: agent}
	}
	return nil
}

// agentOfLocked 返回 ticketID 当前工人所属的 agent。dsh 代理记成 yoyo。调用时要已持有 mu。
func (e *Executor) agentOfLocked(ticketID string) string {
	if e.workers[ticketID] != nil {
		return agentKey(e.workerOf[ticketID])
	}
	return YoyoAgent
}

// swapWorkerLocked 把 ticketID 的工人换成 w，agent 是它所属的 agent。返回被换下的外部工人，由调用方在锁外关掉。调用时要已持有 mu。
func (e *Executor) swapWorkerLocked(ticketID, agent string, w Worker) Worker {
	if e.agents == nil {
		e.agents = map[string]*dsh.Agent{}
	}
	if e.workers == nil {
		e.workers = map[string]Worker{}
	}
	if e.workerOf == nil {
		e.workerOf = map[string]string{}
	}
	old := e.workers[ticketID]
	if d, ok := w.(*dshWorker); ok {
		e.agents[ticketID] = d.agent
		delete(e.workers, ticketID)
		delete(e.workerOf, ticketID)
	} else {
		delete(e.agents, ticketID)
		e.workers[ticketID] = w
		e.workerOf[ticketID] = agentKey(agent)
	}
	if old == w {
		return nil
	}
	return old
}

// holdAgent 把 ticketID 记到 agent 名下，算它正在跑的一张。limit 大于 0 且已满时不记，返回假。已经记过时返回真。
func (e *Executor) holdAgent(ticketID, agent string, limit int) bool {
	agent = agentKey(agent)
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.liveAgent[ticketID]; ok {
		return true
	}
	if limit > 0 && e.agentRun[agent] >= limit {
		return false
	}
	if e.agentRun == nil {
		e.agentRun = map[string]int{}
	}
	if e.liveAgent == nil {
		e.liveAgent = map[string]string{}
	}
	e.agentRun[agent]++
	e.liveAgent[ticketID] = agent
	return true
}

// dropAgent 去掉 ticketID 在 agent 名下的那一张。没记过时什么都不做。
func (e *Executor) dropAgent(ticketID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	agent, ok := e.liveAgent[ticketID]
	if !ok {
		return
	}
	delete(e.liveAgent, ticketID)
	if e.agentRun[agent] > 1 {
		e.agentRun[agent]--
	} else {
		delete(e.agentRun, agent)
	}
}

// AgentRunning 返回每个 agent 正在跑的工单数。yoyo 自带工人记在 yoyo 名下。
func (e *Executor) AgentRunning() map[string]int {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make(map[string]int, len(e.agentRun))
	for agent, n := range e.agentRun {
		out[agent] = n
	}
	return out
}

// limitFor 返回 agent 的并行上限。没接 AgentLimit 时为 0，表示不另外限制。
func (e *Executor) limitFor(agent string) int {
	if e == nil || e.AgentLimit == nil {
		return 0
	}
	return e.AgentLimit(agentKey(agent))
}

// closeIfEnded 在 ticketID 已经结束且这一轮不在跑时关掉它的外部工人，并从表里拿掉。dsh 代理留在表里，行为不变。
func (e *Executor) closeIfEnded(ticketID string) {
	if e == nil || e.Engine == nil {
		return
	}
	if ticket, ok := e.Engine.Ticket(ticketID); ok && ticket.Ended == nil {
		return
	}
	e.mu.Lock()
	w := e.workers[ticketID]
	if w == nil || e.live[ticketID] {
		e.mu.Unlock()
		return
	}
	delete(e.workers, ticketID)
	delete(e.workerOf, ticketID)
	e.mu.Unlock()
	_ = w.Close()
}

// Sweep 关掉所有已经结束、又不在跑的工单的外部工人。两秒轮次调用它，兜住人在空闲时结束工单的情况。
func (e *Executor) Sweep() {
	if e == nil {
		return
	}
	e.mu.Lock()
	ids := make([]string, 0, len(e.workers))
	for id := range e.workers {
		ids = append(ids, id)
	}
	e.mu.Unlock()
	for _, id := range ids {
		e.closeIfEnded(id)
	}
}

// CloseAll 关掉所有外部工人。应用退出时调用；dsh 代理不受影响。
func (e *Executor) CloseAll() {
	if e == nil {
		return
	}
	e.mu.Lock()
	all := make([]Worker, 0, len(e.workers))
	for id, w := range e.workers {
		all = append(all, w)
		delete(e.workers, id)
		delete(e.workerOf, id)
	}
	e.mu.Unlock()
	for _, w := range all {
		_ = w.Close()
	}
}

// Interrupt 打断 ticketID 正在跑的这一轮。工单和目录都留着，人还可以继续说。
func (e *Executor) Interrupt(ticketID string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	w := e.workerLocked(ticketID)
	e.mu.Unlock()
	if w == nil {
		return
	}
	w.Cancel(dsh.CancelCause{Kind: "steer", Reason: "interrupted"}, true)
}

// pausedSummary 是人让工人先停下之后，这张工单最后的那一句。
const pausedSummary = "人让它先停下，在等下一条消息"

// Pause 停下 ticketID 正在跑的这一轮，工单留着。这一轮不再催工人做下去，记成在等下一条消息。
// 还没送进去的话留在收件箱里，下一轮一起读。没有在跑时返回假。
func (e *Executor) Pause(ticketID string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	w := e.workerLocked(ticketID)
	running := w != nil && e.live[ticketID]
	if running {
		if e.held == nil {
			e.held = map[string]bool{}
		}
		e.held[ticketID] = true
	}
	e.mu.Unlock()
	if !running {
		return false
	}
	w.Cancel(dsh.CancelCause{Kind: "steer", Reason: "paused"}, true)
	return true
}

// takeHeld 在人让 ticketID 先停下时为真，并清掉这个记号。
func (e *Executor) takeHeld(ticketID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	held := e.held[ticketID]
	delete(e.held, ticketID)
	return held
}

// Launch 启动 ticketID 上待运行的派工。没有空槽，或派工的 agent 已经跑满 AgentLimit 时返回 ErrBusy。
// 派工不是 pending 时释放刚占的槽并返回 not_pending。
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
	agent := assignmentAgent(ticket, asgID)
	if !e.holdAgent(ticketID, agent, e.limitFor(agent)) {
		return ErrBusy
	}
	release = false
	go e.runAssignment(ticketID, asgID, ticket)
	return nil
}

// assignmentAgent 返回 asgID 这次派工的 agent。派工没写时用工单的 Agent，都没有时为空，即 yoyo。
func assignmentAgent(ticket Ticket, asgID string) string {
	for i := len(ticket.Assignments) - 1; i >= 0; i-- {
		if ticket.Assignments[i].ID == asgID {
			if agent := strings.TrimSpace(ticket.Assignments[i].Agent); agent != "" {
				return agent
			}
			break
		}
	}
	return strings.TrimSpace(ticket.Agent)
}

// openWorker 为这次派工选工人。agent 是外部 agent 且接了 Opener 时打开外部工人，否则建 yoyo 自带代理。
// 外部工人没有 jobs 和 wait；它的后台命令由 agent 自己管。
func (e *Executor) openWorker(ticketID, asgID, agent string, ticket Ticket, dir string) (Worker, func() int, func(time.Duration), error) {
	if externalAgent(agent) && e.Opener != nil {
		if cur, ok := e.Engine.Ticket(ticketID); ok {
			ticket = cur
		}
		spec := WorkerSpec{
			TicketID:     ticketID,
			AssignmentID: asgID,
			Agent:        agent,
			Pinned:       ticket.AgentPinned,
			Dir:          dir,
			Tools:        e.assignmentTools(ticketID, asgID, false),
			OnPointer:    func(pointer string) { e.Engine.SetPointer(ticketID, agent, pointer) },
			OnUsage:      func(usage Usage) { e.Engine.SetUsage(ticketID, usage) },
			OnAttention:  func(text string) { e.Engine.SetApprovalWait(ticketID, text) },
		}
		if ticket.PointerOf == agentKey(agent) {
			spec.Pointer = ticket.Pointer
		}
		if e.Roots != nil {
			spec.Roots = e.Roots(ticketID)
		}
		_, spec.ReadRoots = e.Engine.MemberDirs(ticketID)
		spec.System = externalWorkerPrompt(e, spec)
		w, err := e.Opener(context.Background(), spec)
		if err != nil {
			return nil, nil, nil, err
		}
		if w == nil {
			return nil, nil, nil, ErrAgentOffline
		}
		return w, nil, nil, nil
	}
	w, jobs, wait := newDshWorker(e, ticketID, asgID, dir)
	return w, jobs, wait, nil
}

// failOpen 在外部工人打不开时写一份报告，并写回主会话。工单留着，人可以重试或换人。
// agent 要先登录时报 blocked，等人去 Buddy 登录；离线时报 blocked，问人改用 yoyo 还是等它上线；其他原因记成 failed。
func (e *Executor) failOpen(ticketID, agent string, err error) {
	name := e.agentName(agent)
	switch {
	case needsLogin(err):
		e.stopWith(ticketID, Report{State: "blocked", Summary: name + " 需要登录", Question: name + " 需要登录。去 Buddy 登录后回复继续。", Need: "login"})
	case errors.Is(err, ErrAgentOffline):
		e.stopWith(ticketID, Report{State: "blocked", Summary: name + " 离线", Question: name + " 离线。改用 yoyo，还是等它上线？"})
	default:
		summary := clip(strings.TrimSpace(err.Error()), 200)
		if summary == "" {
			summary = ErrAgentOffline.Error()
		}
		e.stopWith(ticketID, Report{State: "failed", Summary: summary})
	}
}

// needsLogin 在 err 说明 agent 要先登录时为真。Opener 和单轮结束都用 ReasonAuthRequired 标明。
func needsLogin(err error) bool {
	return err != nil && strings.Contains(err.Error(), ReasonAuthRequired)
}

// handoverPrefix 加上派工编号，是交接消息的编号。
const handoverPrefix = "handover-"

// handoverID 是 asgID 这次交接写进日志的那条消息的编号。重启后看到它就不再写第二条。
func handoverID(asgID string) string {
	return handoverPrefix + asgID
}

// handoverMarker 是 asgID 这次交接写进日志的那条消息。asgID 不是交接派工时为 nil。
func (e *Executor) handoverMarker(ticket Ticket, asgID string) *dsh.UserMessage {
	for _, asg := range ticket.Assignments {
		if asg.ID != asgID {
			continue
		}
		if strings.TrimSpace(asg.From) == "" {
			return nil
		}
		return &dsh.UserMessage{ID: handoverID(asgID), Source: "handover", Text: "改由 " + e.agentName(assignmentAgent(ticket, asgID)) + " 接手"}
	}
	return nil
}

// retire 在 ticketID 已经改派之后关掉旧工人 w，并把它从表里拿掉。它的日志已经写过。
func (e *Executor) retire(ticketID string, w Worker) {
	e.mu.Lock()
	if cur, ok := e.workers[ticketID]; ok && cur == w {
		delete(e.workers, ticketID)
		delete(e.workerOf, ticketID)
	}
	if d, ok := w.(*dshWorker); ok && e.agents[ticketID] == d.agent {
		delete(e.agents, ticketID)
	}
	e.mu.Unlock()
	_ = w.Close()
}

// runAssignment 准备目录、选工人，再跑这一轮。准备失败就结束工单，不自动重试。
func (e *Executor) runAssignment(ticketID, asgID string, ticket Ticket) {
	defer e.Slots.Free(ticketID)
	defer e.dropAgent(ticketID)
	defer func() {
		e.mu.Lock()
		delete(e.live, ticketID)
		e.mu.Unlock()
		e.closeIfEnded(ticketID)
	}()
	dir, err := e.ensureWorkDir(ticketID)
	if err != nil {
		_ = e.Engine.FailStart(ticketID, err.Error())
		if e.OnReport != nil {
			e.OnReport(ticketID, "failed", err.Error(), "")
		}
		return
	}
	if err := e.Engine.AssignmentStarted(ticketID, asgID); err != nil {
		return
	}
	agent := assignmentAgent(ticket, asgID)
	marker := e.handoverMarker(ticket, asgID)
	w, jobs, wait, err := e.openWorker(ticketID, asgID, agent, ticket, dir)
	if errors.Is(err, ErrAgentOffline) && !needsLogin(err) {
		if from := e.Engine.FallBack(ticketID, asgID); from != "" {
			marker = &dsh.UserMessage{ID: handoverID(asgID), Source: "handover", Text: e.agentName(from) + " 离线，改由 yoyo 做"}
			agent = YoyoAgent
			e.dropAgent(ticketID)
			e.holdAgent(ticketID, agent, 0)
			w, jobs, wait, err = e.openWorker(ticketID, asgID, agent, ticket, dir)
		}
	}
	if err != nil {
		e.failOpen(ticketID, agent, err)
		return
	}
	e.Slots.Running(ticketID)
	e.mu.Lock()
	old := e.swapWorkerLocked(ticketID, agent, w)
	e.live[ticketID] = true
	delete(e.held, ticketID)
	e.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	restored := e.restoreAgent(w, ticketID, marker)
	resumed := e.Engine.TakeResume(ticketID) && restored
	if resumed {
		// 日志里已经有的输入不再重发。重启前还没送到的话跟在「继续」后面，送到了才标成已投递。
		texts, ids := e.Engine.PendingInputs(ticketID)
		w.Followup(dsh.UserMessage{Text: "继续。", Source: "retry"})
		for _, text := range texts {
			w.Followup(dsh.UserMessage{Text: text, Source: "human"})
		}
		e.Engine.AckInputs(ticketID, ids)
	} else {
		e.ackSnapshot(ticketID, ticket)
		w.Followup(dsh.UserMessage{Text: assignmentText(ticket, dir, e.mandateLine(ticket), e.groupLine(ticket)), Source: "assignment"})
	}
	stopSave := make(chan struct{})
	defer close(stopSave)
	go e.watchLog(w, ticketID, stopSave)
	caps := w.Caps()
	var deadline time.Time
	tries := 0
	nudges := 0
	for {
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			w.Cancel(dsh.CancelCause{Kind: "limit", Reason: "background"}, false)
			_ = e.Engine.Report(ticketID, Report{State: "waiting", Summary: "后台命令超过三十分钟，已停掉"})
			e.storeAgent(w, ticketID)
			return
		}
		w.WhenIdle()
		e.storeAgent(w, ticketID)
		if e.Engine.LatestAssignment(ticketID) != asgID {
			e.retire(ticketID, w)
			return
		}
		if stop, again := e.afterTurn(w, ticketID, &tries); stop {
			return
		} else if again {
			continue
		}
		if e.busyLeft(ticketID, jobs) == 0 {
			if e.takeHeld(ticketID) {
				e.park(ticketID, pausedSummary)
				return
			}
			if nudges < caps.Nudges && e.openWork(ticketID) {
				nudges++
				w.Followup(dsh.UserMessage{Text: workerContinueNudge, Source: "retry"})
				continue
			}
			e.settle(ticketID, w)
			return
		}
		if deadline.IsZero() {
			deadline = time.Now().Add(30 * time.Minute)
		}
		remain := time.Until(deadline)
		if remain <= 0 {
			w.Cancel(dsh.CancelCause{Kind: "limit", Reason: "background"}, false)
			_ = e.Engine.Report(ticketID, Report{State: "waiting", Summary: "后台命令超过三十分钟，已停掉"})
			e.storeAgent(w, ticketID)
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

// maxWorkerNudges 是工人没写完成报告时，再催它做下去的次数。
const maxWorkerNudges = 4

const workerContinueNudge = "任务说明里的完成标准还没达到。从停下的地方继续做完，不要停下来让人选择下一步。只有说明没授权的破坏、必须改说明和工单目录以外的文件、或缺少密钥时才提问。"

// openWork 在这一轮还没有完成、受阻或等待报告时为真。上一份报告之后又来了新输入，也算还没做完。
func (e *Executor) openWork(ticketID string) bool {
	if e == nil || e.Engine == nil {
		return false
	}
	ticket, ok := e.Engine.Ticket(ticketID)
	if !ok || ticket.Ended != nil || len(ticket.Assignments) == 0 {
		return false
	}
	asg := ticket.Assignments[len(ticket.Assignments)-1]
	if len(asg.Reports) == 0 {
		return true
	}
	last := asg.Reports[len(asg.Reports)-1]
	switch last.State {
	case "waiting", "ready", "blocked", "failed", "dropped":
		return inputAfter(&ticket, last.Seq)
	default:
		return true
	}
}

// settle 在一轮做完且没有后台命令时，补一条 waiting。上一份完成报告之后又有新输入时，也要记下这一轮已经停下。
// 外部工人常常不写报告，这时摘要用它最后一段正文。
func (e *Executor) settle(ticketID string, w Worker) {
	summary := "这一轮做完了，在等下一条消息"
	if w != nil && w.Caps().External {
		if text := lastAssistantText(w); text != "" {
			summary = clipRunes(text, 200)
		}
	}
	e.park(ticketID, summary)
}

// park 用 summary 补一条 waiting，把这一轮记成停下。工单已经结束，或上一份停下的报告之后没有新输入时不补。
func (e *Executor) park(ticketID, summary string) {
	ticket, ok := e.Engine.Ticket(ticketID)
	if !ok || ticket.Ended != nil || len(ticket.Assignments) == 0 {
		return
	}
	asg := ticket.Assignments[len(ticket.Assignments)-1]
	if len(asg.Reports) > 0 {
		last := asg.Reports[len(asg.Reports)-1]
		switch last.State {
		case "waiting", "ready", "blocked", "failed", "dropped":
			if !inputAfter(&ticket, last.Seq) {
				return
			}
		}
	}
	_ = e.Engine.Report(ticketID, Report{State: "waiting", Summary: summary})
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

// afterTurn 看 w 的上一回合。暂时失败时再开一次并返回 again。其他失败记下 failed 并返回 stop。
// 外部工人一轮超时记 waiting，要登录记 blocked，都返回 stop。
func (e *Executor) afterTurn(w Worker, ticketID string, tries *int) (stop, again bool) {
	kind, reason := lastTurnEnd(w)
	if kind == "aborted" && reason == ReasonTurnTimeout {
		summary := "单轮超时，已停下"
		if n := w.Caps().TurnMinutes; n > 0 {
			summary = fmt.Sprintf("单轮超过 %d 分钟，已停下", n)
		}
		e.stopWith(ticketID, Report{State: "waiting", Summary: summary})
		return true, false
	}
	if kind != "error" {
		return false, false
	}
	if strings.HasPrefix(reason, ReasonAuthRequired) {
		name := e.agentName(e.agentOf(ticketID))
		e.stopWith(ticketID, Report{State: "blocked", Summary: name + " 需要登录", Question: name + " 需要登录。去 Buddy 登录后回复继续。", Need: "login"})
		return true, false
	}
	if tries != nil && *tries < 3 && passingError(reason) {
		*tries++
		w.Followup(dsh.UserMessage{
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
		summary = clip(summary, 200)
		_ = e.Engine.Report(ticketID, Report{State: "failed", Summary: summary})
		if e.OnReport != nil {
			e.OnReport(ticketID, "failed", summary, "")
		}
	}
	return true, false
}

// stopWith 记下 report。它该回到主会话时再交给 OnReport。
func (e *Executor) stopWith(ticketID string, report Report) {
	if e.Engine == nil || e.Engine.Report(ticketID, report) != nil {
		return
	}
	if e.OnReport != nil && terminalReport(report.State, report.Question) {
		e.OnReport(ticketID, report.State, report.Summary, report.Question)
	}
}

// agentOf 返回 ticketID 当前工人所属的 agent，dsh 代理和还没有工人时为 yoyo。
func (e *Executor) agentOf(ticketID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.agentOfLocked(ticketID)
}

// agentName 返回 agent 给人看的名字。没有 AgentName 或它返回空串时用编号。
func (e *Executor) agentName(agent string) string {
	agent = agentKey(agent)
	if e.AgentName != nil {
		if name := strings.TrimSpace(e.AgentName(agent)); name != "" {
			return name
		}
	}
	return agent
}

// askReport 把「做完了，但还有问题」收成等人。ready 会结束工单，问题就不能写回主会话。
func askReport(state, question string) string {
	if state == "ready" && strings.TrimSpace(question) != "" {
		return "blocked"
	}
	return state
}

// terminalReport 在这份报告该回到主会话时为真。进行中、普通进度和等人的等待不回去。
func terminalReport(state, question string) bool {
	if strings.TrimSpace(question) != "" {
		return true
	}
	switch state {
	case "ready", "blocked", "failed", "dropped", "stopped":
		return true
	default:
		return false
	}
}

// HasAgent 在 ticketID 的工人已经创建、还没关掉时为真。
func (e *Executor) HasAgent(ticketID string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.agents[ticketID] != nil || e.workers[ticketID] != nil
}

// DeliverPending 把还没投递的输入送进 ticketID 的代理。代理还没创建时返回 ErrLater。
// 只把真正送进去的标成已投递。槽满送不进去时返回 ErrBusy，剩下的留给下一次投递。
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
	for i, text := range texts {
		if !e.Deliver(ticketID, text) {
			e.Engine.AckInputs(ticketID, ids[:i])
			return ErrBusy
		}
	}
	e.Engine.AckInputs(ticketID, ids)
	return nil
}

// Deliver 把 text 交给 ticketID 的工人，作为人的下一步消息。工人还在跑时先插话，不支持插话就排在这一轮后面。
// 这一轮已经结束时，先占人的额外槽，再开一轮。工人还没启动、正文为空或槽满时返回假，这句没有送出。
func (e *Executor) Deliver(ticketID, text string) bool {
	e.mu.Lock()
	w := e.workerLocked(ticketID)
	running := e.live[ticketID]
	agent := e.agentOfLocked(ticketID)
	e.mu.Unlock()
	if w == nil || text == "" {
		return false
	}
	msg := dsh.UserMessage{Text: text, Source: "human"}
	if running {
		steerOrFollow(w, msg)
		return true
	}
	fresh, err := e.Slots.Reserve(ticketID, "person")
	if err != nil {
		return false
	}
	if !fresh {
		steerOrFollow(w, msg)
		return true
	}
	e.holdAgent(ticketID, agent, 0)
	go e.wake(ticketID, w, text)
	return true
}

// wake 在已经占好的人的额外槽里，把 text 送进停过一轮的工人 w，跑到这一轮停下。
// 中途改派时关掉 w，不再催它。
func (e *Executor) wake(ticketID string, w Worker, text string) {
	asgID := e.Engine.LatestAssignment(ticketID)
	e.mu.Lock()
	e.live[ticketID] = true
	delete(e.held, ticketID)
	e.mu.Unlock()
	e.Slots.Running(ticketID)
	stopSave := make(chan struct{})
	defer close(stopSave)
	go e.watchLog(w, ticketID, stopSave)
	defer e.Slots.Free(ticketID)
	defer e.dropAgent(ticketID)
	defer func() {
		e.mu.Lock()
		delete(e.live, ticketID)
		e.mu.Unlock()
		e.closeIfEnded(ticketID)
	}()
	steerOrFollow(w, dsh.UserMessage{Text: text, Source: "human"})
	caps := w.Caps()
	var tries, nudges int
	for {
		w.WhenIdle()
		e.storeAgent(w, ticketID)
		if e.Engine.LatestAssignment(ticketID) != asgID {
			e.retire(ticketID, w)
			return
		}
		if stop, again := e.afterTurn(w, ticketID, &tries); stop {
			return
		} else if again {
			continue
		}
		if e.takeHeld(ticketID) {
			e.park(ticketID, pausedSummary)
			return
		}
		if nudges < caps.Nudges && e.openWork(ticketID) {
			nudges++
			w.Followup(dsh.UserMessage{Text: workerContinueNudge, Source: "retry"})
			continue
		}
		break
	}
	e.settle(ticketID, w)
}

// planStillOpen 在这张工单的计划文件里还有未完成步骤时为真。没有计划文件时为假。
func (e *Executor) planStillOpen(ticketID string) bool {
	if e == nil {
		return false
	}
	dir, err := e.workDir(ticketID)
	if err != nil || dir == "" {
		return false
	}
	text := runtime.LoadPlanFile(dir, "ticket-"+ticketID)
	if text == "" {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "[pending]") || strings.Contains(line, "[in_progress]") {
			return true
		}
	}
	return false
}

// reportTool 返回 yoyo 自带工人用的 orchestrator_report，ready 前检查计划文件。
func (e *Executor) reportTool(ticketID string) dsh.Tool {
	return e.reportToolWith(ticketID, true)
}

// reportToolWith 返回 ticketID 使用的 orchestrator_report。progress 不结束回合，其余成功的报告会结束回合。
// planGate 为假时 ready 不检查计划文件；外部工人没有 update_plan，用它。
func (e *Executor) reportToolWith(ticketID string, planGate bool) dsh.Tool {
	return dsh.Tool{
		Name:        "orchestrator_report",
		Description: "向工单报告状态。state 取 progress、waiting、ready、blocked、failed 或 dropped。summary 是给人看的一句，也会留在这条会话末尾。progress 不结束这一轮。ready 只在任务说明里的完成标准达到、并且计划步骤都已 complete 时使用；计划里还有 pending 或 in_progress 会被拒绝。有 question 时不要用 ready。question 只在被硬性挡住时填写：没授权的破坏、说明和工单目录以外的必改路径、缺少密钥。不要为了确认下一步而提问。",
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
			body.State = askReport(body.State, body.Question)
			if planGate && body.State == "ready" && e.planStillOpen(ticketID) {
				return dsh.Result{Text: "计划里还有没完成的步骤。先做完，用 update_plan 标成 complete，然后再 ready。", IsError: true}, nil
			}
			err := e.Engine.Report(ticketID, Report{State: body.State, Summary: body.Summary, Question: body.Question})
			if err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			if e.OnReport != nil && terminalReport(body.State, body.Question) {
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

// assignmentText 把 ticket 的编号、目录、任务组、授权说明和输入拼成派工正文。dir、group 或 mandate 为空时不写那一行。
func assignmentText(ticket Ticket, dir, mandate, group string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "工单 %s\n", ticket.ID)
	if dir != "" {
		fmt.Fprintf(&b, "工作目录 %s\n", dir)
	}
	if group = strings.TrimSpace(group); group != "" {
		b.WriteString(group)
		b.WriteByte('\n')
	}
	if mandate = strings.TrimSpace(mandate); mandate != "" {
		b.WriteString(mandate)
		b.WriteByte('\n')
	}
	for _, in := range ticket.Inputs {
		if in.Text == "" || strings.TrimSpace(in.Text) == hostRestartNote {
			continue
		}
		b.WriteString(in.Text)
		b.WriteByte('\n')
	}
	b.WriteString("完成后调用 orchestrator_report。")
	return b.String()
}

// mandateLine 写明推断接下的工单依据哪几条规则。人明确交代的工单不写。超出这些规则就报 dropped，不去问人。
func (e *Executor) mandateLine(ticket Ticket) string {
	if e == nil || e.Engine == nil || ticket.Mandate.Strength != "inferred" {
		return ""
	}
	texts := e.Engine.RuleTexts(ticket.Mandate.Rules)
	if len(texts) == 0 {
		return ""
	}
	return mandatePrefix + strings.Join(texts, "；") + "。做的事超出这些规则时，用 orchestrator_report 报 dropped，不要问人。"
}

// mandatePrefix 是派工正文里授权那一行的开头。任务页按它把这一行从对话里拿掉。
const mandatePrefix = "这张工单是按规则主动接下的："

// groupLinePrefix 是派工正文里任务组那一行的开头，后面紧跟任务组编号。任务页按它把这一行从对话里拿掉。
const groupLinePrefix = "任务组 grp-"

// groupLine 是任务组成员派工正文里的一行：「任务组 grp-7「做 todo 应用」（分工）· 你负责：后端」。不是成员时为空。
func (e *Executor) groupLine(ticket Ticket) string {
	if e == nil || e.Engine == nil || ticket.Group == "" {
		return ""
	}
	g, ok := e.Engine.Group(ticket.Group)
	if !ok {
		return ""
	}
	line := "任务组 " + g.ID + "「" + g.Title + "」"
	if g.Mode == GroupCompare {
		return line + "（对比）"
	}
	line += "（分工）"
	if role := firstText(ticket.Role, ticket.Title); role != "" {
		line += " · 你负责：" + role
	}
	return line
}
