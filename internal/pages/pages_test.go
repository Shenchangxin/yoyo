// 锁住页面落在存储根下、修订冲突不覆盖、批准写入草稿，以及路径不能逃出根目录。

package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateGetAndRebuildIndex 确认新页面落在默认空间、修订号为 1，删掉 index.json 后 List 仍能重建出它。
func TestCreateGetAndRebuildIndex(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pg, err := s.Create(Page{Title: "Launch brief", Content: "Do the thing.\n"})
	if err != nil {
		t.Fatal(err)
	}
	if pg.SpaceID != DefaultSpace || pg.Revision != 1 || pg.ID == "" {
		t.Fatalf("page %+v", pg)
	}
	got, err := s.Get("", pg.ID)
	if err != nil || got.Title != "Launch brief" || !strings.Contains(got.Content, "Do the thing") {
		t.Fatalf("get %+v %v", got, err)
	}
	if err := os.Remove(filepath.Join(s.Root(), "index.json")); err != nil {
		t.Fatal(err)
	}
	list := s.List("")
	if len(list) != 1 || list[0].ID != pg.ID {
		t.Fatalf("rebuilt index %+v", list)
	}
}

// TestRevisionConflictKeepsDraft 确认用过期修订号 Save 返回 ConflictError，磁盘内容不变。
func TestRevisionConflictKeepsDraft(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pg, err := s.Create(Page{Title: "A", Content: "one"})
	if err != nil {
		t.Fatal(err)
	}
	pg.Content = "two"
	if _, err := s.Save(pg, pg.Revision); err != nil {
		t.Fatal(err)
	}
	pg.Content = "stale"
	_, err = s.Save(pg, 1)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if _, ok := err.(*ConflictError); !ok {
		t.Fatalf("want ConflictError, got %T %v", err, err)
	}
	got, _ := s.Get("", pg.ID)
	if got.Content != "two" || got.Revision != 2 {
		t.Fatalf("stale write landed: %+v", got)
	}
}

// TestReviewIdempotentAndApprove 确认相同提案不换 ID，批准后能读到页面，再次批准仍成功。
func TestReviewIdempotentAndApprove(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r1, err := s.Propose(Review{ThreadID: "t1", ToolCallID: "c1", Title: "Note", Content: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := s.Propose(Review{ThreadID: "t1", ToolCallID: "c1", Title: "Note", Content: "hello"})
	if err != nil || r2.Hash != r1.Hash {
		t.Fatalf("idempotent %+v %+v %v", r1, r2, err)
	}
	pg, done, err := s.Decide(r1.ID, r1.Hash, true)
	if err != nil || pg.ID == "" || done.Status != ReviewSaved {
		t.Fatalf("approve %+v %+v %v", pg, done, err)
	}
	again, _, err := s.Decide(r1.ID, r1.Hash, true)
	if err != nil || again.ID != pg.ID {
		t.Fatalf("retry approve %v %v", again, err)
	}
}

// TestPathTraversalStaysInsideStore 确认 id 里的路径片段被清掉，根目录外没有写出文件。
func TestPathTraversalStaysInsideStore(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	pg, err := s.Create(Page{ID: "../../evil", SpaceID: "../out", Title: "x", Content: "y"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pg.ID, "/") || strings.Contains(pg.ID, "..") || strings.Contains(pg.SpaceID, ".") {
		t.Fatalf("unsanitized %+v", pg)
	}
	for _, p := range []string{
		filepath.Join(filepath.Dir(root), "evil.md"),
		filepath.Join(root, "..", "evil.md"),
		filepath.Join(root, "..", "out", "evil.md"),
	} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("escaped write %s", p)
		}
	}
	got, err := s.Get(pg.SpaceID, pg.ID)
	if err != nil || got.Content != "y" {
		t.Fatalf("inside store %+v %v", got, err)
	}
	want := filepath.Join(root, pg.SpaceID, pg.ID+".md")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("missing in-store file %s: %v", want, err)
	}
}

