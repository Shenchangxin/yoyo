// 读者对课本里单个块的操作：改正文、插一条自己的笔记、删一块、重写一块，以及作答随堂题和把整本书导成 Markdown。
// 读者改过的块和读者的笔记在重写章节时留着。随堂题在这里批改，答案和解析只在作答之后给出；答错的选择题进题库。
// 这里不排章节也不写正文，那在 book_worker.go。

package learn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// editableBlock 在读者可以直接改这种块的正文时为真。
func editableBlock(kind string) bool {
	return kind == BlockText || kind == BlockCallout || kind == BlockUserNote
}

// UpdateBlock 改一块的正文并返回改完的块。只有正文、提示框和读者笔记能改；提示框的种类不变。
// 改过的块标成读者改过，重写这一章时会留着。body 去掉空白后不能空。
func (s *Store) UpdateBlock(bookID, blockID, body string) (BookBlock, error) {
	if s == nil {
		return BookBlock{}, fmt.Errorf("learn store unavailable")
	}
	body = cleanBody(body)
	if body == "" {
		return BookBlock{}, fmt.Errorf("the text is empty")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	blk, err := loadBlock(s.db, bookID, blockID)
	if err == nil && !editableBlock(blk.Kind) {
		err = fmt.Errorf("a %s block cannot be edited by hand; regenerate it instead", blk.Kind)
	}
	if err == nil {
		var payload any = bookText{Body: body}
		if blk.Kind == BlockCallout {
			old, _ := storedPayload(BlockCallout, blk.Payload).(bookCallout)
			payload = bookCallout{Variant: normalizeVariant(old.Variant, "key_idea"), Body: body}
		}
		_, err = s.db.Exec(`UPDATE book_blocks SET payload_json = ?, status = ?, error = '', edited = 1, updated_at = ? WHERE id = ?`,
			encodePayload(payload), blockReady, now, blockID)
		if err == nil {
			err = touchBook(s.db, bookID, now)
		}
	}
	if err == nil {
		blk, err = loadBlock(s.db, bookID, blockID)
	}
	s.mu.Unlock()
	if err != nil {
		return BookBlock{}, err
	}
	s.notify("book", bookID, "block_ready")
	return blk.public(), nil
}

// InsertBlock 在一章里插一块读者自己写的内容，返回新块。kind 是 user_note 或 text，其余当成 user_note。
// afterBlockID 非空时插在那一块后面，否则放在这一章最前面。body 去掉空白后不能空。
func (s *Store) InsertBlock(bookID, chapterID, afterBlockID, kind, body string) (BookBlock, error) {
	if s == nil {
		return BookBlock{}, fmt.Errorf("learn store unavailable")
	}
	body = cleanBody(body)
	if body == "" {
		return BookBlock{}, fmt.Errorf("the text is empty")
	}
	if kind != BlockText {
		kind = BlockUserNote
	}
	now := formatTime(s.clock())
	id := newKey("blk")
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		if _, err := loadChapter(tx, bookID, chapterID); err != nil {
			return err
		}
		blocks, err := loadBlocks(tx, chapterID)
		if err != nil {
			return err
		}
		if len(blocks) >= maxChapterBlocks {
			return fmt.Errorf("this chapter already holds %d blocks", maxChapterBlocks)
		}
		at := 0
		if afterBlockID != "" {
			at = -1
			for i, blk := range blocks {
				if blk.ID == afterBlockID {
					at = i + 1
				}
			}
			if at < 0 {
				return fmt.Errorf("block not found")
			}
		}
		// 先给后面的块腾位置，再把所有块按顺序重新编号，旧库里重复的序号也一并理顺。
		for i, blk := range blocks {
			ord := i
			if i >= at {
				ord = i + 1
			}
			if _, err := tx.Exec(`UPDATE book_blocks SET ord = ? WHERE id = ?`, ord, blk.ID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`INSERT INTO book_blocks (id, book_id, chapter_id, ord, page_no, kind, body, status, title, params_json, payload_json, edited, updated_at)
			VALUES (?, ?, ?, ?, 0, ?, '', ?, '', '{}', ?, 1, ?)`,
			id, bookID, chapterID, at, kind, blockReady, encodePayload(bookText{Body: body}), now); err != nil {
			return err
		}
		return touchBook(tx, bookID, now)
	})
	var blk blockRow
	if err == nil {
		blk, err = loadBlock(s.db, bookID, id)
	}
	s.mu.Unlock()
	if err != nil {
		return BookBlock{}, err
	}
	s.notify("book", bookID, "block_ready")
	return blk.public(), nil
}

