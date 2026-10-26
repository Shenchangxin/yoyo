// HTML 追加若会把文字留在 </html> 之后则拒绝。覆盖只警告。封存的 game-world 可玩文件拒绝改写。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustWrite 按给定模式写入并返回工具结果，失败则终止测试。
func mustWrite(t *testing.T, tools *WorkspaceTools, rel, mode, content string) ToolResult {
	t.Helper()
	raw := `{"path":"` + rel + `","mode":"` + mode + `","content":` + jsonString(content) + `}`
	return tools.Call("write_file", raw)
}

// jsonString 把字符串编成 JSON 字面量。
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

// readWorkspaceFile 读工作区里的文件，失败则终止测试。
func readWorkspaceFile(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestAppendRefusesPartThatClosesPageEarly 确认第一段提前闭合、第二段以 </script></body></html> 开头再带脚本时被拒绝，避免浏览器把后文渲染到文档外。
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

// TestAppendRefusesAfterClosedDocument 确认文档已经闭合后再追加会被拒绝。
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

// TestAppendAllowsPartsThatKeepDocumentOpen 确认追加后文档仍开着时允许写入。
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

// TestAppendRefusesPartContainingEarlyClose 确认本段自己提前闭合，使后续文字落在 </html> 之后时被拒绝。
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

// TestAppendCaseInsensitiveClose 确认大写的 </HTML> 同样挡住追加。
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

// TestAppendIgnoresNonHTMLFiles 确认非 html 文件不受闭合检查。
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

// TestOverwriteWarnsWhenTextFollowsClose 确认覆盖后 </html> 之后仍有文字时返回警告。
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

// TestOverwriteCleanDocumentHasNoWarning 确认干净闭合的覆盖没有警告。
func TestOverwriteCleanDocumentHasNoWarning(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	res := mustWrite(t, tools, "index.html", "overwrite", "<html></html>\n")
	if res.Err != nil || strings.Contains(res.Content, "warning") {
		t.Fatalf("clean document: %+v", res)
	}
}
