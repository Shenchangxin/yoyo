// 锁住课本在应用这一层的接线：界面用的方法能把一本书从意图走到写完并读一章、答一道随堂题；
// 绑了课本的导师对话开场就有课本工具，读到的章节里没有随堂题的答案，回合简报说得出读者正打开哪一章。

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// fakeBookModel 是写课本用的假模型：按系统提示认出请求的种类，各回一份合格的 JSON。
func fakeBookModel(_ context.Context, req learn.CompleteRequest) (string, error) {
	switch {
	case strings.Contains(req.System, "plan a short book"):
		return `{"title":"导数速成","description":"读完会求导。","scope":"一元函数。","target_level":"高中数学。","estimated_chapters":2,"rationale":"够用。"}`, nil
	case strings.Contains(req.System, "prepare library searches"):
		return `{"queries":["导数定义"]}`, nil
	case strings.Contains(req.System, "outline a book"):
		return `{"chapters":[{"title":"导数的定义","summary":"割线到切线。","objectives":["说出定义"]},{"title":"求导法则","summary":"四则和链式。","objectives":["会用法则"]}]}`, nil
	case strings.Contains(req.System, "plan one chapter"):
		return `{"blocks":[{"kind":"text","focus":"开场"},{"kind":"quiz","focus":"检查"}]}`, nil
	case strings.Contains(req.System, "write one block"):
		if strings.Contains(req.User, "Block kind: quiz") {
			return `{"questions":[{"question":"导数的几何意义是？","qtype":"choice","options":["面积","切线斜率","弧长","体积"],"answer":"切线斜率","explanation":"secret-explanation"}]}`, nil
		}
		return `{"body":"导数是瞬时变化率。"}`, nil
	}
	return "", fmt.Errorf("unexpected request")
}

// TestLearnCallBookRoundTrip 确认界面用的方法能建书、确认提案和书脊、等它写完、读一章、答随堂题和导出；
// 绑了课本的导师对话有课本工具，章节里没有答案，简报带着正打开的那一章。
func TestLearnCallBookRoundTrip(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	a.Learn.Complete = fakeBookModel
	out, err := a.LearnCall("learn.book.create", map[string]any{"intent": "我想学会求导", "depth": "brief"})
	if err != nil {
		t.Fatal(err)
	}
	draft := out.(learn.BookDraft)
	id := draft.Book.ID
	if draft.Proposal.Title != "导数速成" || draft.Book.Status != learn.BookStatusDraft {
		t.Fatalf("draft %+v", draft)
	}
	if _, err := a.LearnCall("learn.book.confirm_proposal", map[string]any{"id": id}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 400; i++ {
		if view, _ := a.Learn.Book(id); view.Book.Status == learn.BookStatusSpineReady {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := a.LearnCall("learn.book.confirm_spine", map[string]any{"id": id}); err != nil {
		t.Fatal(err)
	}
	a.Learn.WaitBook(id)
	got, err := a.LearnCall("learn.book", map[string]any{"id": id})
	if err != nil || got.(learn.BookView).Book.Status != learn.BookStatusReady || len(got.(learn.BookView).Chapters) != 2 {
		t.Fatalf("book %+v %v", got, err)
	}
	opened, err := a.LearnCall("learn.book.chapter", map[string]any{"id": id})
	if err != nil {
		t.Fatal(err)
	}
	chapter := opened.(learn.BookChapterView)
	raw, _ := json.Marshal(chapter)
	if len(chapter.Blocks) != 2 || strings.Contains(string(raw), "secret-explanation") {
		t.Fatalf("chapter %s", raw)
	}
	var parsed struct {
		Blocks []struct {
			ID      string `json:"id"`
			Kind    string `json:"kind"`
			Payload struct {
				Questions []struct {
					ID      string         `json:"id"`
					Options []learn.Option `json:"options"`
				} `json:"questions"`
			} `json:"payload"`
		} `json:"blocks"`
	}
	_ = json.Unmarshal(raw, &parsed)
	quiz := parsed.Blocks[1]
	right := ""
	for _, o := range quiz.Payload.Questions[0].Options {
		if o.Body == "切线斜率" {
			right = o.Label
		}
	}
	answered, err := a.LearnCall("learn.book.quiz", map[string]any{"id": id, "block_id": quiz.ID, "question_id": quiz.Payload.Questions[0].ID, "answer": right})
	if err != nil || answered.(learn.BookQuizResult).Correct == nil || !*answered.(learn.BookQuizResult).Correct || answered.(learn.BookQuizResult).Explanation != "secret-explanation" {
		t.Fatalf("quiz %+v %v", answered, err)
	}
	if _, err := a.LearnCall("learn.book.block.insert", map[string]any{"id": id, "chapter_id": chapter.ID, "kind": "user_note", "body": "我的笔记"}); err != nil {
		t.Fatal(err)
	}
	exported, err := a.LearnCall("learn.book.export", map[string]any{"id": id})
	if err != nil || !strings.Contains(exported.(map[string]any)["markdown"].(string), "> 我的笔记") {
		t.Fatalf("export %+v %v", exported, err)
	}

	sess, err := a.NewSessionOn(t.TempDir(), session.ChannelLearn)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := a.SetSessionLearn(sess.ID, LearnBinding{BookID: id})
	if err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{SessionID: sess.ID, ChatOverlay: true}
	a.attachLearn(tools)
	listed := map[string]bool{}
	for _, spec := range runtime.AllToolJSON(tools) {
		name, _ := spec.Function["name"].(string)
		listed[name] = true
	}
	if !listed["learn_book_outline"] || !listed["learn_book_chapter"] {
		t.Fatal("book tools are not in the opening tool list")
	}
	_, res := callLearn(t, tools, "learn_book_chapter", map[string]any{})
	if res.Err != nil || !strings.Contains(res.Content, "导数是瞬时变化率") || strings.Contains(res.Content, "secret-explanation") || strings.Contains(res.Content, `"answer"`) {
		t.Fatalf("the tutor must read the chapter without quiz answers: %s %v", res.Content, res.Err)
	}
	if brief := a.learnTurnBrief(meta); !strings.Contains(brief, "<book_state>") || !strings.Contains(brief, `"number":1`) || !strings.Contains(brief, "导数的定义") {
		t.Fatalf("brief:\n%s", brief)
	}
	_, frags := applyLearnLanePrompt(artifact.LoopPreset{}, nil, meta)
	if text := runtime.AssembleIdentity(artifact.LoopPreset{}, frags); !strings.Contains(text, "## Book") || !strings.Contains(text, "## Sources") {
		t.Fatalf("a book chat carries the book and source pins:\n%s", text)
	}
	if _, err := a.LearnCall("learn.book.delete", map[string]any{"id": id}); err != nil {
		t.Fatal(err)
	}
	if list, _ := a.LearnCall("learn.books", nil); len(list.([]learn.BookSummary)) != 0 {
		t.Fatalf("books after delete %+v", list)
	}
}
