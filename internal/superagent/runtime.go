// runtime.go 持有编排引擎、两秒轮次和 SQLite 快照。
// 它不画界面。桌面和 HTTP 通过 View、Decide、SetMode 使用它。

package superagent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"

	_ "modernc.org/sqlite"
)

// Runtime 把引擎、副作用、来源轮询和快照放在一起。两秒轮次调用 Pass。
type Runtime struct {
	mu           sync.Mutex
	eng          *Engine
	effects      *Queue
	exec         *Executor
	ready        func() error
	db           *sql.DB
	cancel       context.CancelFunc
	client       *http.Client
	sources      SourceConfig
	status       map[string]SourceState
	health       string
	at           time.Time
	workers      int
	workRoot     string
	sourcesSaved bool
	lastGitHub   time.Time
	lastFeishu   time.Time
	commentSince time.Time
	lastMail     time.Time
	prepareCmd   string
	intake       func() []Trigger
	readMiss     map[string]int
	cursors      map[string]string
	logs         map[string][]byte
	report       func(ticketID, state, summary, question string)
	notice       func(sessionID, text string)
	profile      func() string
	opener       Opener
	agentLimit   func(agent string) int
	agentName    func(agent string) string
	roots        func(ticketID string) []string
}

// Open 打开 path 上的快照并还原引擎。path 为空时只放在内存里。policy 在没有快照时作为初始策略。
func Open(path string, policy Policy) (*Runtime, error) {
	eng := NewEngine(policy)
	rt := &Runtime{
		eng:     eng,
		effects: NewQueue(),
		client:  &http.Client{Timeout: 20 * time.Second},
		sources: DefaultSources(),
		status:  map[string]SourceState{},
		health:  "ready",
	}
	if path == "" {
		return rt, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS snapshot (id INTEGER PRIMARY KEY CHECK (id = 1), body TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS worker_log (ticket TEXT PRIMARY KEY, body TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	rt.db = db
	if err := rt.load(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return rt, nil
}

// SetWorkRoot 记下执行根。每张工单会在这个目录下的 tickets/<id> 里跑命令。还没 Bind 时先记下，Bind 时带上。
func (rt *Runtime) SetWorkRoot(dir string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.workRoot = dir
	if rt.exec != nil {
		rt.exec.Root = dir
	}
}

// Bind 接上模型和命令。ready 返回错误时派工先推迟。model 是工单代理的补全。shell 在给出的目录里执行命令。
func (rt *Runtime) Bind(ready func() error, model dsh.Completer, shell func(ctx context.Context, dir, command string) (string, error)) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.ready = ready
	if rt.workers < 1 {
		rt.workers = 3
	}
	rt.exec = NewExecutor(rt.eng, NewSlots(rt.workers), model)
	rt.exec.Root = rt.workRoot
	rt.exec.PrepareCmd = rt.prepareCmd
	rt.exec.Shell = shell
	rt.refreshEnvLocked()
	rt.attachExec()
	rt.wireEffects()
}

// attachExec 把日志、报告和准备命令接到当前执行器上。Bind 之后调用。
func (rt *Runtime) attachExec() {
	if rt == nil || rt.exec == nil {
		return
	}
	rt.exec.LogLoad = rt.loadWorkerLog
	rt.exec.LogSave = rt.saveWorkerLog
	rt.exec.OnReport = rt.queueReport
	rt.exec.Profile = rt.profile
	rt.exec.Opener = rt.opener
	rt.exec.AgentLimit = rt.agentLimit
	rt.exec.AgentName = rt.agentName
	rt.exec.Roots = rt.roots
}

// wireEffects 登记六类先落盘再执行的副作用。调用时要已持有锁。
func (rt *Runtime) wireEffects() {
	if rt == nil || rt.effects == nil {
		return
	}
	rt.effects.Handle("launch", 3, rt.runLaunch)
	rt.effects.Handle("delivery", 4, rt.runDelivery)
	rt.effects.Handle("assessment", 4, rt.runAssessment)
	rt.effects.Handle("reply", 1, rt.runReply)
	rt.effects.HandleOrdered("relay", 1, rt.runRelay)
	rt.effects.HandleOrdered("notice", 1, rt.runNotice)
	if rt.eng == nil {
		return
	}
	rt.eng.OnLaunch = func(id string) {
		rt.effects.Enqueue("launch", id, id)
	}
	rt.eng.OnInput = func(id, text string) {
		if strings.TrimSpace(text) == "" {
			return
		}
		rt.effects.Enqueue("delivery", id, id)
	}
}

// SetPrepareRunner 接上准备命令。它在执行根里跑，环境变量里没有钥匙。
func (rt *Runtime) SetPrepareRunner(fn func(ctx context.Context, root, command string, env map[string]string) (string, error)) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.exec != nil {
		rt.exec.RunPrepare = fn
	}
}

// Receive 收下 in。第二个返回值是已经落到的工单。
func (rt *Runtime) Receive(in Trigger) (string, []string) {
	if rt == nil || rt.eng == nil {
		return "empty", nil
	}
	status := rt.eng.Receive(in)
	rt.mu.Lock()
	effects := rt.effects
	rt.mu.Unlock()
	if effects != nil {
		effects.Pump(time.Now())
	}
	got, ok := rt.eng.Trigger(in.ID)
	if !ok {
		return status, nil
	}
	return status, append([]string(nil), got.Reached...)
}

// StagedProgress 是这张工单已经报过的阶段性进展。还没有时为空。
func (rt *Runtime) StagedProgress(ticketID string) string {
	if rt == nil || rt.eng == nil {
		return ""
	}
	return rt.eng.StagedProgress(ticketID)
}

// TicketLabel 是工单给人看的短名。先用第一句输入，没有时用目录短名。
func (rt *Runtime) TicketLabel(id string) string {
	if rt == nil || rt.eng == nil {
		return ""
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return ""
	}
	if len(t.Inputs) > 0 {
		if line := oneLine(t.Inputs[0].Text); line != "" {
			return line
		}
	}
	return strings.TrimSpace(t.Name)
}

// TriggerAgents 返回人在 id 这条触发里 @ 过的 agent，按出现顺序。没有这条触发时为空。
func (rt *Runtime) TriggerAgents(id string) []string {
	if rt == nil || rt.eng == nil {
		return nil
	}
	tr, ok := rt.eng.Trigger(id)
	if !ok {
		return nil
	}
	return append([]string(nil), tr.Agents...)
}

// TicketAgent 返回做 id 这张工单的 agent，yoyo 自带工人为 yoyo。没有这张工单时 ok 为假。
func (rt *Runtime) TicketAgent(id string) (string, bool) {
	if rt == nil || rt.eng == nil {
		return "", false
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return "", false
	}
	return agentKey(t.Agent), true
}

// TicketNeed 返回 id 最近一份报告要人先做的事，例如 login。最近那份没有写时为空串。
func (rt *Runtime) TicketNeed(id string) string {
	if rt == nil || rt.eng == nil {
		return ""
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return ""
	}
	asg := latestAssignment(&t)
	if asg == nil || len(asg.Reports) == 0 {
		return ""
	}
	return asg.Reports[len(asg.Reports)-1].Need
}

// TicketTexts 返回这张工单交给工人的正文，不截断。
func (rt *Runtime) TicketTexts(id string) []string {
	if rt == nil || rt.eng == nil {
		return nil
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return nil
	}
	var out []string
	for _, in := range t.Inputs {
		if strings.TrimSpace(in.Text) != "" {
			out = append(out, in.Text)
		}
	}
	return out
}

// WorkerTranscript 返回这张工单已经落下的会话。还没有日志时为空。
func (rt *Runtime) WorkerTranscript(ticketID string) []TranscriptLine {
	if rt == nil || strings.TrimSpace(ticketID) == "" {
		return nil
	}
	events, _ := decodeLog(rt.loadWorkerLog(ticketID))
	return TranscriptLines(events)
}

// DeskSession 返回开工这张工单的桌面会话。来源不是桌面时返回空串。
func (rt *Runtime) DeskSession(ticketID string) string {
	if rt == nil || rt.eng == nil {
		return ""
	}
	return rt.eng.DeskSession(ticketID)
}

// DeskAudiences 返回这份报告该写进的桌面会话。后来跟上的会话优先于开工那一条。
func (rt *Runtime) DeskAudiences(ticketID string) []string {
	if rt == nil || rt.eng == nil {
		return nil
	}
	return rt.eng.DeskAudiences(ticketID)
}

// refreshEnvLocked 把来源里不含钥匙的说明写进工人环境。调用时要已持有锁。
func (rt *Runtime) refreshEnvLocked() {
	if rt == nil || rt.exec == nil || rt.eng == nil {
		return
	}
	cfg := rt.sources
	eng := rt.eng
	rt.exec.Env = func(dir, ticketID string) map[string]string {
		name := ticketID
		if t, ok := eng.Ticket(ticketID); ok && strings.TrimSpace(t.Name) != "" {
			name = t.Name
		}
		return map[string]string{
			"DSH_SUPERAGENT_DIR":               dir,
			"DSH_SUPERAGENT_NAME":              name,
			"DSH_SUPERAGENT_TICKET":            ticketID,
			"DSH_SUPERAGENT_GITHUB_REPOSITORY": cfg.GitHubRepo,
			"DSH_SUPERAGENT_GITHUB_ACCOUNT":    cfg.GitHubAccount,
			"DSH_SUPERAGENT_FEISHU_BOT":        cfg.FeishuAppID,
			"DSH_SUPERAGENT_FEISHU_CHAT":       cfg.FeishuChat,
		}
	}
}

// SetPrepare 记下第一次派工前要跑的命令。空串表示只建空目录。面板才能改它。
func (rt *Runtime) SetPrepare(command string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.prepareCmd = strings.TrimSpace(command)
	if rt.exec != nil {
		rt.exec.PrepareCmd = rt.prepareCmd
	}
}

// SetIntake 接上邮件和日历。fn 返回推断触发，由轮询按固定间隔收下。
func (rt *Runtime) SetIntake(fn func() []Trigger) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.intake = fn
	rt.mu.Unlock()
}

// SetMount 接上工人的工作区工具。dir 是这张工单的目录。
func (rt *Runtime) SetMount(fn func(reg *dsh.Registry, agent *dsh.Agent, dir, ticketID string) (func() int, func(time.Duration))) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.exec != nil {
		rt.exec.Mount = fn
	}
}

