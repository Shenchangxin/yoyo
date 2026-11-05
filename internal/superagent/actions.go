// actions.go 拥有工单外围动作：挂接、回复、交还、备注、退订、驳回和策略变更。
// 不发送渠道消息。发件箱只记账，Runtime 再把它写回 GitHub 或飞书。

package superagent

import "strings"

// Attach 把 triggerID 挂到 ticketID。note 非空时再记一条备注输入。
// 授权失效，或这条触发已经被这个工单交还过，会拒绝。已经停下的工单沿用原目录再打开。
func (e *Engine) Attach(ticketID, triggerID, note string) error {
	return e.finishAttach(ticketID, triggerID, note, false)
}

// Assign 把 brief 交给 ticketID 的工人，作为这一轮唯一要读的说明。
// 人的原话留在触发上，不再写进工人能看到的正文。触发仍算到达，阶段也能看到人已经跟上。
// brief 为空时和 Attach 一样，原话照常送过去。
func (e *Engine) Assign(ticketID, triggerID, brief string) error {
	return e.finishAttach(ticketID, triggerID, brief, true)
}

// finishAttach 挂接触发。fold 为真且 note 非空时，人的原话不进工人正文，note 是唯一的新说明。
func (e *Engine) finishAttach(ticketID, triggerID, note string, fold bool) error {
	e.mu.Lock()
	launch, wake, err := e.attach(ticketID, triggerID, note, fold)
	e.mu.Unlock()
	if err != nil {
		return err
	}
	if launch && e.OnLaunch != nil {
		e.OnLaunch(ticketID)
		return nil
	}
	if wake && e.OnInput != nil {
		wakeText := strings.TrimSpace(note)
		if wakeText == "" {
			wakeText = triggerID
		}
		e.OnInput(ticketID, wakeText)
	}
	return nil
}

// attach 在锁内挂接触发。launch 为真表示工单被重新打开，调用方应在解锁后启动它。
// wake 为真表示工单还开着，调用方要把新输入送给工人。
// fold 为真时，刚记上的原话会从工人正文里拿掉，只留下 note。
func (e *Engine) attach(ticketID, triggerID, note string, fold bool) (bool, bool, error) {
	t := e.tickets[ticketID]
	if t == nil {
		return false, false, errCode("unknownTicket")
	}
	if !e.policy.Revalidate(t.Mandate) {
		return false, false, errCode("mandate")
	}
	tr := e.triggers[triggerID]
	if tr == nil {
		return false, false, errCode("unknownTrigger")
	}
	if contains(tr.Handbacks, ticketID) {
		return false, false, errCode("handedBackHere")
	}
	note = strings.TrimSpace(note)
	e.unqueue(tr)
	if !reachedTrigger(t, tr.ID) {
		e.addInput(t, tr)
		if fold && note != "" {
			e.hideTriggerText(t, tr.ID)
		}
		if !contains(tr.Reached, ticketID) {
			tr.Reached = append(tr.Reached, ticketID)
		}
	}
	if note != "" {
		e.addNote(t, clipRunes(note, 12000), false)
	}
	for _, key := range tr.Threads {
		e.subscribe(ticketID, key)
		e.follow(t, key, false)
	}
	if t.Ended == nil {
		return false, true, nil
	}
	// 人把下一句交到这张工单上时，暂停不应当成工单已死。否则路由会另开一张永远不派工的单。
	if !e.reopen(t, false) {
		return false, false, errCode("ticketEnded")
	}
	return true, false, nil
}

// Reply 按 purpose 把 text 放进发件箱。purpose 取 acknowledgement、answer、result 或 attention。
// ticketID 是工单，triggerID 是相关触发。相同键的第二次调用不新增。
func (e *Engine) Reply(ticketID, purpose, triggerID, text string) (Outbound, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reply(ticketID, purpose, triggerID, text)
}

