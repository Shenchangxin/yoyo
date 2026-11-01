// engine.go 是领域写入的唯一入口。它不跑模型，也不碰进程。
// 执行器在提交之后通过 OnLaunch 和 OnInput 接手。

package superagent

import (
	"fmt"
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
	e.mu.Lock()
	d, launch, err := e.decide(triggerID, kind, reason)
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
	asg.Reports = append(asg.Reports, report)
	if report.State == "ready" {
		t.Attention = ""
	} else if report.Question != "" {
		t.Attention = report.Question
	}
	e.applyReportEnd(t, report)
	return nil
}

// Stop 由人结束 ticketID。reason 记在结束记录上。已经结束时返回 ticketEnded。
func (e *Engine) Stop(ticketID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stop(ticketID, reason)
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
	}
	if len(e.owners(tr)) == 0 {
		e.queue(tr)
	}
	return "stored", launches, inputs
}

// decide 评估 triggerID。kind 与 reason 写进决定。第二个返回值是要启动的工单，没有则为空。
func (e *Engine) decide(triggerID, kind, reason string) (Decision, string, error) {
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
		if strength != "explicit" && e.policy.Mode != "auto" {
			d := e.putDecision(tr, "ignore", "observed", "")
			e.unqueue(tr)
			return d, "", nil
		}
		e.n++
		ticketID := fmt.Sprintf("tkt-%d", e.n)
		d := e.putDecision(tr, "accept", reason, ticketID)
		e.openTicket(tr, strength, ticketID)
		e.unqueue(tr)
		if e.policy.Paused {
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

// openTicket 用 tr 和 strength 开出 ticketID。授权版本取当前策略。
func (e *Engine) openTicket(tr *Trigger, strength, ticketID string) {
	t := &Ticket{ID: ticketID, Origin: tr.ID, Mandate: Mandate{Version: e.policy.Version, Strength: strength, Sender: tr.Sender}}
	if strength == "explicit" {
		t.Mandate.Rules = []string{"addressed"}
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
	e.n++
	e.outbox = append(e.outbox, Outbound{
		ID: fmt.Sprintf("out-%d", e.n), TicketID: ticketID, Kind: "acknowledgement", Text: "已接单", Status: "pending",
	})
}

// applyReportEnd 按 report 结束 t。没有跟随线程的 ready 记 completed。跟随都关闭时用关闭结果。dropped 结束工单。
func (e *Engine) applyReportEnd(t *Ticket, report Report) {
	switch report.State {
	case "dropped":
		outcome := "dropped"
		if done := followsDone(t); done != nil && done.Outcome != "" {
			outcome = done.Outcome
		}
		e.finish(t, "executor", outcome, report.Summary)
	case "ready":
		if len(t.Follows) == 0 {
			e.finish(t, "executor", "completed", report.Summary)
			return
		}
		if done := followsDone(t); done != nil {
			e.finish(t, "executor", done.Outcome, report.Summary)
		}
	}
}

// resume 在没暂停且授权仍有效时重新打开 t。需要新派工时返回真。
func (e *Engine) resume(t *Ticket) bool {
	if e.policy.Paused || !e.policy.Revalidate(t.Mandate) {
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
	t.Ended = &End{By: by, Outcome: outcome, Reason: reason, Seq: e.seq}
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
