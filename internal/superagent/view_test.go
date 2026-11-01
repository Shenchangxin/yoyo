// view_test.go 锁住编排台快照用正文当标题，并分开最近一天的流量和执行负载。

package superagent

import (
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
