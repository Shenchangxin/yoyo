// 阅读台把一份文档收成材料，批注和定位都挂在材料上。定位先查已发布的切块，不把别的库混进来。

package learn

import (
	"fmt"
	"strings"
	"time"
)

// Material 是阅读台上的一份材料。
type Material struct {
	ID         string `json:"id"`
	LibraryID  string `json:"library_id"`
	DocumentID string `json:"document_id,omitempty"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	URL        string `json:"url,omitempty"`
}

// Annotation 是材料上的一条批注。
type Annotation struct {
	ID         string `json:"id"`
	MaterialID string `json:"material_id"`
	Quote      string `json:"quote"`
	Note       string `json:"note"`
	Page       int    `json:"page"`
	CreatedAt  string `json:"created_at"`
}

// OpenMaterial 把库里的一份文档放到阅读台。同库同文档已有材料时返回原来的。
// libraryID 和 documentID 必须匹配，且文档未删除。
func (s *Store) OpenMaterial(libraryID, documentID string) (Material, error) {
	if s == nil {
		return Material{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var title string
	err := s.db.QueryRow(`SELECT title FROM documents WHERE id = ? AND library_id = ? AND removed = 0`, documentID, libraryID).Scan(&title)
	if err != nil && libraryID == "notes" {
		err = s.db.QueryRow(`SELECT title FROM notes WHERE id = ?`, documentID).Scan(&title)
	}
	if err != nil {
		return Material{}, fmt.Errorf("document not found")
	}
	var m Material
	err = s.db.QueryRow(`SELECT id, library_id, document_id, kind, title, url FROM materials WHERE library_id = ? AND document_id = ?`, libraryID, documentID).
		Scan(&m.ID, &m.LibraryID, &m.DocumentID, &m.Kind, &m.Title, &m.URL)
	if err == nil {
		return m, nil
	}
	m = Material{ID: newKey("mat"), LibraryID: libraryID, DocumentID: documentID, Kind: "doc", Title: title}
	_, err = s.db.Exec(`INSERT INTO materials (id, library_id, document_id, kind, title, created_at) VALUES (?, ?, ?, 'doc', ?, ?)`,
		m.ID, libraryID, documentID, title, time.Now().UTC().Format(time.RFC3339Nano))
	return m, err
}

// Materials 列出阅读台上的材料。
func (s *Store) Materials() []Material {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, library_id, document_id, kind, title, url FROM materials ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Material
	for rows.Next() {
		var m Material
		if err := rows.Scan(&m.ID, &m.LibraryID, &m.DocumentID, &m.Kind, &m.Title, &m.URL); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

// Annotate 在材料上记一条批注。quote 和 note 不能同时空。page 小于 0 时写成 0。
// materialID 必须存在。
func (s *Store) Annotate(materialID, quote, note string, page int) (Annotation, error) {
	if s == nil {
		return Annotation{}, fmt.Errorf("learn store unavailable")
	}
	quote = strings.TrimSpace(quote)
	note = strings.TrimSpace(note)
	if quote == "" && note == "" {
		return Annotation{}, fmt.Errorf("quote or note required")
	}
	if page < 0 {
		page = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM materials WHERE id = ?`, materialID).Scan(&n)
	if n == 0 {
		return Annotation{}, fmt.Errorf("material not found")
	}
	a := Annotation{
		ID: newKey("ann"), MaterialID: materialID, Quote: clipRunes(quote, 400), Note: clipRunes(note, 800),
		Page: page, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	_, err := s.db.Exec(`INSERT INTO annotations (id, material_id, quote, note, page_no, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.MaterialID, a.Quote, a.Note, a.Page, a.CreatedAt)
	return a, err
}

// Annotations 列出一份材料上的批注。materialID 传给查询。
func (s *Store) Annotations(materialID string) []Annotation {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, material_id, quote, note, page_no, created_at FROM annotations WHERE material_id = ? ORDER BY created_at`, materialID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Annotation
	for rows.Next() {
		var a Annotation
		if err := rows.Scan(&a.ID, &a.MaterialID, &a.Quote, &a.Note, &a.Page, &a.CreatedAt); err != nil {
			continue
		}
		out = append(out, a)
	}
	return out
}

// Locate 在这份材料所属的库里检索，只留下同一份文档的命中。query 传给 SearchLibrary。
// materialID 决定库和文档，调用方不能改成另一份库。
func (s *Store) Locate(materialID, query string) ([]Hit, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	var libraryID, documentID string
	err := s.db.QueryRow(`SELECT library_id, document_id FROM materials WHERE id = ?`, materialID).Scan(&libraryID, &documentID)
	s.mu.Unlock()
	if err != nil {
		return nil, "", fmt.Errorf("material not found")
	}
	hits, msg, err := s.SearchLibrary(libraryID, query)
	if err != nil || msg != "" {
		return hits, msg, err
	}
	var own []Hit
	for _, h := range hits {
		if h.DocumentID == documentID {
			own = append(own, h)
		}
	}
	if len(own) == 0 {
		return nil, "检索完成，没有命中", nil
	}
	return own, "", nil
}

// FileNames 返回库里最多 limit 个文件名，给提示用。limit 小于 1 时用 20，大于 20 时也用 20。
// libraryID 是 notes 时返回笔记标题。
func (s *Store) FileNames(libraryID string, limit int) []string {
	if s == nil {
		return nil
	}
	if limit < 1 || limit > 20 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	q := `SELECT title FROM documents WHERE library_id = ? AND removed = 0 ORDER BY rel_path LIMIT ?`
	args := []any{libraryID, limit}
	if libraryID == "notes" {
		q = `SELECT title FROM notes ORDER BY updated_at DESC LIMIT ?`
		args = []any{limit}
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && name != "" {
			out = append(out, name)
		}
	}
	return out
}
