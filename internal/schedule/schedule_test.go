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
