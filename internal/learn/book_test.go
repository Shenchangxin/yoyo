// 锁住课本的不变量：提案和书脊各等读者确认一次，确认书脊后逐章写完；随堂题给界面时不带答案，答错进题库；
// 连续两章写不出来自己停下并留下原因，恢复后只补没写完的；重写一章时读者的笔记和改动留着；没接模型时不留下半本书。

package learn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// bookModel 是写课本用的假模型。它按系统提示认出这一次是写提案、想检索词、排书脊、排章节还是写一块，各回一份合格的 JSON。
// fail 非空时，写块的请求里出现这个词就报错。calls 记下每一类请求的次数。
type bookModel struct {
	mu    sync.Mutex
	fail  string
	calls map[string]int
}

// complete 是交给 Store.Complete 的函数。
func (m *bookModel) complete(_ context.Context, req CompleteRequest) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.calls == nil {
		m.calls = map[string]int{}
	}
	switch {
	case strings.Contains(req.System, "plan a short book"):
		m.calls["proposal"]++
		return `{"title":"反向传播入门","description":"读完能手推两层网络的梯度。","scope":"只讲全连接网络。","target_level":"会求导。","estimated_chapters":"2","rationale":"贴合意图。"}`, nil
	case strings.Contains(req.System, "prepare library searches"):
		m.calls["queries"]++
		return `{"queries":["链式法则","计算图"]}`, nil
	case strings.Contains(req.System, "outline a book"):
		m.calls["spine"]++
		return `{"chapters":[{"title":"链式法则","summary":"复合函数怎么求导。","objectives":["写出链式法则"],"content_type":"theory"},{"title":"计算图","summary":"把计算画成图。","objectives":["画出计算图"],"content_type":"concept"}]}`, nil
	case strings.Contains(req.System, "plan one chapter"):
		m.calls["plan"]++
		return `{"blocks":[{"kind":"text","title":"开场","focus":"为什么要学"},{"kind":"section","title":"主体","focus":"讲清楚"},{"kind":"callout","variant":"pitfall","focus":"常见错误"},{"kind":"video","title":"不认得的种类"},{"kind":"quiz","focus":"检查目标"}]}`, nil
	case strings.Contains(req.System, "write one block"):
		kind := ""
		if i := strings.Index(req.User, "Block kind: "); i >= 0 {
			kind = strings.Fields(req.User[i+len("Block kind: "):])[0]
		}
		m.calls["write:"+kind]++
		if m.fail != "" && strings.Contains(req.User, m.fail) {
			return "", fmt.Errorf("boom")
		}
		switch kind {
		case BlockSection:
			return `{"intro":"先看一个例子。","subsections":[{"heading":"第一步","body":"外层求导。"},{"heading":"第二步","body":"乘内层的导数。"}],"key_takeaway":"外导乘内导。"}`, nil
		case BlockCallout:
			return `{"body":"别把 f'(g(x)) 写成 f'(x)。"}`, nil
		case BlockQuiz:
			return `{"questions":[{"question":"y=f(g(x)) 的导数是？","qtype":"choice","options":["f'(x)g'(x)","f'(g(x))g'(x)","f(g'(x))","f'(g(x))+g'(x)"],"answer":"f'(g(x))g'(x)","explanation":"外导乘内导。"},{"question":"链式法则用来做什么？","qtype":"short","answer":"求复合函数的导数","explanation":"它把导数沿着复合的层次乘起来。"}]}`, nil
		default:
			return `{"body":"这一章讲链式法则。"}`, nil
		}
	}
	return "", fmt.Errorf("unexpected request: %s", clipRunes(req.System, 60))
}

// waitBookStatus 等课本离开 from 这个状态，最多等两秒，返回离开之后的全貌。
func waitBookStatus(t *testing.T, s *Store, id string, settled func(BookSummary) bool) BookView {
	t.Helper()
	for i := 0; i < 400; i++ {
		view, err := s.Book(id)
		if err != nil {
			t.Fatal(err)
		}
		if settled(view.Book) {
			return view
		}
		time.Sleep(5 * time.Millisecond)
	}
	view, _ := s.Book(id)
	t.Fatalf("the book did not settle: %+v", view.Book)
	return BookView{}
}

