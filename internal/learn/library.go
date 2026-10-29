// 文库是导入文件或笔记的集合。notes 这个库由打开时保证存在，原文仍是笔记表。

package learn

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// Library 是一个可检索的库。
type Library struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	LinkedPath   string `json:"linked_path,omitempty"`
	Status       string `json:"status"`
	EmbedModel   string `json:"embed_model,omitempty"`
	PublishedGen int    `json:"published_gen"`
	LastError    string `json:"last_error,omitempty"`
	Documents    int    `json:"documents"`
	UpdatedAt    string `json:"updated_at"`
}

// Document 是库里的一份原文。
type Document struct {
	ID        string `json:"id"`
	LibraryID string `json:"library_id"`
	RelPath   string `json:"rel_path"`
	Title     string `json:"title"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
	Kind      string `json:"kind"`
	PageCount int    `json:"page_count"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Removed   bool   `json:"removed,omitempty"`
}

// ensureNotesLibrary 保证 id 为 notes 的库存在。它索引笔记，不接收导入文件。
func (s *Store) ensureNotesLibrary() error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`INSERT OR IGNORE INTO libraries
		(id, name, kind, status, created_at, updated_at) VALUES ('notes', '笔记', 'notes', 'ready', ?, ?)`, now, now)
	return err
}

// Libraries 列出全部库，并带上未删除的文档数。
func (s *Store) Libraries() []Library {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, name, kind, linked_path, status, embed_model, published_gen, last_error, updated_at FROM libraries ORDER BY name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Library
	for rows.Next() {
		var lib Library
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.Kind, &lib.LinkedPath, &lib.Status, &lib.EmbedModel, &lib.PublishedGen, &lib.LastError, &lib.UpdatedAt); err != nil {
			continue
		}
		out = append(out, lib)
	}
	rows.Close()
	for i := range out {
		if out[i].Kind == "notes" {
			_ = s.db.QueryRow(`SELECT COUNT(1) FROM notes`).Scan(&out[i].Documents)
			continue
		}
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM documents WHERE library_id = ? AND removed = 0`, out[i].ID).Scan(&out[i].Documents)
	}
	return out
}

// CreateLibrary 新建一个 indexed 或 linked 库。name 去掉空白后不能空，最长 80 个符文。
// kind 不是 linked 时写成 indexed。linked 必须带 linkedPath。
func (s *Store) CreateLibrary(name, kind, linkedPath string) (Library, error) {
	if s == nil {
		return Library{}, fmt.Errorf("learn store unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Library{}, fmt.Errorf("library name required")
	}
	if utf8.RuneCountInString(name) > 80 {
		name = string([]rune(name)[:80])
	}
	kind = strings.TrimSpace(kind)
	if kind != "linked" {
		kind = "indexed"
	}
	if kind == "linked" && strings.TrimSpace(linkedPath) == "" {
		return Library{}, fmt.Errorf("linked path required")
	}
	now := time.Now().UTC()
	lib := Library{
		ID: now.Format("lib-20060102T150405.000000000"), Name: name, Kind: kind,
		LinkedPath: strings.TrimSpace(linkedPath), Status: "ready", UpdatedAt: now.Format(time.RFC3339Nano),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`INSERT INTO libraries (id, name, kind, linked_path, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'ready', ?, ?)`,
		lib.ID, lib.Name, lib.Kind, lib.LinkedPath, lib.UpdatedAt, lib.UpdatedAt)
	return lib, err
}

// DeleteLibrary 删除库及其文档、切块和检索行。id 为 notes 时拒绝。
func (s *Store) DeleteLibrary(id string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	if id == "notes" {
		return fmt.Errorf("the notes library stays")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.library(id); err != nil {
		return err
	}
	_, _ = s.db.Exec(`DELETE FROM chunks_fts WHERE library_id = ?`, id)
	_, _ = s.db.Exec(`DELETE FROM embeddings WHERE chunk_id IN (SELECT id FROM chunks WHERE library_id = ?)`, id)
	_, _ = s.db.Exec(`DELETE FROM chunks WHERE library_id = ?`, id)
	_, _ = s.db.Exec(`DELETE FROM documents WHERE library_id = ?`, id)
	_, _ = s.db.Exec(`DELETE FROM generations WHERE library_id = ?`, id)
	_, _ = s.db.Exec(`DELETE FROM index_journal WHERE library_id = ?`, id)
	_, err := s.db.Exec(`DELETE FROM libraries WHERE id = ?`, id)
	return err
}

// SetEmbedModel 记下这个库使用的嵌入模型。空字符串表示只做词法检索。
// model 去掉空白后写入。已发布的一代因此标成 needs_reindex。
func (s *Store) SetEmbedModel(id, model string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.library(id); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`UPDATE libraries SET embed_model = ?, status = CASE WHEN published_gen > 0 THEN 'needs_reindex' ELSE status END, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(model), now, id)
	return err
}

