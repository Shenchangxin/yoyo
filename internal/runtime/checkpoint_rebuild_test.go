// 锁住检查点落盘之后，按轨迹重建的消息从检查点开始：之前的话不再回来，摘要、保留尾和之后的新话都在。

package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestCheckpointRebuildsFromTail 确认检查点之前的用户句不再出现，摘要、保留尾和新用户句还在。
func TestCheckpointRebuildsFromTail(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	_ = st.Append(trace.Event{Type: trace.TypeUser, SessionID: "s", Payload: map[string]any{"text": "old user"}})
	_ = st.Append(trace.Event{Type: trace.TypeAssistant, SessionID: "s", Payload: map[string]any{"text": "old asst"}})
	tail := []Message{{Role: RoleUser, Content: "kept user"}, {Role: RoleAssistant, Content: "kept asst"}}
	if err := persistCheckpoint(st, "s", "test", "## Objective\nkeep going", tail, true, nil, "", 0, ""); err != nil {
		t.Fatal(err)
	}
	_ = st.Append(trace.Event{Type: trace.TypeUser, SessionID: "s", Payload: map[string]any{"text": "new"}})
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, m := range MessagesFromEvents(evs) {
		joined += m.Content + "\n"
	}
	if strings.Contains(joined, "old user") {
		t.Fatalf("pre-checkpoint leaked: %s", joined)
	}
	for _, want := range []string{"keep going", "kept user", "new"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q after checkpoint: %s", want, joined)
		}
	}
}
