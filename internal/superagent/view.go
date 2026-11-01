// view.go 把引擎快照收成编排台的几页：标题、线程消息、最近一天的流量和执行负载。
// 不改引擎。Runtime.View 调用它。

package superagent

import (
	"sort"
	"strings"
	"time"
)

// View 是编排台一次快照。时间在 At。Health 为 ready，或带上最近一次失败。
type View struct {
	Mode        string         `json:"mode"`
	Paused      bool           `json:"paused"`
	Version     int            `json:"version"`
	Admission   AdmissionView  `json:"admission"`
	Sources     []SourceState  `json:"sources"`
	Tickets     []TicketRow    `json:"tickets"`
	Threads     []ThreadRow    `json:"threads"`
	Queued      []TriggerRow   `json:"queued"`
	Evals       []EvalRow      `json:"evals"`
	Decisions   []DecisionRow  `json:"decisions"`
	Executions  []ExecutionRow `json:"executions"`
	Problems    []ProblemRow   `json:"problems"`
	Allowed     []string       `json:"allowed"`
	Workers     int            `json:"workers"`
	Health      string         `json:"health"`
	At          string         `json:"at"`
	EvalTimeout int            `json:"evalTimeout"`
	// Flow 是最近一天的到达、滤掉、已评、接单和忽略。没有到达时刻的旧触发不计入。
	Flow FlowView `json:"flow"`
	// Load 是执行器正在跑、排队和空闲的工单数。MaxWorkers 是普通槽位。
	Load LoadView `json:"load"`
}

// FlowView 是最近二十四小时的流量。Received 含 Filtered。Ignored 是这段里被忽略的决定。
type FlowView struct {
	Received int `json:"received"`
	Filtered int `json:"filtered"`
	Judged   int `json:"judged"`
	Accepted int `json:"accepted"`
	Ignored  int `json:"ignored"`
}

// LoadView 是执行器负载。Running 是正在做的工单，Queued 是还在排队的，Idle 是在等下一条触发的。
type LoadView struct {
	Running    int `json:"running"`
	Queued     int `json:"queued"`
	Idle       int `json:"idle"`
	MaxWorkers int `json:"maxWorkers"`
}

// AdmissionView 是填好默认值的三格准入。
type AdmissionView struct {
	ListedUnaddressed   string `json:"listedUnaddressed"`
	UnlistedAddressed   string `json:"unlistedAddressed"`
	UnlistedUnaddressed string `json:"unlistedUnaddressed"`
}

// SourceState 是一个来源现在的状态。State 取 unconfigured、ok 或 error。HasSecret 为真表示凭证已保存，响应里不带回密钥。
type SourceState struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Detail    string `json:"detail"`
	Repo      string `json:"repo,omitempty"`
	Chat      string `json:"chat,omitempty"`
	App       string `json:"app,omitempty"`
	API       string `json:"api,omitempty"`
	HasSecret bool   `json:"hasSecret,omitempty"`
}

// TicketRow 是工单一行。Stage 是阶段名，Action 是这一阶段在等的动作。
type TicketRow struct {
	ID            string          `json:"id"`
	Stage         string          `json:"stage"`
	Action        string          `json:"action"`
	Person        bool            `json:"person"`
	Origin        string          `json:"origin"`
	Outcome       string          `json:"outcome,omitempty"`
	Strength      string          `json:"strength,omitempty"`
	Sender        string          `json:"sender,omitempty"`
	Attention     string          `json:"attention,omitempty"`
	DeliveryError string          `json:"deliveryError,omitempty"`
	OriginURL     string          `json:"originUrl,omitempty"`
	Title         string          `json:"title"`
	Inputs        []InputRow      `json:"inputs"`
	Follows       []FollowRow     `json:"follows"`
	Assignments   []AssignmentRow `json:"assignments"`
}

// InputRow 是工单上的一条输入。Delivered 为假表示执行器还没领走。
type InputRow struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Human     bool   `json:"human"`
	Delivered bool   `json:"delivered"`
}

// FollowRow 是工单跟随的线程。Closed 非空表示这条线程已经关闭。
type FollowRow struct {
	Thread string `json:"thread"`
	Own    bool   `json:"own"`
	Closed string `json:"closed,omitempty"`
}