// reply 在锁内写发件箱。规则不满足时返回错误短码。
func (e *Engine) reply(ticketID, purpose, triggerID, text string) (Outbound, error) {
	t := e.tickets[ticketID]
	if t == nil {
		return Outbound{}, errCode("unknownTicket")
	}
	tr := e.triggers[triggerID]
	if triggerID != "" && tr == nil {
		return Outbound{}, errCode("unknownTrigger")
	}
	key := ""
	switch purpose {
	case "acknowledgement":
		if t.Ended != nil {
			return Outbound{}, errCode("ackEnded")
		}
		if tr == nil || !tr.Human {
			return Outbound{}, errCode("ackNotPerson")
		}
		if tr.Source == "" {
			return Outbound{}, errCode("ackNoChannel")
		}
		key = "ack:" + tr.ID
	case "answer":
		if tr == nil || !tr.Human {
			return Outbound{}, errCode("notQuestion")
		}
		key = "reply:" + tr.ID + ":" + text
	case "result":
		if !resultReady(t) {
			return Outbound{}, errCode("notReady")
		}
		key = "result:" + t.ID
	case "attention":
		if t.Attention == "" {
			return Outbound{}, errCode("notAttention")
		}
		key = "attention:" + t.ID
	default:
		return Outbound{}, errCode("bad_purpose")
	}
	if existing, ok := e.outboundByKey(key); ok {
		return existing, nil
	}
	source, ref := "", ""
	if tr != nil {
		source, ref = tr.Source, tr.Ref
	}
	if source == "" {
		source = "desk"
	}
	e.n++
	msg := Outbound{
		ID: "out-" + itoa(e.n), TicketID: t.ID, Kind: purpose, Text: text,
		Status: "pending", Source: source, Ref: ref, Key: key,
	}
	e.outbox = append(e.outbox, msg)
	return msg, nil
}

// Handback 让 ticketID 交还 triggerID。reason 记在备注里。交还后若没有别的工单还在决定它，就重新排队。
// 返回值表示有没有重新排队。
func (e *Engine) Handback(ticketID, triggerID, reason string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return false, errCode("unknownTicket")
	}
	tr := e.triggers[triggerID]
	if tr == nil {
		return false, errCode("unknownTrigger")
	}
	if !contains(tr.Reached, ticketID) {
		return false, errCode("notReached")
	}
	if contains(tr.Handbacks, ticketID) {
		return false, errCode("handedBack")
	}
	tr.Handbacks = append(tr.Handbacks, ticketID)
	if strings.TrimSpace(reason) != "" {
		e.addNote(t, reason, false)
	}
	if len(e.owners(tr)) > 0 {
		return false, nil
	}
	tr.Requeued = true
	e.queue(tr)
	return true, nil
}

// Say 把人的下一句交给工单，并叫醒工人。做完或被人打断的工单都沿用原目录再开一轮。
// 空正文返回 empty。重新开工时不再单独投递，派工正文里已经带上这一句。
func (e *Engine) Say(ticketID, text string) error {
	text = strings.TrimSpace(text)
	if e == nil || text == "" {
		return errCode("empty")
	}
	e.mu.Lock()
	t := e.tickets[ticketID]
	if t == nil {
		e.mu.Unlock()
		return errCode("unknownTicket")
	}
	launch := false
	if t.Ended != nil {
		if !e.resume(t) {
			e.mu.Unlock()
			return errCode("ticketEnded")
		}
		launch = true
	}
	e.addNote(t, text, true)
	e.mu.Unlock()
	if launch && e.OnLaunch != nil {
		e.OnLaunch(ticketID)
		return nil
	}
	if e.OnInput != nil {
		e.OnInput(ticketID, text)
	}
	return nil
}

// Note 给还开着的 ticketID 记一条事实。text 原样成为输入。工单已结束时返回 ticketEnded。
func (e *Engine) Note(ticketID, text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Ended != nil {
		return errCode("ticketEnded")
	}
	e.addNote(t, text, false)
	return nil
}

