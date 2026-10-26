// Package pages 在 Open 收到的根目录下按空间保存 Markdown。默认空间目录名是 home，文件是 root/home/<id>.md。
// 桌面应用把 root 设成主目录的 pages/，不是编码工作区。
// Create 和 Save 直接写盘。Propose 只记录待审阅。Decide 在 approve 为真时才写入，并把 expected revision 传 0。
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
	// DefaultSpace 是空间名为空时使用的目录名，值为 home。这是存储根下的子目录，不是主目录本身。
	DefaultSpace = "home"
	// MaxTitle 是标题的符文上限，值为 160。
	MaxTitle = 160
	// MaxBody 是正文的符文上限，值为 100000。
	MaxBody = 100_000
)

// Page 是一篇 Markdown。Content 不进 YAML 头，写在正文区。
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

// Meta 是索引和列表用的摘要，Preview 来自正文。
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

// ConflictError 表示保存时的期望修订号和磁盘上的不一致。Page 是磁盘上的当前页。
type ConflictError struct {
	Page Page
}

// Error 返回 page revision conflict。
func (e *ConflictError) Error() string { return "page revision conflict" }

// Store 的 root 是页面根目录。桌面应用传入主目录的 pages/。
type Store struct {
	mu   sync.Mutex
	root string
}

// Open 按 0o755 创建 root 和 root/home，并重建 index.json。
func Open(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, DefaultSpace), 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root}
	_ = s.rebuildIndex()
	return s, nil
}

// Root 返回构造时的根目录。
func (s *Store) Root() string { return s.root }

// List 从 index.json 列出摘要，可按空间过滤。更新时间新的在前，相同则按标题。
// spaceID 交给 normalizeSpace。非空时只留下 SpaceID 与它相等的摘要。
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

// Search 在查询为空时调用 List。否则遍历根下的 .md，在标题和正文里做子串匹配。
// q 去掉空白并转成小写。为空时和 spaceID 一起传给 List。否则在标题、换行和正文拼成的小写文本里做包含判断。
// spaceID 交给 normalizeSpace。q 为空时传给 List。非空时页面的 SpaceID 必须与它相等。
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

// Get 按空间和 id 读取一篇页面。找不到返回 page not found。
// spaceID 传给 getLocked。
func (s *Store) Get(spaceID, id string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(spaceID, id)
}

// Delete 删除页面文件。子页面的父 id 改成被删页面的父 id。
// spaceID 传给 deleteLocked。
func (s *Store) Delete(spaceID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteLocked(spaceID, id)
}

// deleteLocked 在已持有锁时删除，并把直接子页重新挂到原父页下。
// spaceID 传给 getLocked。
// id 传给 getLocked。
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

// Create 在锁内创建页面。它直接写盘，不经过审阅。
// p 传给 createLocked。
func (s *Store) Create(p Page) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createLocked(p)
}

// createLocked 规范化空间和 id。id 为空则生成。ParentID 为 - 或 root 时清空。修订号小于 1 时写成 1。
// p 的 SpaceID、ID、ParentID 先规范化。Title 交给 clipTitle，Content 交给 clipBody。ID 为空时用 uniqueID。CreatedAt 为零时写成当前 UTC。Revision 小于 1 时写成 1。然后传给 writePage。
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

// Save 在锁内按期望修订号保存。expectedRevision 大于 0 且与当前不符时返回 ConflictError。它直接写盘。
func (s *Store) Save(p Page, expectedRevision int) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(p, expectedRevision)
}

// saveLocked 把修订号加 1。ParentID 为空则保留原来的父页，为 - 或 root 则清空。id 与父 id 相同则不改父页。
// p 的 SpaceID 和 ID 传给 getLocked。Title、Content、ParentID、SourceSession 与当前页合并后，Revision 写成当前值加 1，再传给 writePage。
// expectedRevision 大于 0 且与当前 Revision 不同时返回 ConflictError。
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

// ApplyApproved 在锁内按已批准的字段创建或保存。页面不存在时改为创建。
// spaceID 传给 applyApprovedLocked。
// pageID 传给 applyApprovedLocked。
// parentID 传给 applyApprovedLocked。
// title 传给 applyApprovedLocked。
// content 传给 applyApprovedLocked。
// sourceSession 传给 applyApprovedLocked。
// expectedRevision 传给 applyApprovedLocked。
func (s *Store) ApplyApproved(spaceID, pageID, parentID, title, content, sourceSession string, expectedRevision int) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applyApprovedLocked(spaceID, pageID, parentID, title, content, sourceSession, expectedRevision)
}

// applyApprovedLocked 在 pageID 为空或页面不存在时创建。否则更新标题、正文和可选的父页、会话，再 saveLocked。
// spaceID 交给 normalizeSpace，写入草稿，并在更新时传给 getLocked。
// parentID 交给 normalizePageID。非空时在更新分支写入 ParentID。
// title 交给 clipTitle，写入草稿或当前页。
// content 交给 clipBody，写入草稿或当前页。
// sourceSession 写入草稿。非空时在更新分支写入 SourceSession。
// expectedRevision 在更新分支传给 saveLocked。
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

// RebuildIndex 重建 index.json。遍历错误被忽略，函数本身返回 nil。
func (s *Store) RebuildIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.rebuildIndexLocked()
	return nil
}

// getLocked 先找该空间和默认空间下的文件，再走遍根目录里同名 .md。id 为空返回 page not found。
// spaceID 交给 normalizeSpace，用来拼首选路径。不是 DefaultSpace 时再加一条默认空间的路径。
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

