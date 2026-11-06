// engine.go 是领域写入的唯一入口。它不跑模型，也不碰进程。
// 执行器在提交之后通过 OnLaunch 和 OnInput 接手。

package superagent

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Engine 保存触发、线程、决定、工单和发件箱。同一时刻只有它改这些记录。
type Engine struct {
	mu        sync.Mutex
	policy    Policy
	sources   map[string]bool
	seq       int
	n         int
	triggers  map[string]*Trigger
	threads   map[string]*Thread
	tickets   map[string]*Ticket
	decisions []Decision
	outbox    []Outbound

	// OnLaunch 在新派工可以启动时调用。ticketID 是工单。暂停时不调用。
	OnLaunch func(ticketID string)
	// OnInput 在活人的工单收到一条消息时调用。ticketID 是工单，text 是消息正文。
	OnInput func(ticketID, text string)
	clock   func() time.Time
}

// now 返回引擎时钟。clock 为空时用 time.Now。
func (e *Engine) now() time.Time {
	if e != nil && e.clock != nil {
		return e.clock()
	}
	return time.Now()
}

// NewEngine 返回使用 p 的引擎。p.Mode 为空时当作 observe。p.Version 小于 1 时当作 1。
func NewEngine(p Policy) *Engine {
	if p.Mode == "" {
		p.Mode = "observe"
	}
	if p.Version < 1 {
		p.Version = 1
	}
	return &Engine{
		policy:   p,
		triggers: map[string]*Trigger{},
		threads:  map[string]*Thread{},
		tickets:  map[string]*Ticket{},
	}
}

// SetPolicy 换成 p。已经发出的授权不改，恢复时用 Revalidate 再看。
func (e *Engine) SetPolicy(p Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = p
}

// Listen 记下 source 是否在听。on 为假时这个来源之后的触发记为 filtered。
// 从没调用过时，所有来源都在听。
func (e *Engine) Listen(source string, on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sources == nil {
		e.sources = map[string]bool{}
	}
	e.sources[source] = on
}

// Receive 收下 in。相同 ID 返回 duplicate。空正文、没有线程或不在听的来源返回 filtered。其余返回 stored。
// 人的消息若已经落到还开着的订阅工单上，就不再排队评估。
func (e *Engine) Receive(in Trigger) string {
	e.mu.Lock()
	status, launches, inputs := e.receive(in)
	text := in.Text
	paused := e.policy.Paused
	e.mu.Unlock()
	if !paused && e.OnLaunch != nil {
		for _, id := range unique(launches) {
			e.OnLaunch(id)
		}
	}
	if e.OnInput != nil {
		for _, id := range unique(inputs) {
			e.OnInput(id, text)
		}
	}
	return status
}

// Decide 评估 triggerID。kind 取 accept、stop、ignore、unavailable。reason 写进决定。
// 推断候选在 Mode 不是 auto 时记为 ignore，原因是 observed，不开工单。
// 人的后续消息只有 kind 为 stop 时能改工单，否则返回 workerDecides。
func (e *Engine) Decide(triggerID, kind, reason string) (Decision, error) {
	return e.decideOuter(triggerID, kind, reason, nil, false)
}

// DecideMatched 和 Decide 一样。rules 是这条推断命中的规则编号，必须都还启用。
func (e *Engine) DecideMatched(triggerID, kind, reason string, rules []string) (Decision, error) {
	return e.decideOuter(triggerID, kind, reason, rules, false)
}

// Confirm 是人在面板上作出的决定。推断候选不会被观察模式改写成「本来会接」。
func (e *Engine) Confirm(triggerID, kind, reason string) (Decision, error) {
	return e.decideOuter(triggerID, kind, reason, nil, true)
}

// DecideBrief 和 Decide 一样。brief 非空时，工人看到的是这份说明，不是人的原话。
// 原话仍留在触发上，任务列表的短标题不变。rules 是推断接受时还启用的规则编号。
func (e *Engine) DecideBrief(triggerID, kind, reason, brief string, rules []string) (Decision, error) {
	e.mu.Lock()
	d, launch, err := e.decide(triggerID, kind, reason, rules, false, strings.TrimSpace(brief))
	e.mu.Unlock()
	if err == nil && launch != "" && e.OnLaunch != nil {
		e.OnLaunch(launch)
	}
	return d, err
}

// decideOuter 在锁外启动派工。confirmed 为真表示人已经点了接受。
func (e *Engine) decideOuter(triggerID, kind, reason string, rules []string, confirmed bool) (Decision, error) {
	e.mu.Lock()
	d, launch, err := e.decide(triggerID, kind, reason, rules, confirmed, "")
	e.mu.Unlock()
	if err == nil && launch != "" && e.OnLaunch != nil {
		e.OnLaunch(launch)
	}
	return d, err
}

