// 文库是一组可检索的原文：indexed 把文件抄进 learn/raw，linked 让文件留在原处，notes 是内置的笔记库。
// 这里拥有库和文档的增删改、导入和原文字节。导入只登记文档并叫醒索引；解析、切块和检索在 index.go，阅读和批注在 reading.go。
// 读文件、算哈希和抄文件都不持有库锁。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

const (
	// NotesLibrary 是内置笔记库的 id。
	NotesLibrary = "notes"

	// 文库的状态。
	LibraryReady    = "ready"
	LibraryIndexing = "indexing"
	LibraryStale    = "needs_reindex"
	LibraryError    = "error"

	// 文档的状态。pending 是还没解析，needs_ocr 是整份都抽不出文字。
	DocPending  = "pending"
	DocParsed   = "parsed"
	DocFailed   = "failed"
	DocNeedsOCR = "needs_ocr"

	maxImportFiles = 500
	// MaxBlobBytes 是能把原文字节整个交给界面的上限，40 MiB。
	MaxBlobBytes = 40 << 20
)

// IndexProgress 是正在建索引的进度。Phase 是 parsing、embedding 或 publishing。File 是正在处理的文档标题。
type IndexProgress struct {
	Phase string `json:"phase"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	File  string `json:"file"`
}

// Library 是一个可检索的库。Progress 只在建索引时有值。
type Library struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Kind         string         `json:"kind"`
	Description  string         `json:"description"`
	LinkedPath   string         `json:"linked_path"`
	Status       string         `json:"status"`
	Documents    int            `json:"documents"`
	EmbedModel   string         `json:"embed_model"`
	EmbedConn    string         `json:"embed_conn"`
	PublishedGen int            `json:"published_gen"`
	LastError    string         `json:"last_error"`
	Progress     *IndexProgress `json:"progress,omitempty"`
	UpdatedAt    string         `json:"updated_at"`
}

// Document 是库里的一份原文。Kind 是 pdf、markdown、text、html、docx 或 note。Unit 是 page 或 section，PageCount 是单元数。
type Document struct {
	ID        string `json:"id"`
	LibraryID string `json:"library_id"`
	RelPath   string `json:"rel_path"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
	PageCount int    `json:"page_count"`
	Unit      string `json:"unit"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	SourceURL string `json:"source_url"`
	CharCount int    `json:"char_count"`
	UpdatedAt string `json:"updated_at"`
}

// ImportSkip 是一个没导入的路径和原因。
type ImportSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ImportOutcome 是一次导入的结果。Imported 是新登记或内容有变的文档数。
type ImportOutcome struct {
	Imported int          `json:"imported"`
	Skipped  []ImportSkip `json:"skipped"`
}

const libraryCols = `id, name, kind, description, linked_path, status, embed_model, embed_conn, published_gen, last_error, progress_json, updated_at`

// scanLibrary 从一行读出一个库。列顺序和 libraryCols 相同。旧库里的 building 状态读成 indexing。
func scanLibrary(row rower) (Library, error) {
	var lib Library
	var progress string
	err := row.Scan(&lib.ID, &lib.Name, &lib.Kind, &lib.Description, &lib.LinkedPath, &lib.Status, &lib.EmbedModel, &lib.EmbedConn,
		&lib.PublishedGen, &lib.LastError, &progress, &lib.UpdatedAt)
	if err != nil {
		return Library{}, err
	}
	if lib.Status == "building" {
		lib.Status = LibraryIndexing
	}
	if lib.Status == LibraryIndexing && progress != "" {
		var p IndexProgress
		if json.Unmarshal([]byte(progress), &p) == nil {
			lib.Progress = &p
		}
	}
	return lib, nil
}

// library 用 q 读出一个库。id 不存在时返回 library not found。调用方要持有 s.mu。
func (s *Store) library(id string) (Library, error) {
	lib, err := scanLibrary(s.db.QueryRow(`SELECT `+libraryCols+` FROM libraries WHERE id = ?`, id))
	if err != nil {
		return Library{}, fmt.Errorf("library not found")
	}
	return lib, nil
}

// ensureNotesLibrary 保证 id 为 notes 的库存在。它索引笔记，不接收导入文件。
func (s *Store) ensureNotesLibrary() error {
	now := formatTime(s.clock())
	_, err := s.db.Exec(`INSERT OR IGNORE INTO libraries
		(id, name, kind, status, created_at, updated_at) VALUES ('notes', '笔记', 'notes', 'ready', ?, ?)`, now, now)
	return err
}

// Libraries 列出全部库，并带上各自的文档数。笔记库排在最前，其余按名字排。
func (s *Store) Libraries() []Library {
	out := []Library{}
	if s == nil {
		return out
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT ` + libraryCols + ` FROM libraries ORDER BY CASE WHEN id = 'notes' THEN 0 ELSE 1 END, name`)
	if err != nil {
		return out
	}
	for rows.Next() {
		if lib, err := scanLibrary(rows); err == nil {
			out = append(out, lib)
		}
	}
	rows.Close()
	for i := range out {
		if out[i].Kind == NotesLibrary {
			_ = s.db.QueryRow(`SELECT COUNT(1) FROM notes`).Scan(&out[i].Documents)
			continue
		}
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM documents WHERE library_id = ? AND removed = 0`, out[i].ID).Scan(&out[i].Documents)
	}
	return out
}

// Library 读出一个库和它的文档数。id 不存在时返回 library not found。
func (s *Store) Library(id string) (Library, error) {
	if s == nil {
		return Library{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lib, err := s.library(id)
	if err != nil {
		return Library{}, err
	}
	if lib.Kind == NotesLibrary {
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM notes`).Scan(&lib.Documents)
	} else {
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM documents WHERE library_id = ? AND removed = 0`, id).Scan(&lib.Documents)
	}
	return lib, nil
}

// CreateLibrary 新建一个 indexed 或 linked 库。name 去掉空白后不能空，最长 80 个符文。
// kind 不是 linked 时写成 indexed。linked 必须带 linkedPath，建好后立刻扫一遍那个文件夹并开始建索引。
func (s *Store) CreateLibrary(name, kind, linkedPath string) (Library, error) {
	if s == nil {
		return Library{}, fmt.Errorf("learn store unavailable")
	}
	name = clipRunes(strings.TrimSpace(name), 80)
	if name == "" {
		return Library{}, fmt.Errorf("library name required")
	}
	kind = strings.TrimSpace(kind)
	if kind != "linked" {
		kind = "indexed"
	}
	linkedPath = strings.TrimSpace(linkedPath)
	if kind == "linked" {
		if linkedPath == "" {
			return Library{}, fmt.Errorf("linked path required")
		}
		if st, err := os.Stat(linkedPath); err != nil || !st.IsDir() {
			return Library{}, fmt.Errorf("the linked folder could not be opened")
		}
	} else {
		linkedPath = ""
	}
	now := formatTime(s.clock())
	id := newKey("lib")
	s.mu.Lock()
	_, err := s.db.Exec(`INSERT INTO libraries (id, name, kind, linked_path, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'ready', ?, ?)`,
		id, name, kind, linkedPath, now, now)
	s.mu.Unlock()
	if err != nil {
		return Library{}, err
	}
	if kind == "linked" {
		if _, err := s.syncLinked(id, true); err != nil {
			return Library{}, err
		}
	}
	s.notify("library", id, "created")
	return s.Library(id)
}

// UpdateLibrary 改库的名字或说明。指针为 nil 的项不动。名字去掉空白后不能空。笔记库的名字不能改。
func (s *Store) UpdateLibrary(id string, name, description *string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	lib, err := s.library(id)
	if err == nil && name != nil {
		lib.Name = clipRunes(strings.TrimSpace(*name), 80)
		if lib.Name == "" {
			err = fmt.Errorf("library name required")
		} else if id == NotesLibrary {
			err = fmt.Errorf("the notes library keeps its name")
		}
	}
	if err == nil {
		if description != nil {
			lib.Description = clipRunes(strings.TrimSpace(*description), 600)
		}
		_, err = s.db.Exec(`UPDATE libraries SET name = ?, description = ?, updated_at = ? WHERE id = ?`, lib.Name, lib.Description, formatTime(s.clock()), id)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("library", id, "updated")
	return nil
}

// DeleteLibrary 删除库及其文档、索引、阅读材料和批注。抄进 raw 的原文字节留着，别的库可能还在用。id 为 notes 时拒绝。
func (s *Store) DeleteLibrary(id string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	if id == NotesLibrary {
		return fmt.Errorf("the notes library stays")
	}
	s.mu.Lock()
	_, err := s.library(id)
	if err == nil {
		err = s.inTx(func(tx *sql.Tx) error {
			return execArgs(tx, id,
				`DELETE FROM chunks_fts WHERE library_id = ?`,
				`DELETE FROM embeddings WHERE chunk_id IN (SELECT id FROM chunks WHERE library_id = ?)`,
				`DELETE FROM chunks WHERE library_id = ?`,
				`DELETE FROM annotations WHERE material_id IN (SELECT id FROM materials WHERE library_id = ?)`,
				`DELETE FROM bookmarks WHERE material_id IN (SELECT id FROM materials WHERE library_id = ?)`,
				`DELETE FROM reading_positions WHERE material_id IN (SELECT id FROM materials WHERE library_id = ?)`,
				`DELETE FROM materials WHERE library_id = ?`,
				`DELETE FROM documents WHERE library_id = ?`,
				`DELETE FROM generations WHERE library_id = ?`,
				`DELETE FROM index_journal WHERE library_id = ?`,
				`DELETE FROM libraries WHERE id = ?`,
			)
		})
	}
	delete(s.vectors, id)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("library", id, "deleted")
	return nil
}

// SetEmbedModel 记下这个库使用的嵌入模型并重新建索引。model 为空表示只做词法检索。
func (s *Store) SetEmbedModel(id, model string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	_, err := s.library(id)
	if err == nil {
		_, err = s.db.Exec(`UPDATE libraries SET embed_model = ?, updated_at = ? WHERE id = ?`, strings.TrimSpace(model), formatTime(s.clock()), id)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.scheduleIndex(id)
	return nil
}

const documentCols = `id, library_id, rel_path, title, sha256, bytes, page_count, unit, status, error, source_url, char_count, updated_at`

// scanDocument 从一行读出一份文档。列顺序和 documentCols 相同。Kind 按文件名算，不信旧库里存的值。
func scanDocument(row rower) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.LibraryID, &d.RelPath, &d.Title, &d.SHA256, &d.Bytes, &d.PageCount, &d.Unit, &d.Status, &d.Error, &d.SourceURL, &d.CharCount, &d.UpdatedAt)
	if err != nil {
		return Document{}, err
	}
	d.Kind = docKind(d.RelPath)
	if d.Kind != kindPDF {
		d.Unit = UnitSection
	}
	return d, nil
}

// document 读出库里一份没删的文档。对不上时返回 document not found。调用方要持有 s.mu。
func (s *Store) document(libraryID, documentID string) (Document, error) {
	d, err := scanDocument(s.db.QueryRow(`SELECT `+documentCols+` FROM documents WHERE id = ? AND library_id = ? AND removed = 0`, documentID, libraryID))
	if err != nil {
		return Document{}, fmt.Errorf("document not found")
	}
	return d, nil
}

// Documents 列出一个库里没删的文档，按路径排。libraryID 必须存在。笔记库列出各篇笔记。
func (s *Store) Documents(libraryID string) ([]Document, error) {
	if s == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lib, err := s.library(libraryID)
	if err != nil {
		return nil, err
	}
	out := []Document{}
	if lib.Kind == NotesLibrary {
		rows, err := s.db.Query(`SELECT id, title, LENGTH(content), updated_at FROM notes ORDER BY updated_at DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			d := Document{LibraryID: NotesLibrary, Kind: kindNote, Unit: UnitSection, Status: DocParsed}
			if err := rows.Scan(&d.ID, &d.Title, &d.Bytes, &d.UpdatedAt); err == nil {
				d.RelPath = d.ID
				out = append(out, d)
			}
		}
		return out, rows.Err()
	}
	rows, err := s.db.Query(`SELECT `+documentCols+` FROM documents WHERE library_id = ? AND removed = 0 ORDER BY rel_path`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if d, err := scanDocument(rows); err == nil {
			out = append(out, d)
		}
	}
	return out, rows.Err()
}

