// view_test.go 锁住编排台快照用正文当标题，并分开最近一天的流量和执行负载。

package superagent

import (
	"strings"
	"testing"
	"time"
)

// TestBuildViewNamesTheWork 锁住工单和线程取第一行正文，旧触发不进最近一天，排队工单计入负载。
func TestBuildViewNamesTheWork(t *testing.T) {
	now := time.Now().UTC()
	snap := Snapshot{
		Policy: Policy{Mode: "auto", Version: 1},
		Triggers: []Trigger{
			{ID: "t1", Source: "github", Threads: []string{"room"}, Primary: "room", Sender: "ada", Text: "修复登录\n细节", At: now, Queued: true},
			{ID: "old", Source: "github", Threads: []string{"room"}, Primary: "room", Text: "旧的", At: now.Add(-48 * time.Hour)},
			{ID: "drop", Source: "github", Text: "噪声", At: now, Filtered: "empty"},
		},
		Threads:   []Thread{{Key: "room", Context: []string{"old", "t1"}, Queued: []string{"t1"}}},
		Tickets:   []Ticket{{ID: "k1", Origin: "t1", Mandate: Mandate{Sender: "ada"}}},
		Decisions: []Decision{{ID: "d1", TriggerID: "t1", Thread: "room", Kind: "accept", At: now, TicketID: "k1"}},
	}
	view := buildView(snap, nil, "", "", 3)
	if len(view.Tickets) != 1 || view.Tickets[0].Title != "修复登录" {
		t.Fatalf("title %+v", view.Tickets)
	}
	if len(view.Threads) != 1 || view.Threads[0].Title != "修复登录" || !view.Threads[0].Recent || len(view.Threads[0].Items) != 2 {
		t.Fatalf("thread %+v", view.Threads)
	}
	if view.Flow.Received != 2 || view.Flow.Filtered != 1 || view.Flow.Judged != 1 || view.Flow.Accepted != 1 || view.Flow.Ignored != 0 {
		t.Fatalf("flow %+v", view.Flow)
	}
	if view.Load.Queued != 1 || view.Load.Running != 0 || view.Load.MaxWorkers != 3 {
		t.Fatalf("load %+v", view.Load)
	}
	if view.Decisions[0].Thread != "room" {
		t.Fatal(view.Decisions[0].Thread)
	}
}

// TestBuildViewPutsRecentWorkFirst 锁住同一阶段里最近有动静的工单在前，序号按数字比，旧快照用起点触发补上接单时刻。
func TestBuildViewPutsRecentWorkFirst(t *testing.T) {
	base := time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)
	ended := func(id string, at time.Time) Ticket {
		return Ticket{ID: id, Origin: "o-" + id, Ended: &End{Outcome: "completed", At: at}}
	}
	var triggers []Trigger
	for _, id := range []string{"tkt-2", "tkt-10", "tkt-11", "tkt-3"} {
		triggers = append(triggers, Trigger{ID: "o-" + id, Text: id, At: base})
	}
	snap := Snapshot{
		Policy:   Policy{Mode: "observe", Version: 1},
		Triggers: triggers,
		Tickets: []Ticket{
			ended("tkt-2", base.Add(3*time.Hour)),
			ended("tkt-10", base.Add(time.Hour)),
			ended("tkt-11", base.Add(time.Hour)),
			{ID: "tkt-3", Origin: "o-tkt-3", Assignments: []Assignment{{ID: "a", State: "running", Reports: []Report{{State: "progress", At: base.Add(2 * time.Hour)}}}}},
		},
	}
	view := buildView(snap, nil, "", "", 3)
	var ids []string
	for _, row := range view.Tickets {
		ids = append(ids, row.ID)
	}
	if got := strings.Join(ids, ","); got != "tkt-3,tkt-2,tkt-11,tkt-10" {
		t.Fatalf("order = %s", got)
	}
	if view.Tickets[0].OpenedAt != base.Format(time.RFC3339) || view.Tickets[0].UpdatedAt != base.Add(2*time.Hour).Format(time.RFC3339) {
		t.Fatalf("times = %+v", view.Tickets[0])
	}
}

// TestOriginURLKeepsRepo 锁住每件工作自己的线程仍能打开对应的 GitHub 页面。
func TestOriginURLKeepsRepo(t *testing.T) {
	issue := originURL(Trigger{Source: "github", Ref: "7", Primary: "github:acme/widgets#7", ID: "github:acme/widgets#7"})
	if issue != "https://github.com/acme/widgets/issues/7" {
		t.Fatal(issue)
	}
	pr := originURL(Trigger{Source: "github", Ref: "3", Primary: "github:acme/widgets!3", ID: "github:acme/widgets!pr#3"})
	if pr != "https://github.com/acme/widgets/pull/3" {
		t.Fatal(pr)
	}
}
