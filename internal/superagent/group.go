// group.go 管任务组：一条触发拆给几个 agent，分工（split，可带先后）或对比（compare，同一份说明各做一遍）。
// 有前置的成员先挂着，前置都做完时带上它们的结果放行，前置没做成时挂起等人；整组的状态从成员算出，
// 状态变了才唤醒路由，同一个状态只唤醒一次。
// 这里只改引擎里的数据。建共享目录、排唤醒、任务组摘要在 group_runtime.go，主会话的卡片在 app。

package superagent

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 任务组的模式。
const (
	GroupSplit   = "split"
	GroupCompare = "compare"
)

// 任务组的状态，见 groupState。
const (
	GroupRunning   = "running"
	GroupAttention = "attention"
	GroupDone      = "done"
	GroupFailed    = "failed"
	GroupStopped   = "stopped"
)

// 成员挂着的原因，记在 Ticket.Held。
const (
	HeldDeps       = "deps"
	HeldDepsFailed = "deps_failed"
)

// 前置工单的状态，见 DepsState。
const (
	DepsReady   = "ready"
	DepsWaiting = "waiting"
	DepsFailed  = "failed"
)

// 拆分的份数：split 拆 2 到 6 份，compare 交给 2 到 4 个不同的 agent。
const (
	splitMin   = 2
	splitMax   = 6
	compareMax = 4
)

// SplitPart 是拆出来的一份。DependsOn 是要先做完的那几份在 Parts 里的下标，只能指向前面的。
// compare 的 Brief 可以空着，用 SplitRequest.Brief。
type SplitPart struct {
	Title     string
	Agent     string
	Role      string
	Brief     string
	DependsOn []int
}

// SplitRequest 是一次拆分。Mode 取 split 或 compare，Title 是任务组的短标题，Reason 和 Rules 同接受一张工单。
// Brief 是 compare 的共用说明。Session 是发起的桌面会话，空着时从触发上取。
// Root 是执行根，split 的共享目录是 Root/groups/<编号>，Root 为空时不设。Names 把 agent 编号换成写进说明的名字。
type SplitRequest struct {
	Trigger string
	Mode    string
	Title   string
	Reason  string
	Rules   []string
	Brief   string
	Parts   []SplitPart
	Session string
	Root    string
	Names   map[string]string
}

// GroupWake 是一次该唤醒路由的状态变化。Mark 是「状态:序号」，Count 是算上这一次这个任务组记过几个标记。
type GroupWake struct {
	Group   string
	Session string
	State   string
	Mark    string
	Count   int
}

// Split 把 req.Trigger 接成一个任务组：每份开一张工单，只有第一张给来源回已接单。
// 返回任务组和马上启动的工单，即没有前置的那几份；暂停时推断的工作一份都不启动。启动在锁外经 OnLaunch。
// 同一条触发再拆一次时返回原来的任务组，不再启动。说明不合要求时返回错误短码，什么都不改。
func (e *Engine) Split(req SplitRequest) (Group, []string, error) {
	e.mu.Lock()
	g, launches, err := e.split(req)
	e.mu.Unlock()
	if err == nil && e.OnLaunch != nil {
		for _, id := range launches {
			e.OnLaunch(id)
		}
	}
	return g, launches, err
}

