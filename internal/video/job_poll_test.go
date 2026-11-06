// job_poll_test.go 锁住轮询碰上一时错误时的处理：限流、5xx 和连不上接着等，4xx 和普通错误直接失败，等过宽限期也失败，退避按次数翻倍。

package video

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTransientPollErrors 确认 503、429 和连不上算一时的，401 和普通错误不算；错误文本仍是供应商给的那句。
func TestTransientPollErrors(t *testing.T) {
	e := testEngine(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/busy":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"overloaded"}`))
		case "/slow-down":
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"bad key"}`))
		}
	}))
	defer srv.Close()
	call := func(path string) error {
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, err = e.doJSON(req)
		return err
	}
	if err := call("/busy"); err == nil || !transientPollErr(err) || err.Error() != "overloaded" {
		t.Fatalf("503 %v", err)
	}
	if err := call("/slow-down"); err == nil || !transientPollErr(err) {
		t.Fatalf("429 %v", err)
	}
	if err := call("/auth"); err == nil || transientPollErr(err) || err.Error() != "bad key" {
		t.Fatalf("401 must fail the job: %v", err)
	}
	srv.Close()
	if err := call("/busy"); err == nil || !transientPollErr(err) {
		t.Fatalf("connection refused %v", err)
	}
	if transientPollErr(errors.New("image completed without url")) {
		t.Fatal("a plain error is not transient")
	}
}

// TestKeepPollingGivesUpAfterGrace 确认一时错误在宽限期内接着等，过了宽限期才判失败。
func TestKeepPollingGivesUpAfterGrace(t *testing.T) {
	start := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	var j Job
	if !j.keepPolling(start) || j.PollMisses != 1 || j.PollMissAt == "" {
		t.Fatalf("first miss %+v", j)
	}
	if !j.keepPolling(start.Add(time.Minute)) || j.PollMisses != 2 {
		t.Fatalf("a miss inside the grace window should keep polling: %+v", j)
	}
	if j.keepPolling(start.Add(pollGrace + time.Second)) {
		t.Fatal("misses past the grace window should fail the job")
	}
}

// TestPollRestingBacksOff 确认出错后退避从 1 秒起翻倍，最多 20 秒；没出过错不退避。
func TestPollRestingBacksOff(t *testing.T) {
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	j := Job{UpdatedAt: at.Format(time.RFC3339Nano)}
	if pollResting(j, at) {
		t.Fatal("no misses, no rest")
	}
	j.PollMisses = 1
	if !pollResting(j, at.Add(500*time.Millisecond)) || pollResting(j, at.Add(1100*time.Millisecond)) {
		t.Fatal("first miss rests one second")
	}
	j.PollMisses = 3
	if !pollResting(j, at.Add(3*time.Second)) || pollResting(j, at.Add(4100*time.Millisecond)) {
		t.Fatal("third miss rests four seconds")
	}
	j.PollMisses = 9
	if !pollResting(j, at.Add(19*time.Second)) || pollResting(j, at.Add(21*time.Second)) {
		t.Fatal("rest caps at twenty seconds")
	}
}
