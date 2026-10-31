// stage.go 按原型 core/stage.ts 的表计算工单阶段。先匹配的行赢。本文件不改工单。

package superagent

// Stage 是工单正在等什么。Person 为真表示没有人输入就继续不了。
// Name 取 person、working、idle、ended。Action 在 person 时取 delivery、answer、review、blocked、failed，在 queued 时取 queued。
type Stage struct {
	Name   string
	Person bool
	Action string
}

// TicketStage 返回 ticket 的阶段。ticket 为 nil 或已经结束时阶段是 ended。
func TicketStage(ticket *Ticket) Stage {
	if ticket == nil || ticket.Ended != nil {
		return Stage{Name: "ended"}
	}
	if ticket.DeliveryErr != "" {
		return Stage{Name: "person", Person: true, Action: "delivery"}
	}
	if ticket.Attention != "" {
		return Stage{Name: "person", Person: true, Action: "answer"}
	}
	asg := latestAssignment(ticket)
	report := ""
	if asg != nil && len(asg.Reports) > 0 {
		report = asg.Reports[len(asg.Reports)-1].State
	}
	if report == "ready" {
		return Stage{Name: "person", Person: true, Action: "review"}
	}
	if (report == "blocked" || report == "failed") && WaitsForPerson(ticket) {
		return Stage{Name: "person", Person: true, Action: report}
	}
	if asg == nil || asg.State == "pending" {
		return Stage{Name: "working", Action: "queued"}
	}
	if report == "waiting" || asg.State == "ended" {
		return Stage{Name: "idle"}
	}
	return Stage{Name: "working"}
}

// WaitsForPerson 在 ticket 最近一次派工报了 blocked 或 failed，且之后没有人的消息时为真。
func WaitsForPerson(ticket *Ticket) bool {
	if ticket == nil {
		return false
	}
	asg := latestAssignment(ticket)
	if asg == nil || len(asg.Reports) == 0 {
		return false
	}
	rep := asg.Reports[len(asg.Reports)-1]
	if rep.State != "blocked" && rep.State != "failed" {
		return false
	}
	for _, in := range ticket.Inputs {
		if in.Human && in.Seq > rep.Seq {
			return false
		}
	}
	return true
}

// CanRetry 在 ticket 还开着，并且在等人处理失败或投递失败时为真。
func CanRetry(ticket *Ticket) bool {
	return ticket != nil && ticket.Ended == nil && (WaitsForPerson(ticket) || ticket.DeliveryErr != "")
}

// latestAssignment 返回 ticket 最后一次派工。ticket 没有派工时返回 nil。
func latestAssignment(ticket *Ticket) *Assignment {
	if ticket == nil || len(ticket.Assignments) == 0 {
		return nil
	}
	return &ticket.Assignments[len(ticket.Assignments)-1]
}
