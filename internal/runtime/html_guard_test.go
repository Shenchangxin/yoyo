package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustWrite(t *testing.T, tools *WorkspaceTools, rel, mode, content string) ToolResult {
	t.Helper()
	raw := `{"path":"` + rel + `","mode":"` + mode + `","content":` + jsonString(content) + `}`
	return tools.Call("write_file", raw)
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func readWorkspaceFile(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The failure this guards: part one closes the page, part two begins with
// </script></body></html> and then carries the rest of the script. Browsers
// render that rest as text after the document, and nothing reported it.
func TestAppendRefusesPartThatClosesPageEarly(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	first := "<!doctype html>\n<html><body>\n<script>\nvar a = 1;\n</script>\n</body></html>\n"
	if res := mustWrite(t, tools, "index.html", "create", first); res.Err != nil {
		t.Fatal(res.Err)
	}
	second := "</script></body></html>\nfunction later() { return 2; }\n"
	res := mustWrite(t, tools, "index.html", "append", second)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "after </html>") {
		t.Fatalf("append after the document closed must be refused, got %+v", res)
	}
	if got := readWorkspaceFile(t, dir, "index.html"); got != first {
		t.Fatalf("refused append must not touch the file:\n%s", got)
	}
}

func TestAppendRefusesAfterClosedDocument(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	if res := mustWrite(t, tools, "page.html", "create", "<html><body></body></html>\n"); res.Err != nil {
		t.Fatal(res.Err)
	}
	res := mustWrite(t, tools, "page.html", "append", "<script>x()</script>\n")
	if res.Err == nil {
		t.Fatalf("a document that already ends with </html> must not take more parts, got %+v", res)
	}
}

func TestAppendAllowsPartsThatKeepDocumentOpen(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	if res := mustWrite(t, tools, "index.html", "create", "<html><body>\n<script>\nvar a = 1;\n"); res.Err != nil {
		t.Fatal(res.Err)
	}
	if res := mustWrite(t, tools, "index.html", "append", "var b = 2;\n</script>\n</body></html>\n"); res.Err != nil {
		t.Fatalf("closing the document in the last part is allowed: %v", res.Err)
	}
	if got := readWorkspaceFile(t, dir, "index.html"); !strings.HasSuffix(strings.TrimSpace(got), "</html>") {
		t.Fatalf("document should end at </html>:\n%s", got)
	}
}

func TestAppendRefusesPartContainingEarlyClose(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	if res := mustWrite(t, tools, "index.html", "create", "<html><body>\n<script>\nvar a = 1;\n"); res.Err != nil {
		t.Fatal(res.Err)
	}
	res := mustWrite(t, tools, "index.html", "append", "</script></body></html>\n<script>\nvar late = 3;\n</script>\n")
	if res.Err == nil {
		t.Fatalf("a part whose own text closes the page and then continues must be refused, got %+v", res)
	}
}

func TestAppendCaseInsensitiveClose(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	if res := mustWrite(t, tools, "index.html", "create", "<HTML><BODY></BODY></HTML>\n"); res.Err != nil {
		t.Fatal(res.Err)
	}
	if res := mustWrite(t, tools, "index.html", "append", "<p>x</p>\n"); res.Err == nil {
		t.Fatal("upper-case </HTML> must count as closed")
	}
}

func TestAppendIgnoresNonHTMLFiles(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	if res := mustWrite(t, tools, "notes.txt", "create", "</html> is a literal here\n"); res.Err != nil {
		t.Fatal(res.Err)
	}
	if res := mustWrite(t, tools, "notes.txt", "append", "more text\n"); res.Err != nil {
		t.Fatalf("a text file may contain the string: %v", res.Err)
	}
}

func TestOverwriteWarnsWhenTextFollowsClose(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	res := mustWrite(t, tools, "index.html", "overwrite", "<html></html>\n<script>x()</script>\n")
	if res.Err != nil {
		t.Fatalf("overwrite stays allowed: %v", res.Err)
	}
	if !strings.Contains(res.Content, "warning: index.html has text after </html>") {
		t.Fatalf("overwrite must warn about text after the document:\n%s", res.Content)
	}
}

func TestOverwriteCleanDocumentHasNoWarning(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	res := mustWrite(t, tools, "index.html", "overwrite", "<html></html>\n")
	if res.Err != nil || strings.Contains(res.Content, "warning") {
		t.Fatalf("clean document: %+v", res)
	}
}
