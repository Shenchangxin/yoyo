// 旧 Markdown 页面只迁一次。之后笔记的权威在 SQLite，pages 目录留作档案。

package learn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/pages"
)

// migratePages 在 flags.pages_imported 还不存在时，把 pagesRoot 下的 .md 和待审阅抄进来。
// pagesRoot 不存在或没有文件时也写上标记，避免每次启动再扫。
func (s *Store) migratePages(pagesRoot string) error {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM flags WHERE k = 'pages_imported'`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if pagesRoot != "" {
		_ = filepath.WalkDir(pagesRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
				return nil
			}
			pg, err := pages.ReadFile(path)
			if err != nil || pg.ID == "" {
				return nil
			}
			_ = s.insertNote(pg)
			return nil
		})
		b, err := os.ReadFile(filepath.Join(pagesRoot, "reviews.json"))
		if err == nil {
			var revs []pages.Review
			if json.Unmarshal(b, &revs) == nil {
				for _, r := range revs {
					if r.Status != pages.ReviewAwaiting || r.ID == "" {
						continue
					}
					_ = s.insertReview(r)
				}
			}
		}
	}
	_, err = s.db.Exec(`INSERT INTO flags (k, v) VALUES ('pages_imported', ?)`, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) insertNote(pg pages.Page) error {
	if pg.SpaceID == "" {
		pg.SpaceID = pages.DefaultSpace
	}
	if pg.Revision < 1 {
		pg.Revision = 1
	}
	if pg.CreatedAt.IsZero() {
		pg.CreatedAt = time.Now().UTC()
	}
	if pg.UpdatedAt.IsZero() {
		pg.UpdatedAt = pg.CreatedAt
	}
	_, err := s.db.Exec(`INSERT OR IGNORE INTO notes
		(id, space_id, parent_id, title, content, revision, source_session, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		pg.ID, pg.SpaceID, pg.ParentID, pg.Title, pg.Content, pg.Revision, pg.SourceSession,
		pg.CreatedAt.UTC().Format(time.RFC3339Nano), pg.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) insertReview(r pages.Review) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO note_reviews
		(id, thread_id, tool_call_id, session_id, space_id, page_id, parent_id, title, content, expected_revision, hash, status, created_at, expires_at, saved_page_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.ThreadID, r.ToolCallID, r.SessionID, r.SpaceID, r.PageID, r.ParentID, r.Title, r.Content,
		r.ExpectedRevision, r.Hash, string(r.Status),
		r.CreatedAt.UTC().Format(time.RFC3339Nano), r.ExpiresAt.UTC().Format(time.RFC3339Nano), r.SavedPageID)
	return err
}