// AssignmentRow 是一次派工的摘要。Summary 是最近一次报告的正文。
type AssignmentRow struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Report  string `json:"report,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// ProblemRow 是总览上要人看的一条故障。Kind 取 source、unavailable、outgoing、overdue。
type ProblemRow struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Detail     string `json:"detail,omitempty"`
	TriggerID  string `json:"triggerId,omitempty"`
	DecisionID string `json:"decisionId,omitempty"`
	Thread     string `json:"thread,omitempty"`
	TicketID   string `json:"ticketId,omitempty"`
}

// ThreadRow 是线程一行。Queued 是还没评估的触发数。Subs 是订阅它的工单数。
// Title 是最近一条正文。Items 是这条线程上的触发，按到达顺序，最多四十条。
type ThreadRow struct {
	Key        string       `json:"key"`
	Title      string       `json:"title"`
	Queued     int          `json:"queued"`
	Subs       int          `json:"subs"`
	Followers  []string     `json:"followers"`
	Evaluating bool         `json:"evaluating"`
	Recent     bool         `json:"recent"`
	At         string       `json:"at,omitempty"`
	Sender     string       `json:"sender,omitempty"`
	Items      []TriggerRow `json:"items"`
}

// TriggerRow 是一条触发。Text 最多保留一段给人看的正文。Filtered 非空表示这条没有进入评估。
type TriggerRow struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Sender    string `json:"sender"`
	Text      string `json:"text"`
	Addressed bool   `json:"addressed"`
	Thread    string `json:"thread,omitempty"`
	At        string `json:"at,omitempty"`
	Filtered  string `json:"filtered,omitempty"`
	Queued    bool   `json:"queued,omitempty"`
}

// DecisionRow 是一条评估决定。Dismissed 为真时人已经把它从待处理里拿掉。
type DecisionRow struct {
	ID        string `json:"id"`
	TriggerID string `json:"triggerId"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
	Thread    string `json:"thread,omitempty"`
	TicketID  string `json:"ticketId,omitempty"`
	Dismissed bool   `json:"dismissed,omitempty"`
	At        string `json:"at,omitempty"`
}

// EvalRow 是一轮还没结束的评估。Deadline 是 RFC3339。
type EvalRow struct {
	ID        string `json:"id"`
	Thread    string `json:"thread"`
	Strength  string `json:"strength"`
	Deadline  string `json:"deadline,omitempty"`
	Delivered bool   `json:"delivered"`
	Triggers  int    `json:"triggers"`
}

// ExecutionRow 是一次派工。Report 是最近一次报告的状态。
type ExecutionRow struct {
	TicketID     string `json:"ticketId"`
	AssignmentID string `json:"assignmentId"`
	State        string `json:"state"`
	Report       string `json:"report,omitempty"`
}

