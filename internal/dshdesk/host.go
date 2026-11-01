// host.go 持有每个会话当前这一轮的 dsh 代理。
// 它不决定提示和工具从哪来。App.sendLocked 填好 Turn 再调用 Run。

package dshdesk

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Turn 是一轮桌面发送。User 已经由调用方写入轨迹时，这里不再发一条来源为空的用户事件。
type Turn struct {
	SessionID        string
	TurnID           string
	Prompt           string
	User             string
	Inject           string
	Model            string
	MaxTokens        int
	Budget           float64
	History          []runtime.Message
	Client           runtime.Client
	Tools            *runtime.WorkspaceTools
	Trace            *trace.Store
	OnEvent          func(trace.Event)
	ShowThinking     bool
	HarnessHash      string
	ModelFingerprint string
	Meter            *runtime.Meter
	UserParts        []runtime.ContentPart
	PullSteer        func() string
	StopHooks        []runtime.FileHook
	Checkpoint       string
}

// Host 按会话保存正在跑的代理，供 Steer 和 Interrupt 找到它。
type Host struct {
	mu       sync.Mutex
	sessions map[string]*live
}

// live 是一个会话上长期活着的代理、工具和补全桥。同一会话的下一轮发送复用它，不再重灌历史。
type live struct {
	agent      *dsh.Agent
	reg        *dsh.Registry
	box        *toolBox
	bridge     *Bridge
	ready      bool
	checkpoint string
}

// New 返回空的宿主。
func New() *Host {
	return &Host{sessions: map[string]*live{}}
}

// Run 用这个会话上的代理跑 turn。第一次发送建代理并灌入历史，之后的发送复用同一代理。
// ctx 取消时停掉模型和这一轮的命令。返回最后一条助手正文。
func (h *Host) Run(ctx context.Context, turn Turn) (string, error) {
	if h == nil {
		return "", errString("no desk")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if turn.Tools != nil {
		turn.Tools.DshShell = true
		turn.Tools.Ctx = ctx
	}
	sessionID := turn.SessionID
	if sessionID == "" {
		sessionID = "session"
		turn.SessionID = sessionID
	}
	h.mu.Lock()
	if h.sessions == nil {
		h.sessions = map[string]*live{}
	}
	cur := h.sessions[sessionID]
	h.mu.Unlock()
	var (
		box    *toolBox
		bridge *Bridge
		reg    *dsh.Registry
		agent  *dsh.Agent
	)
	if cur != nil && cur.ready && cur.agent != nil {
		box, bridge, reg, agent = cur.box, cur.bridge, cur.reg, cur.agent
		box.set(turn.Tools)
		bridge.Client = turn.Client
		bridge.Model = turn.Model
		bridge.MaxTokens = turn.MaxTokens
		bridge.ShowThinking = turn.ShowThinking
		bridge.Meter = turn.Meter
		bridge.UserParts = turn.UserParts
		bridge.partsUsed = false
		agent.SetPrompt(turn.Prompt)
		agent.SetRoute(turn.Model, turn.MaxTokens)
		if cur.checkpoint != turn.Checkpoint {
			agent.ResetSession()
			seedHistory(agent, turn.History)
			cur.checkpoint = turn.Checkpoint
		}
	} else {
		box = &toolBox{note: make(chan struct{}, 1)}
		box.set(turn.Tools)
		bridge = &Bridge{
			Client:       turn.Client,
			Model:        turn.Model,
			MaxTokens:    turn.MaxTokens,
			ShowThinking: turn.ShowThinking,
			Meter:        turn.Meter,
			UserParts:    turn.UserParts,
		}
		reg = dsh.NewRegistry()
		agent = dsh.New(sessionID, bridge, reg, dsh.Options{
			Prompt:    turn.Prompt,
			Model:     turn.Model,
			MaxTokens: turn.MaxTokens,
		})
		cur = &live{agent: agent, reg: reg, box: box, bridge: bridge, ready: true, checkpoint: turn.Checkpoint}
		h.mu.Lock()
		h.sessions[sessionID] = cur
		h.mu.Unlock()
		agent.ResumeTurn(recordedTurn(turn.Trace, sessionID))
		seedHistory(agent, turn.History)
	}
	agent.SetBeforeModel(func(context.Context) {
		reg.Reset()
		box.bind(reg, agent)
	})
	reg.Reset()
	box.bind(reg, agent)
	wireStream(bridge, turn)

	stopProject := make(chan struct{})
	projDone := make(chan struct{})
	// 游标要在 Followup 之前记下。放进协程里再读，回合开始事件可能已经写上，投影会把它跳过。
	cursor := 0
	if agent.Session() != nil {
		cursor = len(agent.Session().Events())
	}
	go func() {
		defer close(projDone)
		project(turn, agent, cursor, stopProject)
	}()
	stopPull := make(chan struct{})
	var pullOnce sync.Once
	endPull := func() { pullOnce.Do(func() { close(stopPull) }) }
	defer endPull()
	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stopPull:
				return
			case <-tick.C:
				pullSteer(agent, turn.PullSteer)
			}
		}
	}()
	stopWatch := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			agent.Cancel(dsh.CancelCause{Kind: "user", Reason: "stop"}, false)
		case <-stopWatch:
		}
	}()
	defer func() {
		close(stopWatch)
		close(stopProject)
		<-projDone
	}()

	if ctx.Err() != nil {
		return "", stopCancelled(ctx.Err())
	}
	pullSteer(agent, turn.PullSteer)
	if strings.TrimSpace(turn.Inject) != "" {
		turn.emit(trace.Event{
			Type:    trace.TypeInject,
			Source:  "dsh",
			Payload: map[string]any{"text": turn.Inject},
		}, true)
	}
	agent.Followup(dsh.UserMessage{Text: composeUser(turn), Source: ""})
	waitIdle(ctx, agent, box)
	endPull()
	waitIdle(ctx, agent, box)
	if ctx.Err() != nil {
		return lastAssistant(agent), stopCancelled(ctx.Err())
	}
	end := lastEnd(agent)
	if end.Kind == "completed" {
		for hop := 0; hop < 4 && ctx.Err() == nil; hop++ {
			block, msg := denyStop(turn.StopHooks)
			if !block {
				break
			}
			agent.Followup(dsh.UserMessage{Text: msg, Source: "hook"})
			waitIdle(ctx, agent, box)
		}
		end = lastEnd(agent)
	}
	text := lastAssistant(agent)
	if end.Kind == "error" {
		reason := end.Reason
		if reason == "" {
			reason = "model error"
		}
		return text, &runtime.StopError{Reason: runtime.StopModelError, Msg: reason, Err: errString(reason)}
	}
	if turn.Meter != nil && turn.Budget > 0 {
		if err := turn.Meter.Check(turn.Budget); err != nil {
			return text, err
		}
	}
	return text, nil
}