// TestSaveClipsBody 确认正文被截到 MaxBody 个符文。
func TestSaveClipsBody(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pg, err := s.Create(Page{Title: "clip", Content: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	pg.Content = strings.Repeat("字", MaxBody+50)
	saved, err := s.Save(pg, pg.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(saved.Content)); n != MaxBody {
		t.Fatalf("clipped %d", n)
	}
}

// TestApproveOverridesStaleRevision 确认页面已被改过时，批准仍写入审阅草稿。
func TestApproveOverridesStaleRevision(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pg, err := s.Create(Page{Title: "A", Content: "one"})
	if err != nil {
		t.Fatal(err)
	}
	rev, err := s.Propose(Review{ThreadID: "t", ToolCallID: "c", PageID: pg.ID, Title: "A", Content: "two", ExpectedRevision: pg.Revision})
	if err != nil {
		t.Fatal(err)
	}
	pg.Content = "library edit"
	if _, err := s.Save(pg, pg.Revision); err != nil {
		t.Fatal(err)
	}
	out, done, err := s.Decide(rev.ID, rev.Hash, true)
	if err != nil || done.Status != ReviewSaved {
		t.Fatalf("approve %v %+v %v", out, done, err)
	}
	if out.Content != "two" {
		t.Fatalf("operator approve should write the draft, got %q", out.Content)
	}
}

// TestProposeKeepsReviewIDWhenContentChanges 确认同一工具调用更新内容时 ID 不变、哈希变了。
func TestProposeKeepsReviewIDWhenContentChanges(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r1, err := s.Propose(Review{ThreadID: "t1", ToolCallID: "c1", Title: "Note", Content: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := s.Propose(Review{ThreadID: "t1", ToolCallID: "c1", Title: "Note", Content: "hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if r2.ID != r1.ID {
		t.Fatalf("review id changed %s -> %s", r1.ID, r2.ID)
	}
	if r2.Hash == r1.Hash {
		t.Fatal("hash should change with content")
	}
}

// TestListIncludesPreview 确认列表摘要带有正文预览。
func TestListIncludesPreview(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(Page{Title: "Brief", Content: "alpha beta gamma"}); err != nil {
		t.Fatal(err)
	}
	list := s.List("")
	if len(list) != 1 || !strings.Contains(list[0].Preview, "alpha beta gamma") {
		t.Fatalf("preview %+v", list)
	}
}

// TestDeletePromotesChildren 确认删除父页后，子页的父 id 改成被删页原来的父 id。
func TestDeletePromotesChildren(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	parent, err := s.Create(Page{Title: "Parent", Content: "p"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.Create(Page{Title: "Child", Content: "c", ParentID: parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("", parent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("", parent.ID); err == nil {
		t.Fatal("parent still present")
	}
	got, err := s.Get("", child.ID)
	if err != nil || got.ParentID != "" {
		t.Fatalf("child parent %q %v", got.ParentID, err)
	}
}

// TestSaveDashClearsParent 确认父 id 为 - 时清空父页。
func TestSaveDashClearsParent(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	parent, err := s.Create(Page{Title: "Parent", Content: "p"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.Create(Page{Title: "Child", Content: "c", ParentID: parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	child.ParentID = "-"
	saved, err := s.Save(child, child.Revision)
	if err != nil || saved.ParentID != "" {
		t.Fatalf("parent %q %v", saved.ParentID, err)
	}
}

// TestAtomicRenameLeavesReadableMarkdown 确认写完的文件仍是可读的 Markdown。
func TestAtomicRenameLeavesReadableMarkdown(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	pg, err := s.Create(Page{Title: "Readable", Content: "operator can open this"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, DefaultSpace, pg.ID+".md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "title: Readable") || !strings.Contains(text, "operator can open this") {
		t.Fatalf("not inspectable markdown:\n%s", text)
	}
}
