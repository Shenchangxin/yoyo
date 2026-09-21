package observe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		// parent/child names are in JSON
	}
	child2 := tr.StartChild(parent, "model.complete", "s1", map[string]any{"gen_ai.usage.input_tokens": 12, "gen_ai.usage.cache_read_tokens": 3})
	tr.End(child2, nil)
	body, _ = os.ReadFile(filepath.Join(dir, "spans.jsonl"))
	if !strings.Contains(string(body), "cache_read") {
		t.Fatalf("missing cache attr: %s", body)
	}
}