// Unsubscribe 退订 thread 上全部非自有跟随。返回有没有退掉至少一条。
func (e *Engine) Unsubscribe(thread string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	th := e.threads[thread]
	dropped := false
	if th != nil && len(th.Subs) > 0 {
		th.Subs = nil
		dropped = true
	}
	for _, t := range e.tickets {
		if t == nil {
			continue
		}
		kept := t.Follows[:0]
		for _, f := range t.Follows {
			if f.Thread == thread && !f.Own {
				dropped = true
				continue
			}
			kept = append(kept, f)
		}
		t.Follows = kept
	}
	return dropped
}

// Dismiss 把 unavailable 的 decisionID 标成不再出现在待处理里。别的决定返回 notUnavailable。
func (e *Engine) Dismiss(decisionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.decisions {
		if e.decisions[i].ID != decisionID {
			continue
		}
		if e.decisions[i].Kind != "unavailable" {
			return errCode("notUnavailable")
		}
		e.decisions[i].Dismissed = true
		return nil
	}
	return errCode("unknownDecision")
}

// Reevaluate 把 triggerID 重新放回队列。它正处在一轮评估里时返回 false。未知触发返回错误。
func (e *Engine) Reevaluate(triggerID string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	tr := e.triggers[triggerID]
	if tr == nil {
		return false, errCode("unknownTrigger")
	}
	for _, th := range e.threads {
		if th == nil || th.Evaluating == nil {
			continue
		}
		if contains(th.Evaluating.TriggerIDs, triggerID) {
			return false, nil
		}
	}
	tr.DecisionID = ""
	tr.Requeued = true
	e.queue(tr)
	return true, nil
}

// PolicyChanged 让进行中的评估重新排队，并给还开着的工单记一条策略变更。from 和 to 写进备注。
func (e *Engine) PolicyChanged(from, to string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, th := range e.threads {
		if th == nil || th.Evaluating == nil {
			continue
		}
		ids := append([]string(nil), th.Evaluating.TriggerIDs...)
		th.Evaluating = nil
		for _, id := range ids {
			tr := e.triggers[id]
			if tr == nil || tr.DecisionID != "" {
				continue
			}
			e.queue(tr)
		}
	}
	note := "policy " + from + " -> " + to
	for _, t := range e.tickets {
		if t == nil || t.Ended != nil {
			continue
		}
		e.addNote(t, note, false)
	}
}

// MarkUnrelated 去掉 ticketID 对 thread 的跟随。thread 是工单起点时拒绝。reason 记成备注。
func (e *Engine) MarkUnrelated(ticketID, thread, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return errCode("unknownTicket")
	}
	if t.Origin != "" {
		if tr := e.triggers[t.Origin]; tr != nil && (tr.Primary == thread || contains(tr.Threads, thread)) {
			return errCode("originThread")
		}
	}
	found := false
	kept := t.Follows[:0]
	for _, f := range t.Follows {
		if f.Thread == thread {
			found = true
			continue
		}
		kept = append(kept, f)
	}
	if !found {
		return errCode("notFollowed")
	}
	t.Follows = kept
	if th := e.threads[thread]; th != nil {
		subs := th.Subs[:0]
		for _, id := range th.Subs {
			if id != ticketID {
				subs = append(subs, id)
			}
		}
		th.Subs = subs
		watch := th.Watch[:0]
		for _, id := range th.Watch {
			if id != ticketID {
				watch = append(watch, id)
			}
		}
		th.Watch = watch
	}
	if strings.TrimSpace(reason) != "" {
		e.addNote(t, reason, false)
	}
	return nil
}

// FollowForContext 让 ticketID 为了上下文跟随 thread。已经跟随或工单已结束时按规则返回。
func (e *Engine) FollowForContext(ticketID, thread string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return false, errCode("unknownTicket")
	}
	if t.Ended != nil {
		return false, errCode("ticketEnded")
	}
	for _, f := range t.Follows {
		if f.Thread == thread {
			return false, nil
		}
	}
	e.follow(t, thread, false)
	e.watch(ticketID, thread)
	th := e.ensureThread(thread)
	if !contains(th.Context, ticketID) {
		th.Context = append(th.Context, ticketID)
	}
	return true, nil
}