// SetOpener 接上外部 agent 的工人。fn 为 nil 时所有工单都走 yoyo 自带工人。Bind 前后调用都可以。
func (rt *Runtime) SetOpener(fn Opener) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.opener = fn
	if rt.exec != nil {
		rt.exec.Opener = fn
	}
}

// SetAgentLimit 接上每个 agent 的并行上限。fn 返回小于 1 时这个 agent 只受全局槽限制。
func (rt *Runtime) SetAgentLimit(fn func(agent string) int) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.agentLimit = fn
	if rt.exec != nil {
		rt.exec.AgentLimit = fn
	}
}

// SetAgentName 接上 agent 给人看的名字。fn 返回空串时用编号。
func (rt *Runtime) SetAgentName(fn func(agent string) string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.agentName = fn
	if rt.exec != nil {
		rt.exec.AgentName = fn
	}
}

// SetRoots 接上每张工单除工作目录外可读写的目录。外部工人的说明、文件读写和审批都用它。
func (rt *Runtime) SetRoots(fn func(ticketID string) []string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.roots = fn
	if rt.exec != nil {
		rt.exec.Roots = fn
	}
}

// CloseWorkers 关掉所有外部工人。应用退出时调用。
func (rt *Runtime) CloseWorkers() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	exec := rt.exec
	rt.mu.Unlock()
	exec.CloseAll()
}

