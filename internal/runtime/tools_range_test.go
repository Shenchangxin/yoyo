package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Session 3b9f1bbb660e4236: reads past EOF came back as silent empty results,
// and an invalid grep regex was a hard error the model retried verbatim.
func TestReadFileRangeNamesTheRealExtent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}

	past := tools.Call("read_file", `{"path":"app.js","offset":270,"limit":20}`)
	if past.Err != nil {
		t.Fatalf("past-EOF read must not error: %v", past.Err)
	}
	if !strings.Contains(past.Content, "has 3 lines") || !strings.Contains(past.Content, "past the end") {
		t.Fatalf("past-EOF read should name the file length, got %q", past.Content)
	}

	mid := tools.Call("read_file", `{"path":"app.js","offset":2,"limit":1}`)
	if mid.Err != nil || !strings.Contains(mid.Content, "     2|b") || strings.Contains(mid.Content, "c") {
		t.Fatalf("in-range window changed: %+v", mid)
	}

	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	empty := tools.Call("read_file", `{"path":"empty.txt","offset":1,"limit":5}`)
	if empty.Err != nil || !strings.Contains(empty.Content, "is empty") {
		t.Fatalf("empty file should say so: %+v", empty)
	}
}

func TestGrepFallsBackToLiteralOnInvalidRegex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("function a() {}\n})();\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}

	res := tools.Call("grep", `{"pattern":"})();"}`)
	if res.Err != nil {
		t.Fatalf("invalid regex must fall back to literal search, got error: %v", res.Err)
	}
	if !strings.Contains(res.Content, "literal text") || !strings.Contains(res.Content, "app.js") {
		t.Fatalf("expected literal-match note and hit, got %q", res.Content)
	}

	valid := tools.Call("grep", `{"pattern":"function a"}`)
	if valid.Err != nil || strings.Contains(valid.Content, "literal text") {
		t.Fatalf("valid regex must not get the fallback note: %+v", valid)
	}
}