// RequestStatus 在工单最近一次报告是 waiting 时，记下一条只问状态的输入。返回有没有记下。
func (e *Engine) RequestStatus(ticketID string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil {
		return false, errCode("unknownTicket")
	}
	asg := latestAssignment(t)
	if asg == nil || len(asg.Reports) == 0 || asg.Reports[len(asg.Reports)-1].State != "waiting" {
		return false, nil
	}
	e.addNote(t, "Report the current ticket status with orchestrator_report. This is a status-only request.", false)
	return true, nil
}

// PendingOutbox 返回还要发送的发件箱副本。失败未满三次的仍在里面。同一工单、同一来源按原顺序。
func (e *Engine) PendingOutbox() []Outbound {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []Outbound
	for _, msg := range e.outbox {
		if msg.Status == "sent" || msg.Status == "dropped" {
			continue
		}
		if msg.Status == "error" && msg.Attempts >= 3 {
			continue
		}
		out = append(out, msg)
	}
	return out
}

// MarkOutbound 把 id 的状态写成 status。error 累加失败次数。没找到时什么都不做。
func (e *Engine) MarkOutbound(id, status string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.outbox {
		if e.outbox[i].ID != id {
			continue
		}
		if status == "error" {
			e.outbox[i].Attempts++
			e.outbox[i].Status = "error"
			return
		}
		e.outbox[i].Status = status
		return
	}
}

// hideTriggerText 把 t 上这条触发的正文清掉，工人的派工和投递都看不到。
// 输入还在，人和序号都留着，阶段才知道人已经跟上。
func (e *Engine) hideTriggerText(t *Ticket, triggerID string) {
	for i := range t.Inputs {
		if t.Inputs[i].TriggerID != triggerID {
			continue
		}
		t.Inputs[i].Text = ""
		id := t.Inputs[i].ID
		kept := make([]string, 0, len(t.Undelivered))
		for _, pending := range t.Undelivered {
			if pending != id {
				kept = append(kept, pending)
			}
		}
		t.Undelivered = kept
		return
	}
}

// addNote 给 t 追加一条不绑定触发的输入。text 原样保存。human 为真表示人写的。
func (e *Engine) addNote(t *Ticket, text string, human bool) {
	e.seq++
	e.n++
	id := "in-" + itoa(e.n)
	t.Inputs = append(t.Inputs, Input{ID: id, Human: human, Text: text, Seq: e.seq})
	t.Undelivered = append(t.Undelivered, id)
}

// follow 让 t 跟随 thread。own 为真时已有跟随也会变成自有。
func (e *Engine) follow(t *Ticket, thread string, own bool) {
	for i := range t.Follows {
		if t.Follows[i].Thread == thread {
			if own {
				t.Follows[i].Own = true
			}
			return
		}
	}
	t.Follows = append(t.Follows, Follow{Thread: thread, Own: own})
}

// outboundByKey 按 key 找发件箱。没有时 ok 为假。
func (e *Engine) outboundByKey(key string) (Outbound, bool) {
	for _, msg := range e.outbox {
		if msg.Key == key {
			return msg, true
		}
	}
	return Outbound{}, false
}

// resultReady 在 t 已经完成、合并、丢弃，或最近报告是 ready 时为真。
func resultReady(t *Ticket) bool {
	if t == nil {
		return false
	}
	if t.Ended != nil {
		switch t.Ended.Outcome {
		case "completed", "merged", "dropped":
			return true
		}
	}
	asg := latestAssignment(t)
	return asg != nil && len(asg.Reports) > 0 && asg.Reports[len(asg.Reports)-1].State == "ready"
}

// reachedTrigger 在 t 的输入里已经有 triggerID 时为真。
func reachedTrigger(t *Ticket, triggerID string) bool {
	for _, in := range t.Inputs {
		if in.TriggerID == triggerID {
			return true
		}
	}
	return false
}

// itoa 把 n 收成十进制。n 为负时带负号。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