// SetProfile 接上操作者画像。fn 返回已经留下的偏好和本人说过的话，写进每个新工人的系统提示，只读。
func (rt *Runtime) SetProfile(fn func() string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.profile = fn
	if rt.exec != nil {
		rt.exec.Profile = fn
	}
}

// SetOnReport 在工人报告需要人时调用。progress 不进来。报告先入队，再按顺序写到原会话。
func (rt *Runtime) SetOnReport(fn func(ticketID, state, summary, question string)) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.report = fn
	rt.mu.Unlock()
}

// SetOnNotice 把一条不叫醒路由的通知写进桌面会话。关闭和合并用它。
func (rt *Runtime) SetOnNotice(fn func(sessionID, text string)) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.notice = fn
	rt.mu.Unlock()
}

// Start 每两秒跑一次 Pass。重复调用不会再开一条计时器。启动时先按从旧到新恢复被打断的派工。
func (rt *Runtime) Start() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	if rt.cancel != nil {
		rt.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	eng := rt.eng
	launch := eng.OnLaunch
	effects := rt.effects
	rt.mu.Unlock()
	if eng != nil && launch != nil {
		for _, id := range eng.RecoverHost() {
			launch(id)
		}
	}
	rt.recoverRejectedAsks()
	if effects != nil {
		effects.Pump(time.Now())
	}
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				rt.Pass(now)
			}
		}
	}()
}