// split 是 Split 在锁内的部分。
func (e *Engine) split(req SplitRequest) (Group, []string, error) {
	tr := e.triggers[req.Trigger]
	if tr == nil || tr.Filtered != "" {
		return Group{}, nil, errCode("unknownTrigger")
	}
	if tr.DecisionID != "" {
		if d, ok := e.decision(tr.DecisionID); ok && d.Kind == "accept" && e.groups[d.Group] != nil {
			return cloneGroup(e.groups[d.Group]), nil, nil
		}
		return Group{}, nil, errCode("settled")
	}
	parts, err := splitParts(req)
	if err != nil {
		return Group{}, nil, err
	}
	strength, matched, observed, err := e.admitAccept(tr, req.Rules, false)
	if err != nil {
		return Group{}, nil, err
	}
	if observed {
		e.putDecision(tr, "ignore", "observed", "")
		e.unqueue(tr)
		return Group{}, nil, errCode("observed")
	}
	e.n++
	g := &Group{
		ID: fmt.Sprintf("grp-%d", e.n), Mode: req.Mode, Title: clipRunes(strings.TrimSpace(req.Title), 80),
		Trigger: tr.ID, Session: strings.TrimSpace(req.Session), Opened: e.now(),
	}
	if g.Session == "" {
		g.Session = deskSessionOf(tr)
	}
	if root := strings.TrimSpace(req.Root); root != "" && g.Mode == GroupSplit {
		g.SharedDir = filepath.Join(root, "groups", g.ID)
	}
	for range parts {
		e.n++
		g.Tickets = append(g.Tickets, fmt.Sprintf("tkt-%d", e.n))
	}
	e.groups[g.ID] = g
	e.putDecision(tr, "accept", firstText(req.Reason, g.Title), g.Tickets[0])
	e.decisions[len(e.decisions)-1].Group = g.ID
	start := strength == "explicit" || !e.policy.Paused
	var launches []string
	for i, part := range parts {
		id := g.Tickets[i]
		e.openTicket(tr, strength, id, part.Title, matched, memberBrief(g, parts, i, req.Names), i == 0)
		t := e.tickets[id]
		t.Group, t.Role, t.Title = g.ID, part.Role, part.Title
		t.Agent = part.Agent
		t.AgentPinned = t.Agent != "" && contains(tr.Agents, t.Agent)
		for _, dep := range part.DependsOn {
			t.DependsOn = append(t.DependsOn, g.Tickets[dep])
		}
		if len(t.DependsOn) > 0 {
			t.Held = HeldDeps
			continue
		}
		if start {
			launches = append(launches, id)
		}
	}
	e.unqueue(tr)
	return cloneGroup(g), launches, nil
}

// splitParts 检查 req 并把每份收拾好：agent 记成小写编号，yoyo 记成空；compare 没写说明的份用共用说明。
// 违反时返回错误短码：empty_title、bad_mode、bad_parts、empty_agent、empty_brief、bad_deps、compare_deps、dup_agents。
func splitParts(req SplitRequest) ([]SplitPart, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, errCode("empty_title")
	}
	most := splitMax
	switch req.Mode {
	case GroupSplit:
	case GroupCompare:
		most = compareMax
	default:
		return nil, errCode("bad_mode")
	}
	if len(req.Parts) < splitMin || len(req.Parts) > most {
		return nil, errCode("bad_parts")
	}
	shared := strings.TrimSpace(req.Brief)
	seen := map[string]bool{}
	out := make([]SplitPart, len(req.Parts))
	for i, p := range req.Parts {
		p.Agent = strings.ToLower(strings.TrimSpace(p.Agent))
		if p.Agent == "" {
			return nil, errCode("empty_agent")
		}
		key := agentKey(p.Agent)
		if !externalAgent(p.Agent) {
			p.Agent = ""
		}
		p.Title = clipRunes(strings.TrimSpace(p.Title), 80)
		p.Role = clipRunes(strings.TrimSpace(p.Role), 40)
		p.Brief = clipRunes(strings.TrimSpace(p.Brief), 12000)
		if req.Mode == GroupCompare {
			if len(p.DependsOn) > 0 {
				return nil, errCode("compare_deps")
			}
			if seen[key] {
				return nil, errCode("dup_agents")
			}
			seen[key] = true
			if p.Brief == "" {
				p.Brief = clipRunes(shared, 12000)
			}
		}
		var deps []int
		for _, d := range p.DependsOn {
			if d < 0 || d >= i {
				return nil, errCode("bad_deps")
			}
			if !containsInt(deps, d) {
				deps = append(deps, d)
			}
		}
		p.DependsOn = deps
		if p.Brief == "" {
			return nil, errCode("empty_brief")
		}
		p.Title = firstText(p.Title, p.Role, clipRunes(strings.TrimSpace(req.Title), 80))
		out[i] = p
	}
	return out, nil
}