// Documents 列出一个库里未删除的文档。
// libraryID 必须存在。
func (s *Store) Documents(libraryID string) ([]Document, error) {
	if s == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.library(libraryID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, library_id, rel_path, title, sha256, bytes, kind, page_count, status, error, removed FROM documents WHERE library_id = ? AND removed = 0 ORDER BY rel_path`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Document
	for rows.Next() {
		var d Document
		var removed int
		if err := rows.Scan(&d.ID, &d.LibraryID, &d.RelPath, &d.Title, &d.SHA256, &d.Bytes, &d.Kind, &d.PageCount, &d.Status, &d.Error, &removed); err != nil {
			continue
		}
		d.Removed = removed != 0
		out = append(out, d)
	}
	return out, nil
}

// Import 把一个文件或目录抄进库。目录最多收 500 个支持的文件。笔记库拒绝导入。
// libraryID 必须是 indexed 或 linked。path 是操作者机器上的路径。
func (s *Store) Import(libraryID, path string) (int, error) {
	if s == nil {
		return 0, fmt.Errorf("learn store unavailable")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, fmt.Errorf("path required")
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	var files []string
	if st.IsDir() {
		_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			if !supportedExt(p) {
				return nil
			}
			if len(files) < 500 {
				files = append(files, p)
			}
			return nil
		})
	} else {
		if !supportedExt(path) {
			return 0, fmt.Errorf("unsupported import type")
		}
		files = []string{path}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lib, err := s.library(libraryID)
	if err != nil {
		return 0, err
	}
	if lib.Kind == "notes" {
		return 0, fmt.Errorf("import files into another library")
	}
	n := 0
	for _, file := range files {
		if err := s.importOne(lib, path, file); err != nil {
			continue
		}
		n++
	}
	if n > 0 {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, _ = s.db.Exec(`UPDATE libraries SET status = 'needs_reindex', updated_at = ? WHERE id = ?`, now, libraryID)
	}
	return n, nil
}

func (s *Store) importOne(lib Library, root, file string) error {
	st, err := os.Stat(file)
	if err != nil || st.Size() > MaxImportBytes {
		return fmt.Errorf("skip")
	}
	sum, err := hashFile(file)
	if err != nil {
		return err
	}
	rel := filepath.Base(file)
	if stRoot, err := os.Stat(root); err == nil && stRoot.IsDir() {
		if r, err := filepath.Rel(root, file); err == nil {
			rel = filepath.ToSlash(r)
		}
	}
	external := ""
	ext := strings.ToLower(filepath.Ext(file))
	if lib.Kind == "linked" {
		external = file
	} else {
		dest := filepath.Join(s.root, "raw", sum+ext)
		if _, err := os.Stat(dest); err != nil {
			if err := copyFile(file, dest); err != nil {
				return err
			}
		}
	}
	id := "doc-" + sum[:12] + "-" + sanitizeRel(rel)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.Exec(`INSERT INTO documents
		(id, library_id, rel_path, title, sha256, bytes, kind, external_path, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'file', ?, 'pending', ?)
		ON CONFLICT(library_id, rel_path) DO UPDATE SET sha256 = excluded.sha256, bytes = excluded.bytes, external_path = excluded.external_path, removed = 0, status = 'pending'`,
		id, lib.ID, rel, strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), sum, st.Size(), external, now)
	return err
}

// RemoveDocument 把一份原文标成已删除，不拆库，也不改已发布的一代。库状态改为 needs_reindex。
// libraryID 和 documentID 必须匹配。
func (s *Store) RemoveDocument(libraryID, documentID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE documents SET removed = 1, status = 'removed' WHERE id = ? AND library_id = ?`, documentID, libraryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("document not found")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.Exec(`UPDATE libraries SET status = 'needs_reindex', updated_at = ? WHERE id = ?`, now, libraryID)
	return err
}

func (s *Store) library(id string) (Library, error) {
	var lib Library
	err := s.db.QueryRow(`SELECT id, name, kind, linked_path, status, embed_model, published_gen, last_error, updated_at FROM libraries WHERE id = ?`, id).
		Scan(&lib.ID, &lib.Name, &lib.Kind, &lib.LinkedPath, &lib.Status, &lib.EmbedModel, &lib.PublishedGen, &lib.LastError, &lib.UpdatedAt)
	if err != nil {
		return Library{}, fmt.Errorf("library not found")
	}
	return lib, nil
}

func supportedExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".html", ".htm", ".docx", ".pdf":
		return true
	default:
		return false
	}
}

func sanitizeRel(rel string) string {
	var b strings.Builder
	for _, r := range rel {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		out = "file"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