// DeleteBlock 删掉一块和它上面的作答记录。blockID 不在这本书里时返回 block not found。
func (s *Store) DeleteBlock(bookID, blockID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		if _, err := loadBlock(tx, bookID, blockID); err != nil {
			return err
		}
		if err := execArgs(tx, blockID, `DELETE FROM book_attempts WHERE block_id = ?`, `DELETE FROM book_blocks WHERE id = ?`); err != nil {
			return err
		}
		return touchBook(tx, bookID, now)
	})
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("book", bookID, "block_ready")
	return nil
}

// RegenerateBlock 让模型把一块重写一遍。读者的笔记不能重写。这一块原来的内容和读者的改动作废，作答记录清掉。
// sessionID 决定用哪段对话的模型。
func (s *Store) RegenerateBlock(bookID, blockID, sessionID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		blk, err := loadBlock(tx, bookID, blockID)
		if err != nil {
			return err
		}
		if blk.Kind == BlockUserNote {
			return fmt.Errorf("your own note is not rewritten by the model")
		}
		if _, err := tx.Exec(`UPDATE book_blocks SET status = ?, error = '', edited = 0, updated_at = ? WHERE id = ?`, blockPending, now, blockID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM book_attempts WHERE block_id = ?`, blockID); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE book_chapters SET status = ?, error = '', updated_at = ? WHERE id = ?`, chapterGenerating, now, blk.ChapterID); err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE books SET status = ?, pause_reason = '', updated_at = ? WHERE id = ?`, BookStatusCompiling, now, bookID)
		return err
	})
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.startBook(bookID, sessionID)
	s.notify("book", bookID, "updated")
	return nil
}

// BookQuizRequest 是作答一道随堂题的入参。Answer 是读者的作答：选择题给选项字母或选项正文。SessionID 决定简答题用哪段对话的模型判。
type BookQuizRequest struct {
	BookID     string
	BlockID    string
	QuestionID string
	Answer     string
	SessionID  string
}

// BookQuizResult 是一道随堂题的批改结果。Correct 为 nil 表示没法判定，读者对照答案自己看。
// Answer 是标准答案，选择题是选项字母。Feedback 是模型判简答题时的一句说明。
type BookQuizResult struct {
	Correct     *bool  `json:"correct"`
	Answer      string `json:"answer"`
	Explanation string `json:"explanation"`
	Feedback    string `json:"feedback,omitempty"`
}

// AnswerBookQuiz 批改一道随堂题，并给出答案和解析。选择题在本地批改；简答题和答案一字不差算对，否则接了模型就请它按意思判，
// 判不出来就不给结论。每次作答都记一笔。答错的题进题库当错题，同一道题只进一次。作答读不出来时不算答错。
func (s *Store) AnswerBookQuiz(ctx context.Context, req BookQuizRequest) (BookQuizResult, error) {
	if s == nil {
		return BookQuizResult{}, fmt.Errorf("learn store unavailable")
	}
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return BookQuizResult{}, fmt.Errorf("the answer is empty")
	}
	s.mu.Lock()
	b, err := loadBook(s.db, req.BookID)
	var blk blockRow
	if err == nil {
		blk, err = loadBlock(s.db, req.BookID, req.BlockID)
	}
	s.mu.Unlock()
	if err != nil {
		return BookQuizResult{}, err
	}
	if blk.Kind != BlockQuiz {
		return BookQuizResult{}, fmt.Errorf("this block is not a quiz")
	}
	var q *bookQuizQuestion
	quiz := storedQuiz(blk.Payload)
	for i := range quiz.Questions {
		if quiz.Questions[i].ID == req.QuestionID {
			q = &quiz.Questions[i]
		}
	}
	if q == nil {
		return BookQuizResult{}, fmt.Errorf("question not found")
	}
	out := BookQuizResult{Answer: q.Answer, Explanation: q.Explanation}
	decide := func(correct bool) { out.Correct = &correct }
	if q.Type == QuestionChoice {
		if picked, ok := resolveChoice(q.Options, answer); ok {
			decide(picked == q.Answer)
			answer = picked
		}
	} else if normalizeAnswer(answer) == normalizeAnswer(q.Answer) {
		decide(true)
	} else if s.Complete != nil {
		v, err := s.judgeAnswer(ctx, req.SessionID, storedQuestion{ID: q.ID, Type: QuestionShort, Prompt: q.Question, Expected: q.Answer}, "", answer)
		if err == nil && v.Final {
			out.Feedback = v.Feedback
			if v.Result == ResultCorrect {
				decide(true)
			} else if v.Result == ResultIncorrect {
				decide(false)
			}
		}
	}
	correct := -1
	if out.Correct != nil {
		correct = boolInt(*out.Correct)
	}
	now := s.clock()
	s.mu.Lock()
	err = s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO book_attempts (id, book_id, block_id, question_id, answer, correct, at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			newKey("bat"), req.BookID, req.BlockID, q.ID, clipRunes(answer, 4000), correct, formatTime(now)); err != nil {
			return err
		}
		if correct != 0 {
			return nil
		}
		_, _, err := bankInsert(tx, bankEntry{
			Stem: q.Question, Type: q.Type, Options: q.Options, Answer: q.Answer, Explanation: q.Explanation,
			Tags: cleanTags([]string{b.Title}), Source: "book", SourceID: req.BookID, OriginKey: "book:" + req.BlockID + ":" + q.ID,
			LibraryID: b.LibraryID, Mistake: true, LastAnswer: answer,
		}, now)
		return err
	})
	s.mu.Unlock()
	if err != nil {
		return BookQuizResult{}, err
	}
	if correct == 0 {
		s.notify("practice", "", "added")
	}
	return out, nil
}

// ExportBook 把一本课本导成 Markdown，返回建议的文件名和正文。只导写好的块；随堂题只导题目，不带答案。
func (s *Store) ExportBook(id string) (filename, markdown string, err error) {
	if s == nil {
		return "", "", fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadBook(s.db, id)
	if err != nil {
		return "", "", err
	}
	chapters, err := loadChapters(s.db, id)
	if err != nil {
		return "", "", err
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n", b.Title)
	if b.Proposal.Description != "" {
		fmt.Fprintf(&out, "\n%s\n", b.Proposal.Description)
	}
	for i, ch := range chapters {
		fmt.Fprintf(&out, "\n## %d. %s\n", i+1, ch.Title)
		if ch.Summary != "" {
			fmt.Fprintf(&out, "\n*%s*\n", ch.Summary)
		}
		blocks, err := loadBlocks(s.db, ch.ID)
		if err != nil {
			return "", "", err
		}
		for _, blk := range blocks {
			if blk.Status != blockReady {
				continue
			}
			if text := blockMarkdown(blk); text != "" {
				out.WriteString("\n" + text + "\n")
			}
		}
	}
	return sanitizeName(b.Title) + ".md", out.String(), nil
}

// blockMarkdown 把一块写成 Markdown。认不出的种类和空的块返回空串。
func blockMarkdown(blk blockRow) string {
	var b strings.Builder
	switch p := storedPayload(blk.Kind, blk.Payload).(type) {
	case bookText:
		if blk.Kind == BlockUserNote {
			for _, line := range strings.Split(p.Body, "\n") {
				b.WriteString("> " + line + "\n")
			}
			return strings.TrimRight(b.String(), "\n")
		}
		return p.Body
	case bookSection:
		if blk.Title != "" {
			fmt.Fprintf(&b, "### %s\n\n", blk.Title)
		}
		if p.Intro != "" {
			b.WriteString(p.Intro + "\n")
		}
		for _, sub := range p.Subsections {
			fmt.Fprintf(&b, "\n#### %s\n\n%s\n", sub.Heading, sub.Body)
		}
		if p.KeyTakeaway != "" {
			fmt.Fprintf(&b, "\n**%s**\n", p.KeyTakeaway)
		}
	case bookCallout:
		for _, line := range strings.Split(p.Body, "\n") {
			b.WriteString("> " + line + "\n")
		}
	case bookCode:
		fmt.Fprintf(&b, "```%s\n%s\n```\n", p.Language, p.Code)
		if p.Explanation != "" {
			b.WriteString("\n" + p.Explanation + "\n")
		}
	case bookTimeline:
		for _, e := range p.Events {
			fmt.Fprintf(&b, "- **%s** %s: %s\n", e.Date, e.Title, e.Description)
		}
	case bookFlashCards:
		for _, c := range p.Cards {
			fmt.Fprintf(&b, "- **%s** — %s\n", c.Front, c.Back)
		}
	case bookQuiz:
		for i, q := range p.Questions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, q.Question)
			for _, o := range q.Options {
				fmt.Fprintf(&b, "   - %s. %s\n", o.Label, o.Body)
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
