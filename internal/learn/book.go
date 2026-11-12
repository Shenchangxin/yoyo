// 课本的数据形状，以及不调用模型的读写：列表、摘要、打开一章并记下进度、改名、删除，和打开库时的收拾。
// 写提案在 book_create.go，起草书脊在 book_spine.go，逐章写作在 book_worker.go，块的校验在 book_payload.go，
// 读者对单个块的操作、随堂题和导出在 book_blocks.go。
// 随堂题的答案和解析只留在库里，这里给出去的块不带它们。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// BookStatusDraft 表示提案在等读者确认。书脊正在后台起草时也是这个状态，这时 PauseReason 是 BookPauseBuildingSpine。
	BookStatusDraft = "draft"
	// BookStatusSpineReady 表示书脊起草好了，在等读者确认。
	BookStatusSpineReady = "spine_ready"
	// BookStatusCompiling 表示正在逐章写。
	BookStatusCompiling = "compiling"
	// BookStatusPaused 表示写作停着，原因在 PauseReason 里。
	BookStatusPaused = "paused"
	// BookStatusReady 表示没有还没写的章节了。
	BookStatusReady = "ready"
	// BookStatusError 表示一章都没写出来。
	BookStatusError = "error"

	// BookPauseBuildingSpine 是草稿状态下书脊正在起草时的 PauseReason。起草完就清掉。
	BookPauseBuildingSpine = "building_spine"
	// BookPauseUser 是读者自己按了暂停时的 PauseReason。
	BookPauseUser = "user"
	// BookPauseInterrupted 是上次退出时还在写的课本在打开库之后的 PauseReason。
	BookPauseInterrupted = "interrupted"

	// BlockText 是一段正文。
	BlockText = "text"
	// BlockSection 是分小节的主体讲解。
	BlockSection = "section"
	// BlockCallout 是一个提示框。
	BlockCallout = "callout"
	// BlockCode 是一段代码和它的说明。
	BlockCode = "code"
	// BlockTimeline 是按时间排的事件。
	BlockTimeline = "timeline"
	// BlockFlashCards 是一组记忆卡片。
	BlockFlashCards = "flash_cards"
	// BlockQuiz 是随堂题。
	BlockQuiz = "quiz"
	// BlockUserNote 是读者自己写的笔记，模型不写也不改它。
	BlockUserNote = "user_note"

	chapterPending    = "pending"
	chapterPlanning   = "planning"
	chapterGenerating = "generating"
	chapterReady      = "ready"
	chapterPartial    = "partial"
	chapterError      = "error"

	blockPending    = "pending"
	blockGenerating = "generating"
	blockReady      = "ready"
	blockError      = "error"

	// bookReader 是本机唯一的读者。进度按它存。
	bookReader = "local"

	maxBookTitleRunes = 120
	maxSpineChapters  = 12
	maxChapterBlocks  = 40
)

// BookSummary 是列表和学习计划里的一本课本。Chapters 是章节总数，ReadyChapters 是已经写好的章节数。
// ChapterID 是读者上次读到的那一章，Percent 是读过的章节占比，0 到 1。
type BookSummary struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	LibraryID     string  `json:"library_id"`
	Status        string  `json:"status"`
	Depth         string  `json:"depth"`
	Language      string  `json:"language"`
	Chapters      int     `json:"chapters"`
	ReadyChapters int     `json:"ready_chapters"`
	PauseReason   string  `json:"pause_reason"`
	ChapterID     string  `json:"chapter_id"`
	Percent       float64 `json:"percent"`
	UpdatedAt     string  `json:"updated_at"`
}

// BookProposal 是一本课本的提案：写什么、给谁写、大约几章。读者确认之前可以改。
type BookProposal struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	Scope             string `json:"scope"`
	TargetLevel       string `json:"target_level"`
	EstimatedChapters int    `json:"estimated_chapters"`
	Rationale         string `json:"rationale"`
}

