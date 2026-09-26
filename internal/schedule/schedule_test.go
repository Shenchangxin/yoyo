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
