// group_runtime.go 是任务组在运行时这一侧的事：拆分时建共享目录和 NOTES.md；两秒轮次和每份报告之后看一遍任务组，
// 放行或挂起成员，状态变了就排一次唤醒路由（有次数上限，超过只写一句通知）；以及写给路由的任务组摘要和单张工单的详情。
// 不决定怎样拆、交给谁：那是路由的事，校验在 app 的 task_split 里。

package superagent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 唤醒路由的上限：每个任务组 groupWakeCap 次，每个会话每小时 sessionWakeCap 次。超过后只写一句通知，不再开路由回合。
const (
	groupWakeCap   = 6
	sessionWakeCap = 10
)

// digestMax 是任务组摘要最多几个字。
const digestMax = 3000

type wakeBody struct {
	Session string `json:"session"`
	Group   string `json:"group"`
	State   string `json:"state"`
	Mark    string `json:"mark"`
}

// SetOnWake 接上唤醒路由的回调。session 是发起任务组的桌面会话，可能为空；digest 是写给路由的任务组摘要。
// 回调返回 ErrLater 时两秒后再试，例如那条会话正在跑。
func (rt *Runtime) SetOnWake(fn func(session, group, state, digest string) error) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.onWake = fn
	rt.mu.Unlock()
}

// Split 开出任务组，建好共享目录和 NOTES.md，落盘后启动没有前置的成员。req.Root 为空时用执行根，req.Names 由这里填。
// 同一条触发再拆一次时返回原来的任务组。
func (rt *Runtime) Split(req SplitRequest) (Group, error) {
	if rt == nil || rt.eng == nil {
		return Group{}, errCode("empty")
	}
	var agents []string
	for _, p := range req.Parts {
		agents = append(agents, p.Agent)
	}
	req.Names = rt.agentNames(agents)
	rt.mu.Lock()
	if strings.TrimSpace(req.Root) == "" {
		req.Root = rt.workRoot
	}
	g, _, err := rt.eng.Split(req)
	if err != nil {
		rt.mu.Unlock()
		return Group{}, err
	}
	rt.saveLocked()
	effects := rt.effects
	rt.mu.Unlock()
	rt.writeNotes(g, req.Names)
	if effects != nil {
		effects.Pump(time.Now())
	}
	return g, nil
}

