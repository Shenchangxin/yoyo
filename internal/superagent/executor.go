// executor.go 用 dsh.Agent 执行派工。它不决定接单。
// 每个工单一个代理。前台命令超过预算后转入后台，完成时 Steer 回同一代理。

package superagent

import (
	"context"
	"encoding/json"
	"fmt"
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

// Executor 把工单派工跑成 dsh 回合。Shell 执行命令。Foreground 小于等于 0 时用 dsh.DefaultForeground。
type Executor struct {
	Engine     *Engine
	Slots      *Slots
	Model      dsh.Completer
	Shell      func(ctx context.Context, command string) (string, error)
	Foreground time.Duration

	mu     sync.Mutex
	agents map[string]*dsh.Agent
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
	asg := ticket.Assignments[len(ticket.Assignments)-1]
	if err := e.Engine.AssignmentStarted(ticketID, asg.ID); err != nil {
		return err
	}
	reg := dsh.NewRegistry()
	agent := dsh.New("ticket-"+ticketID, e.Model, reg, dsh.Options{
		Prompt: "你是工单执行器。用 orchestrator_report 报告 progress、waiting、ready、blocked、failed 或 dropped。前台命令超过预算会转入后台。",
		Model:  "yoyo-dsh",
	})
	reg.Add(e.reportTool(ticketID))
	reg.Add(e.shellTool(agent))
	e.mu.Lock()
	e.agents[ticketID] = agent
	e.mu.Unlock()
	e.Slots.Running(ticketID)
	release = false
	text := assignmentText(ticket)
	go func() {
		defer e.Slots.Free(ticketID)
		agent.Followup(dsh.UserMessage{Text: text, Source: "assignment"})
		agent.WhenIdle()
	}()
	return nil
}

// Deliver 把 text 交给 ticketID 的代理，作为人的下一步消息。代理还没启动时什么都不做。
func (e *Executor) Deliver(ticketID, text string) {
	e.mu.Lock()
	agent := e.agents[ticketID]
	e.mu.Unlock()
	if agent == nil || text == "" {
		return
	}
	agent.Steer(dsh.UserMessage{Text: text, Source: "human"})
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
			return dsh.Result{Text: "recorded " + body.State, Concludes: body.State != "progress"}, nil
		},
	}
}

// shellTool 返回 agent 上的 shell。命令在前台预算内结束就把输出交回，否则转入后台。
func (e *Executor) shellTool(agent *dsh.Agent) dsh.Tool {
	return dsh.Tool{
		Name:        "shell",
		Description: "运行一条命令。超过前台预算后转入后台，完成时再投递输出。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			budget := e.Foreground
			view := agent.RunJob(call.ID, budget, func(ctx context.Context) (string, error) {
				if e.Shell == nil {
					return "", errCode("no_shell")
				}
				return e.Shell(ctx, body.Command)
			})
			if view.Err != nil && !view.Background {
				return dsh.Result{Text: view.Err.Error(), IsError: true}, nil
			}
			if view.Background {
				return dsh.Result{Text: "background job " + view.ID}, nil
			}
			return dsh.Result{Text: view.Output}, nil
		},
	}
}

// assignmentText 把 ticket 的编号和输入拼成派工正文。
func assignmentText(ticket Ticket) string {
	var b strings.Builder
	fmt.Fprintf(&b, "工单 %s\n", ticket.ID)
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
