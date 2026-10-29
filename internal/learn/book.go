// 课本把文库摘录编译成块。编译过程不调用聊天模型。连续两份材料没有摘录就暂停，已经写好的块留着。

package learn

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Book 是一本课本。进度不在这里，按读者另存。
type Book struct {
	ID        string      `json:"id"`
	LibraryID string      `json:"library_id,omitempty"`
	Title     string      `json:"title"`
	Status    string      `json:"status"`
	Depth     string      `json:"depth"`
	Proposal  string      `json:"proposal,omitempty"`
	Spine     []SpineItem `json:"spine,omitempty"`
	Blocks    []Block     `json:"blocks,omitempty"`
	Page      int         `json:"page,omitempty"`
}

// SpineItem 是书脊上的一节。Query 用来在文库里找摘录。
type SpineItem struct {
	Title string `json:"title"`
	Query string `json:"query"`
	Kind  string `json:"kind,omitempty"`
}

// Block 是编译出来的一块。Kind 是 section、text、callout、code、quiz、flash_cards、figure 或 timeline。
type Block struct {
	ID   string `json:"id"`
	Ord  int    `json:"ord"`
	Kind string `json:"kind"`
	Body string `json:"body"`
	Page int    `json:"page"`
}

// CreateBook 新建一本草稿。title 去掉空白后不能空。libraryID 可空。
func (s *Store) CreateBook(title, libraryID string) (Book, error) {
	if s == nil {
		return Book{}, fmt.Errorf("learn store unavailable")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Book{}, fmt.Errorf("title required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	b := Book{ID: newKey("book"), LibraryID: strings.TrimSpace(libraryID), Title: title, Status: "draft", Depth: "standard"}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`INSERT INTO books (id, library_id, title, status, depth, created_at, updated_at) VALUES (?, ?, ?, 'draft', 'standard', ?, ?)`,
		b.ID, b.LibraryID, b.Title, now, now)
	return b, err
}

// RecentBooks 返回最近几本课本，不含块正文。limit 小于 1 时当成 4。
// 页码是 local 这位读者的进度。
func (s *Store) RecentBooks(limit int) []Book {
	if s == nil {
		return nil
	}
	if limit < 1 {
		limit = 4
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT b.id, b.library_id, b.title, b.status,
		COALESCE((SELECT page_no FROM book_progress p WHERE p.book_id = b.id AND p.reader = 'local'), 1)
		FROM books b WHERE b.status != 'error' ORDER BY b.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.LibraryID, &b.Title, &b.Status, &b.Page); err != nil {
			continue
		}
		out = append(out, b)
	}
	return out
}

// Books 列出课本，不含块正文。
func (s *Store) Books() []Book {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, library_id, title, status, depth, proposal FROM books ORDER BY updated_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.LibraryID, &b.Title, &b.Status, &b.Depth, &b.Proposal); err != nil {
			continue
		}
		out = append(out, b)
	}
	return out
}

// SetProposal 写下选题说明，并把状态改成 draft。bookID 必须存在。proposal 截到 2000 个符文。
func (s *Store) SetProposal(bookID, proposal string) error {
	return s.touchBook(bookID, `UPDATE books SET proposal = ?, status = 'draft', updated_at = ? WHERE id = ?`,
		clipRunes(strings.TrimSpace(proposal), 2000))
}

// SetSpine 写下书脊。spine 编码成 JSON。空书脊也可以。bookID 必须存在。成功后状态是 spine_ready。
func (s *Store) SetSpine(bookID string, spine []SpineItem) error {
	raw, err := json.Marshal(spine)
	if err != nil {
		return err
	}
	return s.touchBook(bookID, `UPDATE books SET spine_json = ?, status = 'spine_ready', updated_at = ? WHERE id = ?`, string(raw))
}

func (s *Store) touchBook(bookID, q string, extra ...any) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	args := append(extra, time.Now().UTC().Format(time.RFC3339Nano), bookID)
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("book not found")
	}
	return nil
}