// memberBrief 是第 i 份交给工人的说明：路由写的说明，后面写明同组还有谁。
// split 再写共享目录和要等的前置；compare 写明同一份说明也交给了谁，各做各的。
func memberBrief(g *Group, parts []SplitPart, i int, names map[string]string) string {
	var b strings.Builder
	b.WriteString(parts[i].Brief)
	b.WriteString("\n\n")
	if g.Mode == GroupCompare {
		var others []string
		for j, p := range parts {
			if j != i {
				others = append(others, memberName(p.Agent, names)+"（"+g.Tickets[j]+"）")
			}
		}
		b.WriteString("同一份说明也交给了 " + strings.Join(others, "、") + "，独立完成，不要看别人的目录。")
		return b.String()
	}
	if g.SharedDir != "" {
		// 路径单独一行，后面不接别的字，免得被当成路径的一部分。
		b.WriteString("任务组共享目录（别的成员要用的产物、接口和约定放这里；开工前先读里面的 NOTES.md，做完在你那一节记下来）：\n")
		b.WriteString(g.SharedDir + "\n")
	}
	var others []string
	for j, p := range parts {
		if j == i {
			continue
		}
		line := g.Tickets[j] + " " + memberName(p.Agent, names)
		if role := firstText(p.Role, p.Title); role != "" {
			line += " · " + role
		}
		others = append(others, line)
	}
	b.WriteString("同组的其他成员：" + strings.Join(others, "；") + "。")
	if deps := parts[i].DependsOn; len(deps) > 0 {
		var ids []string
		for _, d := range deps {
			ids = append(ids, g.Tickets[d])
		}
		b.WriteString("\n" + strings.Join(ids, "、") + " 做完后你才开工，开工时会收到它们的结果和目录。")
	}
	return b.String()
}

// DepsState 说明 ticketID 的前置工单：ready 是都做完了，failed 是有没做成的，waiting 是还有没做完的。没有前置时为 ready。
func (e *Engine) DepsState(ticketID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.depsStateLocked(e.tickets[ticketID])
}

// depsStateLocked 是 DepsState 在锁内的部分。只有结束成 completed 或 merged 才算做完；
// 结束成别的、或者最近一份报告是 failed 还在等人时算没做成。有一份没做成就是 failed。
func (e *Engine) depsStateLocked(t *Ticket) string {
	if t == nil {
		return DepsReady
	}
	state := DepsReady
	for _, id := range t.DependsOn {
		dep := e.tickets[id]
		switch {
		case dep == nil:
			return DepsFailed
		case dep.Ended != nil:
			if !completedEnd(dep.Ended) {
				return DepsFailed
			}
		case failedReport(dep):
			return DepsFailed
		default:
			state = DepsWaiting
		}
	}
	return state
}

// ReleaseDependent 放行挂着的 ticketID：清掉 Held，追加一条不算人写的输入，再启动。note 为空时写前置任务的结果。
// 暂停时推断的工作只放行不启动。没有挂着时返回 notHeld。
func (e *Engine) ReleaseDependent(ticketID, note string) error {
	e.mu.Lock()
	t := e.tickets[ticketID]
	if err := heldCheck(t); err != nil {
		e.mu.Unlock()
		return err
	}
	e.releaseLocked(t, note, nil)
	start := t.Mandate.Strength == "explicit" || !e.policy.Paused
	e.mu.Unlock()
	if start && e.OnLaunch != nil {
		e.OnLaunch(ticketID)
	}
	return nil
}

