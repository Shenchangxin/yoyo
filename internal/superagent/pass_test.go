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

// TestEvaluateDueAcceptsExplicit 锁住自动模式会在轮次里接受显式触发，观察模式则留在队列里。
func TestEvaluateDueAcceptsExplicit(t *testing.T) {
	eng := NewEngine(autoPolicy())
	if eng.Receive(human("m1", "fix")) != "stored" {
		t.Fatal("stored")
	}
	eng.EvaluateDue()
	tr, ok := eng.Trigger("m1")
	if !ok || tr.Queued || tr.DecisionID == "" {
		t.Fatalf("trigger %+v", tr)
	}
	quiet := NewEngine(Policy{Version: 1, Mode: "observe", Allowed: map[string]bool{"ada": true}})
	quiet.Receive(human("m2", "fix"))
	quiet.EvaluateDue()
	tr, ok = quiet.Trigger("m2")
	if !ok || !tr.Queued {
		t.Fatalf("observe %+v", tr)
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