// BookChapter 是书脊上的一章，不带块。Order 从 0 起。Objectives 没有内容时是空数组。
type BookChapter struct {
	ID          string   `json:"id"`
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Objectives  []string `json:"objectives"`
	ContentType string   `json:"content_type"`
	Status      string   `json:"status"`
	Error       string   `json:"error"`
}

// BookChapterView 是打开的一章：章节本身加上它的块。Blocks 没有内容时是空数组。
type BookChapterView struct {
	BookChapter
	Blocks []BookBlock `json:"blocks"`
}

// BookBlock 是章节里的一块。Payload 的形状由 Kind 决定；随堂题的 Payload 里没有答案和解析。
// 还没写出来的块带着这种块的空载荷。Edited 为真表示读者改过正文，重写这一章时会留着它。
type BookBlock struct {
	ID        string `json:"id"`
	ChapterID string `json:"chapter_id"`
	Order     int    `json:"order"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Title     string `json:"title"`
	Payload   any    `json:"payload"`
	Error     string `json:"error"`
	Edited    bool   `json:"edited"`
}

// BookProgress 是读者在一本课本里的进度。ChapterID 是上次打开的那一章，Visited 是打开过的章节。
// Percent 是打开过而且已经写好的章节数除以章节总数。
type BookProgress struct {
	ChapterID string   `json:"chapter_id"`
	Visited   []string `json:"visited"`
	Percent   float64  `json:"percent"`
	UpdatedAt string   `json:"updated_at"`
}

// BookView 是一本课本的全貌：摘要、提案、不带块的章节和进度。
type BookView struct {
	Book     BookSummary   `json:"book"`
	Proposal BookProposal  `json:"proposal"`
	Chapters []BookChapter `json:"chapters"`
	Progress BookProgress  `json:"progress"`
}

// bookRow 是库里的一本课本。Proposal 已经从 proposal_json 解出来；旧库只有一段说明文字时放在 Scope 里。
type bookRow struct {
	ID, LibraryID, Title, Status, Depth, Intent, Language, Description, PauseReason, UpdatedAt string
	Proposal                                                                                   BookProposal
}

// chapterRow 是库里的一章。
type chapterRow struct {
	ID, BookID, Title, Summary, ContentType, Status, Error string
	Ord                                                    int
	Objectives                                             []string
}

// blockRow 是库里的一块。Params 和 Payload 是原样的 JSON 文本。
type blockRow struct {
	ID, BookID, ChapterID, Kind, Status, Title, Params, Payload, Error, UpdatedAt string
	Ord                                                                           int
	Edited                                                                        bool
}

const chapterCols = `id, book_id, ord, title, summary, objectives_json, content_type, status, error`

const blockCols = `id, book_id, chapter_id, ord, kind, status, title, params_json, payload_json, error, edited, updated_at`

// loadBook 用 q 读出一本课本。id 不存在时返回 book not found。
func loadBook(q querier, id string) (bookRow, error) {
	var b bookRow
	var proposalJSON, legacy string
	err := q.QueryRow(`SELECT id, library_id, title, status, depth, intent, language, description, proposal_json, proposal, pause_reason, updated_at
		FROM books WHERE id = ?`, id).Scan(&b.ID, &b.LibraryID, &b.Title, &b.Status, &b.Depth, &b.Intent, &b.Language, &b.Description,
		&proposalJSON, &legacy, &b.PauseReason, &b.UpdatedAt)
	if err != nil {
		return bookRow{}, fmt.Errorf("book not found")
	}
	if strings.TrimSpace(proposalJSON) == "" || json.Unmarshal([]byte(proposalJSON), &b.Proposal) != nil {
		b.Proposal = BookProposal{Scope: legacy}
	}
	if b.Proposal.Title == "" {
		b.Proposal.Title = b.Title
	}
	if b.Proposal.Description == "" {
		b.Proposal.Description = b.Description
	}
	b.Proposal.EstimatedChapters = clampChapters(b.Proposal.EstimatedChapters)
	b.Depth = normalizeDepth(b.Depth)
	return b, nil
}

// scanChapter 从一行读出章节。列顺序和 chapterCols 相同。
func scanChapter(row rower) (chapterRow, error) {
	var c chapterRow
	var objectives string
	if err := row.Scan(&c.ID, &c.BookID, &c.Ord, &c.Title, &c.Summary, &objectives, &c.ContentType, &c.Status, &c.Error); err != nil {
		return chapterRow{}, err
	}
	_ = json.Unmarshal([]byte(objectives), &c.Objectives)
	if c.Objectives == nil {
		c.Objectives = []string{}
	}
	return c, nil
}

// loadChapters 用 q 按顺序读出 bookID 的全部章节。
func loadChapters(q querier, bookID string) ([]chapterRow, error) {
	rows, err := q.Query(`SELECT `+chapterCols+` FROM book_chapters WHERE book_id = ? ORDER BY ord, id`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []chapterRow{}
	for rows.Next() {
		c, err := scanChapter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// loadChapter 用 q 读出 bookID 里的一章。chapterID 不在这本书里时返回 chapter not found。
func loadChapter(q querier, bookID, chapterID string) (chapterRow, error) {
	c, err := scanChapter(q.QueryRow(`SELECT `+chapterCols+` FROM book_chapters WHERE id = ? AND book_id = ?`, chapterID, bookID))
	if err != nil {
		return chapterRow{}, fmt.Errorf("chapter not found")
	}
	return c, nil
}

// public 把章节收成给界面的形状。
func (c chapterRow) public() BookChapter {
	return BookChapter{
		ID: c.ID, Order: c.Ord, Title: c.Title, Summary: c.Summary, Objectives: c.Objectives,
		ContentType: normalizeContentType(c.ContentType), Status: c.Status, Error: c.Error,
	}
}

// unfinished 在这一章还轮得到写作任务时为真。
func (c chapterRow) unfinished() bool {
	return c.Status == chapterPending || c.Status == chapterPlanning || c.Status == chapterGenerating
}

// scanBlock 从一行读出块。列顺序和 blockCols 相同。
func scanBlock(row rower) (blockRow, error) {
	var b blockRow
	var edited int
	if err := row.Scan(&b.ID, &b.BookID, &b.ChapterID, &b.Ord, &b.Kind, &b.Status, &b.Title, &b.Params, &b.Payload, &b.Error, &edited, &b.UpdatedAt); err != nil {
		return blockRow{}, err
	}
	b.Edited = edited != 0
	return b, nil
}

// loadBlocks 用 q 按顺序读出 chapterID 的全部块。
func loadBlocks(q querier, chapterID string) ([]blockRow, error) {
	rows, err := q.Query(`SELECT `+blockCols+` FROM book_blocks WHERE chapter_id = ? ORDER BY ord, id`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []blockRow{}
	for rows.Next() {
		b, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// loadBlock 用 q 读出 bookID 里的一块。blockID 不在这本书里时返回 block not found。
func loadBlock(q querier, bookID, blockID string) (blockRow, error) {
	b, err := scanBlock(q.QueryRow(`SELECT `+blockCols+` FROM book_blocks WHERE id = ? AND book_id = ?`, blockID, bookID))
	if err != nil {
		return blockRow{}, fmt.Errorf("block not found")
	}
	return b, nil
}

// public 把块收成给界面的形状。随堂题的答案和解析在这里去掉。
func (b blockRow) public() BookBlock {
	return BookBlock{
		ID: b.ID, ChapterID: b.ChapterID, Order: b.Ord, Kind: b.Kind, Status: b.Status, Title: b.Title,
		Payload: publicPayload(b.Kind, b.Payload), Error: b.Error, Edited: b.Edited,
	}
}

// kept 在重写这一章时要留下这一块时为真：读者的笔记，或读者改过的块。
func (b blockRow) kept() bool {
	return b.Kind == BlockUserNote || b.Edited
}

// loadBookProgress 用 q 读出读者在 bookID 里的进度。chapters 是这本书现有的章节，已经不存在的章节不算。
// 旧库只记了读到哪一章，那一章也算读过。
func loadBookProgress(q querier, bookID string, chapters []chapterRow) BookProgress {
	out := BookProgress{Visited: []string{}}
	var visitedJSON string
	if err := q.QueryRow(`SELECT chapter_id, visited_json, updated_at FROM book_progress WHERE book_id = ? AND reader = ?`, bookID, bookReader).
		Scan(&out.ChapterID, &visitedJSON, &out.UpdatedAt); err != nil {
		return out
	}
	status := map[string]string{}
	for _, c := range chapters {
		status[c.ID] = c.Status
	}
	if _, ok := status[out.ChapterID]; !ok {
		out.ChapterID = ""
	}
	var visited []string
	_ = json.Unmarshal([]byte(visitedJSON), &visited)
	seen := map[string]bool{}
	ready := 0
	for _, id := range append(visited, out.ChapterID) {
		if _, ok := status[id]; !ok || seen[id] {
			continue
		}
		seen[id] = true
		out.Visited = append(out.Visited, id)
		if status[id] == chapterReady {
			ready++
		}
	}
	if len(chapters) > 0 {
		out.Percent = float64(ready) / float64(len(chapters))
	}
	return out
}

// summarizeBook 用 q 把一本课本收成摘要，章节数和进度现算。
func summarizeBook(q querier, b bookRow) (BookSummary, error) {
	chapters, err := loadChapters(q, b.ID)
	if err != nil {
		return BookSummary{}, err
	}
	return bookSummaryOf(b, chapters, loadBookProgress(q, b.ID, chapters)), nil
}

// bookSummaryOf 用已经读出的章节和进度拼出摘要。
func bookSummaryOf(b bookRow, chapters []chapterRow, progress BookProgress) BookSummary {
	out := BookSummary{
		ID: b.ID, Title: b.Title, Description: b.Description, LibraryID: b.LibraryID, Status: b.Status, Depth: b.Depth,
		Language: b.Language, Chapters: len(chapters), PauseReason: b.PauseReason, ChapterID: progress.ChapterID,
		Percent: progress.Percent, UpdatedAt: b.UpdatedAt,
	}
	for _, c := range chapters {
		if c.Status == chapterReady {
			out.ReadyChapters++
		}
	}
	return out
}

// bookSummaries 用 q 读出 ids 对应的摘要，顺序不变。读不出来的跳过。
func bookSummaries(q querier, ids []string) []BookSummary {
	out := make([]BookSummary, 0, len(ids))
	for _, id := range ids {
		b, err := loadBook(q, id)
		if err != nil {
			continue
		}
		sum, err := summarizeBook(q, b)
		if err != nil {
			continue
		}
		out = append(out, sum)
	}
	return out
}

// bookIDs 跑一条只选 id 的查询并把结果读完。args 是查询参数。
func bookIDs(q querier, query string, args ...any) []string {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// Books 按最近更新列出全部课本的摘要，出错的也在内。
func (s *Store) Books() []BookSummary {
	if s == nil {
		return []BookSummary{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return bookSummaries(s.db, bookIDs(s.db, `SELECT id FROM books ORDER BY updated_at DESC, id`))
}

// RecentBooks 返回最近写过或读过的几本课本的摘要。limit 小于 1 时当成 4。出错的书不列出来。
func (s *Store) RecentBooks(limit int) []BookSummary {
	if s == nil {
		return []BookSummary{}
	}
	if limit < 1 {
		limit = 4
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := bookIDs(s.db, `SELECT b.id FROM books b WHERE b.status != ?
		ORDER BY MAX(b.updated_at, COALESCE((SELECT p.updated_at FROM book_progress p WHERE p.book_id = b.id AND p.reader = ?), '')) DESC, b.id
		LIMIT ?`, BookStatusError, bookReader, limit)
	return bookSummaries(s.db, ids)
}

// Book 读出一本课本的摘要、提案、章节和进度。章节不带块。id 必须存在。
func (s *Store) Book(id string) (BookView, error) {
	if s == nil {
		return BookView{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return bookView(s.db, id)
}

// bookView 用 q 拼出一本课本的全貌。调用方要已经持有 s.mu。
func bookView(q querier, id string) (BookView, error) {
	b, err := loadBook(q, id)
	if err != nil {
		return BookView{}, err
	}
	chapters, err := loadChapters(q, id)
	if err != nil {
		return BookView{}, err
	}
	progress := loadBookProgress(q, id, chapters)
	view := BookView{Book: bookSummaryOf(b, chapters, progress), Proposal: b.Proposal, Chapters: make([]BookChapter, 0, len(chapters)), Progress: progress}
	for _, c := range chapters {
		view.Chapters = append(view.Chapters, c.public())
	}
	return view, nil
}

// bookSummary 读出一本课本的摘要。id 必须存在。
func (s *Store) bookSummary(id string) (BookSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := loadBook(s.db, id)
	if err != nil {
		return BookSummary{}, err
	}
	return summarizeBook(s.db, b)
}

// OpenBookChapter 读出一章和它的块，并记下读者打开过它。chapterID 为空时取读者上次读到的那一章，没有就取第一章。
// 记进度不发变化通知：界面收到课本的通知会重新读这一章，发了就成了回路。
func (s *Store) OpenBookChapter(bookID, chapterID string) (BookChapterView, error) {
	if s == nil {
		return BookChapterView{}, fmt.Errorf("learn store unavailable")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	defer s.mu.Unlock()
	var view BookChapterView
	err := s.inTx(func(tx *sql.Tx) error {
		chapters, err := loadChapters(tx, bookID)
		if err != nil {
			return err
		}
		if len(chapters) == 0 {
			if _, err := loadBook(tx, bookID); err != nil {
				return err
			}
			return fmt.Errorf("chapter not found")
		}
		progress := loadBookProgress(tx, bookID, chapters)
		target := pickChapter(chapters, chapterID, progress.ChapterID)
		if target == nil {
			return fmt.Errorf("chapter not found")
		}
		blocks, err := loadBlocks(tx, target.ID)
		if err != nil {
			return err
		}
		view = BookChapterView{BookChapter: target.public(), Blocks: make([]BookBlock, 0, len(blocks))}
		for _, blk := range blocks {
			view.Blocks = append(view.Blocks, blk.public())
		}
		visited := progress.Visited
		seen := false
		for _, id := range visited {
			seen = seen || id == target.ID
		}
		if !seen {
			visited = append(visited, target.ID)
		}
		raw, _ := json.Marshal(visited)
		_, err = tx.Exec(`INSERT INTO book_progress (book_id, reader, page_no, chapter_id, visited_json, updated_at) VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(book_id, reader) DO UPDATE SET page_no = excluded.page_no, chapter_id = excluded.chapter_id,
				visited_json = excluded.visited_json, updated_at = excluded.updated_at`,
			bookID, bookReader, target.Ord+1, target.ID, string(raw), now)
		return err
	})
	if err != nil {
		return BookChapterView{}, err
	}
	return view, nil
}

// BookChapter 读出一章和它的块，不记进度。导师读章节用它，读者读到哪里不会因此变。chapterID 必须在这本书里。
func (s *Store) BookChapter(bookID, chapterID string) (BookChapterView, error) {
	if s == nil {
		return BookChapterView{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, err := loadChapter(s.db, bookID, chapterID)
	if err != nil {
		return BookChapterView{}, err
	}
	blocks, err := loadBlocks(s.db, ch.ID)
	if err != nil {
		return BookChapterView{}, err
	}
	view := BookChapterView{BookChapter: ch.public(), Blocks: make([]BookBlock, 0, len(blocks))}
	for _, blk := range blocks {
		view.Blocks = append(view.Blocks, blk.public())
	}
	return view, nil
}

// pickChapter 在 chapters 里找出要打开的那一章：先认 want，再认 last，都空时取第一章。want 非空却找不到时返回 nil。
func pickChapter(chapters []chapterRow, want, last string) *chapterRow {
	pick := strings.TrimSpace(want)
	if pick == "" {
		pick = last
	}
	for i := range chapters {
		if chapters[i].ID == pick {
			return &chapters[i]
		}
	}
	if strings.TrimSpace(want) == "" && len(chapters) > 0 {
		return &chapters[0]
	}
	return nil
}

// RenameBook 改课本的名字，提案里的标题跟着改。title 去掉空白后不能空，最长 120 个符文。
func (s *Store) RenameBook(id, title string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	title = clipRunes(strings.TrimSpace(title), maxBookTitleRunes)
	if title == "" {
		return fmt.Errorf("book title required")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		b, err := loadBook(tx, id)
		if err != nil {
			return err
		}
		b.Proposal.Title = title
		raw, _ := json.Marshal(b.Proposal)
		_, err = tx.Exec(`UPDATE books SET title = ?, proposal_json = ?, updated_at = ? WHERE id = ?`, title, string(raw), now, id)
		return err
	})
	s.mu.Unlock()
	if err == nil {
		s.notify("book", id, "updated")
	}
	return err
}

// DeleteBook 在一个事务里删掉课本和它的章节、块、进度与作答记录，并让正在写它的任务停下。答错后进了题库的题留着。
func (s *Store) DeleteBook(id string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM books WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("book not found")
		}
		return execArgs(tx, id,
			`DELETE FROM book_chapters WHERE book_id = ?`,
			`DELETE FROM book_blocks WHERE book_id = ?`,
			`DELETE FROM book_progress WHERE book_id = ?`,
			`DELETE FROM book_attempts WHERE book_id = ?`,
		)
	})
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.interruptBook(id)
	s.notify("book", id, "deleted")
	return nil
}

// recoverBooks 在打开库时收拾上次没写完的课本：还在写的停成暂停，原因是 interrupted；写到一半的章节和块退回待写；
// 书脊起草到一半的草稿清掉标记，读者可以再确认一次提案。这里不启动任何后台任务。
func (s *Store) recoverBooks() {
	now := formatTime(s.clock())
	_ = s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE book_blocks SET status = ? WHERE status = ?`, blockPending, blockGenerating); err != nil {
			return err
		}
		// 排到一半的章节退回待排。已经排好、正在写块的章节留在写作中，恢复时接着写剩下的块，不重排。
		if _, err := tx.Exec(`UPDATE book_chapters SET status = ? WHERE status = ?`, chapterPending, chapterPlanning); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE books SET status = ?, pause_reason = ?, updated_at = ? WHERE status = ?`,
			BookStatusPaused, BookPauseInterrupted, now, BookStatusCompiling); err != nil {
			return err
		}
		_, err := tx.Exec(`UPDATE books SET pause_reason = '' WHERE status = ? AND pause_reason = ?`, BookStatusDraft, BookPauseBuildingSpine)
		return err
	})
}

// touchBook 用 q 把课本的更新时间写成 now。
func touchBook(q querier, bookID, now string) error {
	_, err := q.Exec(`UPDATE books SET updated_at = ? WHERE id = ?`, now, bookID)
	return err
}

// normalizeDepth 把 depth 收成 brief、standard 或 deep。认不出的写成 standard。
func normalizeDepth(depth string) string {
	switch d := strings.ToLower(strings.TrimSpace(depth)); d {
	case "brief", "deep":
		return d
	default:
		return "standard"
	}
}

// normalizeContentType 把 kind 收成五种章节类型之一。认不出的写成 concept。
func normalizeContentType(kind string) string {
	switch k := strings.ToLower(strings.TrimSpace(kind)); k {
	case "concept", "theory", "practice", "history", "overview":
		return k
	default:
		return "concept"
	}
}

// clampChapters 把提案里估计的章节数收进 2 到 8。小于等于 0 时取 4。
func clampChapters(n int) int {
	switch {
	case n <= 0:
		return 4
	case n < 2:
		return 2
	case n > 8:
		return 8
	default:
		return n
	}
}

// cleanObjectives 去掉空的学习目标，每条截到 200 个符文，最多留 limit 条。结果不会是 nil。
func cleanObjectives(in []string, limit int) []string {
	out := []string{}
	for _, item := range in {
		if item = clipRunes(strings.TrimSpace(item), 200); item != "" && len(out) < limit {
			out = append(out, item)
		}
	}
	return out
}
