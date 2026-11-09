// Package acpworker 让任何讲 ACP 的 agent 当 yoyo 工单的工人。
//
// worker.go 拥有一个外部工人：实现 superagent.Worker，第一次有活时才连接（先 resume，再 load，都不行就新建），
// 一个后台循环一轮一轮地发提示，处理取消、单轮超时、空闲回收、崩溃和要求登录。
// 日志归一化在 journal.go，权限在 perm.go，文件读写在 fs.go，会话参数在 prompt.go，进程上限在 pool.go。
// 不拥有 agent 的安装、检测和启动方式（buddy），也不拥有控制面（mcpserve）和工单状态（superagent）。
package acpworker

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// defaultIdle 是空闲多久之后关掉进程。工人对象留着，下次有活再连。
const defaultIdle = 10 * time.Minute

var (
	// errTurnTimeout 是一轮超过 TurnTimeout 时取消的原因。
	errTurnTimeout = errors.New("turn timeout")
	// errWorkerClosed 是工人已经关掉后还要连接。
	errWorkerClosed = errors.New("acp worker is closed")
)

// Config 是打开一个外部工人需要的全部输入。Spec 来自执行器，其余由 app 填。
type Config struct {
	Spec superagent.WorkerSpec
	// Launch 是 agent 的启动方式。Env 已经是白名单变量加这个 agent 的密钥；Dir 会换成工单目录。
	Launch acp.Launch
	// Dial 连上 agent。为 nil 时用 acp.Dial，测试换成同进程的假 agent。
	Dial func(ctx context.Context, opts acp.Options) (*acp.Client, error)
	// Version 写进 clientInfo。
	Version string
	// MCPURL 和 MCPToken 是控制面的地址和这张工单的 token。任一为空时不给 agent 配控制面。
	MCPURL, MCPToken string
	// Exe 是 stdio 桥要启动的 yoyo 可执行文件。为空时用 os.Executable。
	Exe string
	// Model 是人为这个 agent 选的模型。为空时用 agent 自己的默认。
	Model string
	// Approve 决定一次权限请求。为 nil 时一律拒绝。
	Approve func(ctx context.Context, req Approval) (bool, error)
	// CheckWrite 在 fs 写入前检查路径，返回错误时不写。为 nil 时只按目录限制。
	CheckWrite func(ctx context.Context, path string) error
	// TurnTimeout 是一轮最长多久，0 表示不限。IdleTimeout 是空闲多久后关掉进程，0 时用 10 分钟。
	TurnTimeout, IdleTimeout time.Duration
	// OnAuth 在 agent 要求登录时调用，detail 打过码。
	OnAuth func(detail string)
	// OnRun 在每次连接之后调用，err 为 nil 表示连上了。
	OnRun func(err error)
	// Redact 给要写进日志的错误打码。为 nil 时原样。
	Redact func(string) string
	// OnClose 在工人 Close 之后调用一次。
	OnClose func()
	// Log 记连接诊断，可以为 nil。
	Log func(string)
}

// Worker 是一张工单上的一个外部 agent。进程可以被回收再重连，对象和日志一直留到工单结束。
type Worker struct {
	cfg  Config
	spec superagent.WorkerSpec
	sess *dsh.Session
	jr   *journal
	pool *pool

	busy   atomic.Bool
	usedAt atomic.Int64

	mu      sync.Mutex
	idle    *sync.Cond
	queue   []dsh.UserMessage
	running bool
	closed  bool
	halted  bool
	turn    int
	client  *acp.Client
	sid     string
	pointer string
	cancel  context.CancelCauseFunc
	timer   *time.Timer
}

// Open 建一个外部工人。不启动进程，第一次 Followup 时才连接。
func Open(cfg Config) *Worker { return open(cfg, processes) }

// open 建一个记在 p 里的工人。
func open(cfg Config, p *pool) *Worker {
	w := &Worker{cfg: cfg, spec: cfg.Spec, sess: dsh.NewSession(), pool: p, pointer: strings.TrimSpace(cfg.Spec.Pointer)}
	w.idle = sync.NewCond(&w.mu)
	w.jr = newJournal(w.sess, cfg.Spec.Dir, w.onUsage)
	w.usedAt.Store(time.Now().UnixNano())
	p.track(w)
	return w
}

