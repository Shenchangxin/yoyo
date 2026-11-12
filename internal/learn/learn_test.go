// 测试共用的开库助手，以及笔记和手记的不变量：掌握分上限、迁移保留页面 id、审阅哈希和文件库一致、手记只注入使命和上一课。
// 掌握路径的测试在 mastery_test.go，练习在 practice_test.go，文库在 library_test.go。

package learn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/pages"
)

// TestComputeMasteryCaps 确认一次答对封顶 0.5，两次封顶 0.8，三次才过 0.9 的门。
func TestComputeMasteryCaps(t *testing.T) {
	if ComputeMastery(nil) != 0 {
		t.Fatal("empty")
	}
	if ComputeMastery([]float64{1}) != 0.5 {
		t.Fatal("one correct must cap at 0.5")
	}
	if ComputeMastery([]float64{1, 1}) != 0.8 {
		t.Fatal("two correct must cap at 0.8")
	}
	if ComputeMastery([]float64{1, 1, 1}) < 0.9 {
		t.Fatal("three correct must clear the 0.9 gate")
	}
}

// TestMigrateKeepsPageID 确认旧的 Markdown 页面抄进笔记后 id、标题和正文不变。
func TestMigrateKeepsPageID(t *testing.T) {
	root := t.TempDir()
	pagesRoot := filepath.Join(root, "pages", "home")
	if err := os.MkdirAll(pagesRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: pg-keep\nspace_id: home\ntitle: 旧笔记\nrevision: 2\ncreated_at: 2020-01-02T03:04:05Z\nupdated_at: 2020-01-02T03:04:05Z\n---\n\n正文\n"
	if err := os.WriteFile(filepath.Join(pagesRoot, "pg-keep.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(filepath.Join(root, "learn"), filepath.Join(root, "pages"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	pg, err := s.Get("home", "pg-keep")
	if err != nil || pg.Title != "旧笔记" || pg.Content != "正文\n" && pg.Content != "正文" {
		t.Fatalf("%+v %v", pg, err)
	}
}

// TestReviewHashMatchesPages 确认待审阅的哈希和文件库用的是同一个公式。
func TestReviewHashMatchesPages(t *testing.T) {
	s := openTemp(t)
	rev, err := s.Propose(pages.Review{Title: "题", Content: "文", PageID: "pg-1", ExpectedRevision: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := pages.HashDraft("题", "文", "pg-1", "home", 3)
	if rev.Hash != want {
		t.Fatalf("hash %s want %s", rev.Hash, want)
	}
}

// TestJournalInjectOmitsRecords 确认注入给模型的手记只有使命和上一课，记录不进提示；那一行坏了就报错，不编一份空使命。
func TestJournalInjectOmitsRecords(t *testing.T) {
	s := openTemp(t)
	if err := s.SaveJournal("线性代数", "为了考试", "入门", "学了行列式", "下一课特征值"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddJournal("SECRET_RECORD"); err != nil {
		t.Fatal(err)
	}
	text, err := s.JournalInject()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "SECRET_RECORD") || !strings.Contains(text, "线性代数") || !strings.Contains(text, "特征值") {
		t.Fatal(text)
	}
	if _, err := s.db.Exec(`DELETE FROM journal`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.JournalInject(); err == nil {
		t.Fatal("corrupt journal must fail closed")
	}
}

// openTemp 在临时目录里开一份空库，测试结束时关掉。
func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
