// 笔记读写实现 pages.Catalog。审阅哈希和文件库相同。人直接保存，智能体只写待审阅。

package learn

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/pages"
)

// List 按更新时间列出笔记摘要。spaceID 为空时不过滤空间。
// spaceID 先规范化。非空时只留下相等的空间。
func (s *Store) List(spaceID string) []pages.Meta {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list(spaceID)
}

func (s *Store) list(spaceID string) []pages.Meta {
	spaceID = strings.TrimSpace(spaceID)
	q := `SELECT id, space_id, parent_id, title, revision, source_session, created_at, updated_at, content FROM notes`
	var args []any
	if spaceID != "" {
		q += ` WHERE space_id = ?`
		args = append(args, pages.NormalizeSpace(spaceID))
	}
	q += ` ORDER BY updated_at DESC, title`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []pages.Meta
	for rows.Next() {
		pg, err := scanNote(rows)
		if err != nil {
			continue
		}
		out = append(out, metaOf(pg))
	}
	return out
}

// Search 在标题和正文里做子串匹配。q 为空时等同 List。
// q 去掉空白并转成小写。spaceID 的过滤和 List 相同。
func (s *Store) Search(q, spaceID string) []pages.Meta {
	if s == nil {
		return nil
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return s.List(spaceID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.list(spaceID)
	var out []pages.Meta
	for _, m := range all {
		pg, err := s.get(m.SpaceID, m.ID)
		if err != nil {
			continue
		}
		blob := strings.ToLower(pg.Title + "\n" + pg.Content)
		if strings.Contains(blob, q) {
			out = append(out, metaOf(pg))
		}
	}
	return out
}

// Get 按空间和 id 读取一篇笔记。找不到返回 page not found。
// spaceID 为空时用 home。id 先规范化。
func (s *Store) Get(spaceID, id string) (pages.Page, error) {
	if s == nil {
		return pages.Page{}, fmt.Errorf("pages store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.get(spaceID, id)
}

func (s *Store) get(spaceID, id string) (pages.Page, error) {
	id = pages.NormalizePageID(id)
	spaceID = pages.NormalizeSpace(spaceID)
	row := s.db.QueryRow(`SELECT id, space_id, parent_id, title, revision, source_session, created_at, updated_at, content
		FROM notes WHERE id = ? AND space_id = ?`, id, spaceID)
	pg, err := scanNote(row)
	if err == sql.ErrNoRows {
		return pages.Page{}, fmt.Errorf("page not found")
	}
	return pg, err
}

// Delete 删除笔记，并把直接子页挂到被删笔记的父页上。
// spaceID 和 id 传给 get。
func (s *Store) Delete(spaceID, id string) error {
	if s == nil {
		return fmt.Errorf("pages store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pg, err := s.get(spaceID, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE notes SET parent_id = ? WHERE parent_id = ?`, pg.ParentID, pg.ID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM notes WHERE id = ?`, pg.ID)
	if err == nil {
		_ = s.markNotesStale()
	}
	return err
}

// Create 直接写入一篇笔记，不经过审阅。id 为空时生成 pg- 前缀。
// p 的标题和正文会被截断。
func (s *Store) Create(p pages.Page) (pages.Page, error) {
	if s == nil {
		return pages.Page{}, fmt.Errorf("pages store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.create(p)
}

func (s *Store) create(p pages.Page) (pages.Page, error) {
	now := time.Now().UTC()
	p.SpaceID = pages.NormalizeSpace(p.SpaceID)
	p.ID = pages.NormalizePageID(p.ID)
	p.ParentID = pages.NormalizePageID(p.ParentID)
	if p.ParentID == "-" || strings.EqualFold(p.ParentID, "root") {
		p.ParentID = ""
	}
	p.Title = pages.ClipTitle(p.Title)
	if strings.TrimSpace(p.Title) == "" {
		p.Title = "Untitled"
	}
	p.Content = pages.ClipBody(p.Content)
	if p.ID == "" {
		p.ID = now.Format("pg-20060102T150405.000000000")
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Revision < 1 {
		p.Revision = 1
	}
	_, err := s.db.Exec(`INSERT INTO notes
		(id, space_id, parent_id, title, content, revision, source_session, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.SpaceID, p.ParentID, p.Title, p.Content, p.Revision, p.SourceSession,
		p.CreatedAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return pages.Page{}, err
	}
	_ = s.markNotesStale()
	return p, nil
}

// Save 按期望修订号保存。expectedRevision 大于 0 且与当前不符时返回冲突。
// p 的 id 必须已经存在。
func (s *Store) Save(p pages.Page, expectedRevision int) (pages.Page, error) {
	if s == nil {
		return pages.Page{}, fmt.Errorf("pages store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(p, expectedRevision)
}

func (s *Store) save(p pages.Page, expectedRevision int) (pages.Page, error) {
	cur, err := s.get(p.SpaceID, p.ID)
	if err != nil {
		return pages.Page{}, err
	}
	if expectedRevision > 0 && cur.Revision != expectedRevision {
		return pages.Page{}, &pages.ConflictError{Page: cur}
	}
	p.ID = cur.ID
	p.SpaceID = cur.SpaceID
	p.Title = pages.ClipTitle(firstNonEmpty(p.Title, cur.Title))
	p.Content = pages.ClipBody(p.Content)
	switch {
	case p.ParentID == "-" || strings.EqualFold(p.ParentID, "root"):
		p.ParentID = ""
	case strings.TrimSpace(p.ParentID) == "":
		p.ParentID = cur.ParentID
	default:
		p.ParentID = pages.NormalizePageID(p.ParentID)
		if p.ParentID == p.ID {
			p.ParentID = cur.ParentID
		}
	}
	p.CreatedAt = cur.CreatedAt
	p.SourceSession = firstNonEmpty(p.SourceSession, cur.SourceSession)
	p.Revision = cur.Revision + 1
	p.UpdatedAt = time.Now().UTC()
	_, err = s.db.Exec(`UPDATE notes SET parent_id = ?, title = ?, content = ?, revision = ?, source_session = ?, updated_at = ? WHERE id = ?`,
		p.ParentID, p.Title, p.Content, p.Revision, p.SourceSession, p.UpdatedAt.UTC().Format(time.RFC3339Nano), p.ID)
	if err != nil {
		return pages.Page{}, err
	}
	_ = s.markNotesStale()
	return p, nil
}

// Propose 记下待审阅，不改笔记。同一工具调用或同一哈希会替换未决草稿，但保留原来的 id。
// r 的标题和正文会被截断，哈希按批准公式重算。
func (s *Store) Propose(r pages.Review) (pages.Review, error) {
	if s == nil {
		return pages.Review{}, fmt.Errorf("pages store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r.SpaceID = pages.NormalizeSpace(r.SpaceID)
	r.PageID = pages.NormalizePageID(r.PageID)
	r.ParentID = pages.NormalizePageID(r.ParentID)
	r.Title = pages.ClipTitle(r.Title)
	r.Content = pages.ClipBody(r.Content)
	r.Hash = pages.HashDraft(r.Title, r.Content, r.PageID, r.SpaceID, r.ExpectedRevision)
	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.ExpiresAt.IsZero() {
		r.ExpiresAt = now.Add(30 * time.Minute)
	}
	r.Status = pages.ReviewAwaiting
	if r.ID == "" {
		r.ID = pages.ReviewID(r.Hash)
	}
	rows, err := s.db.Query(`SELECT id, thread_id, tool_call_id, session_id, space_id, page_id, parent_id, title, content, expected_revision, hash, status, created_at, expires_at, saved_page_id
		FROM note_reviews WHERE status = ?`, string(pages.ReviewAwaiting))
	if err != nil {
		return pages.Review{}, err
	}
	var waiting []pages.Review
	for rows.Next() {
		existing, err := scanReview(rows)
		if err != nil {
			continue
		}
		waiting = append(waiting, existing)
	}
	rows.Close()
	for _, existing := range waiting {
		sameCall := r.ToolCallID != "" && existing.ToolCallID == r.ToolCallID && existing.ThreadID == r.ThreadID
		if !sameCall && existing.Hash != r.Hash {
			continue
		}
		if existing.SavedPageID != "" {
			return existing, nil
		}
		r.ID = existing.ID
		r.CreatedAt = existing.CreatedAt
		if err := s.insertReviewReplace(r); err != nil {
			return pages.Review{}, err
		}
		return r, nil
	}
	if err := s.insertReview(r); err != nil {
		return pages.Review{}, err
	}
	return r, nil
}

func (s *Store) insertReviewReplace(r pages.Review) error {
	_, err := s.db.Exec(`UPDATE note_reviews SET thread_id=?, tool_call_id=?, session_id=?, space_id=?, page_id=?, parent_id=?, title=?, content=?, expected_revision=?, hash=?, status=?, created_at=?, expires_at=?, saved_page_id=? WHERE id=?`,
		r.ThreadID, r.ToolCallID, r.SessionID, r.SpaceID, r.PageID, r.ParentID, r.Title, r.Content, r.ExpectedRevision, r.Hash, string(r.Status),
		r.CreatedAt.UTC().Format(time.RFC3339Nano), r.ExpiresAt.UTC().Format(time.RFC3339Nano), r.SavedPageID, r.ID)
	return err
}

// Pending 返回仍在等待的审阅。过期的改成 expired。
func (s *Store) Pending() []pages.Review {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	rows, err := s.db.Query(`SELECT id, thread_id, tool_call_id, session_id, space_id, page_id, parent_id, title, content, expected_revision, hash, status, created_at, expires_at, saved_page_id FROM note_reviews WHERE status = ?`, string(pages.ReviewAwaiting))
	if err != nil {
		return nil
	}
	var all []pages.Review
	for rows.Next() {
		r, err := scanReview(rows)
		if err != nil {
			continue
		}
		all = append(all, r)
	}
	rows.Close()
	var out []pages.Review
	for _, r := range all {
		if now.After(r.ExpiresAt) {
			_, _ = s.db.Exec(`UPDATE note_reviews SET status = ? WHERE id = ?`, string(pages.ReviewExpired), r.ID)
			continue
		}
		out = append(out, r)
	}
	return out
}

// Decide 按哈希批准或拒绝。approve 为假只改状态。为真时以修订号 0 写入，并发改过也不会把卡片卡死。
// hash 非空且不符时返回 review hash mismatch。
func (s *Store) Decide(id, hash string, approve bool) (pages.Page, pages.Review, error) {
	if s == nil {
		return pages.Page{}, pages.Review{}, fmt.Errorf("pages store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.findReview(id)
	if err != nil {
		return pages.Page{}, pages.Review{}, err
	}
	if hash != "" && hash != r.Hash {
		return pages.Page{}, pages.Review{}, fmt.Errorf("review hash mismatch")
	}
	if r.Status == pages.ReviewSaved && r.SavedPageID != "" {
		pg, err := s.get(r.SpaceID, r.SavedPageID)
		return pg, r, err
	}
	if r.Status != pages.ReviewAwaiting {
		return pages.Page{}, r, fmt.Errorf("review is %s", r.Status)
	}
	if !approve {
		_, err = s.db.Exec(`UPDATE note_reviews SET status = ? WHERE id = ?`, string(pages.ReviewDeclined), r.ID)
		r.Status = pages.ReviewDeclined
		return pages.Page{}, r, err
	}
	pg, err := s.applyApproved(r)
	if err != nil {
		return pages.Page{}, r, err
	}
	_, err = s.db.Exec(`UPDATE note_reviews SET status = ?, saved_page_id = ? WHERE id = ?`, string(pages.ReviewSaved), pg.ID, r.ID)
	r.Status = pages.ReviewSaved
	r.SavedPageID = pg.ID
	return pg, r, err
}

func (s *Store) applyApproved(r pages.Review) (pages.Page, error) {
	if r.PageID == "" {
		return s.create(pages.Page{SpaceID: r.SpaceID, ParentID: r.ParentID, Title: r.Title, Content: r.Content, SourceSession: r.SessionID})
	}
	if _, err := s.get(r.SpaceID, r.PageID); err != nil {
		return s.create(pages.Page{ID: r.PageID, SpaceID: r.SpaceID, ParentID: r.ParentID, Title: r.Title, Content: r.Content, SourceSession: r.SessionID})
	}
	return s.save(pages.Page{ID: r.PageID, SpaceID: r.SpaceID, ParentID: r.ParentID, Title: r.Title, Content: r.Content, SourceSession: r.SessionID}, 0)
}

func (s *Store) findReview(id string) (pages.Review, error) {
	id = strings.TrimSpace(id)
	row := s.db.QueryRow(`SELECT id, thread_id, tool_call_id, session_id, space_id, page_id, parent_id, title, content, expected_revision, hash, status, created_at, expires_at, saved_page_id
		FROM note_reviews WHERE id = ? OR hash = ?`, id, id)
	r, err := scanReview(row)
	if err == sql.ErrNoRows {
		return pages.Review{}, fmt.Errorf("review not found")
	}
	return r, err
}

func (s *Store) markNotesStale() error {
	_, err := s.db.Exec(`UPDATE libraries SET status = 'needs_reindex', updated_at = ? WHERE id = 'notes' AND published_gen > 0`, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

type rower interface {
	Scan(dest ...any) error
}

func scanNote(row rower) (pages.Page, error) {
	var pg pages.Page
	var created, updated string
	err := row.Scan(&pg.ID, &pg.SpaceID, &pg.ParentID, &pg.Title, &pg.Revision, &pg.SourceSession, &created, &updated, &pg.Content)
	if err != nil {
		return pages.Page{}, err
	}
	pg.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	pg.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return pg, nil
}

func scanReview(row rower) (pages.Review, error) {
	var r pages.Review
	var status, created, expires string
	err := row.Scan(&r.ID, &r.ThreadID, &r.ToolCallID, &r.SessionID, &r.SpaceID, &r.PageID, &r.ParentID, &r.Title, &r.Content, &r.ExpectedRevision, &r.Hash, &status, &created, &expires, &r.SavedPageID)
	if err != nil {
		return pages.Review{}, err
	}
	r.Status = pages.ReviewStatus(status)
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	r.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return r, nil
}

func metaOf(pg pages.Page) pages.Meta {
	return pages.Meta{
		ID: pg.ID, SpaceID: pg.SpaceID, ParentID: pg.ParentID, Title: pg.Title,
		Revision: pg.Revision, UpdatedAt: pg.UpdatedAt, CreatedAt: pg.CreatedAt,
		SourceSession: pg.SourceSession, Preview: pages.Preview(pg.Content),
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
