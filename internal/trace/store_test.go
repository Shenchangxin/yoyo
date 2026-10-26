// 本文件锁住追加后能读回，以及 ListSessions 能看见该 jsonl。由 go test 调用。

package trace

import (
	"path/filepath"
	"testing"
)

// TestAppendRead 锁住 Append 一条 user 后 Read 得到一行，ListSessions 只列出这个会话 id。
func TestAppendRead(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "sess"))
	if err := s.Append(Event{Type: TypeUser, SessionID: "abc", HarnessSnapshot: "h", Payload: map[string]any{"text": "hi"}}); err != nil {
		t.Fatal(err)
	}
	evs, err := s.Read("abc")
	if err != nil || len(evs) != 1 {
		t.Fatalf("%v %d", err, len(evs))
	}
	ids, err := s.ListSessions()
	if err != nil || len(ids) != 1 || ids[0] != "abc" {
		t.Fatalf("%v %v", err, ids)
	}
}