// writePage 确认路径在根内，再按 0o644 原子写入 Markdown。
// p 的 SpaceID 和 ID 传给 pagePath，整页传给 encodePage。
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

// pagePath 返回 root/<空间>/<id>.md。
// spaceID 交给 normalizeSpace，再和 id 加 .md 拼到 root 下。
func (s *Store) pagePath(spaceID, id string) string {
	return filepath.Join(s.root, normalizeSpace(spaceID), id+".md")
}

// loadIndex 读取 index.json。失败或 JSON 无效时重建。
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

// rebuildIndex 在锁内重建索引。
func (s *Store) rebuildIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.rebuildIndexLocked()
	return nil
}

// rebuildIndexLocked 遍历 .md 生成摘要，按更新时间排序，写入 index.json。
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

// frontmatter 是 Markdown 文件 YAML 头里的字段，不含正文。
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

// encodePage 把元数据写成 YAML 头，正文接在第二个 --- 之后。
// p 的 ID、SpaceID、ParentID、Title、Revision、SourceSession、CreatedAt、UpdatedAt 写入 YAML 头，Content 写在第二个 --- 之后。Content 非空且不以换行结尾时再补一个换行。
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

// readPageFile 读取路径并解码成 Page。
// path 交给 os.ReadFile，读到的字节传给 decodePage。
func readPageFile(path string) (Page, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Page{}, err
	}
	return decodePage(b)
}

// decodePage 要求以 --- 开头的 YAML 头。空间为空时用 DefaultSpace。修订号小于 1 时写成 1。
// raw 转成字符串。不是 --- 加换行开头时返回 missing frontmatter。没有结束标记或 YAML 解析失败时返回错误。
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

// metaOf 从页面抽出列表摘要，并生成预览。
// p 的各元数据字段写入摘要，Content 传给 previewOf。
func metaOf(p Page) Meta {
	return Meta{
		ID: p.ID, SpaceID: p.SpaceID, ParentID: p.ParentID, Title: p.Title,
		Revision: p.Revision, UpdatedAt: p.UpdatedAt, CreatedAt: p.CreatedAt, SourceSession: p.SourceSession,
		Preview: previewOf(p.Content),
	}
}

// previewOf 把空白折叠成单行，超过 160 个符文则截断。
// content 折叠空白。rune 数大于 160 时截到 160。
func previewOf(content string) string {
	s := strings.Join(strings.Fields(content), " ")
	n := utf8.RuneCountInString(s)
	if n <= 160 {
		return s
	}
	return string([]rune(s)[:160])
}

// normalizeSpace 清掉非法字符。结果为空时返回 DefaultSpace。
// id 以 allowDot 为 false 传给 sanitizeSegment。结果为空时返回 DefaultSpace。
func normalizeSpace(id string) string {
	id = sanitizeSegment(id, false)
	if id == "" {
		return DefaultSpace
	}
	return id
}

// normalizePageID 清掉非法字符，并删掉剩余的 ..。
// id 以 allowDot 为 true 传给 sanitizeSegment，再删掉剩余的 ..。
func normalizePageID(id string) string {
	id = sanitizeSegment(id, true)
	for strings.Contains(id, "..") {
		id = strings.ReplaceAll(id, "..", "")
	}
	return id
}

// sanitizeSegment 只保留字母、数字、减号和下划线。allowDot 为真时还保留点。
// id 去掉首尾空白后逐个 rune 过滤。
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

// uniqueID 用当前 UTC 时间生成 id，碰撞时加后缀，最多试 8 次。
// spaceID 传给 pagePath。该路径 Stat 失败时返回这次生成的 id。
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

// ensureInsideRoot 用绝对路径判断目标是否在 root 之下。相对结果是 .. 或以 .. 加分隔符开头则返回错误。
// path 转成绝对路径，再和 root 的绝对路径算相对路径。
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

// clipTitle 去掉空白。空标题变成 Untitled。超过 MaxTitle 个符文则截断。
// s 去掉首尾空白。为空时写成 Untitled。rune 数大于 MaxTitle 时截到 MaxTitle。
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

// clipBody 在正文超过 MaxBody 个符文时截断。
// s 的 rune 数大于 MaxBody 时截到 MaxBody。
func clipBody(s string) string {
	if utf8.RuneCountInString(s) > MaxBody {
		s = string([]rune(s)[:MaxBody])
	}
	return s
}

// firstNonEmpty 在 a 去掉空白后非空时返回 a，否则返回 b。
func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// newID 用当前 UTC 时间生成 pg- 前缀的 id。
func newID() string {
	return time.Now().UTC().Format("pg-20060102T150405.000000000")
}

// atomicWrite 先写 path.tmp，再改名替换 path。
// b 写入 path 加 .tmp。
// mode 作为该临时文件的权限传给 os.WriteFile。
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// hashDraft 对空间、页面 id、标题、正文和修订号做 SHA-256，返回十六进制。
// title 按 spaceID、pageID、title、content、rev 的顺序用换行拼进待哈希文本。
// content 按同样顺序拼进待哈希文本。
// pageID 按同样顺序拼进待哈希文本。
// spaceID 放在待哈希文本的第一段。
// rev 以十进制写在最后一段。
func hashDraft(title, content, pageID, spaceID string, rev int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n%s\n%d", spaceID, pageID, title, content, rev)))
	return hex.EncodeToString(sum[:])
}