var _ superagent.Worker = (*Worker)(nil)

// Followup 把 msg 排进队列。空闲时立刻开一轮，正在跑时等这一轮结束后一起发。
func (w *Worker) Followup(msg dsh.UserMessage) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.halted {
		return
	}
	w.queue = append(w.queue, msg)
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if !w.running {
		w.running = true
		w.busy.Store(true)
		go w.loop()
	}
}

// Steer 返回假。ACP 没有标准的插话，执行器改用 Followup。
func (w *Worker) Steer(dsh.UserMessage) bool { return false }

// Cancel 停下这一轮：取消提示，客户端会发 session/cancel。keepInbox 为假时清掉还没发的输入。
func (w *Worker) Cancel(cause dsh.CancelCause, keepInbox bool) {
	w.mu.Lock()
	if !keepInbox {
		w.queue = nil
	}
	cancel := w.cancel
	w.mu.Unlock()
	if cancel != nil {
		cancel(cause)
	}
}

// WhenIdle 阻塞到这一轮和排队的输入都跑完。
func (w *Worker) WhenIdle() {
	w.mu.Lock()
	for w.running {
		w.idle.Wait()
	}
	w.mu.Unlock()
}

// Status 返回 running 或 idle。
func (w *Worker) Status() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return "running"
	}
	return "idle"
}

// Session 返回归一化的日志。
func (w *Worker) Session() *dsh.Session { return w.sess }

// Restore 把已存的日志灌回去，回合号接在 turn 后面。之后新开的会话会带上延续说明。
func (w *Worker) Restore(events []dsh.Event, turn int) {
	w.sess.Restore(events)
	w.mu.Lock()
	if turn > w.turn {
		w.turn = turn
	}
	w.mu.Unlock()
}

// Caps 是外部工人的规则：只催一次，不查计划文件，不能插话，单轮上限按配置。
func (w *Worker) Caps() superagent.WorkerCaps {
	return superagent.WorkerCaps{Nudges: 1, External: true, TurnMinutes: int(w.cfg.TurnTimeout / time.Minute)}
}

// Close 在工单结束时调用：取消这一轮、关掉会话和进程，再调 OnClose。重复调用只做一次。
func (w *Worker) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.queue = nil
	cancel := w.cancel
	c := w.client
	w.client, w.sid = nil, ""
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.mu.Unlock()
	if cancel != nil {
		cancel(dsh.CancelCause{Kind: "closed"})
	}
	if c != nil {
		_ = c.Close()
	}
	w.pool.forget(w)
	if w.cfg.OnClose != nil {
		w.cfg.OnClose()
	}
	return nil
}

// loop 一轮一轮地发队列里的消息，队列空了就停下并开始计空闲。
func (w *Worker) loop() {
	for {
		w.mu.Lock()
		if w.closed || len(w.queue) == 0 {
			w.running = false
			w.busy.Store(false)
			w.armIdleLocked()
			w.idle.Broadcast()
			w.mu.Unlock()
			return
		}
		msgs := w.queue
		w.queue = nil
		w.turn++
		turn := w.turn
		ctx, cancel := context.WithCancelCause(context.Background())
		w.cancel = cancel
		w.mu.Unlock()

		w.runTurn(ctx, turn, msgs)
		cancel(nil)

		w.mu.Lock()
		w.cancel = nil
		w.mu.Unlock()
		w.usedAt.Store(time.Now().UnixNano())
	}
}

