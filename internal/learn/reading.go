// 阅读台把一份文档收成材料。读到哪里、书签和批注都挂在材料上，文档从库里拿掉再导回来它们还在。
// 批注用引文加它在单元文字里的位置来锚定，另存前后各一小段文字；原文挪动之后靠这些重新找到它，找不到就标成未定位，不丢。
// 这里不解析文件也不建索引，读正文走 index.go 的 ReadUnit。

package learn

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

const (
	// 批注的三种样子。
	AnnotHighlight = "highlight"
	AnnotUnderline = "underline"
	AnnotNote      = "note"

	// 批注锚点的两种状态：在原文里找得到，或者暂时找不到。
	AnchorResolved   = "resolved"
	AnchorUnresolved = "unresolved"

	anchorContext = 32
	maxQuoteRunes = 2000
	maxNoteRunes  = 4000
	maxLocateHits = 20
)

var annotColors = map[string]bool{"yellow": true, "green": true, "blue": true, "pink": true, "purple": true}

// Material 是阅读台上的一份材料。Kind 是文档格式，Unit 是 page 或 section，PageCount 是单元数。
// Locator 和 Percent 是上次读到的位置，Annotations 是批注条数。
type Material struct {
	ID           string  `json:"id"`
	LibraryID    string  `json:"library_id"`
	DocumentID   string  `json:"document_id"`
	Title        string  `json:"title"`
	Kind         string  `json:"kind"`
	Unit         string  `json:"unit"`
	PageCount    int     `json:"page_count"`
	LastOpenedAt string  `json:"last_opened_at"`
	Locator      int     `json:"locator"`
	Percent      float64 `json:"percent"`
	Annotations  int     `json:"annotations"`
}