// Report 把 report 记到 ticketID 正在跑的派工上。没有正在跑的就记到最后一次还没结束的派工。
// ready 且没有未完成的跟随线程时结束工单。dropped 结束工单。progress 不结束。
func (e *Engine) Report(ticketID string, report Report) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	if report.Question != "" && report.State == "ready" {
		return errCode("questionWithResult")
	}
	switch report.State {
	case "progress", "waiting", "ready", "blocked", "failed", "dropped":
	default:
		return errCode("bad_report")
	}
	asg := runningAssignment(t)
	if asg == nil {
		asg = latestAssignment(t)
	}
	if asg == nil {
		return errCode("unknownAssignment")
	}
	if asg.State == "ended" {
		return errCode("assignmentEnded")
	}
	e.seq++
	report.Seq = e.seq
	if report.At.IsZero() {
		report.At = e.now()
	}
	asg.Reports = append(asg.Reports, report)
	if report.State == "ready" {
		t.Attention = ""
	} else if report.Question != "" {
		t.Attention = report.Question
	}
	e.applyReportEnd(t, report)
	return nil
}

// StagedProgress 把 ticketID 上给人看的阶段性进展收成一段。空等待和重复的句子去掉。
func (e *Engine) StagedProgress(ticketID string) string {
	if e == nil {
		return ""
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return ""
	}
	var lines []string
	seen := map[string]bool{}
	for _, asg := range t.Assignments {
		for _, report := range asg.Reports {
			if report.State != "progress" {
				continue
			}
			text := strings.TrimSpace(report.Summary)
			if q := strings.TrimSpace(report.Question); q != "" {
				if text != "" {
					text += "\n"
				}
				text += q
			}
			text = clipRunes(text, 180)
			if text == "" || seen[text] {
				continue
			}
			seen[text] = true
			lines = append(lines, text)
		}
	}
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	return strings.Join(lines, "\n")
}

// FailStart 在准备目录失败时结束 ticketID。outcome 是 failed。已经结束时返回 ticketEnded。
func (e *Engine) FailStart(ticketID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	e.finish(t, "executor", "failed", reason)
	return nil
}

// SetDirectory 记下 ticketID 的目录和短名。已经记过目录时不改路径，重试沿用它。
func (e *Engine) SetDirectory(ticketID, dir, name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return
	}
	if t.Directory == "" {
		t.Directory = dir
	}
	if t.Name == "" && name != "" {
		t.Name = name
	}
}

// DeskAudiences 返回这份报告该写进的桌面会话。先用后来跟上的会话。没有时才用开工那一条。
func (e *Engine) DeskAudiences(ticketID string) []string {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, f := range t.Follows {
		if sid, ok := strings.CutPrefix(f.Thread, "desk:"); ok {
			add(sid)
		}
	}
	if len(out) == 0 {
		add(e.deskSessionLocked(t))
	}
	return out
}

// DeskSession 返回开工这张工单的桌面会话。来源不是桌面时返回空串。
func (e *Engine) DeskSession(ticketID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.deskSessionLocked(e.tickets[ticketID])
}

// deskSessionLocked 返回 t 的桌面会话。调用时要已持有锁。
func (e *Engine) deskSessionLocked(t *Ticket) string {
	if t == nil {
		return ""
	}
	tr := e.triggers[t.Origin]
	if tr == nil {
		return ""
	}
	for _, key := range tr.Threads {
		if strings.HasPrefix(key, "desk:") {
			return strings.TrimPrefix(key, "desk:")
		}
	}
	return ""
}