// runTurn 写回合开始和这一轮的输入，发提示，再写回合结束。
func (w *Worker) runTurn(ctx context.Context, turn int, msgs []dsh.UserMessage) {
	w.sess.Append(dsh.EventTurnStart, turn, 0, nil)
	for _, msg := range msgs {
		w.sess.Append(dsh.EventUser, turn, 0, msg)
	}
	w.jr.begin(turn)
	end := w.prompt(ctx, turn, msgs)
	w.jr.finish(end.Kind == "completed")
	if end.Kind == "error" {
		diaglog.Warn("acp worker turn failed", "component", "acp", "ticket", w.spec.TicketID, "agent", w.spec.Agent, "reason", end.Reason)
	}
	w.sess.Append(dsh.EventTurnEnd, turn, 0, end)
}

// prompt 连上 agent 并发这一轮，返回回合结束。超过 TurnTimeout 时取消。
func (w *Worker) prompt(ctx context.Context, turn int, msgs []dsh.UserMessage) dsh.TurnEnd {
	if w.cfg.TurnTimeout > 0 {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeoutCause(ctx, w.cfg.TurnTimeout, errTurnTimeout)
		defer stop()
	}
	c, sid, preface, err := w.ensure(ctx, turn)
	if err != nil {
		return w.endFor(ctx, err)
	}
	stop, err := c.Prompt(ctx, sid, promptBlocks(preface, msgs))
	if ctx.Err() != nil || err != nil {
		return w.endFor(ctx, err)
	}
	return stopEnd(stop)
}

// endFor 把失败换成回合结束：取消和超时记 aborted；要求登录记 auth_required 并通知 Buddy；进程断开记 agent exited，下一轮重连。
func (w *Worker) endFor(ctx context.Context, err error) dsh.TurnEnd {
	if ctx.Err() != nil {
		cause := context.Cause(ctx)
		if errors.Is(cause, errTurnTimeout) {
			return dsh.TurnEnd{Kind: "aborted", Reason: superagent.ReasonTurnTimeout}
		}
		var cc dsh.CancelCause
		if errors.As(cause, &cc) {
			return dsh.TurnEnd{Kind: "aborted", Reason: cc.Error()}
		}
		return dsh.TurnEnd{Kind: "aborted", Reason: "cancelled"}
	}
	if err == nil {
		err = errors.New("prompt failed")
	}
	if errors.Is(err, acp.ErrAuthRequired) {
		w.drop()
		if w.cfg.OnAuth != nil {
			w.cfg.OnAuth(w.redact(err.Error()))
		}
		return dsh.TurnEnd{Kind: "error", Reason: superagent.ReasonAuthRequired}
	}
	reason := err.Error()
	if errors.Is(err, acp.ErrClosed) {
		w.drop()
		if !strings.HasPrefix(reason, "agent exited") {
			reason = "agent exited: " + reason
		}
	}
	return dsh.TurnEnd{Kind: "error", Reason: clipRunes(w.redact(reason), 300)}
}

// stopEnd 把 stopReason 换成回合结束。
func stopEnd(stop acp.StopReason) dsh.TurnEnd {
	switch stop {
	case acp.StopMaxTokens:
		return dsh.TurnEnd{Kind: "max-tokens"}
	case acp.StopCancelled:
		return dsh.TurnEnd{Kind: "aborted", Reason: "cancelled"}
	case acp.StopRefusal:
		return dsh.TurnEnd{Kind: "error", Reason: "refusal"}
	default:
		return dsh.TurnEnd{Kind: "completed"}
	}
}

// ensure 返回能用的连接。进程不在或已经断了就重连；新会话要带的前言一起返回。
func (w *Worker) ensure(ctx context.Context, turn int) (*acp.Client, string, string, error) {
	w.mu.Lock()
	c, sid := w.client, w.sid
	w.mu.Unlock()
	if c != nil {
		select {
		case <-c.Done():
			w.drop()
		default:
			return c, sid, "", nil
		}
	}
	return w.connect(ctx, turn)
}

