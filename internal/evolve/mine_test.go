// 锁住没有写文件的轨迹会被收成 missing_artifact，并且启发式提议带上非空片段。

package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestMineMissingArtifact 要求只有 list_dir 和 read_file 的失败题机制为 missing_artifact，且启发式片段非空。
func TestMineMissingArtifact(t *testing.T) {
	evs := []trace.Event{
		{TaskID: "t1", Type: trace.TypeToolCall, Payload: map[string]any{"name": "list_dir"}},
		{TaskID: "t1", Type: trace.TypeToolCall, Payload: map[string]any{"name": "read_file"}},
	}
	b := Mine(evs, map[string]string{"t1": "missing file"})
	if len(b.Clusters) != 1 {
		t.Fatalf("%+v", b)
	}
	if b.Clusters[0].Signature.AgentMechanism != "missing_artifact" {
		t.Fatalf("%+v", b.Clusters[0])
	}
	props := HeuristicPropose(b, 3)
	if props[0].Fragment == nil || props[0].Fragment.Text == "" {
		t.Fatal(props)
	}
}
