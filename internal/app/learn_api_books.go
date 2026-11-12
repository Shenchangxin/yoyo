// 课本的桌面接口：建书、确认提案和书脊、读章节、暂停和恢复、改块、作答随堂题和导出。
// 这里只做参数转接，写作和批改在 internal/learn。形状见 docs/architecture/learn.md。

package app

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/learn"
)

// learnBooksCall 处理课本的方法。ok 为假表示这个方法不归它管。
func (a *App) learnBooksCall(method string, params map[string]any) (any, bool, error) {
	s := a.Learn
	id := strParam(params, "id")
	session := strParam(params, "session")
	switch method {
	case "learn.books":
		return s.Books(), true, nil
	case "learn.book.create":
		out, err := s.CreateBook(context.Background(), learn.CreateBookRequest{
			Intent: strParam(params, "intent"), LibraryID: strParam(params, "library_id"), Depth: strParam(params, "depth"),
			Language: strParam(params, "language"), SessionID: session,
		})
		return out, true, err
	case "learn.book":
		out, err := s.Book(id)
		return out, true, err
	case "learn.book.proposal":
		var proposal learn.BookProposal
		if err := decodeParam(params, "proposal", &proposal); err != nil {
			return nil, true, err
		}
		return okMap(), true, s.SetBookProposal(id, proposal)
	case "learn.book.confirm_proposal":
		var proposal *learn.BookProposal
		if hasParam(params, "proposal") {
			var p learn.BookProposal
			if err := decodeParam(params, "proposal", &p); err != nil {
				return nil, true, err
			}
			proposal = &p
		}
		out, err := s.ConfirmProposal(id, proposal, session)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"book": out}, true, nil
	case "learn.book.spine":
		var chapters []learn.BookChapterInput
		if err := decodeParam(params, "chapters", &chapters); err != nil {
			return nil, true, err
		}
		out, err := s.SetBookSpine(id, chapters)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"chapters": out}, true, nil
	case "learn.book.confirm_spine":
		out, err := s.ConfirmSpine(id, session)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"book": out}, true, nil
	case "learn.book.chapter":
		out, err := s.OpenBookChapter(id, strParam(params, "chapter_id"))
		return out, true, err
	case "learn.book.pause":
		return okMap(), true, s.PauseBook(id)
	case "learn.book.resume":
		return okMap(), true, s.ResumeBook(id, session)
	case "learn.book.delete":
		return okMap(), true, s.DeleteBook(id)
	case "learn.book.rename":
		return okMap(), true, s.RenameBook(id, strParam(params, "title"))
	case "learn.book.chapter.regenerate":
		return okMap(), true, s.RegenerateChapter(id, strParam(params, "chapter_id"), session)
	case "learn.book.block.regenerate":
		return okMap(), true, s.RegenerateBlock(id, strParam(params, "block_id"), session)
	case "learn.book.block.delete":
		return okMap(), true, s.DeleteBlock(id, strParam(params, "block_id"))
	case "learn.book.block.update":
		out, err := s.UpdateBlock(id, strParam(params, "block_id"), strParam(params, "body"))
		return out, true, err
	case "learn.book.block.insert":
		out, err := s.InsertBlock(id, strParam(params, "chapter_id"), strParam(params, "after_block_id"), strParam(params, "kind"), strParam(params, "body"))
		return out, true, err
	case "learn.book.quiz":
		out, err := s.AnswerBookQuiz(context.Background(), learn.BookQuizRequest{
			BookID: id, BlockID: strParam(params, "block_id"), QuestionID: strParam(params, "question_id"),
			Answer: strParam(params, "answer"), SessionID: session,
		})
		return out, true, err
	case "learn.book.export":
		name, body, err := s.ExportBook(id)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"filename": name, "markdown": body}, true, nil
	}
	return nil, false, nil
}
