package lease

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestTryIsExclusiveAndReleaseHandsItOver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lead.lock")
	first, err := Try(path, Holder{Role: "daemon", PID: 11, Since: "2026-11-28T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Try(path, Holder{Role: "desktop", PID: 12}); !errors.Is(err, ErrHeld) {
		t.Fatalf("second try = %v, want ErrHeld", err)
	}
	h, ok := ReadHolder(path)
	if !ok || h.Role != "daemon" || h.PID != 11 {
		t.Fatalf("holder = %+v %v", h, ok)
	}
	first.Release()
	first.Release()
	if _, ok := ReadHolder(path); ok {
		t.Fatal("released lease still has a holder")
	}
	second, err := Try(path, Holder{Role: "desktop", PID: 12})
	if err != nil {
		t.Fatalf("after release: %v", err)
	}
	defer second.Release()
	if h, ok := ReadHolder(path); !ok || h.Role != "desktop" {
		t.Fatalf("holder = %+v %v", h, ok)
	}
}

// TestUpdateRewritesHolderOnlyWhileHeld 确认拿着租约时 Update 改写持有人记录（收尾时标 Draining），
// 放开之后再调用什么都不写，免得留下一份没人拿着的记录。
func TestUpdateRewritesHolderOnlyWhileHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lead.lock")
	h := Holder{Role: "daemon", PID: 11, Since: "2026-11-28T00:00:00Z"}
	l, err := Try(path, h)
	if err != nil {
		t.Fatal(err)
	}
	h.Draining = true
	if err := l.Update(h); err != nil {
		t.Fatal(err)
	}
	if got, ok := ReadHolder(path); !ok || !got.Draining || got.Since != h.Since || got.PID != 11 {
		t.Fatalf("holder after update = %+v %v", got, ok)
	}
	l.Release()
	if err := l.Update(h); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".json"); !os.IsNotExist(err) {
		t.Fatalf("update after release wrote a record: %v", err)
	}
	var none *Lease
	if err := none.Update(h); err != nil {
		t.Fatal(err)
	}
}

func TestReadHolderIgnoresAStaleRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lead.lock")
	if err := writeHolder(path+".json", Holder{Role: "daemon", PID: 99}); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadHolder(path); ok {
		t.Fatal("a record nobody locks is a crashed holder")
	}
}

func TestLockSerializesReadModifyWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.lock")
	data := filepath.Join(dir, "n")
	if err := os.WriteFile(data, []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				unlock, err := Lock(path)
				if err != nil {
					t.Error(err)
					return
				}
				b, _ := os.ReadFile(data)
				time.Sleep(time.Millisecond)
				_ = os.WriteFile(data, []byte{b[0] + 1}, 0o600)
				unlock()
			}
		}()
	}
	wg.Wait()
	b, _ := os.ReadFile(data)
	if b[0] != 80 {
		t.Fatalf("lost updates: %d", b[0])
	}
}