// Steer 把 text 交给 sessionID 正在跑的代理，作为下一步消息。没有这一轮的代理时返回假。
func (h *Host) Steer(sessionID, text string) bool {
	if h == nil || strings.TrimSpace(text) == "" {
		return false
	}
	h.mu.Lock()
	cur := h.sessions[sessionID]
	h.mu.Unlock()
	if cur == nil || cur.agent == nil {
		return false
	}
	cur.agent.Steer(dsh.UserMessage{Text: text, Source: "steer"})
	return true
}

// Forget 丢掉 sessionID 上的代理。下一轮发送会按当时的历史重新建一个。
func (h *Host) Forget(sessionID string) {
	if h == nil || sessionID == "" {
		return
	}
	h.mu.Lock()
	delete(h.sessions, sessionID)
	h.mu.Unlock()
}

// Interrupt 取消 sessionID 正在跑的代理。没有代理时什么都不做。已投递的收件箱清掉。
func (h *Host) Interrupt(sessionID string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	cur := h.sessions[sessionID]
	h.mu.Unlock()
	if cur == nil || cur.agent == nil {
		return
	}
	cur.agent.Cancel(dsh.CancelCause{Kind: "user", Reason: "stop"}, false)
}

// denyStop 在 rules 里第一条 Deny 上返回真和要继续的消息。reason 为空时用钩子文件的默认句子。
func denyStop(rules []runtime.FileHook) (bool, string) {
	for _, rule := range rules {
		if !rule.Deny {
			continue
		}
		msg := rule.Reason
		if msg == "" {
			msg = "blocked by .yoyo/hooks.json stop hook"
		}
		return true, msg
	}
	return false, ""
}

// composeUser 拼出这一轮要跟进的用户文本。turn 的 User 和 Inject 都空时用 Continue.。
func composeUser(turn Turn) string {
	text := strings.TrimSpace(turn.User)
	inject := strings.TrimSpace(turn.Inject)
	switch {
	case text != "" && inject != "":
		return text + "\n\n" + inject
	case text != "":
		return text
	case inject != "":
		return inject
	default:
		return "Continue."
	}
}

// pullSteer 把 pull 取到的文本交给 agent。pull 为 nil 或文本为空时什么都不做。
func pullSteer(agent *dsh.Agent, pull func() string) {
	if agent == nil || pull == nil {
		return
	}
	text := strings.TrimSpace(pull())
	if text == "" {
		return
	}
	agent.Steer(dsh.UserMessage{Text: text, Source: "steer"})
}

// waitIdle 等到 agent 空闲且 box 没有后台命令。ctx 取消时返回。
func waitIdle(ctx context.Context, agent *dsh.Agent, box *toolBox) {
	if agent == nil {
		return
	}
	for {
		agent.WhenIdle()
		if ctx.Err() != nil {
			return
		}
		left := 0
		if box != nil {
			left = box.jobsLeft()
		}
		if left == 0 && agent.Status() != "running" {
			if box == nil || box.jobsLeft() == 0 {
				return
			}
		}
		note := box.note
		if note == nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-note:
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// stopCancelled 把 err 包成取消。err 为 nil 时用 context.Canceled 的文本。
func stopCancelled(err error) error {
	if err == nil {
		err = context.Canceled
	}
	return &runtime.StopError{Reason: runtime.StopCancelled, Msg: err.Error(), Err: err}
}

// lastAssistant 返回 agent 日志里最后一条非空助手正文。没有则返回空串。
func lastAssistant(agent *dsh.Agent) string {
	if agent == nil || agent.Session() == nil {
		return ""
	}
	evs := agent.Session().Events()
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Type != dsh.EventAssistant {
			continue
		}
		msg, ok := evs[i].Data.(dsh.Assistant)
		if ok && strings.TrimSpace(msg.Text) != "" {
			return msg.Text
		}
	}
	return ""
}

// lastEnd 返回 agent 日志里最后一条回合结束。没有则返回空结束。
func lastEnd(agent *dsh.Agent) dsh.TurnEnd {
	if agent == nil || agent.Session() == nil {
		return dsh.TurnEnd{}
	}
	evs := agent.Session().Events()
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Type != dsh.EventTurnEnd {
			continue
		}
		if end, ok := evs[i].Data.(dsh.TurnEnd); ok {
			return end
		}
	}
	return dsh.TurnEnd{}
}
