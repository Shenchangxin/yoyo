// presence.go 是头像的全局状态源：导航轨上的头像、桌面伴侣和托盘读同一个状态。
// 状态从已经有的东西里收：暂停、要人批准的动作、在等人的工单（含「最后一下交给你」）、受阻的工单、在做的工单和回合、刚做完的工单。
// 每个状态带一句给人看的话，以及点开时要看的那张工单或那条会话：悬停看这句话，点开在右侧预览那张工单，预览里再放到中间接管。
// 只读已经打开的编排；跟着别的 yoyo 进程时读领头进程写下的那一份。不为了算状态去开数据库或叫模型。

package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// presenceDoneFor 是做完的工单在头像上留多久。
const presenceDoneFor = 10 * time.Minute

// Presence 是头像此刻的状态。State 取 paused、needs、blocked、working、done、idle。
// Ticket 是点开要看的工单，Session 是点开要看的会话（只有批准时有）。Count 是这一类有几件。
type Presence struct {
	State   string `json:"state"`
	Line    string `json:"line"`
	Ticket  string `json:"ticket,omitempty"`
	Session string `json:"session,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// presenceInput 是算头像状态的材料。
type presenceInput struct {
	Paused          bool
	Approvals       int
	ApprovalSession string
	Running         []string
	Tickets         []superagent.TicketRow
	Now             time.Time
	Zh              bool
}

// Presence 返回头像此刻的状态。
func (a *App) Presence() Presence {
	if a == nil {
		return Presence{State: "idle"}
	}
	in := presenceInput{Paused: a.IsPaused(), Running: a.RunningIDs(), Now: time.Now(), Zh: a.briefChinese()}
	if a.Gate != nil {
		offers := a.Gate.Pending()
		in.Approvals = len(offers)
		if len(offers) > 0 {
			in.ApprovalSession = offers[0].Request.SessionID
		}
	}
	if rt := a.liveOrch(); rt != nil {
		in.Tickets = rt.View().Tickets
	} else if a.managed() && !a.leading() {
		in.Tickets = a.followView().Tickets
	}
	return presenceOf(in)
}

// presenceOf 按暂停、等你、受阻、在做、刚做完、空闲的顺序取状态。
func presenceOf(in presenceInput) Presence {
	zh := in.Zh
	if in.Paused {
		return Presence{State: "paused", Line: pick(zh, "暂停中：例行、任务和主动提醒都停着", "Paused: routines, tasks and notes are on hold")}
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	var needs, blocked, working []superagent.TicketRow
	var done *superagent.TicketRow
	for i := range in.Tickets {
		t := in.Tickets[i]
		switch {
		case t.Stage == "ended":
			if endedWell(t) && recentAt(t.UpdatedAt, now, presenceDoneFor) && (done == nil || t.UpdatedAt > done.UpdatedAt) {
				row := t
				done = &row
			}
		case t.Person && (t.Action == "failed" || t.Action == "delivery" || t.Held == "deps_failed"):
			blocked = append(blocked, t)
		case t.Person:
			needs = append(needs, t)
		case t.Stage == "working":
			working = append(working, t)
		}
	}
	if n := in.Approvals + len(needs); n > 0 {
		p := Presence{State: "needs", Count: n}
		if in.Approvals > 0 {
			p.Session = in.ApprovalSession
			p.Line = pick(zh, fmt.Sprintf("有 %d 件要你批准", in.Approvals), fmt.Sprintf("%d approvals waiting", in.Approvals))
			if len(needs) > 0 {
				p.Line += pick(zh, fmt.Sprintf("，另有 %d 件任务在等你", len(needs)), fmt.Sprintf(", %d tasks waiting", len(needs)))
			}
			return p
		}
		t := needs[0]
		p.Ticket = t.ID
		title := presenceTitle(t.Title, zh)
		if line, ok := lastReport(t); ok && line.Finish != nil {
			p.Line = pick(zh, title+"最后一步交给你", "The last step of "+title+" is yours")
		} else {
			p.Line = pick(zh, title+"在等你", title+" is waiting for you")
		}
		if len(needs) > 1 {
			p.Line += pick(zh, fmt.Sprintf("（共 %d 件）", len(needs)), fmt.Sprintf(" (%d in all)", len(needs)))
		}
		return p
	}
	if len(blocked) > 0 {
		t := blocked[0]
		return Presence{State: "blocked", Ticket: t.ID, Count: len(blocked),
			Line: pick(zh, presenceTitle(t.Title, zh)+"没做成，要你看一眼", presenceTitle(t.Title, zh)+" did not finish; take a look")}
	}
	if len(working) > 0 {
		t := working[0]
		p := Presence{State: "working", Ticket: t.ID, Count: len(working), Line: pick(zh, presenceTitle(t.Title, zh)+"在做", "Working on "+presenceTitle(t.Title, zh))}
		if len(working) > 1 {
			p.Line += pick(zh, fmt.Sprintf("，共 %d 件在做", len(working)), fmt.Sprintf(", %d in progress", len(working)))
		}
		return p
	}
	if len(in.Running) > 0 {
		return Presence{State: "working", Session: in.Running[0], Count: len(in.Running), Line: pick(zh, "在回复", "Replying")}
	}
	if done != nil {
		return Presence{State: "done", Ticket: done.ID, Line: pick(zh, presenceTitle(done.Title, zh)+"做完了", presenceTitle(done.Title, zh)+" is done")}
	}
	return Presence{State: "idle", Line: pick(zh, "空闲，有事叫我", "Idle. Ask me anything")}
}

// lastReport 是工单最近一份给人看的报告。
func lastReport(t superagent.TicketRow) (superagent.ReportLine, bool) {
	for i := len(t.Assignments) - 1; i >= 0; i-- {
		if lines := t.Assignments[i].Lines; len(lines) > 0 {
			return lines[len(lines)-1], true
		}
	}
	return superagent.ReportLine{}, false
}

// endedWell 在工单是做成了结束时为真。失败、放下和停下不算。
func endedWell(t superagent.TicketRow) bool {
	out := strings.ToLower(t.Outcome)
	return !strings.Contains(out, "fail") && !strings.Contains(out, "drop") && !strings.Contains(out, "stop")
}

// recentAt 在 RFC3339 时刻 at 离 now 不到 d 时为真。认不出时为假。
func recentAt(at string, now time.Time, d time.Duration) bool {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(at))
	if err != nil {
		return false
	}
	return now.Sub(t) >= 0 && now.Sub(t) < d
}

// presenceTitle 是工单标题在一句话里的样子。像密钥的不写。
func presenceTitle(title string, zh bool) string {
	s := scrubSense(strings.TrimSpace(title))
	if s == "" {
		return pick(zh, "一件任务", "a task")
	}
	s = clipBrief(s, 30)
	return pick(zh, "「"+s+"」", "“"+s+"”")
}