// importFile 是准备登记的一个文件：它在库里的相对路径、标题、字节哈希、大小，以及留在原处时的绝对路径。
type importFile struct {
	rel, title, sum, external, sourceURL string
	size                                 int64
}

// Import 把一个文件或目录导入库，返回登记的文档数。它是 ImportPaths 的单路径写法。
func (s *Store) Import(libraryID, path string) (int, error) {
	out, err := s.ImportPaths(libraryID, []string{path})
	if err != nil {
		return 0, err
	}
	if out.Imported == 0 && len(out.Skipped) > 0 {
		return 0, fmt.Errorf("%s", out.Skipped[0].Reason)
	}
	return out.Imported, nil
}

// ImportPaths 把一组文件或目录导入库，登记成待解析的文档，然后在后台建索引。目录里最多收 500 个支持的文件。
// indexed 库把文件按内容哈希抄进 learn/raw；linked 库只记下路径。同一路径再导一次时内容没变就不算导入。
// 不支持的类型、超过 100 MiB 的文件和读不了的路径记在 Skipped 里，不让整批失败。笔记库拒绝导入。
func (s *Store) ImportPaths(libraryID string, paths []string) (ImportOutcome, error) {
	out := ImportOutcome{Skipped: []ImportSkip{}}
	if s == nil {
		return out, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	lib, err := s.library(libraryID)
	s.mu.Unlock()
	if err != nil {
		return out, err
	}
	if lib.Kind == NotesLibrary {
		return out, fmt.Errorf("import files into another library")
	}
	var files []importFile
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		found, skipped := s.collectFiles(lib, path, maxImportFiles-len(files))
		files = append(files, found...)
		out.Skipped = append(out.Skipped, skipped...)
	}
	if len(files) == 0 && len(out.Skipped) == 0 {
		return out, fmt.Errorf("path required")
	}
	n, err := s.registerFiles(libraryID, files)
	if err != nil {
		return out, err
	}
	out.Imported = n
	if n > 0 {
		s.scheduleIndex(libraryID)
	}
	return out, nil
}