// Stop 停掉两秒轮次，把快照最后落一次盘，再关掉数据库。没有启动过时也先落盘再关。
// 执行器直接写进引擎的报告和续接编号要等下一轮才落盘，不在这里补一次就会丢。之后工人被打断不再落盘，还在做的派工下次启动续接。
func (rt *Runtime) Stop() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	cancel := rt.cancel
	rt.cancel = nil
	rt.saveLocked()
	db := rt.db
	rt.db = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if db != nil {
		_ = db.Close()
	}
}

// Pass 先关掉已结束工单的外部工人，再执行到期副作用和到期评估，然后轮询来源，最后落盘。now 传给副作用和来源的时间比较。
func (rt *Runtime) Pass(now time.Time) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	effects := rt.effects
	eng := rt.eng
	exec := rt.exec
	rt.mu.Unlock()
	exec.Sweep()
	if effects != nil {
		effects.Pump(now)
	}
	if eng != nil {
		eng.EvaluateDue()
		if effects != nil {
			for _, id := range eng.UndeliveredEvals() {
				effects.Enqueue("assessment", id, id)
			}
		}
	}
	if effects != nil {
		effects.Pump(time.Now())
	}
	rt.mu.Lock()
	rt.health = "ready"
	if effects != nil && !effects.Healthy() {
		rt.health = "effect"
	}
	rt.at = now
	rt.mu.Unlock()
	rt.poll(now)
	rt.flushOutbox()
	rt.save()
}

// SetEvalTimeout 改评估时限。d 小于等于 0 时改回两分钟。改完立刻落盘。
func (rt *Runtime) SetEvalTimeout(d time.Duration) {
	if rt == nil || rt.eng == nil {
		return
	}
	if d <= 0 {
		d = 120 * time.Second
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.eng.mu.Lock()
	rt.eng.policy.EvalTimeout = d
	rt.eng.mu.Unlock()
	rt.saveLocked()
}

// Decide 评估 triggerID。kind 和 reason 写进决定。成功后立刻泵一次派工并落盘。
func (rt *Runtime) Decide(triggerID, kind, reason string) error {
	return rt.decide(triggerID, kind, reason, nil, false)
}

// DecideBrief 接受 triggerID，并把 brief 写成工人看到的任务说明。人的原话不进工人正文。
func (rt *Runtime) DecideBrief(triggerID, kind, reason, brief string, rules []string) error {
	return rt.DecideBriefFor(triggerID, kind, reason, brief, rules, "", false)
}

// DecideBriefFor 和 DecideBrief 一样，接受时把工单交给 agent；pinned 表示人点名了它。
func (rt *Runtime) DecideBriefFor(triggerID, kind, reason, brief string, rules []string, agent string, pinned bool) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	_, err := rt.eng.DecideBriefFor(triggerID, kind, reason, brief, rules, agent, pinned)
	if err != nil {
		rt.mu.Unlock()
		return err
	}
	rt.saveLocked()
	effects := rt.effects
	rt.mu.Unlock()
	if effects != nil {
		effects.Pump(time.Now())
	}
	return nil
}

// Reassign 把 ticketID 改派给 agent。agent 为空或 yoyo 时交回 yoyo 自带工人。
// triggerID 是人这一句，可以为空；brief 是给新工人的说明；pinned 表示人点名了 agent。
func (rt *Runtime) Reassign(ticketID, triggerID, agent, brief string, pinned bool) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	if err := rt.reassignLocked(ticketID, triggerID, agent, brief, pinned); err != nil {
		rt.mu.Unlock()
		return err
	}
	rt.saveLocked()
	effects := rt.effects
	rt.mu.Unlock()
	if effects != nil {
		effects.Pump(time.Now())
	}
	return nil
}

