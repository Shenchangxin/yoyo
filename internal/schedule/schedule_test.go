package schedule

import (
	"testing"
	"time"
)

func TestJobsIsolateByDefault(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	j := s.Create(Job{Kind: KindOnce, Spec: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), Prompt: "draft weekly"})
	if !j.Isolate {
		t.Fatal("unattended jobs must isolate")
	}
	due := s.Due(time.Now().UTC())
	if len(due) != 1 {
		t.Fatalf("due %d", len(due))
	}
}

func TestFollowStaysInSessionAndDoesNotIsolate(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	j := s.Create(Job{Kind: KindFollow, Spec: "1h", Prompt: "check in", SessionID: "sess-1", Isolate: true})
	if j.Isolate {
		t.Fatal("follow must not isolate")
	}
	if j.SessionID != "sess-1" {
		t.Fatal("session binding lost")
	}
	now := time.Now().UTC()
	if due := s.Due(now); len(due) != 0 {
		t.Fatalf("follow must wait the delay: %+v", due)
	}
	s.Interrupt(j.ID, "lease expired")
	if due := s.Due(now.Add(time.Hour + time.Second)); len(due) != 0 {
		t.Fatalf("interrupted follow must wait for retry: %+v", due)
	}
	got, ok := s.Retry(j.ID)
	if !ok || got.Status != "" {
		t.Fatalf("retry %+v %v", got, ok)
	}
	if due := s.Due(now.Add(time.Hour + time.Second)); len(due) != 1 || due[0].SessionID != "sess-1" {
		t.Fatalf("follow should fire in-session after delay: %+v", due)
	}
}

func TestWebhookIsNotTimeDueButTriggerWorks(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	j := s.Create(Job{Kind: KindWebhook, Spec: "inbox-hook", Prompt: "triage"})
	if due := s.Due(time.Now().UTC()); len(due) != 0 {
		t.Fatalf("webhook must not be time-due: %+v", due)
	}
	got, ok := s.Trigger(j.ID)
	if !ok || got.ID != j.ID {
		t.Fatalf("trigger %+v %v", got, ok)
	}
	if _, ok := s.Trigger("inbox-hook"); !ok {
		t.Fatal("trigger by spec")
	}
}