// buildView 用 snap、sources、health、at 和 workers 拼一份快照。sources 按名字排序。工单和线程也按名字排序。
func buildView(snap Snapshot, sources []SourceState, health, at string, workers int) View {
	if health == "" {
		health = "ready"
	}
	if workers < 1 {
		workers = 3
	}
	adm := snap.Policy.Admission.Resolved()
	view := View{
		Mode:    snap.Policy.Mode,
		Paused:  snap.Policy.Paused,
		Version: snap.Policy.Version,
		Admission: AdmissionView{
			ListedUnaddressed:   adm.ListedUnaddressed,
			UnlistedAddressed:   adm.UnlistedAddressed,
			UnlistedUnaddressed: adm.UnlistedUnaddressed,
		},
		Sources:     append([]SourceState{}, sources...),
		Tickets:     []TicketRow{},
		Threads:     []ThreadRow{},
		Queued:      []TriggerRow{},
		Evals:       []EvalRow{},
		Decisions:   []DecisionRow{},
		Executions:  []ExecutionRow{},
		Problems:    []ProblemRow{},
		Allowed:     allowedNames(snap.Policy.Allowed),
		Workers:     workers,
		Load:        LoadView{MaxWorkers: workers},
		EvalTimeout: int(snap.Policy.evalTimeout() / time.Second),
		Health:      health,
		At:          at,
	}
	sort.Slice(view.Sources, func(i, j int) bool { return view.Sources[i].Name < view.Sources[j].Name })
	byID := map[string]Trigger{}
	for _, tr := range snap.Triggers {
		byID[tr.ID] = tr
	}
	for i := range snap.Tickets {
		ticket := snap.Tickets[i]
		stage := TicketStage(&ticket)
		row := ticketRow(ticket, stage, byID)
		view.Tickets = append(view.Tickets, row)
		for _, asg := range row.Assignments {
			view.Executions = append(view.Executions, ExecutionRow{
				TicketID: ticket.ID, AssignmentID: asg.ID, State: asg.State, Report: asg.Report,
			})
		}
	}
	sort.Slice(view.Tickets, func(i, j int) bool {
		ri, rj := stageRank(view.Tickets[i].Stage), stageRank(view.Tickets[j].Stage)
		if ri != rj {
			return ri < rj
		}
		return view.Tickets[i].ID < view.Tickets[j].ID
	})
	for _, row := range view.Tickets {
		switch {
		case row.Stage == "working" && row.Action == "queued":
			view.Load.Queued++
		case row.Stage == "working":
			view.Load.Running++
		case row.Stage == "idle":
			view.Load.Idle++
		}
	}
	now := time.Now().UTC()
	since := now.Add(-24 * time.Hour)
	for _, th := range snap.Threads {
		items := threadItems(th.Context, byID)
		row := ThreadRow{
			Key: th.Key, Title: threadTitle(th, byID), Queued: len(th.Queued), Subs: len(th.Subs),
			Followers: append([]string{}, th.Subs...), Evaluating: th.Evaluating != nil, Items: items,
		}
		if row.Followers == nil {
			row.Followers = []string{}
		}
		for _, item := range items {
			if item.At >= row.At {
				row.At = item.At
				row.Sender = item.Sender
			}
			if inDay(item.At, since) {
				row.Recent = true
			}
		}
		view.Threads = append(view.Threads, row)
		if th.Evaluating == nil {
			continue
		}
		eval := EvalRow{
			ID: th.Evaluating.ID, Thread: th.Key, Strength: th.Evaluating.Strength,
			Delivered: !th.Evaluating.Delivered.IsZero(), Triggers: len(th.Evaluating.TriggerIDs),
		}
		if eval.Delivered {
			eval.Deadline = th.Evaluating.Deadline.UTC().Format(time.RFC3339)
			if !th.Evaluating.Deadline.After(now) {
				view.Problems = append(view.Problems, ProblemRow{
					Kind: "overdue", ID: eval.ID, Title: row.Title, Detail: eval.Deadline,
					TriggerID: th.Evaluating.Primary, Thread: th.Key,
				})
			}
		}
		view.Evals = append(view.Evals, eval)
	}
	sort.Slice(view.Threads, func(i, j int) bool {
		if view.Threads[i].At != view.Threads[j].At {
			return view.Threads[i].At > view.Threads[j].At
		}
		return view.Threads[i].Key < view.Threads[j].Key
	})
	for _, tr := range snap.Triggers {
		if inDay(tr.At.UTC().Format(time.RFC3339), since) {
			view.Flow.Received++
			if tr.Filtered != "" {
				view.Flow.Filtered++
			}
		}
		if !tr.Queued {
			continue
		}
		view.Queued = append(view.Queued, triggerRow(tr))
	}
	sort.Slice(view.Queued, func(i, j int) bool { return view.Queued[i].ID < view.Queued[j].ID })
	decisions := snap.Decisions
	if len(decisions) > 80 {
		decisions = decisions[len(decisions)-80:]
	}
	for _, d := range decisions {
		at := ""
		if !d.At.IsZero() {
			at = d.At.UTC().Format(time.RFC3339)
		}
		view.Decisions = append(view.Decisions, DecisionRow{
			ID: d.ID, TriggerID: d.TriggerID, Kind: d.Kind, Reason: d.Reason, Thread: d.Thread,
			TicketID: d.TicketID, Dismissed: d.Dismissed, At: at,
		})
		if inDay(at, since) {
			view.Flow.Judged++
			if d.Kind == "accept" {
				view.Flow.Accepted++
			}
			if d.Kind == "ignore" {
				view.Flow.Ignored++
			}
		}
		if d.Kind == "unavailable" && !d.Dismissed {
			title := d.TriggerID
			if tr, ok := byID[d.TriggerID]; ok {
				if line := oneLine(tr.Text); line != "" {
					title = line
				}
			}
			view.Problems = append(view.Problems, ProblemRow{
				Kind: "unavailable", ID: d.ID, Title: title, Detail: d.Reason,
				TriggerID: d.TriggerID, DecisionID: d.ID, Thread: d.Thread,
			})
		}
	}
	for _, src := range view.Sources {
		if src.State == "error" {
			view.Problems = append(view.Problems, ProblemRow{Kind: "source", ID: src.Name, Title: src.Name, Detail: src.Detail})
		}
	}
	for _, msg := range snap.Outbox {
		if msg.Status == "error" {
			view.Problems = append(view.Problems, ProblemRow{
				Kind: "outgoing", ID: msg.ID, Title: msg.Source, Detail: clip(msg.Text, 160), TicketID: msg.TicketID,
			})
		}
	}
	return view
}