// connect 启动 agent、握手、续上或新建会话。新建时按人选的模型换一次，并返回说明块和延续说明。
func (w *Worker) connect(ctx context.Context, turn int) (*acp.Client, string, string, error) {
	if w.stopped() {
		return nil, "", "", errWorkerClosed
	}
	opts := w.cfg.Launch.Options()
	opts.Dir = w.spec.Dir
	opts.Version = w.cfg.Version
	opts.Name = w.spec.Agent
	opts.Log = w.cfg.Log
	opts.Handler = acp.Handler{Update: w.jr.update, Permission: w.onPermission, ReadFile: w.readFile, WriteFile: w.writeFile}
	dial := w.cfg.Dial
	if dial == nil {
		dial = acp.Dial
	}
	c, err := dial(ctx, opts)
	if err != nil {
		w.noteRun(err)
		return nil, "", "", err
	}
	caps := c.Caps()
	servers := mcpServers(caps, w.cfg.MCPURL, w.cfg.MCPToken, w.exe())
	sid, how, resp, err := w.open(ctx, c, caps, servers, extraDirs(caps, w.spec))
	if err != nil {
		_ = c.Close()
		if !errors.Is(err, acp.ErrAuthRequired) {
			w.noteRun(err)
		}
		return nil, "", "", err
	}
	preface := ""
	if how == "new" {
		w.applyModel(ctx, c, sid, resp)
		preface = w.preface(caps, turn)
	}
	w.mu.Lock()
	closed, halted := w.closed, w.halted
	changed := sid != w.pointer
	if !closed && !halted {
		w.client, w.sid, w.pointer = c, sid, sid
	}
	w.mu.Unlock()
	if closed {
		_ = c.Close()
		return nil, "", "", errWorkerClosed
	}
	if halted {
		_ = c.Suspend()
		return nil, "", "", errWorkerClosed
	}
	w.pool.add(w)
	w.usedAt.Store(time.Now().UnixNano())
	if changed && w.spec.OnPointer != nil {
		w.spec.OnPointer(sid)
	}
	w.noteRun(nil)
	diaglog.Info("acp worker connected", "component", "acp", "ticket", w.spec.TicketID, "agent", w.spec.Agent, "session", how)
	return c, sid, preface, nil
}

// open 先 resume、再 load（都要对方声明了才试），都不行就新建。连接断了、要求登录或已经取消时不再往下试。
func (w *Worker) open(ctx context.Context, c *acp.Client, caps acp.AgentCapabilities, servers []acp.McpServer, dirs []string) (string, string, acp.NewSessionResponse, error) {
	w.mu.Lock()
	pointer := w.pointer
	w.mu.Unlock()
	if pointer != "" {
		old := acp.LoadSessionRequest{SessionID: pointer, Cwd: w.spec.Dir, McpServers: servers, AdditionalDirectories: dirs}
		if caps.SessionCapabilities.Resume {
			err := c.ResumeSession(ctx, old)
			if err == nil {
				return pointer, "resume", acp.NewSessionResponse{}, nil
			}
			if stopTrying(err) {
				return "", "", acp.NewSessionResponse{}, err
			}
		}
		if caps.LoadSession {
			err := c.LoadSession(ctx, old)
			if err == nil {
				return pointer, "load", acp.NewSessionResponse{}, nil
			}
			if stopTrying(err) {
				return "", "", acp.NewSessionResponse{}, err
			}
		}
	}
	resp, err := c.NewSession(ctx, acp.NewSessionRequest{
		Cwd: w.spec.Dir, McpServers: servers, AdditionalDirectories: dirs, SystemPrompt: systemPrompt(caps, w.spec.System),
	})
	if err != nil {
		return "", "", resp, err
	}
	return resp.SessionID, "new", resp, nil
}

