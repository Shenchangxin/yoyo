// view.go 把引擎快照收成界面要的几列。
// 不改引擎。Runtime.View 调用它。

package superagent

import "sort"

// View 是编排台一次快照。时间在 At。Health 为 ready，或带上最近一次失败。
type View struct {
	Mode      string        `json:"mode"`
	Paused    bool          `json:"paused"`
	Version   int           `json:"version"`
	Admission AdmissionView `json:"admission"`
	Sources   []SourceState `json:"sources"`
	Tickets   []TicketRow   `json:"tickets"`
	Threads   []ThreadRow   `json:"threads"`
	Queued    []TriggerRow  `json:"queued"`
	Decisions []DecisionRow `json:"decisions"`
	Health    string        `json:"health"`
	At        string        `json:"at"`
}

// AdmissionView 是填好默认值的三格准入。
type AdmissionView struct {
	ListedUnaddressed   string `json:"listedUnaddressed"`
	UnlistedAddressed   string `json:"unlistedAddressed"`
	UnlistedUnaddressed string `json:"unlistedUnaddressed"`
}

// SourceState 是一个来源现在的状态。State 取 unconfigured、ok 或 error。
type SourceState struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}

// TicketRow 是工单一行。Stage 是阶段名，Action 是这一阶段在等的动作。
type TicketRow struct {
	ID      string `json:"id"`
	Stage   string `json:"stage"`
	Action  string `json:"action"`
	Person  bool   `json:"person"`
	Origin  string `json:"origin"`
	Outcome string `json:"outcome,omitempty"`
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

// DecisionRow 是一条评估决定。
type DecisionRow struct {
	ID        string `json:"id"`
	TriggerID string `json:"triggerId"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
	TicketID  string `json:"ticketId,omitempty"`
}

// buildView 用 snap、sources、health 和 at 拼一份快照。sources 按名字排序。工单和线程也按名字排序。
func buildView(snap Snapshot, sources []SourceState, health, at string) View {
	if health == "" {
		health = "ready"
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
		Sources:   append([]SourceState{}, sources...),
		Tickets:   []TicketRow{},
		Threads:   []ThreadRow{},
		Queued:    []TriggerRow{},
		Decisions: []DecisionRow{},
		Health:    health,
		At:        at,
	}
	sort.Slice(view.Sources, func(i, j int) bool { return view.Sources[i].Name < view.Sources[j].Name })
	for i := range snap.Tickets {
		ticket := snap.Tickets[i]
		stage := TicketStage(&ticket)
		row := TicketRow{ID: ticket.ID, Stage: stage.Name, Action: stage.Action, Person: stage.Person, Origin: ticket.Origin}
		if ticket.Ended != nil {
			row.Outcome = ticket.Ended.Outcome
		}
		view.Tickets = append(view.Tickets, row)
	}
	sort.Slice(view.Tickets, func(i, j int) bool { return view.Tickets[i].ID < view.Tickets[j].ID })
	for _, th := range snap.Threads {
		view.Threads = append(view.Threads, ThreadRow{Key: th.Key, Queued: len(th.Queued), Subs: len(th.Subs)})
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
	if len(decisions) > 40 {
		decisions = decisions[len(decisions)-40:]
	}
	for _, d := range decisions {
		view.Decisions = append(view.Decisions, DecisionRow{
			ID: d.ID, TriggerID: d.TriggerID, Kind: d.Kind, Reason: d.Reason, TicketID: d.TicketID,
		})
	}
	return view
}

// clip 把 text 收成最多 n 个字节。n 小于 1 时返回空串。
func clip(text string, n int) string {
	if n < 1 || len(text) <= n {
		return text
	}
	return text[:n]
}