// Annotation 是材料上的一条批注。Start 和 End 是引文在单元文字里的符文位置，找不到时都是 -1。
// Author 是 user 或 assistant。Resolution 说明锚点现在找不找得到。
type Annotation struct {
	ID         string `json:"id"`
	MaterialID string `json:"material_id"`
	Kind       string `json:"kind"`
	Color      string `json:"color"`
	Locator    int    `json:"locator"`
	Quote      string `json:"quote"`
	Note       string `json:"note"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Prefix     string `json:"prefix"`
	Suffix     string `json:"suffix"`
	Author     string `json:"author"`
	Resolution string `json:"resolution"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// Bookmark 是材料里的一个书签。一个定位号上最多一个。
type Bookmark struct {
	ID         string `json:"id"`
	MaterialID string `json:"material_id"`
	Locator    int    `json:"locator"`
	Label      string `json:"label"`
	CreatedAt  string `json:"created_at"`
}

// Position 是读到的位置。Percent 是 0 到 1 之间的整体进度。
type Position struct {
	Locator int     `json:"locator"`
	Percent float64 `json:"percent"`
}

// MaterialView 是打开一份材料时界面需要的全部东西。
type MaterialView struct {
	Material    Material       `json:"material"`
	Document    Document       `json:"document"`
	Outline     []OutlineEntry `json:"outline"`
	Position    Position       `json:"position"`
	Bookmarks   []Bookmark     `json:"bookmarks"`
	Annotations []Annotation   `json:"annotations"`
}

const materialCols = `m.id, m.library_id, m.document_id, m.title, m.last_opened_at, m.created_at,
	COALESCE(p.locator, 1), COALESCE(p.percent, 0),
	(SELECT COUNT(1) FROM annotations a WHERE a.material_id = m.id)`

const materialFrom = ` FROM materials m LEFT JOIN reading_positions p ON p.material_id = m.id `

// scanMaterial 从一行读出一份材料，并按它的文档补上格式、单元种类和单元数。列顺序和 materialCols 相同。调用方要持有 s.mu。
func (s *Store) scanMaterial(row rower) (Material, error) {
	var m Material
	var created string
	if err := row.Scan(&m.ID, &m.LibraryID, &m.DocumentID, &m.Title, &m.LastOpenedAt, &created, &m.Locator, &m.Percent, &m.Annotations); err != nil {
		return Material{}, err
	}
	if m.LastOpenedAt == "" {
		m.LastOpenedAt = created
	}
	m.Kind, m.Unit, m.PageCount = kindNote, UnitSection, 1
	if m.LibraryID != NotesLibrary {
		if d, err := s.document(m.LibraryID, m.DocumentID); err == nil {
			m.Kind, m.Unit, m.PageCount = d.Kind, d.Unit, d.PageCount
		}
	}
	return m, nil
}

// OpenMaterial 把库里的一份文档放到阅读台，并返回界面要的全部东西。同库同文档已有材料时接着用原来的，并记下这次打开的时刻。
// libraryID 和 documentID 必须对得上一份没删的文档或一篇笔记。
func (s *Store) OpenMaterial(libraryID, documentID string) (MaterialView, error) {
	if s == nil {
		return MaterialView{}, fmt.Errorf("learn store unavailable")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	var title string
	var err error
	if libraryID == NotesLibrary {
		err = s.db.QueryRow(`SELECT title FROM notes WHERE id = ?`, documentID).Scan(&title)
	} else {
		err = s.db.QueryRow(`SELECT title FROM documents WHERE id = ? AND library_id = ? AND removed = 0`, documentID, libraryID).Scan(&title)
	}
	if err != nil {
		s.mu.Unlock()
		return MaterialView{}, fmt.Errorf("document not found")
	}
	var id string
	if s.db.QueryRow(`SELECT id FROM materials WHERE library_id = ? AND document_id = ?`, libraryID, documentID).Scan(&id) == nil {
		_, err = s.db.Exec(`UPDATE materials SET last_opened_at = ?, title = ? WHERE id = ?`, now, title, id)
	} else {
		id = newKey("mat")
		_, err = s.db.Exec(`INSERT INTO materials (id, library_id, document_id, kind, title, created_at, last_opened_at) VALUES (?, ?, ?, 'doc', ?, ?, ?)`,
			id, libraryID, documentID, title, now, now)
	}
	s.mu.Unlock()
	if err != nil {
		return MaterialView{}, err
	}
	return s.MaterialView(id)
}

// MaterialView 读出一份材料现在的样子：材料、文档、提纲、位置、书签和批注。materialID 不存在时返回 material not found。
func (s *Store) MaterialView(materialID string) (MaterialView, error) {
	if s == nil {
		return MaterialView{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	m, err := s.scanMaterial(s.db.QueryRow(`SELECT `+materialCols+materialFrom+`WHERE m.id = ?`, materialID))
	if err != nil {
		s.mu.Unlock()
		return MaterialView{}, fmt.Errorf("material not found")
	}
	view := MaterialView{Material: m, Position: Position{Locator: m.Locator, Percent: m.Percent}}
	if m.LibraryID == NotesLibrary {
		view.Document = Document{ID: m.DocumentID, LibraryID: NotesLibrary, RelPath: m.DocumentID, Title: m.Title, Kind: kindNote, Unit: UnitSection, Status: DocParsed}
	} else if d, err := s.document(m.LibraryID, m.DocumentID); err == nil {
		view.Document = d
	}
	view.Bookmarks = s.bookmarks(materialID)
	view.Annotations = s.annotations(materialID)
	s.mu.Unlock()
	view.Outline = s.outlineOf(m.LibraryID, m.DocumentID)
	if m.LibraryID == NotesLibrary {
		// 笔记没有登记单元数，现数一遍。
		if unit, err := s.ReadUnit(NotesLibrary, m.DocumentID, 1, ""); err == nil && unit.Pages > 0 {
			view.Material.PageCount = unit.Pages
		}
	}
	return view, nil
}

// Materials 列出阅读台上的材料，最近打开的在前，最多 100 份。文档已经从库里拿掉的材料不列。
func (s *Store) Materials() []Material {
	return s.RecentMaterials(100)
}

// RecentMaterials 返回最近打开过的几份材料，新的在前。limit 小于 1 时当成 4。文档已经从库里拿掉的材料不列。
func (s *Store) RecentMaterials(limit int) []Material {
	out := []Material{}
	if s == nil {
		return out
	}
	if limit < 1 {
		limit = 4
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT `+materialCols+materialFrom+`
		WHERE m.library_id = 'notes' AND EXISTS (SELECT 1 FROM notes n WHERE n.id = m.document_id)
		   OR EXISTS (SELECT 1 FROM documents d WHERE d.id = m.document_id AND d.library_id = m.library_id AND d.removed = 0)
		ORDER BY CASE WHEN m.last_opened_at != '' THEN m.last_opened_at ELSE m.created_at END DESC LIMIT ?`, limit)
	if err != nil {
		return out
	}
	type raw struct {
		m       Material
		created string
	}
	var list []raw
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.m.ID, &r.m.LibraryID, &r.m.DocumentID, &r.m.Title, &r.m.LastOpenedAt, &r.created, &r.m.Locator, &r.m.Percent, &r.m.Annotations); err == nil {
			list = append(list, r)
		}
	}
	rows.Close()
	for _, r := range list {
		m := r.m
		if m.LastOpenedAt == "" {
			m.LastOpenedAt = r.created
		}
		m.Kind, m.Unit, m.PageCount = kindNote, UnitSection, 1
		if m.LibraryID != NotesLibrary {
			if d, err := s.document(m.LibraryID, m.DocumentID); err == nil {
				m.Kind, m.Unit, m.PageCount = d.Kind, d.Unit, d.PageCount
			}
		}
		out = append(out, m)
	}
	return out
}

// materialTarget 返回一份材料指向的库和文档。materialID 不存在时返回 material not found。调用方要持有 s.mu。
func (s *Store) materialTarget(materialID string) (libraryID, documentID string, err error) {
	if s.db.QueryRow(`SELECT library_id, document_id FROM materials WHERE id = ?`, materialID).Scan(&libraryID, &documentID) != nil {
		return "", "", fmt.Errorf("material not found")
	}
	return libraryID, documentID, nil
}

// SavePosition 记下读到的位置。locator 小于 1 时按 1 算，percent 夹进 0 到 1。materialID 必须存在。
func (s *Store) SavePosition(materialID string, locator int, percent float64) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	if locator < 1 {
		locator = 1
	}
	if percent < 0 || percent != percent {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.materialTarget(materialID); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO reading_positions (material_id, locator, percent, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(material_id) DO UPDATE SET locator = excluded.locator, percent = excluded.percent, updated_at = excluded.updated_at`,
		materialID, locator, percent, formatTime(s.clock()))
	return err
}

// AnnotateRequest 是记一条批注的入参。Kind 为空时有引文算 highlight，没有算 note。Color 为空或不认得时用 yellow。
// Locator 是引文所在的单元，小于 1 时按 1 算。Start 是界面量出的引文起点，没有时给 -1；Prefix 和 Suffix 是引文前后的一小段，可空。
// Author 是 user 或 assistant，其余按 user 算。
type AnnotateRequest struct {
	MaterialID string
	Kind       string
	Color      string
	Locator    int
	Quote      string
	Note       string
	Start      int
	Prefix     string
	Suffix     string
	Author     string
}

// Annotate 在材料上记一条批注。引文和笔记不能同时空。引文先在那个单元的文字里定位：起点对得上就用它，
// 否则在文字里找引文，有几处时挑前后文最像、离给出的起点最近的一处。找不到也照样记下，标成未定位。
func (s *Store) Annotate(req AnnotateRequest) (Annotation, error) {
	if s == nil {
		return Annotation{}, fmt.Errorf("learn store unavailable")
	}
	quote := clipRunes(strings.TrimSpace(req.Quote), maxQuoteRunes)
	note := clipRunes(strings.TrimSpace(req.Note), maxNoteRunes)
	if quote == "" && note == "" {
		return Annotation{}, fmt.Errorf("quote or note required")
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	switch kind {
	case AnnotHighlight, AnnotUnderline, AnnotNote:
	default:
		kind = AnnotNote
		if quote != "" {
			kind = AnnotHighlight
		}
	}
	color := strings.ToLower(strings.TrimSpace(req.Color))
	if !annotColors[color] {
		color = "yellow"
	}
	author := "user"
	if req.Author == "assistant" {
		author = "assistant"
	}
	locator := req.Locator
	if locator < 1 {
		locator = 1
	}
	s.mu.Lock()
	libraryID, documentID, err := s.materialTarget(req.MaterialID)
	s.mu.Unlock()
	if err != nil {
		return Annotation{}, err
	}
	a := Annotation{
		ID: newKey("ann"), MaterialID: req.MaterialID, Kind: kind, Color: color, Locator: locator, Quote: quote, Note: note,
		Start: -1, End: -1, Author: author, Resolution: AnchorResolved,
	}
	sourceHash := ""
	if quote != "" {
		a.Resolution = AnchorUnresolved
		if unit, err := s.ReadUnit(libraryID, documentID, locator, ""); err == nil {
			sourceHash = unit.SourceHash
			if start, end, ok := anchorQuote(unit.Text, quote, req.Start, req.Prefix, req.Suffix); ok {
				r := []rune(unit.Text)
				a.Start, a.End, a.Resolution = start, end, AnchorResolved
				a.Quote = string(r[start:end])
				a.Prefix, a.Suffix = string(r[max(0, start-anchorContext):start]), string(r[end:min(len(r), end+anchorContext)])
			}
		}
	}
	now := formatTime(s.clock())
	a.CreatedAt, a.UpdatedAt = now, now
	s.mu.Lock()
	_, err = s.db.Exec(`INSERT INTO annotations
		(id, material_id, quote, note, page_no, created_at, kind, color, start_off, end_off, prefix, suffix, author, source_hash, resolution, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.MaterialID, a.Quote, a.Note, a.Locator, now, a.Kind, a.Color, a.Start, a.End, a.Prefix, a.Suffix, a.Author, sourceHash, a.Resolution, now)
	s.mu.Unlock()
	if err != nil {
		return Annotation{}, err
	}
	s.notify("material", req.MaterialID, "annotated")
	return a, nil
}

// anchorQuote 在单元文字 text 里给引文 quote 找位置，返回起止的符文位置。hint 是界面量出的起点，没有时给负数。
// 起点对得上就用它。否则找出引文出现的每一处，按前后文 prefix、suffix 的吻合程度打分，再看离 hint 多近。
// 逐字找不到时，两边都去掉空白再找一次，这样换行和空格不同的引文也认得出。都找不到时 ok 为假。
func anchorQuote(text, quote string, hint int, prefix, suffix string) (start, end int, ok bool) {
	r, q := []rune(text), []rune(quote)
	if len(q) == 0 || len(q) > len(r) {
		return 0, 0, false
	}
	if hint >= 0 && hint+len(q) <= len(r) && string(r[hint:hint+len(q)]) == quote {
		return hint, hint + len(q), true
	}
	var found []int
	for i := 0; i+len(q) <= len(r); i++ {
		if r[i] == q[0] && string(r[i:i+len(q)]) == quote {
			found = append(found, i)
		}
	}
	if len(found) > 0 {
		best, bestScore := found[0], -1<<30
		pre, suf := []rune(prefix), []rune(suffix)
		for _, at := range found {
			score := 0
			for k := 1; k <= len(pre) && at-k >= 0 && r[at-k] == pre[len(pre)-k]; k++ {
				score += 2
			}
			for k := 0; k < len(suf) && at+len(q)+k < len(r) && r[at+len(q)+k] == suf[k]; k++ {
				score += 2
			}
			if hint >= 0 {
				d := at - hint
				if d < 0 {
					d = -d
				}
				score -= d / 200
			}
			if score > bestScore {
				best, bestScore = at, score
			}
		}
		return best, best + len(q), true
	}
	// 去掉空白再找：PDF 里一句话常被换行切开，中文之间也可能多出空格。记下每个留下的符文在原文里的位置，找到后映射回去。
	squash := func(in []rune) (out []rune, at []int) {
		for i, c := range in {
			if c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\u00a0' || c == '\u3000' {
				continue
			}
			out, at = append(out, c), append(at, i)
		}
		return out, at
	}
	flat, pos := squash(r)
	needle, _ := squash(q)
	if len(needle) == 0 || len(needle) > len(flat) {
		return 0, 0, false
	}
	want := string(needle)
	for i := 0; i+len(needle) <= len(flat); i++ {
		if flat[i] == needle[0] && string(flat[i:i+len(needle)]) == want {
			return pos[i], pos[i+len(needle)-1] + 1, true
		}
	}
	return 0, 0, false
}

const annotationCols = `id, material_id, kind, color, page_no, quote, note, start_off, end_off, prefix, suffix, author, resolution, created_at, updated_at`

// scanAnnotation 从一行读出一条批注。列顺序和 annotationCols 相同。旧库里没有修改时刻的批注用创建时刻。
func scanAnnotation(row rower) (Annotation, error) {
	var a Annotation
	err := row.Scan(&a.ID, &a.MaterialID, &a.Kind, &a.Color, &a.Locator, &a.Quote, &a.Note, &a.Start, &a.End, &a.Prefix, &a.Suffix, &a.Author, &a.Resolution, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Annotation{}, err
	}
	if a.UpdatedAt == "" {
		a.UpdatedAt = a.CreatedAt
	}
	if a.Locator < 1 {
		a.Locator = 1
	}
	return a, nil
}

// annotations 读出一份材料上的批注，按定位号和位置排，未定位的排在那个单元的最后。调用方要持有 s.mu。
func (s *Store) annotations(materialID string) []Annotation {
	out := []Annotation{}
	rows, err := s.db.Query(`SELECT `+annotationCols+` FROM annotations WHERE material_id = ?
		ORDER BY page_no, CASE WHEN start_off < 0 THEN 1 ELSE 0 END, start_off, created_at`, materialID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		if a, err := scanAnnotation(rows); err == nil {
			out = append(out, a)
		}
	}
	return out
}

// Annotations 列出一份材料上的批注，按定位号和位置排。
func (s *Store) Annotations(materialID string) []Annotation {
	if s == nil {
		return []Annotation{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.annotations(materialID)
}

// UpdateAnnotation 改一条批注的笔记、颜色或样子，返回改完的批注。指针为 nil 的项不动。
// 颜色和样子不认得时返回错误。改完之后引文和笔记不能都是空的。
func (s *Store) UpdateAnnotation(id string, note, color, kind *string) (Annotation, error) {
	if s == nil {
		return Annotation{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	a, err := scanAnnotation(s.db.QueryRow(`SELECT `+annotationCols+` FROM annotations WHERE id = ?`, id))
	if err != nil {
		s.mu.Unlock()
		return Annotation{}, fmt.Errorf("annotation not found")
	}
	if note != nil {
		a.Note = clipRunes(strings.TrimSpace(*note), maxNoteRunes)
	}
	if color != nil {
		c := strings.ToLower(strings.TrimSpace(*color))
		if !annotColors[c] {
			s.mu.Unlock()
			return Annotation{}, fmt.Errorf("color must be yellow, green, blue, pink, or purple")
		}
		a.Color = c
	}
	if kind != nil {
		k := strings.ToLower(strings.TrimSpace(*kind))
		if k != AnnotHighlight && k != AnnotUnderline && k != AnnotNote {
			s.mu.Unlock()
			return Annotation{}, fmt.Errorf("kind must be highlight, underline, or note")
		}
		a.Kind = k
	}
	if a.Quote == "" && a.Note == "" {
		s.mu.Unlock()
		return Annotation{}, fmt.Errorf("quote or note required")
	}
	a.UpdatedAt = formatTime(s.clock())
	_, err = s.db.Exec(`UPDATE annotations SET note = ?, color = ?, kind = ?, updated_at = ? WHERE id = ?`, a.Note, a.Color, a.Kind, a.UpdatedAt, id)
	s.mu.Unlock()
	if err != nil {
		return Annotation{}, err
	}
	s.notify("material", a.MaterialID, "annotated")
	return a, nil
}

// DeleteAnnotation 删除一条批注。id 不存在时返回 annotation not found。
func (s *Store) DeleteAnnotation(id string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	var materialID string
	err := s.db.QueryRow(`SELECT material_id FROM annotations WHERE id = ?`, id).Scan(&materialID)
	if err == nil {
		_, err = s.db.Exec(`DELETE FROM annotations WHERE id = ?`, id)
	} else {
		err = fmt.Errorf("annotation not found")
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("material", materialID, "annotated")
	return nil
}

// bookmarks 读出一份材料上的书签，按定位号排。调用方要持有 s.mu。
func (s *Store) bookmarks(materialID string) []Bookmark {
	out := []Bookmark{}
	rows, err := s.db.Query(`SELECT id, material_id, locator, label, created_at FROM bookmarks WHERE material_id = ? ORDER BY locator`, materialID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var b Bookmark
		if rows.Scan(&b.ID, &b.MaterialID, &b.Locator, &b.Label, &b.CreatedAt) == nil {
			out = append(out, b)
		}
	}
	return out
}

// ToggleBookmark 在一个定位号上放书签，已经有时拿掉。返回现在有没有，以及这份材料的全部书签。
// locator 小于 1 时按 1 算。label 是书签的名字，可空，最长 120 个符文。
func (s *Store) ToggleBookmark(materialID string, locator int, label string) (bool, []Bookmark, error) {
	if s == nil {
		return false, nil, fmt.Errorf("learn store unavailable")
	}
	if locator < 1 {
		locator = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.materialTarget(materialID); err != nil {
		return false, nil, err
	}
	res, err := s.db.Exec(`DELETE FROM bookmarks WHERE material_id = ? AND locator = ?`, materialID, locator)
	if err != nil {
		return false, nil, err
	}
	on := false
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.db.Exec(`INSERT INTO bookmarks (id, material_id, locator, label, created_at) VALUES (?, ?, ?, ?, ?)`,
			newKey("bm"), materialID, locator, clipRunes(strings.TrimSpace(label), 120), formatTime(s.clock())); err != nil {
			return false, nil, err
		}
		on = true
	}
	return on, s.bookmarks(materialID), nil
}

// Locate 在一份材料自己的文字里找 query。先逐字找（不分大小写），每个单元最多记一处；一处都没有时再用检索，只留这份文档的命中。
// 最多返回 20 处，按定位号排。materialID 决定库和文档，调用方不能借它查别的文档。
func (s *Store) Locate(materialID, query string) ([]Hit, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("learn store unavailable")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("query required")
	}
	s.mu.Lock()
	libraryID, documentID, err := s.materialTarget(materialID)
	s.mu.Unlock()
	if err != nil {
		return nil, "", err
	}
	const none = "检索完成，没有命中"
	src, _, err := s.readSource(libraryID, documentID)
	if err != nil {
		return nil, "", err
	}
	hits := []Hit{}
	if doc, err := s.parseSource(src); err == nil {
		needle := strings.ToLower(query)
		for _, u := range doc.Units {
			if len(hits) >= maxLocateHits {
				break
			}
			at := strings.Index(strings.ToLower(u.Text), needle)
			if at < 0 {
				continue
			}
			// 小写之后字节位置可能和原文差几个字节，摘录按词重新取，不直接切原文。
			hits = append(hits, Hit{
				LibraryID: libraryID, DocumentID: documentID, Title: src.title, Page: u.Page, SourceHash: src.sum,
				Excerpt: excerptAround(u.Text, []string{query}, excerptRunes), Text: query, Score: 1,
			})
		}
	}
	if len(hits) > 0 {
		return hits, "", nil
	}
	found, msg, err := s.SearchIn(libraryID, query, maxHits)
	if err != nil {
		return nil, "", err
	}
	for _, h := range found {
		if h.DocumentID == documentID {
			hits = append(hits, h)
		}
	}
	if len(hits) == 0 {
		if msg == "" {
			msg = none
		}
		return hits, msg, nil
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Page < hits[j].Page })
	return hits, "", nil
}

// ExportMaterial 把一份材料上的书签和批注导成 Markdown，按定位号排。返回建议的文件名和正文。
func (s *Store) ExportMaterial(materialID string) (filename, markdown string, err error) {
	view, err := s.MaterialView(materialID)
	if err != nil {
		return "", "", err
	}
	unit := "p."
	if view.Material.Unit == UnitSection {
		unit = "§"
	}
	marks := map[int]string{}
	for _, b := range view.Bookmarks {
		marks[b.Locator] = b.Label
	}
	byLocator := map[int][]Annotation{}
	var order []int
	seen := map[int]bool{}
	note := func(locator int) {
		if !seen[locator] {
			seen[locator] = true
			order = append(order, locator)
		}
	}
	for _, a := range view.Annotations {
		byLocator[a.Locator] = append(byLocator[a.Locator], a)
		note(a.Locator)
	}
	for locator := range marks {
		note(locator)
	}
	sort.Ints(order)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", view.Material.Title)
	for _, locator := range order {
		fmt.Fprintf(&b, "\n## %s %d", unit, locator)
		if label, ok := marks[locator]; ok {
			b.WriteString(" 🔖")
			if label != "" {
				b.WriteString(" " + label)
			}
		}
		b.WriteString("\n")
		for _, a := range byLocator[locator] {
			b.WriteString("\n")
			if a.Quote != "" {
				for _, line := range strings.Split(a.Quote, "\n") {
					b.WriteString("> " + line + "\n")
				}
			}
			if a.Note != "" {
				if a.Quote != "" {
					b.WriteString("\n")
				}
				b.WriteString(a.Note + "\n")
			}
		}
	}
	return sanitizeName(view.Material.Title) + " - notes.md", b.String(), nil
}

// purgeMaterial 删除一份材料连同它的位置、书签和批注。tx 是进行中的事务。
func purgeMaterial(tx *sql.Tx, materialID string) error {
	return execArgs(tx, materialID,
		`DELETE FROM annotations WHERE material_id = ?`,
		`DELETE FROM bookmarks WHERE material_id = ?`,
		`DELETE FROM reading_positions WHERE material_id = ?`,
		`DELETE FROM materials WHERE id = ?`,
	)
}

// CloseMaterial 把一份材料从阅读台拿掉，连同它的位置、书签和批注。文档留在库里。materialID 不存在时返回 material not found。
func (s *Store) CloseMaterial(materialID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	_, _, err := s.materialTarget(materialID)
	if err == nil {
		err = s.inTx(func(tx *sql.Tx) error { return purgeMaterial(tx, materialID) })
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("material", materialID, "closed")
	return nil
}
