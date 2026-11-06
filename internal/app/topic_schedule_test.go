// topic_schedule_test.go 锁住选题排期的开拍时间：默认十分钟后，按分钟往后排，认本机时间写法，拒绝已经过去的时间。

package app

import (
	"testing"
	"time"
)

// TestTopicRunAt 确认排期时间的几种写法都算对，过去的时间被拒。
func TestTopicRunAt(t *testing.T) {
	now := time.Date(2026, 11, 6, 15, 30, 0, 0, time.Local)
	if got, err := topicRunAt("", nil, now); err != nil || !got.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("default %v %v", got, err)
	}
	if got, err := topicRunAt("", float64(0), now); err != nil || !got.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("zero delay should mean default: %v %v", got, err)
	}
	if got, err := topicRunAt("", float64(60), now); err != nil || !got.Equal(now.Add(time.Hour)) {
		t.Fatalf("delay %v %v", got, err)
	}
	if got, err := topicRunAt("2026-11-06T20:00", nil, now); err != nil || got.Hour() != 20 || got.Day() != 6 {
		t.Fatalf("local stamp %v %v", got, err)
	}
	if _, err := topicRunAt("2026-11-06T15:00", nil, now); err == nil {
		t.Fatal("a time in the past must be refused")
	}
	if _, err := topicRunAt("tonight", nil, now); err == nil {
		t.Fatal("unreadable time must be refused")
	}
}
