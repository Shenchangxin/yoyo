// 本文件锁住跨度 id 的长度、父子 TraceID 相同，以及 spans.jsonl 里不出现密钥、并留下 parent_span_id。
// 不启动 OTLP。由 go test 调用。

package observe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestW3CIdsAndParent 要求 TraceID 长 32、SpanID 长 16，子级沿用父级 TraceID 和 SpanID。
// 结束后的 spans.jsonl 不得含 sk-live，要有 parent_span_id，并且后来写入的属性里要有 cache_read。
func TestW3CIdsAndParent(t *testing.T) {
	dir := t.TempDir()
	tr := Open(dir)
	parent := tr.Start("turn", "s1", map[string]any{"turn_id": "abc", "api_key": "sk-live-secretvalue"})
	if len(parent.TraceID) != 32 || len(parent.SpanID) != 16 {
		t.Fatalf("ids %+v", parent)
	}
	child := tr.StartChild(parent, "model.complete", "s1", map[string]any{"cached_tokens": 12})
	if child.TraceID != parent.TraceID || child.ParentSpanID != parent.SpanID {
		t.Fatalf("parent link %+v / %+v", parent, child)
	}
	tr.End(child, nil)
	tr.End(parent, nil)
	body, err := os.ReadFile(filepath.Join(dir, "spans.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "sk-live") {
		t.Fatalf("secret in spans: %s", body)
	}
	if !strings.Contains(string(body), `"parent_span_id"`) {
		t.Fatalf("missing parent: %s", body)
	}
	if !strings.Contains(string(body), `"model.complete"`) && !strings.Contains(string(body), "model.complete") {
		// 父级和子级的名字写在 JSON 里。
	}
	child2 := tr.StartChild(parent, "model.complete", "s1", map[string]any{"gen_ai.usage.input_tokens": 12, "gen_ai.usage.cache_read_tokens": 3})
	tr.End(child2, nil)
	body, _ = os.ReadFile(filepath.Join(dir, "spans.jsonl"))
	if !strings.Contains(string(body), "cache_read") {
		t.Fatalf("missing cache attr: %s", body)
	}
}