// CompileBook 按书脊把摘录写成块。library 上没有发布索引时，摘录算空。连续两节为空就暂停。
// bookID 必须存在。reader 不参与编译。
func (s *Store) CompileBook(bookID string) (Book, error) {
	if s == nil {
		return Book{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	var libraryID, spineRaw string
	err := s.db.QueryRow(`SELECT library_id, spine_json FROM books WHERE id = ?`, bookID).Scan(&libraryID, &spineRaw)
	if err != nil {
		s.mu.Unlock()
		return Book{}, fmt.Errorf("book not found")
	}
	_, _ = s.db.Exec(`UPDATE books SET status = 'compiling', updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), bookID)
	_, _ = s.db.Exec(`DELETE FROM book_blocks WHERE book_id = ?`, bookID)
	s.mu.Unlock()

	var spine []SpineItem
	_ = json.Unmarshal([]byte(spineRaw), &spine)
	emptyRun := 0
	paused := false
	ord := 0
	page := 1
	for _, item := range spine {
		if paused {
			break
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "未命名"
		}
		hits, _, _ := s.SearchLibrary(libraryID, item.Query)
		body := excerpts(hits)
		if strings.TrimSpace(body) == "" {
			emptyRun++
			if emptyRun >= 2 {
				paused = true
			}
			continue
		}
		emptyRun = 0
		if err := s.insertBlock(bookID, ord, "section", title, page); err != nil {
			return Book{}, err
		}
		ord++
		kind := blockKind(item.Kind, body)
		if err := s.insertBlock(bookID, ord, kind, body, page); err != nil {
			return Book{}, err
		}
		ord++
		page++
	}
	status := "ready"
	if paused {
		status = "paused"
	}
	if ord == 0 && !paused {
		status = "error"
	}
	s.mu.Lock()
	_, _ = s.db.Exec(`UPDATE books SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC().Format(time.RFC3339Nano), bookID)
	s.mu.Unlock()
	return s.Book(bookID, "local")
}

func excerpts(hits []Hit) string {
	var b strings.Builder
	for i, h := range hits {
		if i >= 3 {
			break
		}
		text := strings.TrimSpace(h.Text)
		if text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(text)
	}
	return clipRunes(b.String(), 4000)
}

func blockKind(want, body string) string {
	switch strings.TrimSpace(want) {
	case "callout", "quiz", "flash_cards", "figure", "code", "timeline", "text":
		return want
	}
	if strings.Contains(body, "```") {
		return "code"
	}
	return "text"
}

func (s *Store) insertBlock(bookID string, ord int, kind, body string, page int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`INSERT INTO book_blocks (id, book_id, ord, kind, body, page_no) VALUES (?, ?, ?, ?, ?, ?)`,
		newKey("blk"), bookID, ord, kind, body, page)
	return err
}

// Book 读取课本和指定读者的页码。reader 空时用 local。
// bookID 必须存在。
func (s *Store) Book(bookID, reader string) (Book, error) {
	if s == nil {
		return Book{}, fmt.Errorf("learn store unavailable")
	}
	if strings.TrimSpace(reader) == "" {
		reader = "local"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var b Book
	var spineRaw string
	err := s.db.QueryRow(`SELECT id, library_id, title, status, depth, proposal, spine_json FROM books WHERE id = ?`, bookID).
		Scan(&b.ID, &b.LibraryID, &b.Title, &b.Status, &b.Depth, &b.Proposal, &spineRaw)
	if err != nil {
		return Book{}, fmt.Errorf("book not found")
	}
	_ = json.Unmarshal([]byte(spineRaw), &b.Spine)
	rows, err := s.db.Query(`SELECT id, ord, kind, body, page_no FROM book_blocks WHERE book_id = ? ORDER BY ord`, bookID)
	if err != nil {
		return Book{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var blk Block
		if err := rows.Scan(&blk.ID, &blk.Ord, &blk.Kind, &blk.Body, &blk.Page); err != nil {
			continue
		}
		b.Blocks = append(b.Blocks, blk)
	}
	_ = s.db.QueryRow(`SELECT page_no FROM book_progress WHERE book_id = ? AND reader = ?`, bookID, reader).Scan(&b.Page)
	if b.Page == 0 {
		b.Page = 1
	}
	return b, nil
}

// SetBookPage 记下这位读者读到的页。reader 空时用 local。page 小于 1 时写成 1。
// bookID 必须存在。
func (s *Store) SetBookPage(bookID, reader string, page int) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	if strings.TrimSpace(reader) == "" {
		reader = "local"
	}
	if page < 1 {
		page = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM books WHERE id = ?`, bookID).Scan(&n)
	if n == 0 {
		return fmt.Errorf("book not found")
	}
	_, err := s.db.Exec(`INSERT INTO book_progress (book_id, reader, page_no) VALUES (?, ?, ?)
		ON CONFLICT(book_id, reader) DO UPDATE SET page_no = excluded.page_no`, bookID, reader, page)
	return err
}