// writeNotes 建 g 的共享目录，还没有 NOTES.md 时写一份：标题、成员和分工、协作约定。没有共享目录时什么都不做。
func (rt *Runtime) writeNotes(g Group, names map[string]string) {
	if g.SharedDir == "" {
		return
	}
	if err := os.MkdirAll(g.SharedDir, 0o755); err != nil {
		return
	}
	path := filepath.Join(g.SharedDir, "NOTES.md")
	if _, err := os.Stat(path); err == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# 任务组 %s：%s\n\n", g.ID, g.Title)
	b.WriteString("分工协作。每份在自己的工单目录里做；别的成员要用的产物、接口和约定放在这个目录里。\n\n## 成员\n\n")
	for _, t := range rt.GroupMembers(g.ID) {
		line := "- " + memberLabel(t.ID, &t, names)
		if len(t.DependsOn) > 0 {
			line += "（等 " + strings.Join(t.DependsOn, "、") + " 做完再开始）"
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n## 约定\n\n- 开工前先读这份文件。\n- 做完在文件末尾加一节「## 工单编号 分工」，写清产出放在哪、接口和约定是什么。不要改别人写的段落。\n")
	_ = os.WriteFile(path, []byte(b.String()), 0o644)
}

// agentNames 把 agents 换成给人看的名字，键是 agent 编号，yoyo 自带工人是 yoyo。
// 名字回调在锁外调用：它会去读 Buddy，Buddy 又可能来读编排台。
func (rt *Runtime) agentNames(agents []string) map[string]string {
	out := map[string]string{}
	if rt == nil {
		return out
	}
	rt.mu.Lock()
	fn := rt.agentName
	rt.mu.Unlock()
	for _, agent := range agents {
		key := agentKey(agent)
		if _, ok := out[key]; ok {
			continue
		}
		name := ""
		if fn != nil && key != YoyoAgent {
			name = fn(key)
		}
		out[key] = firstText(name, key)
	}
	return out
}

// watchGroups 看一遍任务组：放行或挂起成员，状态变了就排唤醒，超过上限时改写一句通知。有改动时落一次盘。
func (rt *Runtime) watchGroups() {
	if rt == nil || rt.eng == nil {
		return
	}
	launches, wakes := rt.eng.WatchGroups(rt.agentNames(rt.eng.OpenGroupAgents()))
	if len(launches) == 0 && len(wakes) == 0 {
		return
	}
	now := time.Now()
	rt.mu.Lock()
	effects := rt.effects
	if rt.wakes == nil {
		rt.wakes = map[string][]time.Time{}
	}
	var capped []GroupWake
	for _, w := range wakes {
		recent := recentWakes(rt.wakes[w.Session], now)
		if w.Count > groupWakeCap || len(recent) >= sessionWakeCap {
			rt.wakes[w.Session] = recent
			capped = append(capped, w)
			continue
		}
		rt.wakes[w.Session] = append(recent, now)
		raw, _ := json.Marshal(wakeBody{Session: w.Session, Group: w.Group, State: w.State, Mark: w.Mark})
		if effects != nil {
			effects.Enqueue("wake", wakeKey(w.Session, w.Group, w.Mark), string(raw))
		}
	}
	rt.saveLocked()
	rt.mu.Unlock()
	for _, w := range capped {
		rt.capNotice(w)
	}
}

// recentWakes 是 times 里一小时之内的那些。
func recentWakes(times []time.Time, now time.Time) []time.Time {
	var out []time.Time
	for _, at := range times {
		if now.Sub(at) < time.Hour {
			out = append(out, at)
		}
	}
	return out
}

// wakeKey 是唤醒在副作用队列里的键。会话打头，PumpWake 按它找。
func wakeKey(session, group, mark string) string {
	return session + "\n" + group + "\n" + mark
}

// capNotice 在唤醒次数用完时给发起的会话写一句通知，不开路由回合。一个任务组超过上限后只写这一次。
func (rt *Runtime) capNotice(w GroupWake) {
	if rt.effects == nil || w.Session == "" || w.Count > groupWakeCap+1 {
		return
	}
	title := w.Group
	if g, ok := rt.eng.Group(w.Group); ok && g.Title != "" {
		title = g.Title
	}
	raw, _ := json.Marshal(noticeBody{Session: w.Session, Text: "任务组「" + title + "」停下了。自动汇总的次数用完了，去任务页看各成员的情况。"})
	rt.effects.Enqueue("notice", "group\n"+w.Group+"\n"+w.Mark, string(raw))
}

// PumpWake 让 session 上等着的唤醒马上再试一次。那条会话刚空下来时调用。
func (rt *Runtime) PumpWake(session string) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	effects := rt.effects
	rt.mu.Unlock()
	if effects != nil && effects.Hurry("wake", session+"\n") > 0 {
		effects.Pump(time.Now())
	}
}

// runWake 唤醒路由。任务组已经变了样（标记不是排队时那个）时作废，后来的那次唤醒会带上新状态。回调没接上时稍后再试。
func (rt *Runtime) runWake(payload any) error {
	raw, _ := payload.(string)
	var body wakeBody
	if json.Unmarshal([]byte(raw), &body) != nil || body.Group == "" || rt == nil || rt.eng == nil {
		return nil
	}
	if rt.eng.GroupMark(body.Group) != body.Mark {
		return nil
	}
	rt.mu.Lock()
	fn := rt.onWake
	rt.mu.Unlock()
	if fn == nil {
		return fmt.Errorf("%w", ErrLater)
	}
	return fn(body.Session, body.Group, body.State, rt.GroupDigest(body.Group))
}

// GroupMembers 返回任务组 gid 的成员工单副本，按拆分时的顺序。
func (rt *Runtime) GroupMembers(gid string) []Ticket {
	if rt == nil || rt.eng == nil {
		return nil
	}
	g, ok := rt.eng.Group(gid)
	if !ok {
		return nil
	}
	var out []Ticket
	for _, id := range g.Tickets {
		if t, ok := rt.eng.Ticket(id); ok {
			out = append(out, t)
		}
	}
	return out
}

// Group 返回任务组 gid 的副本。
func (rt *Runtime) Group(gid string) (Group, bool) {
	if rt == nil || rt.eng == nil {
		return Group{}, false
	}
	return rt.eng.Group(gid)
}

// GroupState 是任务组 gid 现在的状态。没有这个任务组时为空串。
func (rt *Runtime) GroupState(gid string) string {
	if rt == nil || rt.eng == nil {
		return ""
	}
	return rt.eng.GroupState(gid)
}

// SetGroupVerdict 记下路由给任务组 gid 的最后汇总，并落盘。
func (rt *Runtime) SetGroupVerdict(gid, text string) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	if err := rt.eng.SetGroupVerdict(gid, text); err != nil {
		return err
	}
	rt.save()
	return nil
}

// MemberDirs 是 ticketID 作为任务组成员另外能用的目录：共享目录可读写，前置工单的目录只读。不是成员时都为空。
func (rt *Runtime) MemberDirs(ticketID string) (shared string, reads []string) {
	if rt == nil || rt.eng == nil {
		return "", nil
	}
	return rt.eng.MemberDirs(ticketID)
}