// stopTrying 在连接已断、要求登录或已经取消时为真，换一种续接方式也没用。
func stopTrying(err error) bool {
	return errors.Is(err, acp.ErrClosed) || errors.Is(err, acp.ErrAuthRequired) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// applyModel 在人选了模型、且新会话当前不是它时换一次：有 models 用 session/set_model，否则改 category 为 model 的设置。失败不影响这一轮。
func (w *Worker) applyModel(ctx context.Context, c *acp.Client, sid string, resp acp.NewSessionResponse) {
	model := strings.TrimSpace(w.cfg.Model)
	if model == "" {
		return
	}
	var err error
	switch {
	case resp.Models != nil:
		if resp.Models.CurrentModelID != model {
			err = c.SetModel(ctx, sid, model)
		}
	default:
		for _, opt := range resp.ConfigOptions {
			if opt.Category == "model" {
				if opt.CurrentValue != model {
					err = c.SetConfigOption(ctx, sid, opt.ID, model)
				}
				break
			}
		}
	}
	if err != nil {
		diaglog.Warn("acp worker could not switch model", "component", "acp", "agent", w.spec.Agent, "err", w.redact(err.Error()))
	}
}

// preface 是新会话第一条消息前的话：对方不支持 systemPrompt 时的说明块，以及有旧日志时的延续说明。
func (w *Worker) preface(caps acp.AgentCapabilities, turn int) string {
	var parts []string
	if systemPrompt(caps, w.spec.System) == nil && strings.TrimSpace(w.spec.System) != "" {
		parts = append(parts, instructionsBlock(w.spec.System))
	}
	var prior []dsh.Event
	for _, ev := range w.sess.Events() {
		if ev.Turn < turn {
			prior = append(prior, ev)
		}
	}
	if note := continuationNote(prior); note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, "\n\n")
}

// drop 丢掉当前连接并停掉进程，不关会话，下一轮重连时续接。
func (w *Worker) drop() {
	w.mu.Lock()
	c := w.client
	w.client, w.sid = nil, ""
	w.mu.Unlock()
	if c != nil {
		_ = c.Suspend()
		w.pool.remove(w)
	}
}

// suspend 在空闲时停掉进程，留着会话编号下次续接。正在跑时什么都不做。
func (w *Worker) suspend() {
	w.mu.Lock()
	if w.running || w.client == nil {
		w.mu.Unlock()
		return
	}
	c := w.client
	w.client, w.sid = nil, ""
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.mu.Unlock()
	_ = c.Suspend()
	w.pool.remove(w)
	diaglog.Info("acp worker suspended", "component", "acp", "ticket", w.spec.TicketID, "agent", w.spec.Agent)
}

// stopped 在工人已经关掉或在应用退出时停下后为真，这时不再连接。
func (w *Worker) stopped() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed || w.halted
}

// halt 在应用退出时调用：取消这一轮，停掉进程，之后不再连接。不关会话，下次启动还能续接。
func (w *Worker) halt() {
	w.mu.Lock()
	w.halted = true
	w.queue = nil
	cancel := w.cancel
	c := w.client
	w.client, w.sid = nil, ""
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.mu.Unlock()
	if cancel != nil {
		cancel(dsh.CancelCause{Kind: "shutdown"})
	}
	if c != nil {
		_ = c.Suspend()
	}
}

// armIdleLocked 在连着进程时开始计空闲，到点就回收。调用时要已持有锁。
func (w *Worker) armIdleLocked() {
	if w.closed || w.client == nil {
		return
	}
	d := w.cfg.IdleTimeout
	if d <= 0 {
		d = defaultIdle
	}
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(d, w.suspend)
}

// onUsage 把用量交给 OnUsage。
func (w *Worker) onUsage(u acp.UsageUpdate) {
	if w.spec.OnUsage == nil {
		return
	}
	usage := superagent.Usage{Used: u.Used, Size: u.Size}
	if u.Cost != nil {
		usage.Cost, usage.Currency = u.Cost.Amount, u.Cost.Currency
	}
	w.spec.OnUsage(usage)
}

// noteRun 把连接结果交给 OnRun。
func (w *Worker) noteRun(err error) {
	if w.cfg.OnRun != nil {
		w.cfg.OnRun(err)
	}
}

// redact 用 Redact 打码。
func (w *Worker) redact(text string) string {
	if w.cfg.Redact == nil {
		return text
	}
	return w.cfg.Redact(text)
}

// exe 是 stdio 桥要启动的程序。
func (w *Worker) exe() string {
	if w.cfg.Exe != "" {
		return w.cfg.Exe
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe
}
