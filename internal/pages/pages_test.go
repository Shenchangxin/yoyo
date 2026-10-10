package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