// reassignLocked 先改派再打断旧工人。反过来时，旧工人这一轮停下后还会被催着做下去。调用时要已持有 rt.mu。
func (rt *Runtime) reassignLocked(ticketID, triggerID, agent, brief string, pinned bool) error {
	name := agentKey(agent)
	if rt.exec != nil {
		name = rt.exec.agentName(agent)
	}
	if err := rt.eng.Reassign(ticketID, Handover{Agent: agent, Name: name, Trigger: triggerID, Brief: brief, Pinned: pinned}); err != nil {
		return err
	}
	if rt.exec != nil {
		rt.exec.Interrupt(ticketID)
	}
	return nil
}

// DecideMatched 接受一条推断。rules 必须是还启用的规则编号。
func (rt *Runtime) DecideMatched(triggerID, kind, reason string, rules []string) error {
	return rt.decide(triggerID, kind, reason, rules, false)
}

// Confirm 记下人在面板上的决定。接受不会被观察模式改写成本来会接。
func (rt *Runtime) Confirm(triggerID, kind, reason string) error {
	return rt.decide(triggerID, kind, reason, nil, true)
}

// decide 写决定、落盘，再泵派工。confirmed 为真时走面板确认。
func (rt *Runtime) decide(triggerID, kind, reason string, rules []string, confirmed bool) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	var err error
	if confirmed {
		_, err = rt.eng.Confirm(triggerID, kind, reason)
	} else if len(rules) > 0 {
		_, err = rt.eng.DecideMatched(triggerID, kind, reason, rules)
	} else {
		_, err = rt.eng.Decide(triggerID, kind, reason)
	}
	if err != nil {
		rt.mu.Unlock()
		return err
	}
	rt.saveLocked()
	effects := rt.effects
	rt.mu.Unlock()
	if effects != nil {
		effects.Pump(time.Now())
	}
	return nil
}

// SetMode 换成 mode。mode 只能是 off、observe 或 auto。
func (rt *Runtime) SetMode(mode string) error {
	switch mode {
	case "off", "observe", "auto":
	default:
		return errCode("bad_mode")
	}
	return rt.updatePolicy(func(p Policy) Policy {
		p.Mode = mode
		return p
	})
}

// SetPaused 设置暂停。paused 为真时两秒轮次不接受触发，也不恢复工单。
func (rt *Runtime) SetPaused(paused bool) error {
	return rt.updatePolicy(func(p Policy) Policy {
		p.Paused = paused
		return p
	})
}

// SetAdmission 写下三格准入。listedUnaddressed、unlistedAddressed、unlistedUnaddressed 只能是空、explicit、inferred 或 ignore。
func (rt *Runtime) SetAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed string) error {
	for _, cell := range []string{listedUnaddressed, unlistedAddressed, unlistedUnaddressed} {
		switch cell {
		case "", "explicit", "inferred", "ignore":
		default:
			return errCode("bad_admission")
		}
	}
	return rt.updatePolicy(func(p Policy) Policy {
		p.Admission = Admission{
			ListedUnaddressed:   listedUnaddressed,
			UnlistedAddressed:   unlistedAddressed,
			UnlistedUnaddressed: unlistedUnaddressed,
		}
		return p
	})
}

// SetSources 换成 cfg。测试用它指向假的接口，不读环境变量。
func (rt *Runtime) SetSources(cfg SourceConfig) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.sources = cfg
	rt.sourcesSaved = true
	rt.lastGitHub = time.Time{}
	rt.lastFeishu = time.Time{}
	rt.mu.Unlock()
}

// Deliver 把还没交给编排台的评估标成已交付。有新交付时落盘。返回新交付的轮数。
func (rt *Runtime) Deliver() int {
	if rt == nil || rt.eng == nil {
		return 0
	}
	n := rt.eng.DeliverOpen()
	if n > 0 {
		rt.save()
	}
	return n
}

