package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	DefaultSpace = "home"
	MaxTitle     = 160
	MaxBody      = 100_000
)

type Page struct {
	ID            string    `json:"id" yaml:"id"`
	SpaceID       string    `json:"space_id" yaml:"space_id"`
	ParentID      string    `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
	Title         string    `json:"title" yaml:"title"`
	Content       string    `json:"content" yaml:"-"`
	Revision      int       `json:"revision" yaml:"revision"`
	SourceSession string    `json:"source_session,omitempty" yaml:"source_session,omitempty"`
	CreatedAt     time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" yaml:"updated_at"`
}

type Meta struct {
	ID            string    `json:"id"`
	SpaceID       string    `json:"space_id"`
	ParentID      string    `json:"parent_id,omitempty"`
	Title         string    `json:"title"`
	Revision      int       `json:"revision"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
	SourceSession string    `json:"source_session,omitempty"`
	Preview       string    `json:"preview,omitempty"`
}

type ConflictError struct {
	Page Page
}

func (e *ConflictError) Error() string { return "page revision conflict" }

type Store struct {
	mu   sync.Mutex
	root string
}

func Open(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, DefaultSpace), 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root}
	_ = s.rebuildIndex()
	return s, nil
}

func (s *Store) Root() string { return s.root }

func (s *Store) List(spaceID string) []Meta {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.loadIndex()
	spaceID = normalizeSpace(spaceID)
	var out []Meta
	for _, m := range idx {
		if spaceID != "" && m.SpaceID != spaceID {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].Title < out[j].Title
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (s *Store) Search(q, spaceID string) []Meta {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return s.List(spaceID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	spaceID = normalizeSpace(spaceID)
	var out []Meta
	_ = filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		pg, err := readPageFile(path)
		if err != nil {
			return nil
		}
		if spaceID != "" && pg.SpaceID != spaceID {
			return nil
		}
		blob := strings.ToLower(pg.Title + "\n" + pg.Content)
		if !strings.Contains(blob, q) {
			return nil
		}
		out = append(out, metaOf(pg))
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

func (s *Store) Get(spaceID, id string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(spaceID, id)
}

func (s *Store) Delete(spaceID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteLocked(spaceID, id)
}

func (s *Store) deleteLocked(spaceID, id string) error {
	pg, err := s.getLocked(spaceID, id)
	if err != nil {
		return err
	}
	path := s.pagePath(pg.SpaceID, pg.ID)
	if err := s.ensureInsideRoot(path); err != nil {
		return err
	}
	for _, m := range s.loadIndex() {
		if m.ParentID != pg.ID || m.ID == pg.ID {
			continue
		}
		child, err := s.getLocked(m.SpaceID, m.ID)
		if err != nil {
			continue
		}
		child.ParentID = pg.ParentID
		child.UpdatedAt = time.Now().UTC()
		if err := s.writePage(child); err != nil {
			return err
		}
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = s.rebuildIndexLocked()
	return nil
}

func (s *Store) Create(p Page) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createLocked(p)
}

func (s *Store) createLocked(p Page) (Page, error) {
	now := time.Now().UTC()
	p.SpaceID = normalizeSpace(p.SpaceID)
	p.ID = normalizePageID(p.ID)
	p.ParentID = normalizePageID(p.ParentID)
	if p.ParentID == "-" || strings.EqualFold(p.ParentID, "root") {
		p.ParentID = ""
	}
	p.Title = clipTitle(p.Title)
	p.Content = clipBody(p.Content)
	if p.ID == "" {
		p.ID = s.uniqueID(p.SpaceID)
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Revision < 1 {
		p.Revision = 1
	}
	if err := s.writePage(p); err != nil {
		return Page{}, err
	}
	_ = s.rebuildIndexLocked()
	return p, nil
}

func (s *Store) Save(p Page, expectedRevision int) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(p, expectedRevision)
}

func (s *Store) saveLocked(p Page, expectedRevision int) (Page, error) {
	cur, err := s.getLocked(p.SpaceID, p.ID)
	if err != nil {
		return Page{}, err
	}
	if expectedRevision > 0 && cur.Revision != expectedRevision {
		return Page{}, &ConflictError{Page: cur}
	}
	p.ID = cur.ID
	p.SpaceID = cur.SpaceID
	p.Title = clipTitle(firstNonEmpty(p.Title, cur.Title))
	p.Content = clipBody(p.Content)
	switch {
	case p.ParentID == "-" || strings.EqualFold(p.ParentID, "root"):
		p.ParentID = ""
	case strings.TrimSpace(p.ParentID) == "":
		p.ParentID = cur.ParentID
	default:
		p.ParentID = normalizePageID(p.ParentID)
		if p.ParentID == p.ID {
			p.ParentID = cur.ParentID
		}
	}
	p.CreatedAt = cur.CreatedAt
	p.SourceSession = firstNonEmpty(p.SourceSession, cur.SourceSession)
	p.Revision = cur.Revision + 1
	p.UpdatedAt = time.Now().UTC()
	if err := s.writePage(p); err != nil {
		return Page{}, err
	}
	_ = s.rebuildIndexLocked()
	return p, nil
}

func (s *Store) ApplyApproved(spaceID, pageID, parentID, title, content, sourceSession string, expectedRevision int) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyApprovedLocked(spaceID, pageID, parentID, title, content, sourceSession, expectedRevision)
}

func (s *Store) applyApprovedLocked(spaceID, pageID, parentID, title, content, sourceSession string, expectedRevision int) (Page, error) {
	title = clipTitle(title)
	content = clipBody(content)
	spaceID = normalizeSpace(spaceID)
	pageID = normalizePageID(pageID)
	parentID = normalizePageID(parentID)
	draft := Page{
		ID: pageID, SpaceID: spaceID, ParentID: parentID, Title: title,
		Content: content, SourceSession: sourceSession,
	}
	if pageID == "" {
		return s.createLocked(draft)
	}
	cur, err := s.getLocked(spaceID, pageID)
	if err != nil {
		return s.createLocked(draft)
	}
	cur.Title = title
	cur.Content = content
	if parentID != "" {
		cur.ParentID = parentID
	}
	if sourceSession != "" {
		cur.SourceSession = sourceSession
	}
	return s.saveLocked(cur, expectedRevision)
}

func (s *Store) RebuildIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.rebuildIndexLocked()
	return nil
}

func (s *Store) getLocked(spaceID, id string) (Page, error) {
	id = normalizePageID(id)
	if id == "" {
		return Page{}, fmt.Errorf("page not found")
	}
	spaceID = normalizeSpace(spaceID)
	candidates := []string{s.pagePath(spaceID, id)}
	if spaceID != DefaultSpace {
		candidates = append(candidates, s.pagePath(DefaultSpace, id))
	}
	_ = filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.TrimSuffix(d.Name(), ".md") == id {
			candidates = append(candidates, path)
		}
		return nil
	})
	seen := map[string]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		pg, err := readPageFile(p)
		if err != nil {
			continue
		}
		if pg.ID == id {
			return pg, nil
		}
	}
	return Page{}, fmt.Errorf("page not found")
}

func (s *Store) writePage(p Page) error {
	path := s.pagePath(p.SpaceID, p.ID)
	if err := s.ensureInsideRoot(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := encodePage(p)
	if err != nil {
		return err
	}
	return atomicWrite(path, raw, 0o644)
}

func (s *Store) pagePath(spaceID, id string) string {
	return filepath.Join(s.root, normalizeSpace(spaceID), id+".md")
}

func (s *Store) loadIndex() []Meta {
	b, err := os.ReadFile(filepath.Join(s.root, "index.json"))
	if err != nil {
		return s.rebuildIndexLocked()
	}
	var out []Meta
	if json.Unmarshal(b, &out) != nil {
		return s.rebuildIndexLocked()
	}
	return out
}

func (s *Store) rebuildIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.rebuildIndexLocked()
	return nil
}

func (s *Store) rebuildIndexLocked() []Meta {
	var out []Meta
	_ = filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		pg, err := readPageFile(path)
		if err != nil {
			return nil
		}
		out = append(out, metaOf(pg))
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	b, _ := json.MarshalIndent(out, "", "  ")
	_ = atomicWrite(filepath.Join(s.root, "index.json"), b, 0o644)
	return out
}

type frontmatter struct {
	ID            string    `yaml:"id"`
	SpaceID       string    `yaml:"space_id"`
	ParentID      string    `yaml:"parent_id,omitempty"`
	Title         string    `yaml:"title"`
	Revision      int       `yaml:"revision"`
	SourceSession string    `yaml:"source_session,omitempty"`
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
}

func encodePage(p Page) ([]byte, error) {
	fm := frontmatter{
		ID: p.ID, SpaceID: p.SpaceID, ParentID: p.ParentID, Title: p.Title,
		Revision: p.Revision, SourceSession: p.SourceSession, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	head, err := yaml.Marshal(&fm)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(head)
	if !strings.HasSuffix(string(head), "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("---\n\n")
	b.WriteString(p.Content)
	if p.Content != "" && !strings.HasSuffix(p.Content, "\n") {
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

func readPageFile(path string) (Page, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Page{}, err
	}
	return decodePage(b)
}

func decodePage(raw []byte) (Page, error) {
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return Page{}, fmt.Errorf("missing frontmatter")
	}
	rest := text[3:]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Page{}, fmt.Errorf("unterminated frontmatter")
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return Page{}, err
	}
	body := rest[end+4:]
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimSuffix(body, "\n")
	body = strings.TrimSuffix(body, "\r")
	if fm.SpaceID == "" {
		fm.SpaceID = DefaultSpace
	}
	if fm.Revision < 1 {
		fm.Revision = 1
	}
	return Page{
		ID: fm.ID, SpaceID: fm.SpaceID, ParentID: fm.ParentID, Title: fm.Title,
		Content: body, Revision: fm.Revision, SourceSession: fm.SourceSession,
		CreatedAt: fm.CreatedAt, UpdatedAt: fm.UpdatedAt,
	}, nil
}

func metaOf(p Page) Meta {
	return Meta{
		ID: p.ID, SpaceID: p.SpaceID, ParentID: p.ParentID, Title: p.Title,
		Revision: p.Revision, UpdatedAt: p.UpdatedAt, CreatedAt: p.CreatedAt, SourceSession: p.SourceSession,
		Preview: previewOf(p.Content),
	}
}

func previewOf(content string) string {
	s := strings.Join(strings.Fields(content), " ")
	n := utf8.RuneCountInString(s)
	if n <= 160 {
		return s
	}
	return string([]rune(s)[:160])
}

func normalizeSpace(id string) string {
	id = sanitizeSegment(id, false)
	if id == "" {
		return DefaultSpace
	}
	return id
}

func normalizePageID(id string) string {
	id = sanitizeSegment(id, true)
	for strings.Contains(id, "..") {
		id = strings.ReplaceAll(id, "..", "")
	}
	return id
}

func sanitizeSegment(id string, allowDot bool) string {
	id = strings.TrimSpace(id)
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case allowDot && r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Store) uniqueID(spaceID string) string {
	for i := 0; i < 8; i++ {
		id := newID()
		if i > 0 {
			id = fmt.Sprintf("%s-%d", id, i)
		}
		if _, err := os.Stat(s.pagePath(spaceID, id)); err != nil {
			return id
		}
	}
	return newID() + "-x"
}

func (s *Store) ensureInsideRoot(path string) error {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return fmt.Errorf("page path escapes store")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("page path escapes store")
	}
	return nil
}

func clipTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "Untitled"
	}
	if utf8.RuneCountInString(s) > MaxTitle {
		s = string([]rune(s)[:MaxTitle])
	}
	return s
}

func clipBody(s string) string {
	if utf8.RuneCountInString(s) > MaxBody {
		s = string([]rune(s)[:MaxBody])
	}
	return s
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func newID() string {
	return time.Now().UTC().Format("pg-20060102T150405.000000000")
}

func atomicWrite(path string, b []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func hashDraft(title, content, pageID, spaceID string, rev int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n%s\n%d", spaceID, pageID, title, content, rev)))
	return hex.EncodeToString(sum[:])
}
