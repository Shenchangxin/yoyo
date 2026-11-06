// studio_task_test.go 锁住后台配音和合成：点下去马上返回并记号，同一条短片不能同时再起一件，做完清掉记号并推 studio_state，失败留下原因，重启清掉挂住的记号。

package video

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// waitStudio 等到 cond 对最新的工程成立，最多两秒。
func waitStudio(t *testing.T, e *Engine, id string, cond func(StudioProject) bool) StudioProject {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		p, err := e.GetStudioProject(id)
		if err != nil {
			t.Fatal(err)
		}
		if cond(p) {
			return p
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting; last %+v", p.Task)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestStudioTaskRunsInBackground 确认后台任务马上返回带记号的工程，跑着时拒绝第二件和一键成片，做完清掉记号并推事件。
func TestStudioTaskRunsInBackground(t *testing.T) {
	e := testEngine(t)
	var mu sync.Mutex
	var events []map[string]any
	e.OnHub = func(kind string, payload map[string]any) {
		if kind == "studio_state" {
			mu.Lock()
			events = append(events, payload)
			mu.Unlock()
		}
	}
	p := approvedStudio(t, e, "第一句。第二句。")
	release := make(chan struct{})
	started, err := e.StartStudioTask(p.ID, "speak", func() error {
		<-release
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.Task == nil || started.Task.Kind != "speak" {
		t.Fatalf("task not marked: %+v", started.Task)
	}
	if _, err := e.StartStudioTask(p.ID, "assemble", func() error { return nil }); err == nil || !strings.Contains(err.Error(), "正在配音") {
		t.Fatalf("second task should be refused: %v", err)
	}
	if _, err := e.StartStudioAuto(p.ID); err == nil {
		t.Fatal("one-click film must wait for the running task")
	}
	close(release)
	waitStudio(t, e, p.ID, func(p StudioProject) bool { return p.Task == nil })
	mu.Lock()
	defer mu.Unlock()
	if len(events) < 2 || events[0]["task"] != "speak" || events[len(events)-1]["task"] != "" {
		t.Fatalf("events %v", events)
	}
}

// TestStudioTaskFailureLeavesReason 确认后台任务失败时记号清掉，原因挂在对应的闸上。
func TestStudioTaskFailureLeavesReason(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。")
	if _, err := e.StartStudioTask(p.ID, "assemble", func() error { return errors.New("ffmpeg 不在") }); err != nil {
		t.Fatal(err)
	}
	got := waitStudio(t, e, p.ID, func(p StudioProject) bool { return p.Task == nil && len(p.Issues) > 0 })
	found := false
	for _, iss := range got.Issues {
		if iss.Gate == GateCompose && strings.Contains(iss.Message, "合成没做成") && strings.Contains(iss.Message, "ffmpeg 不在") {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues %+v", got.Issues)
	}
}

// TestRecoverStudioTaskAfterRestart 确认应用中途关掉后，重启时挂住的记号被清掉并说明。
func TestRecoverStudioTaskAfterRestart(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。")
	if _, err := e.mutateStudio(p.ID, func(cur *StudioProject) error {
		cur.Task = &StudioTask{Kind: "speak", StartedAt: Now()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	e.recoverStudioAuto()
	got, _ := e.GetStudioProject(p.ID)
	if got.Task != nil {
		t.Fatal("stale task should be cleared")
	}
	if len(got.Issues) == 0 || !strings.Contains(got.Issues[len(got.Issues)-1].Message, "上次配音没做完") {
		t.Fatalf("issues %+v", got.Issues)
	}
}

// TestStartStudioAssembleChecksShotsFirst 确认缺画面时当场拒绝，不排进后台再失败。
func TestStartStudioAssembleChecksShotsFirst(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。第二句。")
	_, err := e.StartStudioAssemble(p.ID)
	if err == nil || !strings.Contains(err.Error(), "还没有画面") {
		t.Fatalf("err %v", err)
	}
	got, _ := e.GetStudioProject(p.ID)
	if got.Task != nil {
		t.Fatal("a refused assemble must not leave a task")
	}
}