// DeskSessionsFollowing 返回跟随 thread 的工单所在的桌面会话。
func (e *Engine) DeskSessionsFollowing(thread string) []string {
	if e == nil || strings.TrimSpace(thread) == "" {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	seen := map[string]bool{}
	for _, t := range e.tickets {
		if t == nil {
			continue
		}
		hit := false
		for _, f := range t.Follows {
			if f.Thread == thread {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		sid := e.deskSessionLocked(t)
		if sid == "" || seen[sid] {
			continue
		}
		seen[sid] = true
		out = append(out, sid)
	}
	sort.Strings(out)
	return out
}

// PendingInputs 返回 ticketID 上还没投递给工人的正文，以及对应的输入编号。
func (e *Engine) PendingInputs(ticketID string) (texts []string, ids []string) {
	if e == nil {
		return nil, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return nil, nil
	}
	pending := map[string]bool{}
	for _, id := range t.Undelivered {
		pending[id] = true
	}
	for _, in := range t.Inputs {
		if pending[in.ID] && strings.TrimSpace(in.Text) != "" {
			texts = append(texts, in.Text)
			ids = append(ids, in.ID)
		}
	}
	return texts, ids
}

// AckInputs 把 ids 从 ticketID 的未投递列表里拿掉。ids 为空时什么都不做。
func (e *Engine) AckInputs(ticketID string, ids []string) {
	if e == nil || len(ids) == 0 {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return
	}
	drop := map[string]bool{}
	for _, id := range ids {
		drop[id] = true
	}
	var kept []string
	for _, id := range t.Undelivered {
		if !drop[id] {
			kept = append(kept, id)
		}
	}
	t.Undelivered = kept
}

// UndeliveredEvals 返回还没交给评估者的评估编号。
func (e *Engine) UndeliveredEvals() []string {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, th := range e.threads {
		if th == nil || th.Evaluating == nil || !th.Evaluating.Delivered.IsZero() {
			continue
		}
		out = append(out, th.Evaluating.ID)
	}
	sort.Strings(out)
	return out
}

// DeliverEval 把 id 这一轮评估标成已交付，并从此刻起算截止时间。已经交付时返回假。
func (e *Engine) DeliverEval(id string) bool {
	if e == nil || id == "" {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	now := e.now()
	for _, th := range e.threads {
		if th == nil || th.Evaluating == nil || th.Evaluating.ID != id || !th.Evaluating.Delivered.IsZero() {
			continue
		}
		th.Evaluating.Delivered = now
		th.Evaluating.Deadline = now.Add(e.policy.evalTimeout())
		return true
	}
	return false
}

// ClaimBranch 让短名等于 short 的开着的工单跟随 thread，并记成工单自己打开的。
func (e *Engine) ClaimBranch(thread, short string) {
	if e == nil {
		return
	}
	thread = strings.TrimSpace(thread)
	short = strings.TrimSpace(short)
	if thread == "" || short == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tickets {
		if t == nil || t.Ended != nil || t.Name != short {
			continue
		}
		e.subscribe(t.ID, thread)
		e.follow(t, thread, true)
	}
}

// branchName 把标题收成分支短名。只留小写字母、数字和连字符。
func branchName(text string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !ok {
			r = '-'
		}
		if r == '-' {
			if prevDash || b.Len() == 0 {
				continue
			}
			prevDash = true
		} else {
			prevDash = false
		}
		b.WriteRune(r)
		if b.Len() >= 40 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

// pollHorizon 是工单结束后仍继续读它订阅的线程的时间。与原型两周窗口一致。
const pollHorizon = 14 * 24 * time.Hour

// PollableThreads 返回还要读的线程。活着的订阅一直读。结束后，从最近一条消息或结束时刻起两周内仍读。
func (e *Engine) PollableThreads(now time.Time) []string {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for key, th := range e.threads {
		if th == nil || len(th.Subs) == 0 || !e.pollableLocked(th, now) {
			continue
		}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// pollableLocked 判断 th 是否还在读取窗口里。调用时要已持有锁。
func (e *Engine) pollableLocked(th *Thread, now time.Time) bool {
	latest := time.Time{}
	for _, id := range th.Subs {
		ticket := e.tickets[id]
		if ticket == nil {
			continue
		}
		if ticket.Ended == nil {
			return true
		}
		if ticket.Ended.At.After(latest) {
			latest = ticket.Ended.At
		}
	}
	for _, id := range th.Context {
		tr := e.triggers[id]
		if tr != nil && tr.At.After(latest) {
			latest = tr.At
		}
	}
	if latest.IsZero() {
		return false
	}
	return now.Sub(latest) < pollHorizon
}

// FollowThreads 返回还开着的工单正在跟随的线程。
func (e *Engine) FollowThreads() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	seen := map[string]bool{}
	for _, t := range e.tickets {
		if t == nil || t.Ended != nil {
			continue
		}
		for _, f := range t.Follows {
			if f.Thread == "" || f.Closed != nil || seen[f.Thread] {
				continue
			}
			seen[f.Thread] = true
			out = append(out, f.Thread)
		}
	}
	sort.Strings(out)
	return out
}

// ReopenForInput 在工人已经不在、但还有没送到的话时，结束当前派工并再排一轮。已经在排队时只再通知启动。
func (e *Engine) ReopenForInput(ticketID string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	t := e.tickets[ticketID]
	launch := false
	if t != nil && t.Ended == nil && len(t.Undelivered) > 0 {
		asg := latestAssignment(t)
		if asg == nil || asg.State == "ended" || asg.State == "running" {
			if asg != nil && asg.State == "running" {
				asg.State = "ended"
			}
			e.addAssignment(t)
		}
		launch = true
	}
	fn := e.OnLaunch
	e.mu.Unlock()
	if launch && fn != nil {
		fn(ticketID)
	}
	return launch
}

// TakeResume 取走 ticketID 上的续跑标记。没有时返回假。
func (e *Engine) TakeResume(ticketID string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil || !t.Resume {
		return false
	}
	t.Resume = false
	return true
}

// hostRestartNote 是旧版本重启时写进工单的那句。新的重启不再写它，展示时也把它滤掉。
const hostRestartNote = "宿主重启了。先看工作目录、git 和 GitHub，不要重复已经做过的外部动作。"

// RecoverHost 结束仍标成 running 的派工，并按工单编号从旧到新返回要重开的工单。
// 已有的会话日志会原样接上，不再追加一条新的任务说明。waiting、ready、blocked 和永久性的 failed 保持原样。连接中断这种暂时失败会重开。pending 的派工也包含在返回值里。
func (e *Engine) RecoverHost() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	ids := make([]string, 0, len(e.tickets))
	for id := range e.tickets {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ticketOrder(ids[i]) < ticketOrder(ids[j]) })
	var out []string
	for _, id := range ids {
		t := e.tickets[id]
		if t == nil || t.Ended != nil {
			continue
		}
		asg := latestAssignment(t)
		if asg == nil {
			continue
		}
		// 快照里可能还有只跟着上下文线程、却停在等验收的工单。按现在的规则，它们在报做完时就该结束。
		if n := len(asg.Reports); n > 0 && asg.Reports[n-1].State == "ready" && !ownsThread(t) && !inputAfter(t, asg.Reports[n-1].Seq) {
			e.finish(t, "executor", "completed", asg.Reports[n-1].Summary)
			continue
		}
		if asg.State == "running" {
			if settled(asg) && !inputAfter(t, asg.Reports[len(asg.Reports)-1].Seq) && !transientReport(asg) {
				continue
			}
			asg.State = "ended"
			e.addAssignment(t)
			t.Resume = true
			out = append(out, id)
			continue
		}
		if asg.State == "pending" {
			out = append(out, id)
		}
	}
	return out
}

// transientReport 在最近一次报告是暂时的网络失败时为真。这种派工重启后续上，不留在原地。
func transientReport(asg *Assignment) bool {
	if asg == nil || len(asg.Reports) == 0 {
		return false
	}
	last := asg.Reports[len(asg.Reports)-1]
	return last.State == "failed" && passingError(last.Summary)
}

// settled 在最近一次报告已经是等待、完成、受阻或失败时为真。
func settled(asg *Assignment) bool {
	if asg == nil || len(asg.Reports) == 0 {
		return false
	}
	switch asg.Reports[len(asg.Reports)-1].State {
	case "waiting", "ready", "blocked", "failed", "dropped":
		return true
	default:
		return false
	}
}

// inputAfter 在报告之后还有新输入时为真。那种情况要重新开工。
func inputAfter(t *Ticket, seq int) bool {
	if t == nil {
		return false
	}
	for _, in := range t.Inputs {
		if in.Seq > seq {
			return true
		}
	}
	return false
}

// ticketOrder 取出 tkt- 后面的序号。认不出时放到后面。
func ticketOrder(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "tkt-"))
	if err != nil {
		return 1 << 30
	}
	return n
}

// Stop 由人结束 ticketID。reason 记在结束记录上。已经结束时返回 ticketEnded。
func (e *Engine) Stop(ticketID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stop(ticketID, reason)
}

// Accept 由人验收 ticketID：工人报了做完、工单在等人看结果时，把它记成做完了。不在等验收时返回 notReview。
// 跟随的线程还没关也一样结束，人看过就算数。reason 为空时记成验收通过。
func (e *Engine) Accept(ticketID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	if TicketStage(t).Action != "review" {
		return errCode("notReview")
	}
	if strings.TrimSpace(reason) == "" {
		reason = "验收通过"
	}
	e.finish(t, "human", "completed", reason)
	return nil
}

// Retry 结束 ticketID 上卡住的派工并新开一次。ticketID 不在可重试阶段时返回 notRetryable。
// 没暂停时调用 OnLaunch。
func (e *Engine) Retry(ticketID string) error {
	e.mu.Lock()
	t := e.tickets[ticketID]
	if t == nil {
		e.mu.Unlock()
		return errCode("unknownTicket")
	}
	if !CanRetry(t) {
		e.mu.Unlock()
		return errCode("notRetryable")
	}
	if asg := latestAssignment(t); asg != nil && asg.State != "ended" {
		asg.State = "ended"
	}
	e.addAssignment(t)
	paused := e.policy.Paused
	e.mu.Unlock()
	if !paused && e.OnLaunch != nil {
		e.OnLaunch(ticketID)
	}
	return nil
}

// BindThread 让 ticketID 跟随 thread。own 为真表示工单自己打开的。再次绑定时清掉关闭记录，own 只会变成真。
func (e *Engine) BindThread(ticketID, thread string, own bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	if thread == "" {
		return errCode("empty")
	}
	e.subscribe(ticketID, thread)
	for i := range t.Follows {
		if t.Follows[i].Thread != thread {
			continue
		}
		if own {
			t.Follows[i].Own = true
		}
		t.Follows[i].Closed = nil
		return nil
	}
	t.Follows = append(t.Follows, Follow{Thread: thread, Own: own})
	return nil
}

// ThreadClosed 记下 thread 已关闭。outcome 为 merged 或 closed。at 用来比较先后。
// 跟随都完成且最近报告是 ready 时，工单以这次关闭的 outcome 结束。
func (e *Engine) ThreadClosed(thread, outcome, at string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, t := range e.tickets {
		if t.Ended != nil {
			continue
		}
		for i := range t.Follows {
			f := &t.Follows[i]
			if f.Thread != thread || f.Closed != nil {
				continue
			}
			f.Closed = &Close{At: at, Outcome: outcome}
			if done := followsDone(t); done != nil && reportState(t) == "ready" {
				e.finish(t, "executor", done.Outcome, "")
			}
		}
	}
}

// AssignmentStarted 把 ticketID 上 assignmentID 从 pending 改成 running。已经是 running 时保持不变。
func (e *Engine) AssignmentStarted(ticketID, assignmentID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	for i := range t.Assignments {
		if t.Assignments[i].ID != assignmentID {
			continue
		}
		if t.Assignments[i].State == "ended" {
			return errCode("assignmentEnded")
		}
		t.Assignments[i].State = "running"
		return nil
	}
	return errCode("unknownAssignment")
}

// DeliveryFailed 把 message 记到 ticketID 上，阶段变为 person/delivery。message 为空时清掉失败。
func (e *Engine) DeliveryFailed(ticketID, message string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	t.DeliveryErr = message
	return nil
}

// Ticket 返回 id 的副本。没有这个工单时 ok 为假。
func (e *Engine) Ticket(id string) (Ticket, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[id]
	if t == nil {
		return Ticket{}, false
	}
	return cloneTicket(t), true
}

// RuleTexts 按 ids 的顺序返回还启用的规则正文。addressed、person 和认不出的编号跳过。
func (e *Engine) RuleTexts(ids []string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, id := range ids {
		for _, rule := range e.policy.Rules {
			if rule.ID == id && rule.Enabled && strings.TrimSpace(rule.Text) != "" {
				out = append(out, strings.TrimSpace(rule.Text))
				break
			}
		}
	}
	return out
}

// Trigger 返回 id 的副本。没有这次触发时 ok 为假。
func (e *Engine) Trigger(id string) (Trigger, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.triggers[id]
	if t == nil {
		return Trigger{}, false
	}
	return cloneTrigger(t), true
}

// Thread 返回 key 的副本。没有这条线程时 ok 为假。
func (e *Engine) Thread(key string) (Thread, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	th := e.threads[key]
	if th == nil {
		return Thread{}, false
	}
	out := *th
	out.Subs = append([]string(nil), th.Subs...)
	out.Watch = append([]string(nil), th.Watch...)
	out.Context = append([]string(nil), th.Context...)
	out.Queued = append([]string(nil), th.Queued...)
	return out, true
}

// Outbox 返回待发出消息的副本。
func (e *Engine) Outbox() []Outbound {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Outbound(nil), e.outbox...)
}

// receive 收下 in。status 为 stored、duplicate 或 filtered。launches 是恢复后要启动的工单。inputs 是还开着、要投递正文的工单。
// desk: 开头的线程盖住整段主会话。下一句是不是同一件事由路由判断，这里不自动送进工人。
func (e *Engine) receive(in Trigger) (status string, launches, inputs []string) {
	if in.ID != "" {
		if _, ok := e.triggers[in.ID]; ok {
			return "duplicate", nil, nil
		}
	}
	if e.sources != nil && !e.sources[in.Source] {
		e.storeFiltered(in, "not_listened")
		return "filtered", nil, nil
	}
	threads := append([]string(nil), in.Threads...)
	if len(threads) == 0 && in.Primary != "" {
		threads = []string{in.Primary}
	}
	if in.ID == "" || strings.TrimSpace(in.Text) == "" || len(threads) == 0 {
		e.storeFiltered(in, "empty")
		return "filtered", nil, nil
	}
	cp := in
	stamp(&cp, e.now())
	cp.Threads = threads
	cp.Reached = nil
	cp.Handbacks = nil
	cp.DecisionID = ""
	cp.Queued = false
	cp.Filtered = ""
	cp.Requeued = false
	e.triggers[cp.ID] = &cp
	tr := e.triggers[cp.ID]
	for _, key := range tr.Threads {
		th := e.ensureThread(key)
		if !contains(th.Context, tr.ID) {
			th.Context = append(th.Context, tr.ID)
		}
		if strings.HasPrefix(key, "desk:") {
			continue
		}
		for _, tid := range append([]string(nil), th.Subs...) {
			ticket := e.tickets[tid]
			if ticket == nil || contains(tr.Reached, tid) {
				continue
			}
			if ticket.Ended != nil && (ticket.Ended.By == "human" || !tr.Human || !e.policy.Revalidate(ticket.Mandate)) {
				continue
			}
			e.addInput(ticket, tr)
			tr.Reached = append(tr.Reached, tid)
			if ticket.Ended != nil {
				if e.resume(ticket) {
					launches = append(launches, ticket.ID)
				}
				continue
			}
			inputs = append(inputs, ticket.ID)
		}
		for _, tid := range append([]string(nil), th.Watch...) {
			if contains(th.Subs, tid) || contains(tr.Reached, tid) {
				continue
			}
			ticket := e.tickets[tid]
			if ticket == nil || ticket.Ended != nil {
				continue
			}
			e.addInput(ticket, tr)
			inputs = append(inputs, ticket.ID)
		}
	}
	if len(e.owners(tr)) == 0 {
		e.queue(tr)
	}
	return "stored", launches, inputs
}

// decide 评估 triggerID。kind 与 reason 写进决定。第二个返回值是要启动的工单，没有则为空。
// brief 非空时替换工人看到的第一条正文。人的原话仍留在触发上。
func (e *Engine) decide(triggerID, kind, reason string, rules []string, confirmed bool, brief string) (Decision, string, error) {
	tr := e.triggers[triggerID]
	if tr == nil || tr.Filtered != "" {
		return Decision{}, "", errCode("unknownTrigger")
	}
	if tr.DecisionID != "" {
		d, ok := e.decision(tr.DecisionID)
		if kind == "accept" && ok && d.Kind == "accept" {
			return d, "", nil
		}
		return Decision{}, "", errCode("settled")
	}
	if kind != "stop" && len(e.owners(tr)) > 0 {
		return Decision{}, "", errCode("workerDecides")
	}
	admitted, strength, why := e.policy.Gate(*tr)
	if !admitted {
		if why == "" {
			why = "not_admitted"
		}
		return Decision{}, "", errCode(why)
	}
	switch kind {
	case "accept":
		if !confirmed && strength != "explicit" && e.policy.Mode != "auto" {
			d := e.putDecision(tr, "ignore", "observed", "")
			e.unqueue(tr)
			return d, "", nil
		}
		matched := []string(nil)
		if strength != "explicit" && !confirmed {
			var ok bool
			matched, ok = e.policy.MatchRules(rules)
			if !ok {
				return Decision{}, "", errCode("needs_rule")
			}
		}
		e.n++
		ticketID := fmt.Sprintf("tkt-%d", e.n)
		d := e.putDecision(tr, "accept", reason, ticketID)
		e.openTicket(tr, strength, ticketID, reason, matched, brief)
		e.unqueue(tr)
		// 暂停只挡住推断出来的开工。人明确交代的仍然马上派工。
		if e.policy.Paused && strength != "explicit" {
			return d, "", nil
		}
		return d, ticketID, nil
	case "ignore", "unavailable":
		d := e.putDecision(tr, kind, reason, "")
		e.unqueue(tr)
		return d, "", nil
	case "stop":
		if strength != "explicit" {
			d := e.putDecision(tr, "ignore", "stopNotExplicit", "")
			e.unqueue(tr)
			return d, "", nil
		}
		owners := e.owners(tr)
		if len(owners) == 0 {
			d := e.putDecision(tr, "ignore", "stopUnknown", "")
			e.unqueue(tr)
			return d, "", nil
		}
		for _, id := range owners {
			_ = e.stop(id, reason)
		}
		d := e.putDecision(tr, "stop", reason, owners[0])
		e.unqueue(tr)
		return d, "", nil
	default:
		return Decision{}, "", errCode("bad_kind")
	}
}

// openTicket 用 tr 和 strength 开出 ticketID。name 收成分支短名，空的就先不记。
// brief 非空时，工人读到的是这份说明，不是 tr 的原话。
func (e *Engine) openTicket(tr *Trigger, strength, ticketID, name string, rules []string, brief string) {
	t := &Ticket{ID: ticketID, Origin: tr.ID, Name: branchName(name), Opened: e.now(), Mandate: Mandate{Version: e.policy.Version, Strength: strength, Sender: tr.Sender}}
	switch {
	case strength == "explicit":
		t.Mandate.Rules = []string{"addressed"}
	case len(rules) == 0:
		t.Mandate.Rules = []string{"person"}
	default:
		t.Mandate.Rules = append([]string(nil), rules...)
	}
	e.addAssignment(t)
	e.tickets[ticketID] = t
	keys := append([]string(nil), tr.Threads...)
	if tr.Primary != "" && !contains(keys, tr.Primary) {
		keys = append(keys, tr.Primary)
	}
	for _, key := range keys {
		e.subscribe(ticketID, key)
	}
	e.addInput(t, tr)
	if b := strings.TrimSpace(brief); b != "" && len(t.Inputs) > 0 {
		t.Inputs[len(t.Inputs)-1].Text = clipRunes(b, 12000)
	}
	e.n++
	e.outbox = append(e.outbox, Outbound{
		ID: fmt.Sprintf("out-%d", e.n), TicketID: ticketID, Kind: "acknowledgement", Text: "已接单", Status: "pending",
		Source: tr.Source, Ref: tr.Ref,
	})
}

// applyReportEnd 按 report 结束 t。ready 时没有自己开的线程就记 completed；自己开的线程（比如工人开的拉取请求）要等它关闭，用关闭结果。
// 只为上下文跟着的线程，比如主会话，永远不会关，不挡完成。dropped 结束工单。
func (e *Engine) applyReportEnd(t *Ticket, report Report) {
	switch report.State {
	case "dropped":
		outcome := "dropped"
		if done := followsDone(t); done != nil && done.Outcome != "" {
			outcome = done.Outcome
		}
		e.finish(t, "executor", outcome, report.Summary)
	case "ready":
		if !ownsThread(t) {
			e.finish(t, "executor", "completed", report.Summary)
			return
		}
		if done := followsDone(t); done != nil {
			e.finish(t, "executor", done.Outcome, report.Summary)
		}
	}
}

// resume 在没暂停且授权仍有效时重新打开 t。需要新派工时返回真。
// 主动轮次用它。人点的继续走 reopen，不受暂停影响。
func (e *Engine) resume(t *Ticket) bool {
	return e.reopen(t, true)
}

// reopen 重新打开 t。honorPause 为真时，暂停中的主动恢复不做。
func (e *Engine) reopen(t *Ticket, honorPause bool) bool {
	if t == nil {
		return false
	}
	if honorPause && e.policy.Paused {
		return false
	}
	if !e.policy.Revalidate(t.Mandate) {
		return false
	}
	t.Ended = nil
	asg := latestAssignment(t)
	if asg == nil || asg.State == "ended" {
		e.addAssignment(t)
		return true
	}
	return false
}

// stop 由人结束 ticketID。reason 记在结束记录上。
func (e *Engine) stop(ticketID, reason string) error {
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	e.finish(t, "human", "stopped", reason)
	return nil
}

// finish 把 t 记为结束。by 是 executor 或 human。outcome 与 reason 记在结束记录上。已经结束时不动。
func (e *Engine) finish(t *Ticket, by, outcome, reason string) {
	if t.Ended != nil {
		return
	}
	e.seq++
	t.Ended = &End{By: by, Outcome: outcome, Reason: reason, Seq: e.seq, At: e.now()}
	if asg := latestAssignment(t); asg != nil && asg.State != "ended" {
		asg.State = "ended"
	}
}

// addAssignment 给 t 追加一条 pending 派工，并清掉投递失败。返回新派工的编号。
func (e *Engine) addAssignment(t *Ticket) string {
	e.n++
	id := fmt.Sprintf("asg-%d", e.n)
	t.Assignments = append(t.Assignments, Assignment{ID: id, State: "pending"})
	t.DeliveryErr = ""
	return id
}

// addInput 把 tr 记成 t 的一条输入。同一个触发只记一次。
func (e *Engine) addInput(t *Ticket, tr *Trigger) {
	for _, in := range t.Inputs {
		if in.TriggerID == tr.ID {
			return
		}
	}
	e.seq++
	e.n++
	id := fmt.Sprintf("in-%d", e.n)
	t.Inputs = append(t.Inputs, Input{ID: id, TriggerID: tr.ID, Human: tr.Human, Text: tr.Text, Seq: e.seq})
	t.Undelivered = append(t.Undelivered, id)
}

// putDecision 记下 tr 的决定。kind、reason 和 ticketID 原样写入。返回这条决定。
func (e *Engine) putDecision(tr *Trigger, kind, reason, ticketID string) Decision {
	e.n++
	thread := tr.Primary
	if thread == "" && len(tr.Threads) > 0 {
		thread = tr.Threads[0]
	}
	d := Decision{ID: fmt.Sprintf("dec-%d", e.n), TriggerID: tr.ID, Thread: thread, Kind: kind, Reason: reason, TicketID: ticketID, At: e.now()}
	e.decisions = append(e.decisions, d)
	tr.DecisionID = d.ID
	return d
}

// decision 按 id 查找决定。没有时 ok 为假。
func (e *Engine) decision(id string) (Decision, bool) {
	for i := range e.decisions {
		if e.decisions[i].ID == id {
			return e.decisions[i], true
		}
	}
	return Decision{}, false
}

// owners 返回还由工单工人决定的人的消息 tr 落到了哪些工单。机器人消息没有属主。
func (e *Engine) owners(tr *Trigger) []string {
	if !tr.Human || tr.Requeued {
		return nil
	}
	var out []string
	for _, id := range tr.Reached {
		if contains(tr.Handbacks, id) {
			continue
		}
		t := e.tickets[id]
		if t == nil {
			continue
		}
		if t.Ended == nil || (t.Ended.By != "human" && undeliveredTrigger(t, tr.ID)) {
			out = append(out, id)
		}
	}
	return out
}

// queue 把 tr 放进主线程的待评估列。
func (e *Engine) queue(tr *Trigger) {
	tr.Queued = true
	key := tr.Primary
	if key == "" && len(tr.Threads) > 0 {
		key = tr.Threads[0]
	}
	th := e.ensureThread(key)
	if !contains(th.Queued, tr.ID) {
		th.Queued = append(th.Queued, tr.ID)
	}
}

// unqueue 把 tr 从所有线程的待评估列拿走，并把它从正在进行的评估里摘出来。
// 同一轮里其余还没决定的触发回到队列，等下一轮评估。
func (e *Engine) unqueue(tr *Trigger) {
	if tr == nil {
		return
	}
	tr.Queued = false
	for _, th := range e.threads {
		if th == nil {
			continue
		}
		out := th.Queued[:0]
		for _, id := range th.Queued {
			if id != tr.ID {
				out = append(out, id)
			}
		}
		th.Queued = out
	}
	e.releaseEval(tr.ID)
}

// releaseEval 把 id 从正在进行的评估里拿走。同轮其余触发重新排队。id 不在任何评估里时什么都不做。
func (e *Engine) releaseEval(id string) {
	for _, th := range e.threads {
		if th == nil || th.Evaluating == nil {
			continue
		}
		found := false
		var keep []string
		for _, tid := range th.Evaluating.TriggerIDs {
			if tid == id {
				found = true
				continue
			}
			keep = append(keep, tid)
		}
		if !found {
			continue
		}
		th.Evaluating = nil
		for _, tid := range keep {
			other := e.triggers[tid]
			if other != nil && other.DecisionID == "" {
				e.queue(other)
			}
		}
	}
}

// ensureThread 返回 key 对应的线程。没有就建一条。
func (e *Engine) ensureThread(key string) *Thread {
	th := e.threads[key]
	if th == nil {
		th = &Thread{Key: key}
		e.threads[key] = th
	}
	return th
}

// watch 让 ticketID 只为上下文跟随 key。不授予后续，也不恢复已结束的工单。
func (e *Engine) watch(ticketID, key string) {
	th := e.ensureThread(key)
	if contains(th.Subs, ticketID) || contains(th.Watch, ticketID) {
		return
	}
	th.Watch = append(th.Watch, ticketID)
}

// subscribe 让 ticketID 订阅 key。已经订阅时不重复加。
func (e *Engine) subscribe(ticketID, key string) {
	th := e.ensureThread(key)
	if !contains(th.Subs, ticketID) {
		th.Subs = append(th.Subs, ticketID)
	}
}

// storeFiltered 把 in 存成被滤掉的触发。code 是原因。ID 为空时不存。
func (e *Engine) storeFiltered(in Trigger, code string) {
	if in.ID == "" {
		return
	}
	cp := in
	stamp(&cp, e.now())
	cp.Filtered = code
	cp.Queued = false
	e.triggers[cp.ID] = &cp
}

// stamp 在 in 的 At 还是零时写上 now。in 为 nil 时直接返回。
func stamp(in *Trigger, now time.Time) {
	if in == nil || !in.At.IsZero() {
		return
	}
	in.At = now.UTC()
}

// runningAssignment 返回 t 上状态为 running 的派工。没有时返回 nil。
func runningAssignment(t *Ticket) *Assignment {
	for i := range t.Assignments {
		if t.Assignments[i].State == "running" {
			return &t.Assignments[i]
		}
	}
	return nil
}

// reportState 返回 t 最近一条报告的状态。没有报告时返回空串。
func reportState(t *Ticket) string {
	asg := latestAssignment(t)
	if asg == nil || len(asg.Reports) == 0 {
		return ""
	}
	return asg.Reports[len(asg.Reports)-1].State
}

// ownsThread 在 t 跟着一条自己开的线程时为真，比如工人开的拉取请求。
func ownsThread(t *Ticket) bool {
	for _, f := range t.Follows {
		if f.Own {
			return true
		}
	}
	return false
}

// followsDone 在 t 要跟着的线程都已关闭时返回最晚一次关闭。还有开着的就返回 nil。有自己打开的线程时只数那些。
func followsDone(t *Ticket) *Close {
	if t == nil || len(t.Follows) == 0 {
		return nil
	}
	own := 0
	for _, f := range t.Follows {
		if f.Own {
			own++
		}
	}
	var latest *Close
	for i := range t.Follows {
		f := &t.Follows[i]
		if own > 0 && !f.Own {
			continue
		}
		if f.Closed == nil {
			return nil
		}
		if latest == nil || f.Closed.At > latest.At {
			latest = f.Closed
		}
	}
	return latest
}

// undeliveredTrigger 在 t 还有没投递的 triggerID 输入时为真。
func undeliveredTrigger(t *Ticket, triggerID string) bool {
	for _, in := range t.Inputs {
		if in.TriggerID == triggerID && contains(t.Undelivered, in.ID) {
			return true
		}
	}
	return false
}

// contains 在 list 里有 item 时为真。
func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// unique 按原来的顺序去掉 ids 里的空串和重复。
func unique(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// cloneTicket 返回 t 的副本。调用方改副本不会改引擎里的工单。
func cloneTicket(t *Ticket) Ticket {
	out := *t
	out.Mandate.Rules = append([]string(nil), t.Mandate.Rules...)
	out.Assignments = make([]Assignment, len(t.Assignments))
	for i, asg := range t.Assignments {
		asg.Reports = append([]Report(nil), asg.Reports...)
		out.Assignments[i] = asg
	}
	out.Inputs = append([]Input(nil), t.Inputs...)
	out.Follows = make([]Follow, len(t.Follows))
	for i, f := range t.Follows {
		if f.Closed != nil {
			closed := *f.Closed
			f.Closed = &closed
		}
		out.Follows[i] = f
	}
	out.Undelivered = append([]string(nil), t.Undelivered...)
	if t.Ended != nil {
		end := *t.Ended
		out.Ended = &end
	}
	return out
}

// cloneTrigger 返回 t 的副本。切片单独拷贝。
func cloneTrigger(t *Trigger) Trigger {
	out := *t
	out.Threads = append([]string(nil), t.Threads...)
	out.Reached = append([]string(nil), t.Reached...)
	out.Handbacks = append([]string(nil), t.Handbacks...)
	return out
}