// collectFiles 把 path 展开成要登记的文件，并算好哈希；indexed 库顺手把字节抄进 raw。最多收 room 个。
// path 是文件时相对路径是文件名，是目录时是相对那个目录的路径。读不了或不支持的记进 skipped。这里不碰数据库。
func (s *Store) collectFiles(lib Library, path string, room int) (files []importFile, skipped []ImportSkip) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, []ImportSkip{{Path: path, Reason: "this path could not be opened"}}
	}
	var list []string
	if st.IsDir() {
		_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil || d == nil {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") && p != path {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() && supportedExt(p) {
				list = append(list, p)
			}
			return nil
		})
	} else if supportedExt(path) {
		list = []string{path}
	} else {
		return nil, []ImportSkip{{Path: path, Reason: "unsupported file type; use pdf, md, txt, html, or docx"}}
	}
	for _, file := range list {
		if len(files) >= room {
			skipped = append(skipped, ImportSkip{Path: file, Reason: fmt.Sprintf("a library takes at most %d files per import", maxImportFiles)})
			break
		}
		info, err := os.Stat(file)
		if err != nil {
			skipped = append(skipped, ImportSkip{Path: file, Reason: "this file could not be opened"})
			continue
		}
		if info.Size() > MaxImportBytes {
			skipped = append(skipped, ImportSkip{Path: file, Reason: "larger than 100 MiB"})
			continue
		}
		sum, err := hashFile(file)
		if err != nil {
			skipped = append(skipped, ImportSkip{Path: file, Reason: "this file could not be read"})
			continue
		}
		rel := filepath.Base(file)
		if st.IsDir() {
			if r, err := filepath.Rel(path, file); err == nil {
				rel = filepath.ToSlash(r)
			}
		}
		item := importFile{rel: rel, title: strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), sum: sum, size: info.Size()}
		if lib.Kind == "linked" {
			item.external = file
		} else if err := s.keepRaw(file, sum); err != nil {
			skipped = append(skipped, ImportSkip{Path: file, Reason: "this file could not be copied into the library"})
			continue
		}
		files = append(files, item)
	}
	return files, skipped
}

