package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func memoryEvents(t *testing.T, a *App, sessionID string) []trace.Event {
	t.Helper()
	evs, err := a.Traces.Read(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	var out []trace.Event
	for _, ev := range evs {
		if ev.Type == trace.TypeInject && ev.Source == "memory" {
			out = append(out, ev)
		}
	}
	return out
}

// TestMemorySeenWritesReceiptsOnlyWhenSomethingChanged 锁住：路由记下的先写成回执；召回了旧记忆、或画像变了才写「带进了哪些」；
// 同一份画像不每轮重复；这些事件回放历史时不进模型。
func TestMemorySeenWritesReceiptsOnlyWhenSomethingChanged(t *testing.T) {
	dir := t.TempDir()
	mem, err := memory.Open(filepath.Join(dir, "memory"))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Home: &home.Dir{Root: dir}, Memory: mem, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	kept := mem.Keep(memory.Item{Kind: memory.KindSemantic, Text: "代码注释用中文", Source: "user"}, "")
	a.noteMemoryWrite("s1", kept)
	a.noteMemoryWrite("s1", mem.Keep(memory.Item{Kind: memory.KindSemantic, Text: "代码注释用中文", Source: "user"}, ""))
	mem.Write(memory.Item{ID: "task:tkt-3", Kind: memory.KindEpisodic, Text: "首页改版：标题换成新的，配色改成浅色", Source: "worker"})

	_, pinned := a.profileUse()
	note, recalled := memoryRecall(mem, "首页改版的配色还要再调一下", time.Now())
	if note == "" || len(recalled) != 1 || recalled[0].ID != "task:tkt-3" || len(pinned) != 1 {
		t.Fatalf("recalled %+v pinned %+v", recalled, pinned)
	}
	a.emitMemorySeen("s1", "t1", pinned, recalled)
	evs := memoryEvents(t, a, "s1")
	if len(evs) != 2 || evs[0].Payload["kind"] != "wrote" || evs[1].Payload["kind"] != "used" || evs[1].TurnID != "t1" {
		t.Fatalf("events = %+v", evs)
	}
	// 从会话文件读回来的载荷走过一遍 JSON，列表是 []any。
	items, _ := evs[0].Payload["items"].([]any)
	first, _ := func() (map[string]any, bool) {
		if len(items) == 0 {
			return nil, false
		}
		m, ok := items[0].(map[string]any)
		return m, ok
	}()
	if len(items) != 1 || first["action"] != "added" || first["id"] != kept.Item.ID {
		t.Fatalf("a repeat must not write a second receipt: %+v", evs[0].Payload["items"])
	}

	_, pinned = a.profileUse()
	a.emitMemorySeen("s1", "t2", pinned, nil)
	if got := memoryEvents(t, a, "s1"); len(got) != 2 {
		t.Fatalf("the same pin with nothing recalled must stay quiet: %d", len(got))
	}

	if !a.MemoryMute(kept.Item.ID, true) {
		t.Fatal("mute")
	}
	_, pinned = a.profileUse()
	if len(pinned) != 0 {
		t.Fatalf("muted line still pinned: %+v", pinned)
	}

	all, _ := a.Traces.Read("s1")
	for _, msg := range runtime.MessagesFromEvents(all) {
		t.Fatalf("memory receipts reached the model: %+v", msg)
	}
}