// Act 执行编排台动作。kind 取 attach、assign、reply、note、say、handback、reevaluate、dismiss、unsubscribe、stop、retry、reassign、requestStatus、markUnrelated、follow 或 timeout。
// ticketID、triggerID、text 和 reason 按动作使用。reassign 的 text 是新 agent，人在任务页选的，算点名。成功后立刻泵一次派工并落盘。
func (rt *Runtime) Act(kind, ticketID, triggerID, text, reason string) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	var err error
	relayStop := ""
	switch kind {
	case "attach":
		err = rt.eng.Attach(ticketID, triggerID, text)
	case "assign":
		err = rt.eng.Assign(ticketID, triggerID, text)
	case "reply":
		_, err = rt.eng.Reply(ticketID, reason, triggerID, text)
	case "note":
		err = rt.eng.Note(ticketID, text)
	case "say":
		err = rt.eng.Say(ticketID, text)
	case "handback":
		_, err = rt.eng.Handback(ticketID, triggerID, reason)
	case "reevaluate":
		_, err = rt.eng.Reevaluate(triggerID)
	case "dismiss":
		err = rt.eng.Dismiss(triggerID)
	case "unsubscribe":
		rt.eng.Unsubscribe(text)
	case "stop":
		err = rt.eng.Stop(ticketID, reason)
		if err == nil {
			relayStop = strings.TrimSpace(reason)
			if relayStop == "" {
				relayStop = "已停下"
			}
			if rt.exec != nil {
				rt.exec.Interrupt(ticketID)
			}
		}
	case "pause":
		if rt.exec == nil || !rt.exec.Pause(ticketID) {
			err = errCode("notRunning")
		}
	case "accept":
		err = rt.eng.Accept(ticketID, reason)
	case "retry":
		err = rt.eng.Retry(ticketID)
	case "reassign":
		err = rt.reassignLocked(ticketID, "", text, "", true)
	case "requestStatus":
		_, err = rt.eng.RequestStatus(ticketID)
	case "markUnrelated":
		err = rt.eng.MarkUnrelated(ticketID, text, reason)
	case "follow":
		_, err = rt.eng.FollowForContext(ticketID, text)
	case "timeout":
		sec, conv := strconv.Atoi(strings.TrimSpace(text))
		d := time.Duration(sec) * time.Second
		if conv != nil || d <= 0 {
			d = 120 * time.Second
		}
		rt.eng.mu.Lock()
		rt.eng.policy.EvalTimeout = d
		rt.eng.mu.Unlock()
	case "allowed":
		err = rt.applyAllowed(text)
	case "workers":
		rt.applyWorkers(text)
	case "sources":
		err = rt.applySources(text)
	case "rules":
		err = rt.applyRulesLocked(text)
	case "prepare":
		rt.prepareCmd = strings.TrimSpace(text)
		if rt.exec != nil {
			rt.exec.PrepareCmd = rt.prepareCmd
		}
	default:
		err = errCode("bad_action")
	}
	if err != nil {
		rt.mu.Unlock()
		return err
	}
	rt.saveLocked()
	effects := rt.effects
	rt.mu.Unlock()
	if relayStop != "" {
		rt.queueReport(ticketID, "stopped", relayStop, "")
	}
	if effects != nil {
		effects.Pump(time.Now())
	}
	return nil
}

// View 返回当前快照。引擎还没有记录时，模式和准入仍来自策略。
func (rt *Runtime) View() View {
	if rt == nil || rt.eng == nil {
		return View{Health: "unavailable"}
	}
	rt.mu.Lock()
	sources := rt.sourceList()
	health := rt.health
	at := ""
	if !rt.at.IsZero() {
		at = rt.at.UTC().Format(time.RFC3339)
	}
	workers := rt.workers
	prepare := rt.prepareCmd
	exec := rt.exec
	rt.mu.Unlock()
	snap, err := rt.eng.Export()
	if err != nil {
		return View{Health: err.Error(), Sources: sources, At: at, Tickets: []TicketRow{}, Threads: []ThreadRow{}, Queued: []TriggerRow{}, Evals: []EvalRow{}, Decisions: []DecisionRow{}, Executions: []ExecutionRow{}, Problems: []ProblemRow{}, Allowed: []string{}, Agents: []AgentLoad{}}
	}
	view := buildView(snap, sources, health, at, workers)
	view.Prepare = prepare
	view.Agents = agentLoads(view.Tickets, exec.AgentRunning())
	if rt.effects != nil {
		for _, row := range rt.effects.Failing() {
			view.Problems = append(view.Problems, ProblemRow{
				Kind: "effect", ID: row.Kind + ":" + row.Key, Title: row.Kind, Detail: row.Payload,
			})
		}
	}
	return view
}