// TicketGroup 返回 id 所在的任务组和它在组里的分工。不是任务组成员时都为空。
func (rt *Runtime) TicketGroup(id string) (group, role string) {
	if rt == nil || rt.eng == nil {
		return "", ""
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return "", ""
	}
	return t.Group, t.Role
}

// GroupDigest 是唤醒路由时写给它的任务组摘要：先说明这是任务组的回报，再每个成员一行，最后是这个状态下该怎么做。
// 最多 digestMax 个字，成员多时后面的成员省掉，行动规则总在。
func (rt *Runtime) GroupDigest(gid string) string {
	if rt == nil || rt.eng == nil {
		return ""
	}
	g, ok := rt.eng.Group(gid)
	if !ok {
		return ""
	}
	state := rt.eng.GroupState(gid)
	members := rt.GroupMembers(gid)
	var agents []string
	for _, t := range members {
		agents = append(agents, t.Agent)
	}
	names := rt.agentNames(agents)
	head := fmt.Sprintf("这是任务组 %s「%s」的回报，不是人说的话。%s，%d 份，现在%s。\n", g.ID, g.Title, groupModeText(g.Mode), len(g.Tickets), groupStateText(state))
	if g.SharedDir != "" {
		head += "共享目录 " + g.SharedDir + "\n"
	}
	rules := wakeRules(g.Mode, state, members)
	budget := digestMax - len([]rune(head)) - len([]rune(rules)) - 2
	var body strings.Builder
	now := time.Now()
	used := 0
	for _, t := range members {
		line := rt.memberDigest(t, names, now)
		n := len([]rune(line))
		if used+n > budget {
			body.WriteString("……\n")
			break
		}
		body.WriteString(line)
		used += n
	}
	return head + body.String() + rules
}

// memberDigest 是摘要里一个成员的一行：编号、agent、分工、状态、最近的话、目录、写过的文件、用时和用量。
func (rt *Runtime) memberDigest(t Ticket, names map[string]string, now time.Time) string {
	parts := []string{memberLabel(t.ID, &t, names), memberStateText(&t)}
	if s := oneParagraph(latestWords(&t)); s != "" {
		parts = append(parts, clipRunes(s, 200))
	}
	if dir := strings.TrimSpace(t.Directory); dir != "" {
		parts = append(parts, "目录 "+dir)
	}
	if files := TicketFiles(rt.WorkerTranscript(t.ID)); len(files) > 0 {
		if len(files) > 10 {
			files = files[:10]
		}
		parts = append(parts, "写过 "+strings.Join(files, "、"))
	}
	if !t.Opened.IsZero() {
		end := now
		if t.Ended != nil && !t.Ended.At.IsZero() {
			end = t.Ended.At
		}
		if d := end.Sub(t.Opened); d > 0 {
			parts = append(parts, "用时 "+strconv.Itoa(int(d.Minutes()+0.5))+" 分钟")
		}
	}
	if u := usageText(t.Usage); u != "" {
		parts = append(parts, u)
	}
	return "- " + strings.Join(parts, " · ") + "\n"
}

// wakeRules 是摘要末尾这个状态下路由该怎么做。
func wakeRules(mode, state string, members []Ticket) string {
	switch {
	case state == GroupDone && mode == GroupCompare:
		return "对比做完了。按完成度、正确性、改动大小、耗时和花费比较各份结果，要细节用 task_read 读那张工单。给出推荐，用 task_recommend 记下推荐的那一份，再问人要不要采用；人同意后用 task_pick 记下采用哪一份，要把结果合回项目时对采用的工单用 task_route continue。"
	case state == GroupDone:
		return "分工都做完了。用两三句话告诉人做成了什么、东西在哪、下一步可以做什么。不要再派工，除非人之前要求过。"
	}
	asked, failed := false, false
	for i := range members {
		t := &members[i]
		if t.Ended != nil {
			if !completedEnd(t.Ended) && !(t.Ended.By == "human" && t.Ended.Outcome == "stopped") {
				failed = true
			}
			continue
		}
		if t.Held == HeldDepsFailed {
			failed = true
			continue
		}
		switch TicketStage(t).Action {
		case "answer", "review":
			asked = true
		case "blocked", "failed", "delivery":
			failed = true
		}
	}
	var b strings.Builder
	if asked {
		b.WriteString("有成员在等人回答或验收，它的话已经单独写给了人。不要重复问题，也不要替人回答；只说明这会挡住哪些成员。")
	}
	if failed {
		b.WriteString("有成员没做成：说清是哪一份、原因，再给人三个选项并给出建议：重试（对那张工单用 task_route continue）、换 agent（continue 时填 agent）、停掉整组（task_stop 填 group）。没问过人时最多重试一次。")
	}
	if b.Len() == 0 {
		b.WriteString("用一两句话告诉人任务组现在的情况。")
	}
	return b.String()
}