// HoldDependent 在 ticketID 的前置没做成时挂起它：Held 记成 deps_failed，再记一份不带问题的 blocked。reason 为空时写哪几份没做成。
func (e *Engine) HoldDependent(ticketID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if err := heldCheck(t); err != nil {
		return err
	}
	e.holdLocked(t, reason, nil)
	return nil
}

// heldCheck 在 t 不是一张还挂着的工单时返回错误短码。
func heldCheck(t *Ticket) error {
	switch {
	case t == nil:
		return errCode("unknownTicket")
	case t.Ended != nil:
		return errCode("ticketEnded")
	case t.Held == "":
		return errCode("notHeld")
	}
	return nil
}

// releaseLocked 清掉 t 的 Held 并追加 note。note 为空时用前置任务的结果。调用时要已持有锁。
func (e *Engine) releaseLocked(t *Ticket, note string, names map[string]string) {
	t.Held = ""
	if strings.TrimSpace(note) == "" {
		note = e.depsNoteLocked(t, names)
	}
	e.addNote(t, note, false)
}

// holdLocked 把 t 挂成 deps_failed，并记一份不带问题的 blocked。调用时要已持有锁。
func (e *Engine) holdLocked(t *Ticket, reason string, names map[string]string) {
	t.Held = HeldDepsFailed
	if strings.TrimSpace(reason) == "" {
		var failed []string
		for _, id := range t.DependsOn {
			if dep := e.tickets[id]; dep == nil || (dep.Ended != nil && !completedEnd(dep.Ended)) || (dep.Ended == nil && failedReport(dep)) {
				failed = append(failed, memberLabel(id, dep, names))
			}
		}
		reason = "前置任务没做成，先不开工：" + strings.Join(failed, "；") + "。等人决定重试那一份、换 agent，还是停掉整组。"
	}
	_ = e.reportLocked(t, Report{State: "blocked", Summary: clipRunes(reason, 600)})
}

// depsNoteLocked 是放行时交给工人的那条输入：每个前置任务一行，写谁做的、做成了什么、目录在哪。
func (e *Engine) depsNoteLocked(t *Ticket, names map[string]string) string {
	var b strings.Builder
	b.WriteString("前置任务都完成了：")
	for _, id := range t.DependsOn {
		dep := e.tickets[id]
		b.WriteString("\n" + memberLabel(id, dep, names))
		if dep == nil {
			continue
		}
		if s := strings.TrimRight(strings.TrimSpace(resultSummary(dep)), "。.!！ "); s != "" {
			b.WriteString("：" + clipRunes(s, 300))
		}
		b.WriteString("。")
		if dir := strings.TrimSpace(dep.Directory); dir != "" {
			b.WriteString("目录 " + dir)
		}
	}
	return b.String()
}

// GroupState 是任务组 gid 现在的状态。没有这个任务组时为空串。
func (e *Engine) GroupState(gid string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return groupState(e.groups[gid], e.ticketLocked)
}

// GroupMark 是任务组 gid 现在的唤醒标记「状态:序号」。没有这个任务组时为空串。
func (e *Engine) GroupMark(gid string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	if g == nil {
		return ""
	}
	return groupState(g, e.ticketLocked) + ":" + strconv.Itoa(groupSeq(g, e.ticketLocked))
}

// ticketLocked 按编号找工单。调用时要已持有锁。
func (e *Engine) ticketLocked(id string) *Ticket {
	return e.tickets[id]
}