// updatePolicy 用 edit 改策略并落盘。edit 收到当前策略，返回要写回的策略。
func (rt *Runtime) updatePolicy(edit func(Policy) Policy) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	snap, err := rt.eng.Export()
	if err != nil {
		return err
	}
	from := snap.Policy.Mode
	next := edit(snap.Policy)
	rt.eng.SetPolicy(next)
	if from != next.Mode {
		rt.eng.PolicyChanged(from, next.Mode)
	}
	rt.saveLocked()
	return nil
}

// sourceList 按 github、feishu 的顺序返回状态。还没轮询过的来源是 unconfigured。
func (rt *Runtime) sourceList() []SourceState {
	out := make([]SourceState, 0, 2)
	for _, name := range []string{"github", "feishu"} {
		st, ok := rt.status[name]
		if !ok {
			st = SourceState{Name: name, State: "unconfigured", Detail: "还没轮询"}
		}
		st.Name = name
		if name == "github" {
			st.Repo = rt.sources.GitHubRepo
			st.API = rt.sources.GitHubAPI
			st.Account = rt.sources.GitHubAccount
			st.HasSecret = strings.TrimSpace(rt.sources.GitHubToken) != ""
		}
		if name == "feishu" {
			st.Chat = rt.sources.FeishuChat
			st.App = rt.sources.FeishuAppID
			st.API = rt.sources.FeishuAPI
			st.Account = rt.sources.FeishuMention
			st.HasSecret = strings.TrimSpace(rt.sources.FeishuSecret) != ""
		}
		out = append(out, st)
	}
	return out
}

// applyAllowed 用 text 里逗号分开的名字重写允许发送者。调用时要已持有锁。
func (rt *Runtime) applyAllowed(text string) error {
	snap, err := rt.eng.Export()
	if err != nil {
		return err
	}
	next := snap.Policy
	allowed := map[string]bool{}
	for _, part := range strings.Split(text, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			allowed[name] = true
		}
	}
	next.Allowed = allowed
	rt.eng.SetPolicy(next)
	return nil
}

// applyRulesLocked 用 text 里的 JSON 数组重写主动规则。调用时要已持有锁。编号和正文都空的行丢掉。
func (rt *Runtime) applyRulesLocked(text string) error {
	var rows []Rule
	if strings.TrimSpace(text) != "" && json.Unmarshal([]byte(text), &rows) != nil {
		return errCode("bad_rules")
	}
	var kept []Rule
	seen := map[string]bool{}
	for _, row := range rows {
		row.ID = strings.TrimSpace(row.ID)
		row.Text = strings.TrimSpace(row.Text)
		if row.ID == "" || row.Text == "" || seen[row.ID] || row.ID == "addressed" || row.ID == "person" {
			continue
		}
		seen[row.ID] = true
		kept = append(kept, row)
	}
	snap, err := rt.eng.Export()
	if err != nil {
		return err
	}
	next := snap.Policy
	next.Rules = kept
	rt.eng.SetPolicy(next)
	return nil
}

// applyWorkers 把普通工人槽改成 text 里的数字。调用时要已持有锁。小于 1 当作 1，大于 32 当作 32。
func (rt *Runtime) applyWorkers(text string) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || n < 1 {
		n = 1
	}
	if n > 32 {
		n = 32
	}
	rt.workers = n
	if rt.exec != nil && rt.exec.Slots != nil {
		rt.exec.Slots.SetMax(n)
	}
}

// sourceForm 是编排台提交的来源表。密钥留空表示保持原值，Clear 为真才清掉。
// 点名用指针：没带这个字段就保持原值，带了空串才清掉。
type sourceForm struct {
	GitHubToken       string  `json:"githubToken"`
	GitHubRepo        string  `json:"githubRepo"`
	GitHubAPI         string  `json:"githubApi"`
	GitHubAccount     *string `json:"githubAccount"`
	FeishuAppID       string  `json:"feishuAppId"`
	FeishuSecret      string  `json:"feishuSecret"`
	FeishuChat        string  `json:"feishuChat"`
	FeishuAPI         string  `json:"feishuApi"`
	FeishuMention     *string `json:"feishuMention"`
	ClearGitHubToken  bool    `json:"clearGithubToken"`
	ClearFeishuSecret bool    `json:"clearFeishuSecret"`
}