// rawPath 返回哈希为 sum、扩展名为 ext 的原文在 learn/raw 里的位置。
func (s *Store) rawPath(sum, ext string) string {
	return filepath.Join(s.root, "raw", sum+strings.ToLower(ext))
}

// keepRaw 把 file 抄进 learn/raw，文件名是内容哈希。已经在那里时不再抄。
func (s *Store) keepRaw(file, sum string) error {
	dest := s.rawPath(sum, filepath.Ext(file))
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	return copyFile(file, dest)
}

// registerFiles 把一组文件登记成文档，返回新登记或内容有变的个数。内容有变的文档回到待解析。
func (s *Store) registerFiles(libraryID string, files []importFile) (int, error) {
	if len(files) == 0 {
		return 0, nil
	}
	now := formatTime(s.clock())
	n := 0
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.inTx(func(tx *sql.Tx) error {
		for _, f := range files {
			var id, sum string
			var removed int
			err := tx.QueryRow(`SELECT id, sha256, removed FROM documents WHERE library_id = ? AND rel_path = ?`, libraryID, f.rel).Scan(&id, &sum, &removed)
			switch {
			case err == sql.ErrNoRows:
				if _, err := tx.Exec(`INSERT INTO documents
					(id, library_id, rel_path, title, sha256, bytes, kind, external_path, status, source_url, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)`,
					newKey("doc"), libraryID, f.rel, f.title, f.sum, f.size, docKind(f.rel), f.external, f.sourceURL, now, now); err != nil {
					return err
				}
				n++
			case err != nil:
				return err
			case sum == f.sum && removed == 0:
				// 内容没变。留在原处的文件挪了地方时把路径跟上。
				if _, err := tx.Exec(`UPDATE documents SET external_path = ? WHERE id = ?`, f.external, id); err != nil {
					return err
				}
			default:
				if _, err := tx.Exec(`UPDATE documents SET sha256 = ?, bytes = ?, external_path = ?, removed = 0, status = 'pending', error = '', updated_at = ? WHERE id = ?`,
					f.sum, f.size, f.external, now, id); err != nil {
					return err
				}
				n++
			}
		}
		return nil
	})
	return n, err
}

