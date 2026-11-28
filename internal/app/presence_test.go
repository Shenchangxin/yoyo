package app

import (
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// TestPresenceOrder 锁住头像状态的先后：暂停压过一切，等你压过受阻，受阻压过在做，在做压过刚做完，都没有时空闲。
func TestPresenceOrder(t *testing.T) {
	now := time.Date(2026, 11, 28, 10, 0, 0, 0, time.UTC)
	fresh := now.Add(-3 * time.Minute).Format(time.RFC3339)
	stale := now.Add(-time.Hour).Format(time.RFC3339)
	working := superagent.TicketRow{ID: "t-w", Stage: "working", Title: "写周报"}
	failed := superagent.TicketRow{ID: "t-f", Stage: "person", Person: true, Action: "failed", Title: "部署"}
	waiting := superagent.TicketRow{ID: "t-a", Stage: "person", Person: true, Action: "answer", Title: "选机票"}
	finish := superagent.TicketRow{ID: "t-x", Stage: "person", Person: true, Action: "blocked", Title: "订酒店",
		Assignments: []superagent.AssignmentRow{{Lines: []superagent.ReportLine{{State: "blocked", Finish: &superagent.Finish{What: "付款"}}}}}}
	done := superagent.TicketRow{ID: "t-d", Stage: "ended", Outcome: "ready", Title: "整理文档", UpdatedAt: fresh}
	old := superagent.TicketRow{ID: "t-o", Stage: "ended", Outcome: "ready", Title: "旧事", UpdatedAt: stale}

	cases := []struct {
		name   string
		in     presenceInput
		state  string
		ticket string
		line   string
	}{
		{"paused", presenceInput{Paused: true, Approvals: 2, Tickets: []superagent.TicketRow{waiting}}, "paused", "", "暂停中"},
		{"approval", presenceInput{Approvals: 1, ApprovalSession: "s1", Tickets: []superagent.TicketRow{waiting, failed}}, "needs", "", "有 1 件要你批准，另有 1 件任务在等你"},
		{"finish", presenceInput{Tickets: []superagent.TicketRow{finish, failed, working}}, "needs", "t-x", "「订酒店」最后一步交给你"},
		{"waiting", presenceInput{Tickets: []superagent.TicketRow{waiting, working}}, "needs", "t-a", "「选机票」在等你"},
		{"blocked", presenceInput{Tickets: []superagent.TicketRow{failed, working}}, "blocked", "t-f", "「部署」没做成"},
		{"working", presenceInput{Tickets: []superagent.TicketRow{working, done}}, "working", "t-w", "「写周报」在做"},
		{"replying", presenceInput{Running: []string{"main"}, Tickets: []superagent.TicketRow{done}}, "working", "", "在回复"},
		{"done", presenceInput{Tickets: []superagent.TicketRow{done, old}}, "done", "t-d", "「整理文档」做完了"},
		{"idle", presenceInput{Tickets: []superagent.TicketRow{old}}, "idle", "", "空闲"},
	}
	for _, c := range cases {
		c.in.Now = now
		c.in.Zh = true
		got := presenceOf(c.in)
		if got.State != c.state || got.Ticket != c.ticket || !strings.Contains(got.Line, c.line) {
			t.Fatalf("%s: %+v", c.name, got)
		}
	}
	if got := presenceOf(presenceInput{Approvals: 1, ApprovalSession: "s1", Now: now}); got.Session != "s1" || got.Count != 1 {
		t.Fatalf("approval points at its session: %+v", got)
	}
	if got := (&App{}).Presence(); got.State != "idle" {
		t.Fatalf("an empty app is idle: %+v", got)
	}
}
