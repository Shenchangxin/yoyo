// pass_test.go 锁住准入格、两秒评估、SQLite 快照，以及未配置来源不会假装同步成功。

package superagent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

// TestAdmissionCellOverridesDefault 锁住点名但不在名单里的格子可以改成 ignore。
func TestAdmissionCellOverridesDefault(t *testing.T) {
	p := autoPolicy()
	p.Mode = "auto"
	p.Allowed = nil
	p.Admission.UnlistedAddressed = "ignore"
	msg := human("m1", "hi")
	msg.Sender = "stranger"
	admitted, _, reason := p.Gate(msg)
	if admitted || reason != "not_admitted" {
		t.Fatalf("admitted %v reason %s", admitted, reason)
	}
}

// TestEvaluateDueOpensEval 锁住轮次会把显式触发收成评估，而不是直接开工单。观察模式同样开评，接受仍要 Decide。
func TestEvaluateDueOpensEval(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(human("m1", "fix")) != "stored" {
		t.Fatal("stored")
	}
	eng.EvaluateDue()
	tr, ok := eng.Trigger("m1")
	if !ok || tr.Queued || tr.DecisionID != "" {
		t.Fatalf("trigger %+v", tr)
	}
	th, ok := eng.Thread("room")
	if !ok || th.Evaluating == nil || th.Evaluating.Primary != "m1" {
		t.Fatalf("eval %+v", th.Evaluating)
	}
	quiet := NewEngine(Policy{Version: 1, Mode: "observe", Allowed: map[string]bool{"ada": true}})
	quiet.Receive(human("m2", "fix"))
	quiet.EvaluateDue()
	tr, ok = quiet.Trigger("m2")
	if !ok || tr.Queued || tr.DecisionID != "" {
		t.Fatalf("observe %+v", tr)
	}
}

// TestUndeliveredEvalDoesNotExpire 锁住还没交给编排台的评估，时间过去也不会记成超时。
func TestUndeliveredEvalDoesNotExpire(t *testing.T) {
	eng := NewEngine(autoPolicy())
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	eng.clock = func() time.Time { return now }
	eng.Receive(human("m1", "fix"))
	eng.EvaluateDue()
	now = now.Add(time.Hour)
	eng.EvaluateDue()
	tr, _ := eng.Trigger("m1")
	if tr.DecisionID != "" || tr.Queued {
		t.Fatalf("trigger %+v", tr)
	}
	th, _ := eng.Thread("room")
	if th.Evaluating == nil || !th.Evaluating.Delivered.IsZero() {
		t.Fatalf("eval %+v", th.Evaluating)
	}
}

// TestEvalExpiresThenUnavailable 锁住评估过期两次会重新排队，第三次记 unavailable。
func TestEvalExpiresThenUnavailable(t *testing.T) {
	eng := NewEngine(autoPolicy())
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	eng.clock = func() time.Time { return now }
	if eng.Receive(human("m1", "fix")) != "stored" {
		t.Fatal("stored")
	}
	eng.EvaluateDue()
	if eng.DeliverOpen() != 1 {
		t.Fatal("deliver")
	}
	for i := 0; i < requeues; i++ {
		now = now.Add(121 * time.Second)
		eng.EvaluateDue()
		if eng.DeliverOpen() != 1 {
			t.Fatal("redeliver")
		}
		tr, _ := eng.Trigger("m1")
		if tr.DecisionID != "" {
			t.Fatalf("early decision %+v", tr)
		}
	}
	now = now.Add(121 * time.Second)
	eng.EvaluateDue()
	tr, _ := eng.Trigger("m1")
	if tr.Queued || tr.DecisionID == "" {
		t.Fatalf("trigger %+v", tr)
	}
	d, ok := eng.decision(tr.DecisionID)
	if !ok || d.Kind != "unavailable" || d.Reason != "timeout" {
		t.Fatalf("decision %+v", d)
	}
}

// TestSnapshotRoundTrip 锁住工单和准入格能经 SQLite 还原。
func TestSnapshotRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "superagent.sqlite")
	rt, err := Open(path, autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.SetAdmission("inferred", "explicit", "ignore"); err != nil {
		t.Fatal(err)
	}
	rt.SetSources(SourceConfig{})
	rt.eng.Receive(human("m1", "fix"))
	if _, err := rt.eng.Decide("m1", "accept", "ok"); err != nil {
		t.Fatal(err)
	}
	rt.Pass(time.Now())
	rt.Stop()

	again, err := Open(path, Policy{Mode: "off"})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Stop()
	view := again.View()
	if view.Mode != "auto" || view.Admission.UnlistedAddressed != "explicit" {
		t.Fatalf("view %+v", view)
	}
	if len(view.Tickets) != 1 {
		t.Fatalf("tickets %+v", view.Tickets)
	}
}

// TestUnconfiguredSourcesStayHonest 锁住没有凭证时来源状态是 unconfigured。
func TestUnconfiguredSourcesStayHonest(t *testing.T) {
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.SetSources(SourceConfig{})
	rt.Pass(time.Now())
	view := rt.View()
	if len(view.Sources) != 2 {
		t.Fatalf("%+v", view.Sources)
	}
	for _, src := range view.Sources {
		if src.State != "unconfigured" {
			t.Fatalf("%+v", src)
		}
	}
}

// TestGitHubPollStoresIssue 锁住议题会变成排队触发，而不是一条成功的空同步。
func TestGitHubPollStoresIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widgets/issues" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"number": 7,
			"title":  "Broken",
			"body":   "@yoyo please",
			"user":   map[string]string{"login": "octo"},
		}})
	}))
	defer srv.Close()
	rt, err := Open("", autoPolicy())
	if err != nil {
		t.Fatal(err)
	}
	rt.client = srv.Client()
	rt.SetSources(SourceConfig{GitHubToken: "t", GitHubRepo: "acme/widgets", GitHubAPI: srv.URL})
	rt.Pass(time.Now())
	view := rt.View()
	if len(view.Queued) != 1 || view.Queued[0].ID != "github:acme/widgets#7" {
		t.Fatalf("queued %+v sources %+v", view.Queued, view.Sources)
	}
	var github SourceState
	for _, src := range view.Sources {
		if src.Name == "github" {
			github = src
		}
	}
	if github.State != "ok" {
		t.Fatalf("github %+v", github)
	}
}