// ImportText 把一段文字存成库里的一份 Markdown 文档，例如从网页抓下来的正文。title 不能空，text 不能空。
// sourceURL 记下它从哪里来，可空。同一个标题再存一次会换掉原来的内容。
func (s *Store) ImportText(libraryID, title, text, sourceURL string) (Document, error) {
	if s == nil {
		return Document{}, fmt.Errorf("learn store unavailable")
	}
	title = clipRunes(strings.TrimSpace(title), 160)
	text = normalizeText(text)
	if title == "" || strings.TrimSpace(text) == "" {
		return Document{}, fmt.Errorf("a title and some text are required")
	}
	if len(text) > MaxImportBytes {
		return Document{}, fmt.Errorf("larger than 100 MiB")
	}
	s.mu.Lock()
	lib, err := s.library(libraryID)
	s.mu.Unlock()
	if err != nil {
		return Document{}, err
	}
	if lib.Kind != "indexed" {
		return Document{}, fmt.Errorf("text can only be saved into an indexed library")
	}
	sum := hashText(text)
	dest := s.rawPath(sum, ".md")
	if _, err := os.Stat(dest); err != nil {
		if err := os.WriteFile(dest, []byte(text), 0o644); err != nil {
			return Document{}, err
		}
	}
	rel := sanitizeName(title) + ".md"
	if _, err := s.registerFiles(libraryID, []importFile{{rel: rel, title: title, sum: sum, size: int64(len(text)), sourceURL: strings.TrimSpace(sourceURL)}}); err != nil {
		return Document{}, err
	}
	s.mu.Lock()
	var id string
	_ = s.db.QueryRow(`SELECT id FROM documents WHERE library_id = ? AND rel_path = ?`, libraryID, rel).Scan(&id)
	_, _ = s.db.Exec(`UPDATE documents SET title = ?, source_url = ? WHERE id = ?`, title, strings.TrimSpace(sourceURL), id)
	d, err := s.document(libraryID, id)
	s.mu.Unlock()
	if err != nil {
		return Document{}, err
	}
	s.scheduleIndex(libraryID)
	return d, nil
}

// sanitizeName 把标题收成能当文件名的样子：路径分隔符和控制字符换成空格，最长 80 个符文。收完是空的就叫 untitled。
func sanitizeName(title string) string {
	clean := strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`\/:*?"<>|`, r) {
			return ' '
		}
		return r
	}, title)
	clean = clipRunes(strings.Join(strings.Fields(clean), " "), 80)
	if clean == "" {
		return "untitled"
	}
	return clean
}

