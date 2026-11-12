// 锁住旧库升级：第一版的路径、作答、复习日程、练习题和课本在打开之后都还在，分数不变，迁移只跑一次。

package learn

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

// seedV1 在 root 下建一份第一版的库，写进一条有作答和日程的路径、两道练习题和一本编好的课本。
func seedV1(t *testing.T, root string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(root, "learn.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
	ts := now.Format(time.RFC3339Nano)
	due := now.Add(5 * 24 * time.Hour).Format(time.RFC3339Nano)
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO paths (id, library_id, name, mode, teaching, created_at, updated_at) VALUES ('path-1', '', '微积分', 'study', 'probe_first', ?, ?)`, []any{ts, ts}},
		{`INSERT INTO modules (id, path_id, name, objective, ord) VALUES ('mod-1', 'path-1', '基础', '', 0)`, nil},
		{`INSERT INTO points (id, path_id, module_id, name, kind, ord, mastery, attempts_json) VALUES ('kp-done', 'path-1', 'mod-1', '极限', 'memory', 0, 1, '[1,1,1]')`, nil},
		{`INSERT INTO points (id, path_id, module_id, name, kind, ord, mastery, attempts_json) VALUES ('kp-half', 'path-1', 'mod-1', '导数', 'memory', 1, 0.5, '[0,1]')`, nil},
		{`INSERT INTO points (id, path_id, module_id, name, kind, ord, qualitative, mastery_source) VALUES ('kp-idea', 'path-1', 'mod-1', '连续', 'concept', 2, 1, 'system')`, nil},
		{`INSERT INTO points (id, path_id, module_id, name, kind, ord, mastery_source) VALUES ('kp-claim', 'path-1', 'mod-1', '积分', 'procedure', 3, 'learner')`, nil},
		{`INSERT INTO points (id, path_id, module_id, name, kind, ord) VALUES ('kp-new', 'path-1', 'mod-1', '级数', 'design', 4)`, nil},
		{`INSERT INTO repetitions (point_id, path_id, interval_index, next_review_at, consecutive_correct) VALUES ('kp-done', 'path-1', 3, ?, 1)`, []any{due}},
		{`INSERT INTO interactions (id, path_id, point_id, status, question_type, prompt, expected_answer, options_json, created_at) VALUES ('q-old', 'path-1', 'kp-half', 'pending', 'choice', '选一个', '乙', '["甲","乙","丙"]', ?)`, []any{ts}},
		{`INSERT INTO questions (id, stem, qtype, answer, source, source_id, created_at) VALUES ('bank-1', '首都是哪', 'short', 'paris', 'mastery', 'q-x', ?)`, []any{ts}},
		{`INSERT INTO practice_state (question_id, due_at, interval_days, ease, lapses, review_count, streak) VALUES ('bank-1', ?, 10, 2.5, 1, 4, 2)`, []any{due}},
		{`INSERT INTO questions (id, stem, qtype, answer, source, created_at) VALUES ('bank-2', '没复习过', 'short', 'x', 'import', ?)`, []any{ts}},
		{`INSERT INTO practice_state (question_id, due_at, interval_days, ease) VALUES ('bank-2', ?, 1, 2.5)`, []any{ts}},
		{`INSERT INTO books (id, library_id, title, status, depth, proposal, spine_json, created_at, updated_at) VALUES ('book-1', '', '短书', 'ready', 'standard', '讲导数', '[{"title":"第一节","query":"极限"},{"title":"第二节","query":"导数"},{"title":"没编出来","query":"空"}]', ?, ?)`, []any{ts, ts}},
		{`INSERT INTO book_blocks (id, book_id, ord, kind, body, page_no) VALUES ('blk-1', 'book-1', 0, 'section', '第一节', 1)`, nil},
		{`INSERT INTO book_blocks (id, book_id, ord, kind, body, page_no) VALUES ('blk-2', 'book-1', 1, 'text', '极限的摘录', 1)`, nil},
		{`INSERT INTO book_blocks (id, book_id, ord, kind, body, page_no) VALUES ('blk-3', 'book-1', 2, 'section', '第二节', 2)`, nil},
		{`INSERT INTO book_blocks (id, book_id, ord, kind, body, page_no) VALUES ('blk-4', 'book-1', 3, 'code', 'f(x)', 2)`, nil},
		{`INSERT INTO book_progress (book_id, reader, page_no) VALUES ('book-1', 'local', 2)`, nil},
		{`INSERT INTO materials (id, library_id, document_id, kind, title, created_at) VALUES ('mat-1', 'lib', 'doc', 'doc', '讲义', ?)`, []any{ts}},
		{`INSERT INTO annotations (id, material_id, quote, note, page_no, created_at) VALUES ('ann-1', 'mat-1', '一句原文', '我的想法', 3, ?)`, []any{ts}},
	}
	for _, st := range stmts {
		if _, err := db.Exec(st.q, st.args...); err != nil {
			t.Fatalf("%s: %v", st.q, err)
		}
	}
}

// TestUpgradeFromV1 确认第一版的数据升级后分数和状态不变，旧日程的到期时间保留，再打开一次不会重复迁移。
func TestUpgradeFromV1(t *testing.T) {
	root := t.TempDir()
	seedV1(t, root)
	s, err := Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 2, 2, 8, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	view, err := s.Path("path-1")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]PointView{}
	for _, p := range view.Modules[0].Points {
		got[p.Name] = p
	}
	if p := got["极限"]; p.Status != StatusMastered || p.Source != "system" || p.Mastery < quantitativeGate || p.Attempts != 3 {
		t.Fatalf("极限 %+v", p)
	}
	if want := time.Date(2026, 2, 6, 8, 0, 0, 0, time.UTC); !parseTime(got["极限"].DueAt).Equal(want) {
		t.Fatalf("the old due date must survive: %s", got["极限"].DueAt)
	}
	if p := got["导数"]; p.Status != StatusLearning || p.Mastery != ComputeMastery([]float64{0, 1}) {
		t.Fatalf("导数 %+v", p)
	}
	if p := got["连续"]; p.Status != StatusMastered || p.Source != "system" || p.Gate != "qualitative" {
		t.Fatalf("连续 %+v", p)
	}
	if p := got["积分"]; p.Status != StatusMastered || p.Source != "learner" {
		t.Fatalf("积分 %+v", p)
	}
	if p := got["级数"]; p.Status != StatusNew {
		t.Fatalf("级数 %+v", p)
	}
	if view.Counts.Mastered != 3 || view.Counts.Learning != 1 || view.Counts.New != 1 || view.Retention != DefaultRetention {
		t.Fatalf("%+v", view.Counts)
	}
	// 旧的待答选择题：选项是字符串数组，标准答案是选项正文。
	if view.Next.Action != ActionAnswer || len(view.Next.Question.Options) != 3 || view.Next.Question.Options[1].Label != "B" {
		t.Fatalf("%+v", view.Next)
	}
	res, err := s.Answer(context.Background(), AnswerRequest{PathID: "path-1", Answer: "B"})
	if err != nil || !res.Correct {
		t.Fatalf("%+v %v", res, err)
	}

	var stability, difficulty float64
	var mistake int
	if err := s.db.QueryRow(`SELECT st.stability, st.difficulty, q.mistake FROM practice_state st JOIN questions q ON q.id = st.question_id WHERE q.id = 'bank-1'`).Scan(&stability, &difficulty, &mistake); err != nil {
		t.Fatal(err)
	}
	if stability != 10 || difficulty < 6.9 || difficulty > 7 || mistake != 1 {
		t.Fatalf("S=%v D=%v mistake=%d", stability, difficulty, mistake)
	}
	if err := s.db.QueryRow(`SELECT stability FROM practice_state WHERE question_id = 'bank-2'`).Scan(&stability); err != nil || stability != 0 {
		t.Fatalf("an unreviewed card stays unseen: %v %v", stability, err)
	}

	type chapter struct{ title, status, query string }
	rows, err := s.db.Query(`SELECT title, status, query FROM book_chapters WHERE book_id = 'book-1' ORDER BY ord`)
	if err != nil {
		t.Fatal(err)
	}
	var chapters []chapter
	for rows.Next() {
		var c chapter
		_ = rows.Scan(&c.title, &c.status, &c.query)
		chapters = append(chapters, c)
	}
	rows.Close()
	if len(chapters) != 3 || chapters[0] != (chapter{"第一节", "ready", "极限"}) || chapters[2] != (chapter{"没编出来", "pending", "空"}) {
		t.Fatalf("%+v", chapters)
	}
	var kind, payload, chapterID string
	if err := s.db.QueryRow(`SELECT kind, payload_json, chapter_id FROM book_blocks WHERE id = 'blk-4'`).Scan(&kind, &payload, &chapterID); err != nil {
		t.Fatal(err)
	}
	var body map[string]string
	_ = json.Unmarshal([]byte(payload), &body)
	if kind != "code" || body["code"] != "f(x)" || chapterID == "" {
		t.Fatalf("%s %s %s", kind, payload, chapterID)
	}
	var headings int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM book_blocks WHERE kind = 'section'`).Scan(&headings)
	if headings != 0 {
		t.Fatal("old heading blocks become chapter rows")
	}
	var progress string
	_ = s.db.QueryRow(`SELECT chapter_id FROM book_progress WHERE book_id = 'book-1'`).Scan(&progress)
	if progress != chapterID {
		t.Fatalf("progress %q want %q", progress, chapterID)
	}
	var annKind string
	_ = s.db.QueryRow(`SELECT kind FROM annotations WHERE id = 'ann-1'`).Scan(&annKind)
	if annKind != "highlight" {
		t.Fatalf("annotation kind %q", annKind)
	}

	var evidenceBefore int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM evidence`).Scan(&evidenceBefore)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var evidenceAfter, applied int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM evidence`).Scan(&evidenceAfter)
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM schema_migrations`).Scan(&applied)
	if evidenceAfter != evidenceBefore || applied != len(migrations) {
		t.Fatalf("migrations must run once: evidence %d -> %d, applied %d of %d", evidenceBefore, evidenceAfter, applied, len(migrations))
	}
}

// TestMigrationIDsAreUnique 确认迁移编号不重复，而且每一步都有名字。
func TestMigrationIDsAreUnique(t *testing.T) {
	seen := map[int]string{}
	for _, m := range migrations {
		if m.name == "" || m.run == nil {
			t.Fatalf("migration %d is incomplete", m.id)
		}
		if prev, ok := seen[m.id]; ok {
			t.Fatalf("migration %d registered twice: %s and %s", m.id, prev, m.name)
		}
		seen[m.id] = m.name
	}
}
