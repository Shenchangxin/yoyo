// 锁住默认隔离、视频短片和 follow 不隔离，以及 webhook 不会因时间到期。

package schedule

import (
	"testing"
	"time"
)

// TestJobsIsolateByDefault 确认普通 once 任务 Isolate 为真，并且已经到期。
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

// TestVideoChannelJobDoesNotIsolate 确认频道为 video 的短片任务 Isolate 为假，并保留会话 ID。
func TestVideoChannelJobDoesNotIsolate(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	j := s.Create(Job{Kind: KindOnce, Spec: time.Now().UTC().Add(time.Hour).Format(time.RFC3339), Prompt: "做成短片：x", Channel: "video", SessionID: "vid-1", Isolate: true})
	if j.Isolate {
		t.Fatal("video shorts jobs must not isolate")
	}
	if j.Channel != "video" || j.SessionID != "vid-1" {
		t.Fatalf("%+v", j)
	}
}

// TestLegacyShortsPromptDoesNotIsolate 确认以「做成短片」开头的提示词不隔离，且 IsStudioJob 为真。
func TestLegacyShortsPromptDoesNotIsolate(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	j := s.Create(Job{Kind: KindOnce, Spec: time.Now().UTC().Add(time.Hour).Format(time.RFC3339), Prompt: "做成短片：爱用洗衣凝珠的人天塌了", Isolate: true})
	if j.Isolate {
		t.Fatal("legacy shorts prompt must not isolate")
	}
	if !IsStudioJob(j) {
		t.Fatal("legacy shorts prompt must route to studio")
	}
}

// TestFollowStaysInSessionAndDoesNotIsolate 确认 follow 不隔离、未到间隔不到期，中断后直到 Retry 才再次到期，并仍带原会话。
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

// TestWebhookIsNotTimeDueButTriggerWorks 确认 webhook 不会出现在 Due 里，但能按 ID 和 Spec 触发。
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