// groupState 从成员算出 g 的状态。ticket 按编号取成员。
// 整组已经结束时为 stopped（人停掉了）或 done（采用了一份）。有成员前置没做成或停在等人时为 attention；
// 还有成员没结束时为 running。都结束后：compare 有一份做完就是 done；有成员没做成（不是人停的）为 failed；
// 全部做完为 done；剩下的是人停掉了几份，为 stopped。
func groupState(g *Group, ticket func(string) *Ticket) string {
	if g == nil {
		return ""
	}
	if g.Ended != nil {
		if g.Ended.Outcome == "stopped" {
			return GroupStopped
		}
		return GroupDone
	}
	open, attention := false, false
	completed, stopped := 0, 0
	for _, id := range g.Tickets {
		t := ticket(id)
		switch {
		case t == nil:
		case t.Ended == nil:
			open = true
			if t.Held == HeldDepsFailed || TicketStage(t).Person {
				attention = true
			}
		case completedEnd(t.Ended):
			completed++
		case t.Ended.By == "human" && t.Ended.Outcome == "stopped":
			stopped++
		}
	}
	switch {
	case attention:
		return GroupAttention
	case open:
		return GroupRunning
	case g.Mode == GroupCompare && completed > 0:
		return GroupDone
	case completed+stopped < len(g.Tickets):
		return GroupFailed
	case stopped == 0:
		return GroupDone
	default:
		return GroupStopped
	}
}

// groupSeq 是 g 的成员最后一份不是进度的报告或结束的序号。进度不改唤醒标记，免得同一个状态被反复唤醒。
func groupSeq(g *Group, ticket func(string) *Ticket) int {
	seq := 0
	for _, id := range g.Tickets {
		t := ticket(id)
		if t == nil {
			continue
		}
		if t.Ended != nil && t.Ended.Seq > seq {
			seq = t.Ended.Seq
		}
		for _, asg := range t.Assignments {
			for _, rep := range asg.Reports {
				if rep.State != "progress" && rep.Seq > seq {
					seq = rep.Seq
				}
			}
		}
	}
	return seq
}

// WatchGroups 看一遍开着的任务组：挂着的成员前置都做完时放行并启动，前置没做成时挂起；
// 状态是 attention、done 或 failed 且这个标记还没唤醒过时记下标记，返回要发的唤醒。names 把 agent 编号换成写进说明的名字。
// 暂停时推断的成员不放行，恢复后再放。启动在锁外经 OnLaunch。
func (e *Engine) WatchGroups(names map[string]string) (launches []string, wakes []GroupWake) {
	e.mu.Lock()
	for _, gid := range e.groupIDsLocked() {
		g := e.groups[gid]
		if g.Ended != nil {
			continue
		}
		for _, id := range g.Tickets {
			t := e.tickets[id]
			if t == nil || t.Ended != nil || t.Held == "" {
				continue
			}
			switch e.depsStateLocked(t) {
			case DepsReady:
				if e.policy.Paused && t.Mandate.Strength != "explicit" {
					continue
				}
				e.releaseLocked(t, "", names)
				launches = append(launches, id)
			case DepsFailed:
				if t.Held == HeldDeps {
					e.holdLocked(t, "", names)
				}
			}
		}
		state := groupState(g, e.ticketLocked)
		if state != GroupAttention && state != GroupDone && state != GroupFailed {
			continue
		}
		mark := state + ":" + strconv.Itoa(groupSeq(g, e.ticketLocked))
		if !e.markWokeLocked(g, mark) {
			continue
		}
		wakes = append(wakes, GroupWake{Group: g.ID, Session: g.Session, State: state, Mark: mark, Count: len(g.Woke)})
	}
	e.mu.Unlock()
	if e.OnLaunch != nil {
		for _, id := range launches {
			e.OnLaunch(id)
		}
	}
	return launches, wakes
}

// MarkWoke 记下任务组 gid 的唤醒标记 mark。第一次记下时为真，记过或没有这个任务组时为假。
func (e *Engine) MarkWoke(gid, mark string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	if g == nil {
		return false
	}
	return e.markWokeLocked(g, mark)
}

// markWokeLocked 是 MarkWoke 在锁内的部分。
func (e *Engine) markWokeLocked(g *Group, mark string) bool {
	if mark == "" || contains(g.Woke, mark) {
		return false
	}
	g.Woke = append(g.Woke, mark)
	return true
}