// syncLinked 把 linked 库对着它的文件夹扫一遍：新文件和内容变了的文件登记成待解析，文件夹里已经没有的文档标成删除。
// 返回有没有变化。schedule 为真且有变化时叫醒索引。不是 linked 库时什么都不做。
func (s *Store) syncLinked(libraryID string, schedule bool) (bool, error) {
	s.mu.Lock()
	lib, err := s.library(libraryID)
	s.mu.Unlock()
	if err != nil {
		return false, err
	}
	if lib.Kind != "linked" || lib.LinkedPath == "" {
		return false, nil
	}
	if st, err := os.Stat(lib.LinkedPath); err != nil || !st.IsDir() {
		return false, fmt.Errorf("the linked folder could not be opened")
	}
	files, _ := s.collectFiles(lib, lib.LinkedPath, maxImportFiles)
	n, err := s.registerFiles(libraryID, files)
	if err != nil {
		return false, err
	}
	present := map[string]bool{}
	for _, f := range files {
		present[f.rel] = true
	}
	gone := 0
	s.mu.Lock()
	rows, err := s.db.Query(`SELECT id, rel_path FROM documents WHERE library_id = ? AND removed = 0`, libraryID)
	var missing []string
	if err == nil {
		for rows.Next() {
			var id, rel string
			if rows.Scan(&id, &rel) == nil && !present[rel] {
				missing = append(missing, id)
			}
		}
		rows.Close()
	}
	for _, id := range missing {
		if _, err := s.db.Exec(`UPDATE documents SET removed = 1, status = 'removed' WHERE id = ?`, id); err == nil {
			gone++
		}
	}
	s.mu.Unlock()
	changed := n > 0 || gone > 0
	if changed && schedule {
		s.scheduleIndex(libraryID)
	}
	return changed, nil
}

// RemoveDocument 把一份原文从库里拿掉，并重新建索引。它上面的阅读材料和批注留着，文档再导回来时还在。
func (s *Store) RemoveDocument(libraryID, documentID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	res, err := s.db.Exec(`UPDATE documents SET removed = 1, status = 'removed', updated_at = ? WHERE id = ? AND library_id = ? AND removed = 0`,
		formatTime(s.clock()), documentID, libraryID)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("document not found")
	}
	s.scheduleIndex(libraryID)
	return nil
}

// RenameDocument 改一份文档显示的标题。title 去掉空白后不能空，最长 160 个符文。文件本身不动。
func (s *Store) RenameDocument(libraryID, documentID, title string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	title = clipRunes(strings.TrimSpace(title), 160)
	if title == "" {
		return fmt.Errorf("a title is required")
	}
	s.mu.Lock()
	res, err := s.db.Exec(`UPDATE documents SET title = ?, updated_at = ? WHERE id = ? AND library_id = ? AND removed = 0`, title, formatTime(s.clock()), documentID, libraryID)
	if err == nil {
		_, _ = s.db.Exec(`UPDATE materials SET title = ? WHERE library_id = ? AND document_id = ?`, title, libraryID, documentID)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("document not found")
	}
	s.notify("library", libraryID, "updated")
	return nil
}

// docPath 返回一份文档的原文在磁盘上的位置。linked 库是它原来的路径，indexed 库在 learn/raw 里。调用方要持有 s.mu。
func (s *Store) docPath(libraryID, docID string) string {
	var sum, external, rel string
	_ = s.db.QueryRow(`SELECT sha256, external_path, rel_path FROM documents WHERE id = ? AND library_id = ?`, docID, libraryID).Scan(&sum, &external, &rel)
	if external != "" {
		return external
	}
	if sum == "" {
		return ""
	}
	return s.rawPath(sum, filepath.Ext(rel))
}

// DocumentBlob 读出一份文档的原文字节，给界面按原样显示用。超过 40 MiB 时返回错误。笔记没有原文字节。
func (s *Store) DocumentBlob(libraryID, documentID string) (name, mimeType string, data []byte, err error) {
	if s == nil {
		return "", "", nil, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	d, err := s.document(libraryID, documentID)
	path := ""
	if err == nil {
		path = s.docPath(libraryID, documentID)
	}
	s.mu.Unlock()
	if err != nil {
		return "", "", nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", "", nil, fmt.Errorf("the original file is missing")
	}
	if st.Size() > MaxBlobBytes {
		return "", "", nil, fmt.Errorf("the original file is larger than 40 MiB")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return "", "", nil, fmt.Errorf("the original file could not be read")
	}
	mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(d.RelPath)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return filepath.Base(d.RelPath), mimeType, data, nil
}

// FileNames 返回库里最多 limit 个文档标题，给提示用。limit 小于 1 或大于 20 时用 20。笔记库返回笔记标题。
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
	if libraryID == NotesLibrary {
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

// copyFile 把 src 抄到 dst。先写到临时文件再改名，抄到一半断掉不会留下半个文件。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
