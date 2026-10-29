// 锁住掌握分上限、练习 Again 间隔、答案不外泄、迁移保留页面 id、发布失败不换代，以及手记只注入使命和上一课。

package learn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/pages"
)

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

func TestIntervalSequences(t *testing.T) {
	mem := IntervalDays("memory")
	want := []int{0, 1, 3, 7, 14, 30, 60}
	if len(mem) != len(want) {
		t.Fatal(mem)
	}
	for i := range want {
		if mem[i] != want[i] {
			t.Fatal(mem)
		}
	}
	if IntervalDays("concept")[0] != 3 || IntervalDays("procedure")[0] != 3 || IntervalDays("design")[0] != 14 {
		t.Fatal("type intervals drifted")
	}
}

func TestPracticeStreakPersists(t *testing.T) {
	s := openTemp(t)
	id, err := s.ImportQuestion("题", "short", "答", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.RateCard(id, "good", now); err != nil {
		t.Fatal(err)
	}
	if err := s.RateCard(id, "good", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var streak int
	if err := s.db.QueryRow(`SELECT streak FROM practice_state WHERE question_id = ?`, id).Scan(&streak); err != nil {
		t.Fatal(err)
	}
	if streak != 2 {
		t.Fatalf("streak %d", streak)
	}
}

func TestPracticeAgainIsTenMinutes(t *testing.T) {
	next, err := SchedulePractice(PracticeState{IntervalDays: 1, Ease: 2.5}, "again", time.Unix(1_700_000_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if next.IntervalDays != 0.007 {
		t.Fatalf("again interval = %v, want 0.007 days", next.IntervalDays)
	}
	if next.Ease != 2.3 || next.Lapses != 1 || next.Streak != 0 {
		t.Fatalf("%+v", next)
	}
	good, err := SchedulePractice(PracticeState{IntervalDays: 1, Ease: 2.5}, "good", time.Unix(0, 0).UTC())
	if err != nil || good.IntervalDays != 3 {
		t.Fatalf("first good = %+v %v", good, err)
	}
}

func TestQuizHidesExpectedAnswer(t *testing.T) {
	s := openTemp(t)
	path, err := s.CreatePath("导数", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceOutline(path.ID, []OutlineModule{{Name: "变化", Points: []OutlinePoint{{Name: "极限", Kind: "memory"}}}}); err != nil {
		t.Fatal(err)
	}
	view, _ := s.Path(path.ID)
	point := view.Modules[0].Points[0].ID
	pub, err := s.PoseQuiz(path.ID, point, "sess", "short", "极限是什么", "secret-answer", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(pub)
	if strings.Contains(string(raw), "secret-answer") || strings.Contains(string(raw), "expected_answer") {
		t.Fatal(string(raw))
	}
	again, err := s.Path(path.ID)
	if err != nil || again.Next == nil || again.Next.Question == nil {
		t.Fatal(again)
	}
	raw, _ = json.Marshal(again.Next)
	if strings.Contains(string(raw), "secret-answer") {
		t.Fatal(string(raw))
	}
}

func TestLearnerClaimIsNotAssessed(t *testing.T) {
	s := openTemp(t)
	path, err := s.CreatePath("声明", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = s.ReplaceOutline(path.ID, []OutlineModule{{Name: "章", Points: []OutlinePoint{{Name: "点", Kind: "concept"}}}})
	view, _ := s.Path(path.ID)
	id := view.Modules[0].Points[0].ID
	if err := s.ClaimMastery(path.ID, id, "我已经会了"); err != nil {
		t.Fatal(err)
	}
	view, _ = s.Path(path.ID)
	pt := view.Modules[0].Points[0]
	if pt.Source != "learner" || pt.Status != "mastered" {
		t.Fatalf("%+v", pt)
	}
}

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

func TestSearchEmptyAndCJK(t *testing.T) {
	s := openTemp(t)
	pg, err := s.Create(pages.Page{Title: "材料", Content: "这是学习材料，讲检索。"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildSync("notes"); err != nil {
		t.Fatal(err)
	}
	hits, msg, err := s.SearchLibrary("notes", "zzzz-no-such-token")
	if err != nil || len(hits) != 0 || msg != "检索完成，没有命中" {
		t.Fatalf("%v %q %v", hits, msg, err)
	}
	hits, msg, err = s.SearchLibrary("notes", "学习")
	if err != nil || msg != "" || len(hits) == 0 || hits[0].DocumentID != pg.ID {
		t.Fatalf("%v %q %v", hits, msg, err)
	}
}

func TestFailedPublishKeepsLiveGeneration(t *testing.T) {
	s := openTemp(t)
	lib, err := s.CreateLibrary("讲义", "indexed", "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "note.md")
	if err := os.WriteFile(file, []byte("alpha unique token"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := s.Import(lib.ID, file)
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if err := s.RebuildSync(lib.ID); err != nil {
		t.Fatal(err)
	}
	var gen int
	if err := s.db.QueryRow(`SELECT published_gen FROM libraries WHERE id = ?`, lib.ID).Scan(&gen); err != nil || gen == 0 {
		t.Fatal(gen, err)
	}
	var sum string
	if err := s.db.QueryRow(`SELECT sha256 FROM documents WHERE library_id = ?`, lib.ID).Scan(&sum); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root(), "raw", sum+".md"), []byte("replaced bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildSync(lib.ID); err == nil {
		t.Fatal("changed source must not publish")
	}
	var after int
	if err := s.db.QueryRow(`SELECT published_gen FROM libraries WHERE id = ?`, lib.ID).Scan(&after); err != nil || after != gen {
		t.Fatalf("published %d want %d (%v)", after, gen, err)
	}
	hits, _, err := s.SearchLibrary(lib.ID, "alpha")
	if err != nil || len(hits) == 0 {
		t.Fatalf("live index lost: %v %v", hits, err)
	}
}

func TestMasteryMissWritesQuestion(t *testing.T) {
	s := openTemp(t)
	path, err := s.CreatePath("练习", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = s.ReplaceOutline(path.ID, []OutlineModule{{Name: "章", Points: []OutlinePoint{{Name: "点", Kind: "memory"}}}})
	view, _ := s.Path(path.ID)
	_, err = s.PoseQuiz(path.ID, view.Modules[0].Points[0].ID, "s", "short", "首都是哪", "paris", nil)
	if err != nil {
		t.Fatal(err)
	}
	graded, err := s.GradeQuiz(path.ID, "lyon")
	if err != nil || graded["correct"] != false {
		t.Fatal(graded, err)
	}
	cards := s.Questions()
	if len(cards) != 1 || cards[0].Stem != "首都是哪" {
		t.Fatal(cards)
	}
	raw, _ := json.Marshal(cards[0])
	if strings.Contains(string(raw), "paris") {
		t.Fatal(string(raw))
	}
}

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