// StopGroup 停掉任务组 gid：没结束的成员都记成人停下，任务组记成 stopped。返回刚停下的成员，调用方要打断它们的工人。
func (e *Engine) StopGroup(gid, reason string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	if g == nil {
		return nil, errCode("unknownGroup")
	}
	if g.Ended != nil {
		return nil, errCode("groupEnded")
	}
	reason = firstText(reason, "停掉了整组")
	stopped := e.stopMembersLocked(g, "", reason)
	e.seq++
	g.Ended = &End{By: "human", Outcome: "stopped", Reason: reason, Seq: e.seq, At: e.now()}
	return stopped, nil
}

// Pick 记下对比任务组 gid 采用 ticketID，停掉还没结束的其他成员，任务组就此结束。返回刚停下的成员。
// 不是对比时返回 notCompare，ticketID 不在组里时返回 notMember。
func (e *Engine) Pick(gid, ticketID, reason string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	switch {
	case g == nil:
		return nil, errCode("unknownGroup")
	case g.Mode != GroupCompare:
		return nil, errCode("notCompare")
	case g.Ended != nil:
		return nil, errCode("groupEnded")
	case !contains(g.Tickets, ticketID):
		return nil, errCode("notMember")
	}
	reason = firstText(reason, "采用了 "+ticketID)
	stopped := e.stopMembersLocked(g, ticketID, reason)
	g.Winner = ticketID
	e.seq++
	g.Ended = &End{By: "human", Outcome: "picked", Reason: reason, Seq: e.seq, At: e.now()}
	return stopped, nil
}

// Recommend 记下路由在对比任务组 gid 里推荐 ticketID。只是标出来给人看，不停任何成员，也不结束任务组；采用要等人同意后 Pick。
// 错误码和 Pick 相同。
func (e *Engine) Recommend(gid, ticketID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	switch {
	case g == nil:
		return errCode("unknownGroup")
	case g.Mode != GroupCompare:
		return errCode("notCompare")
	case g.Ended != nil:
		return errCode("groupEnded")
	case !contains(g.Tickets, ticketID):
		return errCode("notMember")
	}
	g.Recommended = ticketID
	return nil
}

// stopMembersLocked 把 g 里除了 keep 之外没结束的成员记成人停下。返回停下的成员。
func (e *Engine) stopMembersLocked(g *Group, keep, reason string) []string {
	var stopped []string
	for _, id := range g.Tickets {
		t := e.tickets[id]
		if id == keep || t == nil || t.Ended != nil {
			continue
		}
		e.finish(t, "human", "stopped", reason)
		stopped = append(stopped, id)
	}
	return stopped
}

// SetGroupVerdict 记下路由给任务组 gid 的最后汇总。
func (e *Engine) SetGroupVerdict(gid, text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	if g == nil {
		return errCode("unknownGroup")
	}
	g.Verdict = clipRunes(strings.TrimSpace(text), 4000)
	return nil
}

// Group 返回任务组 gid 的副本。
func (e *Engine) Group(gid string) (Group, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := e.groups[gid]
	if g == nil {
		return Group{}, false
	}
	return cloneGroup(g), true
}

// Groups 返回全部任务组的副本，按编号排序。
func (e *Engine) Groups() []Group {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []Group
	for _, id := range e.groupIDsLocked() {
		out = append(out, cloneGroup(e.groups[id]))
	}
	return out
}

// OpenGroupAgents 是还开着的任务组里出现过的 agent，yoyo 记成 yoyo。
func (e *Engine) OpenGroupAgents() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, g := range e.groups {
		if g == nil || g.Ended != nil {
			continue
		}
		for _, id := range g.Tickets {
			if t := e.tickets[id]; t != nil && !seen[agentKey(t.Agent)] {
				seen[agentKey(t.Agent)] = true
				out = append(out, agentKey(t.Agent))
			}
		}
	}
	sort.Strings(out)
	return out
}