// ticketRow 把 ticket 收成界面一行。stage 提供阶段，byID 用来找起点触发的来源地址。
func ticketRow(ticket Ticket, stage Stage, byID map[string]Trigger) TicketRow {
	undelivered := map[string]bool{}
	for _, id := range ticket.Undelivered {
		undelivered[id] = true
	}
	row := TicketRow{
		ID: ticket.ID, Stage: stage.Name, Action: stage.Action, Person: stage.Person, Origin: ticket.Origin,
		Strength: ticket.Mandate.Strength, Sender: ticket.Mandate.Sender, Attention: ticket.Attention,
		DeliveryError: ticket.DeliveryErr, Inputs: []InputRow{}, Follows: []FollowRow{}, Assignments: []AssignmentRow{},
	}
	if ticket.Ended != nil {
		row.Outcome = ticket.Ended.Outcome
	}
	if origin, ok := byID[ticket.Origin]; ok {
		row.OriginURL = originURL(origin)
		row.Title = oneLine(origin.Text)
	}
	if row.Title == "" && len(ticket.Inputs) > 0 {
		row.Title = oneLine(ticket.Inputs[0].Text)
	}
	if row.Title == "" {
		row.Title = ticket.ID
	}
	for _, in := range ticket.Inputs {
		row.Inputs = append(row.Inputs, InputRow{ID: in.ID, Text: clip(in.Text, 240), Human: in.Human, Delivered: !undelivered[in.ID]})
	}
	for _, f := range ticket.Follows {
		item := FollowRow{Thread: f.Thread, Own: f.Own}
		if f.Closed != nil {
			item.Closed = f.Closed.Outcome
		}
		row.Follows = append(row.Follows, item)
	}
	for _, asg := range ticket.Assignments {
		item := AssignmentRow{ID: asg.ID, State: asg.State}
		if len(asg.Reports) > 0 {
			last := asg.Reports[len(asg.Reports)-1]
			item.Report = last.State
			item.Summary = clip(last.Summary, 240)
		}
		row.Assignments = append(row.Assignments, item)
	}
	return row
}

// originURL 把 GitHub 议题或拉取请求收成可打开的地址。其他来源返回空串。
func originURL(tr Trigger) string {
	if tr.Source != "github" || tr.Ref == "" {
		return ""
	}
	repo := strings.TrimPrefix(tr.Primary, "github:")
	repo = strings.TrimSuffix(repo, "/pulls")
	kind := "issues"
	if strings.Contains(tr.ID, "!pr#") {
		kind = "pull"
	}
	return "https://github.com/" + repo + "/" + kind + "/" + tr.Ref
}

// allowedNames 返回允许发送者，按名字排序。allowed 为空时返回空切片。
func allowedNames(allowed map[string]bool) []string {
	out := []string{}
	for name, ok := range allowed {
		if ok && name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// clip 把 text 收成最多 n 个字节。n 小于 1 时返回空串。
func clip(text string, n int) string {
	if n < 1 || len(text) <= n {
		return text
	}
	return text[:n]
}

// clipRunes 把 text 收成最多 n 个字符。n 小于 1 时返回空串。
func clipRunes(text string, n int) string {
	if n < 1 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n])
}

// oneLine 取 text 的第一行，最多八十个字符。text 空白时返回空串。
func oneLine(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = text[:i]
	}
	return clipRunes(strings.TrimSpace(text), 80)
}

// stageRank 给 stage 一个总览排序。person 在前，ended 在后。不认识的阶段排在最后。
func stageRank(stage string) int {
	switch stage {
	case "person":
		return 0
	case "working":
		return 1
	case "idle":
		return 2
	case "ended":
		return 3
	default:
		return 4
	}
}

// inDay 判断 at 是否不早于 since。at 为空时为假。
func inDay(at string, since time.Time) bool {
	if at == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return false
	}
	return !parsed.Before(since)
}

// threadTitle 用 th 最近一条有正文的触发当标题。都没有时用 th.Key。byID 用来取正文。
func threadTitle(th Thread, byID map[string]Trigger) string {
	for i := len(th.Context) - 1; i >= 0; i-- {
		tr, ok := byID[th.Context[i]]
		if !ok {
			continue
		}
		if line := oneLine(tr.Text); line != "" {
			return line
		}
	}
	return th.Key
}

// threadItems 按 ids 的顺序收触发。byID 里没有的编号跳过。超过四十条时只留最后四十条。
func threadItems(ids []string, byID map[string]Trigger) []TriggerRow {
	rows := []TriggerRow{}
	for _, id := range ids {
		tr, ok := byID[id]
		if !ok {
			continue
		}
		rows = append(rows, triggerRow(tr))
	}
	if len(rows) > 40 {
		rows = rows[len(rows)-40:]
	}
	return rows
}

// triggerRow 把 tr 收成界面一行。正文按字符截断。
func triggerRow(tr Trigger) TriggerRow {
	at := ""
	if !tr.At.IsZero() {
		at = tr.At.UTC().Format(time.RFC3339)
	}
	thread := tr.Primary
	if thread == "" && len(tr.Threads) > 0 {
		thread = tr.Threads[0]
	}
	return TriggerRow{
		ID: tr.ID, Source: tr.Source, Sender: tr.Sender, Text: clipRunes(strings.TrimSpace(tr.Text), 240),
		Addressed: tr.Addressed, Thread: thread, At: at, Filtered: tr.Filtered, Queued: tr.Queued,
	}
}
