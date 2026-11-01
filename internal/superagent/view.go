// view.go 把引擎快照收成界面要的几列。
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

// ProblemRow 是总览上要人看的一条故障。Kind 取 source、unavailable、outgoing。
type ProblemRow struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Detail     string `json:"detail,omitempty"`
	TriggerID  string `json:"triggerId,omitempty"`
	DecisionID string `json:"decisionId,omitempty"`
}

// ThreadRow 是线程一行。Queued 是还没评估的触发数。Subs 是订阅它的工单数。
type ThreadRow struct {
	Key    string `json:"key"`
	Queued int    `json:"queued"`
	Subs   int    `json:"subs"`
}

// TriggerRow 是排队触发的一行。Text 最多保留一段给人看的正文。
type TriggerRow struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Sender    string `json:"sender"`
	Text      string `json:"text"`
	Addressed bool   `json:"addressed"`
}

// DecisionRow 是一条评估决定。Dismissed 为真时人已经把它从待处理里拿掉。
type DecisionRow struct {
	ID        string `json:"id"`
	TriggerID string `json:"triggerId"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
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
	sort.Slice(view.Tickets, func(i, j int) bool { return view.Tickets[i].ID < view.Tickets[j].ID })
	now := time.Now().UTC()
	for _, th := range snap.Threads {
		view.Threads = append(view.Threads, ThreadRow{Key: th.Key, Queued: len(th.Queued), Subs: len(th.Subs)})
		if th.Evaluating == nil {
			continue
		}
		row := EvalRow{
			ID: th.Evaluating.ID, Thread: th.Key, Strength: th.Evaluating.Strength,
			Delivered: !th.Evaluating.Delivered.IsZero(), Triggers: len(th.Evaluating.TriggerIDs),
		}
		if row.Delivered {
			row.Deadline = th.Evaluating.Deadline.UTC().Format(time.RFC3339)
			if !th.Evaluating.Deadline.After(now) {
				view.Problems = append(view.Problems, ProblemRow{
					Kind: "overdue", ID: row.ID, Title: th.Key, Detail: row.Deadline, TriggerID: th.Evaluating.Primary,
				})
			}
		}
		view.Evals = append(view.Evals, row)
	}
	sort.Slice(view.Threads, func(i, j int) bool { return view.Threads[i].Key < view.Threads[j].Key })
	for _, tr := range snap.Triggers {
		if !tr.Queued {
			continue
		}
		view.Queued = append(view.Queued, TriggerRow{
			ID: tr.ID, Source: tr.Source, Sender: tr.Sender, Text: clip(tr.Text, 240), Addressed: tr.Addressed,
		})
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
			ID: d.ID, TriggerID: d.TriggerID, Kind: d.Kind, Reason: d.Reason, TicketID: d.TicketID, Dismissed: d.Dismissed, At: at,
		})
		if d.Kind == "unavailable" && !d.Dismissed {
			view.Problems = append(view.Problems, ProblemRow{
				Kind: "unavailable", ID: d.ID, Title: d.TriggerID, Detail: d.Reason, TriggerID: d.TriggerID, DecisionID: d.ID,
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
			view.Problems = append(view.Problems, ProblemRow{Kind: "outgoing", ID: msg.ID, Title: msg.Source, Detail: clip(msg.Text, 160)})
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