// applySources 按 form 改来源。仓库、应用号和会话空串会清掉。调用时要已持有锁。
func (rt *Runtime) applySources(text string) error {
	var form sourceForm
	if err := json.Unmarshal([]byte(text), &form); err != nil {
		return err
	}
	next := rt.sources
	next.GitHubRepo = strings.TrimSpace(form.GitHubRepo)
	next.FeishuAppID = strings.TrimSpace(form.FeishuAppID)
	next.FeishuChat = strings.TrimSpace(form.FeishuChat)
	if form.GitHubAccount != nil {
		next.GitHubAccount = strings.TrimPrefix(strings.TrimSpace(*form.GitHubAccount), "@")
	}
	if form.FeishuMention != nil {
		next.FeishuMention = strings.TrimPrefix(strings.TrimSpace(*form.FeishuMention), "@")
	}
	if api := strings.TrimSpace(form.GitHubAPI); api != "" {
		next.GitHubAPI = api
	}
	if api := strings.TrimSpace(form.FeishuAPI); api != "" {
		next.FeishuAPI = api
	}
	if form.ClearGitHubToken {
		next.GitHubToken = ""
	} else if token := strings.TrimSpace(form.GitHubToken); token != "" {
		next.GitHubToken = token
	}
	if form.ClearFeishuSecret {
		next.FeishuSecret = ""
	} else if secret := strings.TrimSpace(form.FeishuSecret); secret != "" {
		next.FeishuSecret = secret
	}
	rt.sources = next
	rt.sourcesSaved = true
	rt.lastGitHub = time.Time{}
	rt.lastFeishu = time.Time{}
	rt.refreshEnvLocked()
	return nil
}

// load 读出快照并还原。没有快照行时保持 Open 时的策略。
func (rt *Runtime) load() error {
	if rt.db == nil {
		return nil
	}
	var body string
	err := rt.db.QueryRow(`SELECT body FROM snapshot WHERE id = 1`).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var stored diskBody
	if err := json.Unmarshal([]byte(body), &stored); err != nil {
		return err
	}
	rt.eng.Import(stored.Snapshot)
	if stored.Sources != nil {
		rt.sources = *stored.Sources
		rt.sourcesSaved = true
	}
	if stored.Workers > 0 {
		rt.workers = stored.Workers
	}
	rt.prepareCmd = strings.TrimSpace(stored.Prepare)
	if len(stored.Cursors) > 0 {
		rt.cursors = stored.Cursors
	}
	if rt.effects != nil {
		rt.effects.Restore(stored.Effects)
	}
	return nil
}

// diskBody 是落盘的快照。Sources 为空指针表示仍用环境变量。Workers 为 0 表示还没改过槽位数。
type diskBody struct {
	Snapshot
	Sources *SourceConfig     `json:"sources,omitempty"`
	Workers int               `json:"workers,omitempty"`
	Prepare string            `json:"prepare,omitempty"`
	Effects []savedEffect     `json:"effects,omitempty"`
	Cursors map[string]string `json:"cursors,omitempty"`
}

// save 在锁内把快照写入数据库。
func (rt *Runtime) save() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.saveLocked()
}

// saveLocked 把快照写入数据库。调用时要已持有锁。没有数据库时什么都不做。
func (rt *Runtime) saveLocked() {
	if rt.db == nil || rt.eng == nil {
		return
	}
	snap, err := rt.eng.Export()
	if err != nil {
		rt.health = err.Error()
		return
	}
	stored := diskBody{Snapshot: snap, Workers: rt.workers, Prepare: rt.prepareCmd, Cursors: rt.cursors}
	if rt.effects != nil {
		stored.Effects = rt.effects.ExportPending()
	}
	if rt.sourcesSaved {
		cfg := rt.sources
		stored.Sources = &cfg
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		rt.health = err.Error()
		return
	}
	_, err = rt.db.Exec(`INSERT INTO snapshot(id, body) VALUES(1, ?) ON CONFLICT(id) DO UPDATE SET body = excluded.body`, string(raw))
	if err != nil {
		rt.health = err.Error()
	}
}