// MemberDirs 是 ticketID 作为任务组成员另外能用的目录：shared 是任务组共享目录，可读写；
// reads 是前置工单的目录，只读。不是成员时都为空。
func (e *Engine) MemberDirs(ticketID string) (shared string, reads []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.tickets[ticketID]
	if t == nil || t.Group == "" {
		return "", nil
	}
	if g := e.groups[t.Group]; g != nil {
		shared = g.SharedDir
	}
	for _, id := range t.DependsOn {
		if dep := e.tickets[id]; dep != nil && strings.TrimSpace(dep.Directory) != "" {
			reads = append(reads, dep.Directory)
		}
	}
	return shared, reads
}

// groupIDsLocked 是任务组编号，按序号排序。调用时要已持有锁。
func (e *Engine) groupIDsLocked() []string {
	ids := make([]string, 0, len(e.groups))
	for id, g := range e.groups {
		if g != nil {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return groupOrder(ids[i]) < groupOrder(ids[j]) })
	return ids
}

// groupOrder 取出 grp- 后面的序号。认不出时放到后面。
func groupOrder(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "grp-"))
	if err != nil {
		return 1 << 30
	}
	return n
}

// cloneGroup 返回 g 的副本。切片单独拷贝。
func cloneGroup(g *Group) Group {
	out := *g
	out.Tickets = append([]string(nil), g.Tickets...)
	out.Woke = append([]string(nil), g.Woke...)
	if g.Ended != nil {
		end := *g.Ended
		out.Ended = &end
	}
	return out
}

// completedEnd 在 end 是做完（completed 或 merged）时为真。
func completedEnd(end *End) bool {
	return end != nil && (end.Outcome == "completed" || end.Outcome == "merged")
}

// failedReport 在 t 最近一份报告是 failed、还在等人处理时为真。
func failedReport(t *Ticket) bool {
	asg := latestAssignment(t)
	if asg == nil || len(asg.Reports) == 0 {
		return false
	}
	return asg.Reports[len(asg.Reports)-1].State == "failed" && WaitsForPerson(t)
}

// resultSummary 是 t 交出来的结果：最后一份 ready 的摘要，没有时用结束原因，再没有用最近的进展。
func resultSummary(t *Ticket) string {
	for i := len(t.Assignments) - 1; i >= 0; i-- {
		reports := t.Assignments[i].Reports
		for j := len(reports) - 1; j >= 0; j-- {
			if reports[j].State == "ready" && strings.TrimSpace(reports[j].Summary) != "" {
				return reports[j].Summary
			}
		}
	}
	if t.Ended != nil && strings.TrimSpace(t.Ended.Reason) != "" {
		return t.Ended.Reason
	}
	return progressSummary(t)
}

// memberLabel 是成员的一行称呼「tkt-21 Codex · 后端」。t 为空时只有编号。
func memberLabel(id string, t *Ticket, names map[string]string) string {
	if t == nil {
		return id
	}
	label := id + " " + memberName(t.Agent, names)
	if role := firstText(t.Role, t.Title); role != "" {
		label += " · " + role
	}
	return label
}

// memberName 是 agent 写进说明的名字。names 里没有时用编号，yoyo 自带工人是 yoyo。
func memberName(agent string, names map[string]string) string {
	key := agentKey(agent)
	if name := strings.TrimSpace(names[key]); name != "" {
		return name
	}
	return key
}

// deskSessionOf 是 tr 所在的桌面会话。不是桌面来的时为空串。
func deskSessionOf(tr *Trigger) string {
	if tr == nil {
		return ""
	}
	for _, key := range tr.Threads {
		if sid, ok := strings.CutPrefix(key, "desk:"); ok {
			return sid
		}
	}
	return ""
}

// firstText 返回第一个去掉空白后不为空的值。
func firstText(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// containsInt 在 list 里有 n 时为真。
func containsInt(list []int, n int) bool {
	for _, v := range list {
		if v == n {
			return true
		}
	}
	return false
}