// readyBook 把一本书从意图一路走到写完，返回它的全貌。
func readyBook(t *testing.T, s *Store, m *bookModel) BookView {
	t.Helper()
	s.Complete = m.complete
	draft, err := s.CreateBook(context.Background(), CreateBookRequest{Intent: "我想学会手推反向传播"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmProposal(draft.Book.ID, nil, ""); err != nil {
		t.Fatal(err)
	}
	waitBookStatus(t, s, draft.Book.ID, func(b BookSummary) bool { return b.Status == BookStatusSpineReady })
	if _, err := s.ConfirmSpine(draft.Book.ID, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(draft.Book.ID)
	view, err := s.Book(draft.Book.ID)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

// TestBookFromIntentToReady 确认一本书从意图走到写完要过提案和书脊两道确认，写完后每章的块都在，随堂题不带答案，
// 打开章节记进度，答错的随堂题进题库，导出的 Markdown 里没有答案。
func TestBookFromIntentToReady(t *testing.T) {
	s := openTemp(t)
	var kinds []string
	var mu sync.Mutex
	s.OnChange = func(c Change) {
		if c.Scope == "book" {
			mu.Lock()
			kinds = append(kinds, c.Kind)
			mu.Unlock()
		}
	}
	m := &bookModel{}
	s.Complete = m.complete
	draft, err := s.CreateBook(context.Background(), CreateBookRequest{Intent: "我想学会手推反向传播", Depth: "brief"})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Book.Status != BookStatusDraft || draft.Proposal.EstimatedChapters != 2 || draft.Book.Title != "反向传播入门" || draft.Book.Language != "zh" {
		t.Fatalf("draft %+v", draft)
	}
	if _, err := s.ConfirmSpine(draft.Book.ID, ""); err == nil {
		t.Fatal("the chapter list cannot be confirmed before the proposal")
	}
	edited := draft.Proposal
	edited.Scope = "只讲两层网络。"
	if _, err := s.ConfirmProposal(draft.Book.ID, &edited, ""); err != nil {
		t.Fatal(err)
	}
	view := waitBookStatus(t, s, draft.Book.ID, func(b BookSummary) bool { return b.Status == BookStatusSpineReady })
	if len(view.Chapters) != 2 || view.Proposal.Scope != "只讲两层网络。" || view.Chapters[0].Status != chapterPending {
		t.Fatalf("spine %+v", view)
	}
	spine := []BookChapterInput{
		{ID: view.Chapters[0].ID, Title: "链式法则（改）", Summary: view.Chapters[0].Summary, Objectives: view.Chapters[0].Objectives, ContentType: "theory"},
		{ID: view.Chapters[1].ID, Title: view.Chapters[1].Title, Summary: view.Chapters[1].Summary, Objectives: view.Chapters[1].Objectives},
	}
	if chapters, err := s.SetBookSpine(draft.Book.ID, spine); err != nil || len(chapters) != 2 || chapters[0].Title != "链式法则（改）" {
		t.Fatalf("edited spine %+v %v", chapters, err)
	}
	if _, err := s.ConfirmSpine(draft.Book.ID, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(draft.Book.ID)
	view, err = s.Book(draft.Book.ID)
	if err != nil || view.Book.Status != BookStatusReady || view.Book.ReadyChapters != 2 || view.Book.PauseReason != "" {
		t.Fatalf("finished book %+v %v", view.Book, err)
	}
	chapter, err := s.OpenBookChapter(draft.Book.ID, "")
	if err != nil || chapter.ID != view.Chapters[0].ID || chapter.Status != chapterReady {
		t.Fatalf("first chapter %+v %v", chapter.BookChapter, err)
	}
	var order []string
	var quiz BookBlock
	for _, blk := range chapter.Blocks {
		order = append(order, blk.Kind)
		if blk.Status != blockReady {
			t.Fatalf("block %s is %s: %s", blk.Kind, blk.Status, blk.Error)
		}
		if blk.Kind == BlockQuiz {
			quiz = blk
		}
	}
	if got := strings.Join(order, ","); got != "text,section,callout,quiz" {
		t.Fatalf("blocks %s, want the planned kinds with the unknown one dropped and the quiz last", got)
	}
	raw, _ := json.Marshal(chapter)
	if strings.Contains(string(raw), "外导乘内导。\"}]") || strings.Contains(string(raw), `"answer"`) || strings.Contains(string(raw), `"explanation":"它把`) {
		t.Fatalf("a chapter must not carry quiz answers: %s", raw)
	}
	public := quiz.Payload.(bookQuizPublic)
	if len(public.Questions) != 2 || len(public.Questions[0].Options) != 4 {
		t.Fatalf("quiz %+v", public)
	}
	wrong, right := "", ""
	for _, o := range public.Questions[0].Options {
		if o.Body == "f'(g(x))g'(x)" {
			right = o.Label
		} else {
			wrong = o.Label
		}
	}
	res, err := s.AnswerBookQuiz(context.Background(), BookQuizRequest{BookID: draft.Book.ID, BlockID: quiz.ID, QuestionID: public.Questions[0].ID, Answer: wrong})
	if err != nil || res.Correct == nil || *res.Correct || res.Answer != right || res.Explanation == "" {
		t.Fatalf("wrong answer %+v %v", res, err)
	}
	if _, err := s.AnswerBookQuiz(context.Background(), BookQuizRequest{BookID: draft.Book.ID, BlockID: quiz.ID, QuestionID: public.Questions[0].ID, Answer: wrong}); err != nil {
		t.Fatal(err)
	}
	page, err := s.ListQuestions(BankQuery{Source: "book"})
	if err != nil || page.Total != 1 || !page.Items[0].Mistake || page.Items[0].SourceID != draft.Book.ID {
		t.Fatalf("a missed quiz question lands in the bank once: %+v %v", page, err)
	}
	res, err = s.AnswerBookQuiz(context.Background(), BookQuizRequest{BookID: draft.Book.ID, BlockID: quiz.ID, QuestionID: public.Questions[0].ID, Answer: right})
	if err != nil || res.Correct == nil || !*res.Correct {
		t.Fatalf("right answer %+v %v", res, err)
	}
	if res, err := s.AnswerBookQuiz(context.Background(), BookQuizRequest{BookID: draft.Book.ID, BlockID: quiz.ID, QuestionID: public.Questions[1].ID, Answer: " 求复合函数的导数 "}); err != nil || res.Correct == nil || !*res.Correct {
		t.Fatalf("an exact short answer is right without a judge: %+v %v", res, err)
	}
	after, _ := s.Book(draft.Book.ID)
	if after.Progress.ChapterID != chapter.ID || after.Book.Percent != 0.5 {
		t.Fatalf("progress %+v %+v", after.Progress, after.Book)
	}
	name, md, err := s.ExportBook(draft.Book.ID)
	if err != nil || !strings.HasSuffix(name, ".md") || !strings.Contains(md, "## 1. 链式法则（改）") || !strings.Contains(md, "#### 第一步") || strings.Contains(md, "外导乘内导。\n\n它把") {
		t.Fatalf("export %q %v\n%s", name, err, md)
	}
	if strings.Contains(md, "求复合函数的导数") {
		t.Fatalf("an export must not carry quiz answers:\n%s", md)
	}
	mu.Lock()
	seen := strings.Join(kinds, ",")
	mu.Unlock()
	for _, want := range []string{"spine_ready", "chapter_started", "block_ready", "chapter_ready", "ready"} {
		if !strings.Contains(seen, want) {
			t.Fatalf("change events %s lack %s", seen, want)
		}
	}
	if list := s.Books(); len(list) != 1 || list[0].Chapters != 2 {
		t.Fatalf("books %+v", list)
	}
}

// TestBookStopsAfterTwoFailedChapters 确认连续两章一块都写不出来时写作自己停下并留下原因，恢复之后只补没写出来的块，写好的不重写。
func TestBookStopsAfterTwoFailedChapters(t *testing.T) {
	s := openTemp(t)
	m := &bookModel{fail: "<book>"}
	s.Complete = m.complete
	draft, err := s.CreateBook(context.Background(), CreateBookRequest{Intent: "我想学会手推反向传播"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmProposal(draft.Book.ID, nil, ""); err != nil {
		t.Fatal(err)
	}
	waitBookStatus(t, s, draft.Book.ID, func(b BookSummary) bool { return b.Status == BookStatusSpineReady })
	if _, err := s.ConfirmSpine(draft.Book.ID, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(draft.Book.ID)
	view, _ := s.Book(draft.Book.ID)
	if view.Book.Status != BookStatusPaused || !strings.Contains(view.Book.PauseReason, "two chapters in a row") || !strings.Contains(view.Book.PauseReason, "boom") {
		t.Fatalf("two failed chapters must pause the book with the reason: %+v", view.Book)
	}
	if view.Chapters[0].Status != chapterError || view.Chapters[1].Status != chapterError {
		t.Fatalf("chapters %+v", view.Chapters)
	}
	if err := s.PauseBook(draft.Book.ID); err == nil {
		t.Fatal("a paused book cannot be paused again")
	}
	m.mu.Lock()
	m.fail = ""
	plans := m.calls["plan"]
	m.mu.Unlock()
	if err := s.ResumeBook(draft.Book.ID, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(draft.Book.ID)
	view, _ = s.Book(draft.Book.ID)
	if view.Book.Status != BookStatusReady || view.Book.ReadyChapters != 2 {
		t.Fatalf("after resuming %+v", view.Book)
	}
	m.mu.Lock()
	replanned := m.calls["plan"] - plans
	m.mu.Unlock()
	if replanned != 0 {
		t.Fatalf("resuming planned %d chapters again, want the stored plans reused", replanned)
	}
}

// TestBookKeepsReaderWorkOnRegenerate 确认重写一章时读者的笔记和读者改过的块留着，其余的块换成新写的；
// 读者的笔记不能交给模型重写，随堂题不能手改。
func TestBookKeepsReaderWorkOnRegenerate(t *testing.T) {
	s := openTemp(t)
	m := &bookModel{}
	view := readyBook(t, s, m)
	id := view.Book.ID
	chapter, err := s.OpenBookChapter(id, view.Chapters[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	text, section, quiz := chapter.Blocks[0], chapter.Blocks[1], chapter.Blocks[3]
	edited, err := s.UpdateBlock(id, text.ID, "我改过的开场。")
	if err != nil || !edited.Edited || edited.Payload.(bookText).Body != "我改过的开场。" {
		t.Fatalf("edit %+v %v", edited, err)
	}
	if _, err := s.UpdateBlock(id, quiz.ID, "手改随堂题"); err == nil {
		t.Fatal("a quiz cannot be edited by hand")
	}
	note, err := s.InsertBlock(id, chapter.ID, section.ID, BlockUserNote, "我的笔记：记住外导乘内导。")
	if err != nil || note.Kind != BlockUserNote || note.Order != 2 {
		t.Fatalf("note %+v %v", note, err)
	}
	if err := s.RegenerateBlock(id, note.ID, ""); err == nil {
		t.Fatal("the reader's note is not rewritten by the model")
	}
	if err := s.RegenerateChapter(id, chapter.ID, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(id)
	again, err := s.OpenBookChapter(id, chapter.ID)
	if err != nil || again.Status != chapterReady {
		t.Fatalf("regenerated chapter %+v %v", again.BookChapter, err)
	}
	kept := map[string]bool{}
	for _, blk := range again.Blocks {
		kept[blk.ID] = true
		if blk.Status != blockReady {
			t.Fatalf("block %s is %s", blk.Kind, blk.Status)
		}
	}
	if !kept[text.ID] || !kept[note.ID] || kept[section.ID] || kept[quiz.ID] {
		t.Fatalf("regenerating must keep the reader's edit and note and replace the rest: %+v", again.Blocks)
	}
	if err := s.DeleteBlock(id, note.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RegenerateBlock(id, again.Blocks[len(again.Blocks)-1].ID, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(id)
	if final, _ := s.Book(id); final.Book.Status != BookStatusReady {
		t.Fatalf("after regenerating one block %+v", final.Book)
	}
	if err := s.DeleteBook(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Book(id); err == nil {
		t.Fatal("a deleted book must be gone")
	}
}

// TestBookPauseIsDurable 确认没接模型时不留下半本书；写到一半关掉再打开，课本停在暂停，写到一半的块退回待写，恢复后接着写完。
func TestBookPauseIsDurable(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateBook(context.Background(), CreateBookRequest{Intent: "没有模型"}); err == nil {
		t.Fatal("creating a book without a model must fail")
	}
	if list := s.Books(); len(list) != 0 {
		t.Fatalf("a failed proposal must leave nothing behind: %+v", list)
	}
	m := &bookModel{}
	view := readyBook(t, s, m)
	id := view.Book.ID
	if _, err := s.db.Exec(`UPDATE books SET status = ? WHERE id = ?`, BookStatusCompiling, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE book_chapters SET status = ? WHERE id = ?`, chapterGenerating, view.Chapters[1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE book_blocks SET status = ? WHERE chapter_id = ? AND kind = ?`, blockGenerating, view.Chapters[1].ID, BlockQuiz); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.Complete = m.complete
	got, _ := s.Book(id)
	if got.Book.Status != BookStatusPaused || got.Book.PauseReason != BookPauseInterrupted {
		t.Fatalf("an interrupted book must reopen paused: %+v", got.Book)
	}
	m.mu.Lock()
	before := m.calls["write:quiz"]
	sections := m.calls["write:section"]
	m.mu.Unlock()
	if err := s.ResumeBook(id, ""); err != nil {
		t.Fatal(err)
	}
	s.WaitBook(id)
	got, _ = s.Book(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	if got.Book.Status != BookStatusReady || m.calls["write:quiz"] != before+1 || m.calls["write:section"] != sections {
		t.Fatalf("resuming must write only the unfinished block: %+v quiz %d->%d section %d->%d", got.Book, before, m.calls["write:quiz"], sections, m.calls["write:section"])
	}
}