// TicketRead 是 task_read 读到的一张工单。State 是给路由看的状态，Text 是工人最后一段话，
// Report 是最后一份不是进度的报告，Files 是写过的文件，新的在前，Usage 是用量的说法，没有时为空。
type TicketRead struct {
	ID        string
	Agent     string
	Title     string
	Role      string
	Group     string
	Directory string
	State     string
	Text      string
	Report    *Report
	Files     []string
	Usage     string
}

// ReadTicket 读出 id 这张工单给路由比较用的详情：最后一段话最多 2000 字，写过的文件最多 20 个。没有这张工单时 ok 为假。
func (rt *Runtime) ReadTicket(id string) (TicketRead, bool) {
	if rt == nil || rt.eng == nil {
		return TicketRead{}, false
	}
	t, ok := rt.eng.Ticket(id)
	if !ok {
		return TicketRead{}, false
	}
	out := TicketRead{
		ID: t.ID, Agent: agentKey(t.Agent), Title: firstText(t.Title, rt.TicketLabel(id)), Role: t.Role, Group: t.Group,
		Directory: t.Directory, State: memberStateText(&t), Usage: usageText(t.Usage),
	}
	lines := rt.WorkerTranscript(id)
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].Role == "assistant" && strings.TrimSpace(lines[i].Text) != "" {
			out.Text = clipRunes(strings.TrimSpace(lines[i].Text), 2000)
			break
		}
	}
	if rep, ok := lastReport(&t); ok {
		out.Report = &rep
	}
	out.Files = TicketFiles(lines)
	if len(out.Files) > 20 {
		out.Files = out.Files[:20]
	}
	return out, true
}

// lastReport 是 t 最后一份不是进度的报告。
func lastReport(t *Ticket) (Report, bool) {
	for i := len(t.Assignments) - 1; i >= 0; i-- {
		reports := t.Assignments[i].Reports
		for j := len(reports) - 1; j >= 0; j-- {
			if reports[j].State != "progress" {
				return reports[j], true
			}
		}
	}
	return Report{}, false
}

// latestWords 是 t 最近该给路由看的一句：最后一份不是进度的报告的摘要和问题，没有时用结束原因或进展。
func latestWords(t *Ticket) string {
	if rep, ok := lastReport(t); ok {
		text := strings.TrimSpace(rep.Summary)
		if q := strings.TrimSpace(rep.Question); q != "" && !strings.Contains(text, q) {
			text = strings.TrimSpace(text + " " + q)
		}
		if text != "" {
			return text
		}
	}
	return resultSummary(t)
}

// memberStateText 是成员给路由看的状态。
func memberStateText(t *Ticket) string {
	if t.Ended != nil {
		switch t.Ended.Outcome {
		case "completed", "merged":
			return "做完了"
		case "stopped":
			return "被停下"
		case "dropped":
			return "放下了"
		default:
			return "没做成"
		}
	}
	switch t.Held {
	case HeldDeps:
		return "等前置"
	case HeldDepsFailed:
		return "前置没做成，挂着"
	}
	st := TicketStage(t)
	switch st.Action {
	case "answer":
		return "在等人回答"
	case "review":
		return "等验收"
	case "blocked":
		return "卡住了"
	case "failed":
		return "没做成，等人处理"
	case "delivery":
		return "结果没送出"
	case "queued":
		return "排队"
	}
	if st.Name == "idle" {
		return "停着"
	}
	return "在做"
}

// groupModeText 是任务组模式的说法。
func groupModeText(mode string) string {
	if mode == GroupCompare {
		return "对比"
	}
	return "分工"
}

// groupStateText 是任务组状态的说法。
func groupStateText(state string) string {
	switch state {
	case GroupAttention:
		return "有成员要人处理"
	case GroupDone:
		return "都结束了"
	case GroupFailed:
		return "有成员没做成"
	case GroupStopped:
		return "停掉了"
	default:
		return "还在做"
	}
}

// usageText 是用量的说法「上下文 53k/200k · 0.05 USD」。没有用量时为空串。
func usageText(u *Usage) string {
	if u == nil {
		return ""
	}
	var parts []string
	if u.Size > 0 {
		parts = append(parts, "上下文 "+kTokens(u.Used)+"/"+kTokens(u.Size))
	}
	if u.Cost > 0 {
		parts = append(parts, strconv.FormatFloat(u.Cost, 'f', 2, 64)+" "+firstText(u.Currency, "USD"))
	}
	return strings.Join(parts, " · ")
}

// kTokens 把 token 数写成 53k 这样。不到一千时照写。
func kTokens(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	return strconv.Itoa((n+500)/1000) + "k"
}

// oneParagraph 把 text 的换行收成空格。
func oneParagraph(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
