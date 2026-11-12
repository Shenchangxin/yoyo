// 课本上的学习工具和导师守则。绑了课本的导师对话能看目录、读一章的内容。
// 随堂题给导师的也是不带答案的那一份：答案由引擎在读者作答之后亮出，导师不能提前拿到。

package app

import (
	"encoding/json"
	"fmt"
	"strings"
)

// learnBookPin 是绑了课本时的导师守则。
const learnBookPin = `## Book
- The learner is reading a generated book. Each message carries which chapter they have open. "This chapter" and "this part" mean that chapter.
- learn_book_outline lists the chapters. learn_book_chapter returns a chapter's blocks. Read the chapter before you explain, summarise, or extend it.
- The book's quiz blocks come without answers. Do not solve them for the learner before they try; give a hint and let them answer on the page.
- The book was written by a model. If the learner doubts a claim, check it against the bound library when there is one, and say so plainly when the book is wrong.`

// mountBookTools 在学习对话上挂课本工具。book 是会话绑定的课本。
func (a *App) mountBookTools(m *learnMount, book string) {
	s := a.Learn
	m.add("learn_book_outline", "List the chapters of the book the learner is reading: id, title, summary, objectives, status, and which chapter they last opened.", true, nil, nil, func(map[string]any) (learnOut, error) {
		view, err := s.Book(book)
		if err != nil {
			return learnOut{}, err
		}
		return learnOut{Value: map[string]any{
			"title": view.Book.Title, "status": view.Book.Status, "description": view.Proposal.Description,
			"chapters": view.Chapters, "current_chapter_id": view.Progress.ChapterID,
		}}, nil
	})
	m.add("learn_book_chapter", "Read one chapter of the book the learner is reading: its blocks in order. Omit chapter_id for the chapter they have open. Quiz blocks carry no answers.", true, map[string]any{
		"chapter_id": map[string]any{"type": "string"},
	}, nil, func(args map[string]any) (learnOut, error) {
		view, err := s.Book(book)
		if err != nil {
			return learnOut{}, err
		}
		target := strings.TrimSpace(strParam(args, "chapter_id"))
		if target == "" {
			target = view.Progress.ChapterID
		}
		for _, c := range view.Chapters {
			if c.ID != target && target != "" {
				continue
			}
			chapter, err := s.BookChapter(book, c.ID)
			if err != nil {
				return learnOut{}, err
			}
			raw, _ := json.Marshal(chapter)
			if len(raw) > 60000 {
				return learnOut{Value: map[string]any{"chapter": chapter.BookChapter, "clipped": true, "note": "This chapter is too long to return whole; ask the learner which part they mean."}}, nil
			}
			return learnOut{Value: map[string]any{"chapter": chapter}}, nil
		}
		return learnOut{}, fmt.Errorf("chapter not found; call learn_book_outline for the chapter ids")
	})
}

// learnBookBrief 返回这一轮开头给导师的课本简报：书名、写作状态，以及读者正打开的那一章。没绑课本或课本已经不在时返回空串。
func (a *App) learnBookBrief(meta SessionMeta) string {
	if a == nil || a.Learn == nil || meta.BookID == "" {
		return ""
	}
	view, err := a.Learn.Book(meta.BookID)
	if err != nil {
		return ""
	}
	state := map[string]any{"book": view.Book.Title, "status": view.Book.Status, "chapters": len(view.Chapters)}
	for i, c := range view.Chapters {
		if c.ID == view.Progress.ChapterID {
			state["open_chapter"] = map[string]any{"id": c.ID, "number": i + 1, "title": c.Title, "summary": c.Summary, "status": c.Status}
		}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return ""
	}
	return "<book_state>\n" + string(raw) + "\n</book_state>\n"
}
