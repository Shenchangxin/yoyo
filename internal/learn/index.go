// 建索引和检索。导入、删文档或换嵌入模型之后，索引在后台自己重建：解析、切块、嵌入都不持有库锁，
// 写好的新一代在一个事务里发布，查询始终只读已发布的那一代。一份文档解析失败只记在它自己身上，别的文档照常可检索。
// 检索把 FTS5 的词法排名和向量排名按倒数排名融合。这里不管库和文档的增删改，那在 library.go。

package learn

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// 检索时每一路取多少个候选参加融合，以及融合公式里的常数。
	searchCandidates = 30
	rrfK             = 60.0
	defaultHits      = 8
	maxHits          = 20
	excerptRunes     = 200
	embedBatch       = 48
)

// Hit 是一次检索命中。Page 是定位号，Heading 是它所在的标题路径。Excerpt 是围着命中词的一小段，Text 是整个切块。
type Hit struct {
	LibraryID  string  `json:"library_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	RelPath    string  `json:"rel_path"`
	Page       int     `json:"page"`
	Heading    string  `json:"heading"`
	ChunkID    string  `json:"chunk_id"`
	SourceHash string  `json:"source_hash"`
	Excerpt    string  `json:"excerpt"`
	Text       string  `json:"text"`
	Score      float64 `json:"score"`
}

// indexJob 是一个库的后台索引状态。dirty 表示在跑的这一轮之后还要再跑一轮。idle 在没有活的时候是关着的通道。
type indexJob struct {
	running bool
	dirty   bool
	idle    chan struct{}
}

// vectorSet 是一个库已发布那一代的全部向量，检索时留在内存里，发布新一代时作废。
type vectorSet struct {
	gen  int
	ids  []string
	vecs [][]float32
}

// indexSource 是建索引时的一份原文快照。note 为真时 text 是笔记正文，否则 path 是文件位置。
type indexSource struct {
	docID, title, sum, path string
	note                    bool
	text                    string
}

// builtChunk 是准备写进新一代的一个切块。
type builtChunk struct {
	docID, sum, heading, text string
	locator                   int
}

// scheduleIndex 让 libraryID 这个库在后台重建索引。已经在建时只记下还要再建一轮。调用方不能持有 s.mu。
func (s *Store) scheduleIndex(libraryID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.index == nil {
		s.index = map[string]*indexJob{}
	}
	job := s.index[libraryID]
	if job == nil {
		job = &indexJob{}
		s.index[libraryID] = job
	}
	job.dirty = true
	if job.running {
		s.mu.Unlock()
		return
	}
	job.running = true
	job.idle = make(chan struct{})
	_, _ = s.db.Exec(`UPDATE libraries SET status = ?, last_error = '', progress_json = '' WHERE id = ?`, LibraryIndexing, libraryID)
	s.mu.Unlock()
	started := s.spawn(func(<-chan struct{}) { s.indexLoop(libraryID, job) })
	if !started {
		s.mu.Lock()
		job.running = false
		close(job.idle)
		s.mu.Unlock()
		return
	}
	s.notify("library", libraryID, "indexing")
}

// indexLoop 一轮一轮地建索引，直到没有新的改动或者库在关闭。job 是这个库的索引状态。
func (s *Store) indexLoop(libraryID string, job *indexJob) {
	for {
		s.mu.Lock()
		if !job.dirty || s.stopping() {
			job.running = false
			if job.dirty {
				// 关闭时还有没建的改动，留给下次打开。
				_, _ = s.db.Exec(`UPDATE libraries SET status = ?, progress_json = '' WHERE id = ? AND status = ?`, LibraryStale, libraryID, LibraryIndexing)
			}
			close(job.idle)
			s.mu.Unlock()
			return
		}
		job.dirty = false
		s.mu.Unlock()
		err := s.buildGeneration(libraryID, job)
		if err != nil && !s.stopping() {
			s.mu.Lock()
			_, _ = s.db.Exec(`UPDATE libraries SET status = ?, last_error = ?, progress_json = '', updated_at = ? WHERE id = ?`,
				LibraryError, clipRunes(err.Error(), 400), formatTime(s.clock()), libraryID)
			s.mu.Unlock()
			s.notify("library", libraryID, "error")
		}
	}
}

// WaitIndex 等 libraryID 这个库的后台索引停下来。没有在建时立刻返回。测试和需要马上检索的调用方用它。
func (s *Store) WaitIndex(libraryID string) {
	if s == nil {
		return
	}
	for {
		s.mu.Lock()
		job := s.index[libraryID]
		if job == nil || !job.running {
			s.mu.Unlock()
			return
		}
		idle := job.idle
		s.mu.Unlock()
		<-idle
	}
}

// Rebuild 让一个库重新建索引。linked 库在每一轮开头都会对着文件夹扫一遍，新增、改动和删掉的文件由此跟上。
// libraryID 必须存在。
func (s *Store) Rebuild(libraryID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	_, err := s.library(libraryID)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.scheduleIndex(libraryID)
	return nil
}

// RebuildSync 重新建索引并等它建完，返回这一轮留下的错误。测试和操作者的「试检索」之前用它。
func (s *Store) RebuildSync(libraryID string) error {
	if err := s.Rebuild(libraryID); err != nil {
		return err
	}
	s.WaitIndex(libraryID)
	s.mu.Lock()
	defer s.mu.Unlock()
	lib, err := s.library(libraryID)
	if err != nil {
		return err
	}
	if lib.Status == LibraryError {
		return fmt.Errorf("%s", lib.LastError)
	}
	return nil
}

// recoverLibraries 在打开库时收拾上次没建完的索引：还标着在建的库改成待重建。真正重新开始要等 ResumeIndexing。
func (s *Store) recoverLibraries() {
	_, _ = s.db.Exec(`UPDATE libraries SET status = ?, progress_json = '' WHERE status IN ('indexing', 'building')`, LibraryStale)
	_, _ = s.db.Exec(`UPDATE generations SET state = 'failed', error = 'interrupted' WHERE state = 'building'`)
}

// ResumeIndexing 把待重建的库和有待解析文档的库重新排进后台。应用在接好嵌入函数之后调一次。
func (s *Store) ResumeIndexing() {
	if s == nil {
		return
	}
	s.mu.Lock()
	rows, err := s.db.Query(`SELECT id FROM libraries WHERE status = ?
		OR id IN (SELECT library_id FROM documents WHERE removed = 0 AND status = 'pending')`, LibraryStale)
	var ids []string
	if err == nil {
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				ids = append(ids, id)
			}
		}
		rows.Close()
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.scheduleIndex(id)
	}
}

// markNotesStale 在笔记有改动之后叫醒笔记库的索引。调用方持有 s.mu，所以这里只起一个后台协程去排队。
func (s *Store) markNotesStale() error {
	s.spawn(func(<-chan struct{}) { s.scheduleIndex(NotesLibrary) })
	return nil
}

// setProgress 记下建索引的进度并通知界面。phase 是阶段，done 和 total 是这个阶段里的进度，file 是正在处理的文档。
func (s *Store) setProgress(libraryID, phase string, done, total int, file string) {
	raw, _ := json.Marshal(IndexProgress{Phase: phase, Done: done, Total: total, File: clipRunes(file, 120)})
	s.mu.Lock()
	_, _ = s.db.Exec(`UPDATE libraries SET progress_json = ? WHERE id = ?`, string(raw), libraryID)
	s.mu.Unlock()
	s.notify("library", libraryID, "progress")
}

// buildGeneration 建出新的一代并发布。job.dirty 在这一轮里又被置上时，发布后库仍标成在建，下一轮接着来。
// 返回的错误是整轮都做不下去的那种，例如库不见了；单份文档的失败记在文档上，不算在这里。
func (s *Store) buildGeneration(libraryID string, job *indexJob) error {
	// linked 库的文件留在原处，随时可能被改。每一轮开头先对一遍文件夹，登记的哈希才和磁盘一致。
	_, _ = s.syncLinked(libraryID, false)
	s.mu.Lock()
	lib, err := s.library(libraryID)
	if err != nil {
		s.mu.Unlock()
		return nil
	}
	sources, err := s.indexSources(lib)
	var gen int
	if err == nil {
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(gen), 0) + 1 FROM generations WHERE library_id = ?`, libraryID).Scan(&gen)
		_, err = s.db.Exec(`INSERT INTO generations (library_id, gen, state, created_at) VALUES (?, ?, 'building', ?)`, libraryID, gen, formatTime(s.clock()))
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}

	var chunks []builtChunk
	kept := make([]indexSource, 0, len(sources))
	for i, src := range sources {
		if s.stopping() {
			return nil
		}
		s.setProgress(libraryID, "parsing", i, len(sources), src.title)
		doc, err := s.parseSource(src)
		if !src.note {
			s.recordParse(src.docID, doc, err)
		}
		if err != nil {
			continue
		}
		kept = append(kept, src)
		for _, c := range chunkDoc(doc) {
			chunks = append(chunks, builtChunk{docID: src.docID, sum: src.sum, heading: c.Heading, text: c.Text, locator: c.Locator})
		}
	}

	var vecs [][]float32
	embedNote := ""
	if lib.EmbedModel != "" && s.Embed != nil && len(chunks) > 0 {
		vecs, err = s.embedChunks(libraryID, lib.EmbedModel, chunks)
		if err != nil {
			// 嵌入失败不拦发布，这一代只有词法检索。
			vecs, embedNote = nil, "embedding failed, keyword search only: "+clipRunes(err.Error(), 200)
		}
	}
	if s.stopping() {
		return nil
	}

	s.setProgress(libraryID, "publishing", len(sources), len(sources), "")
	// 留在原处的文件可能在这一轮里被改了。发布前再对一次哈希，变了的这一代不收，下一轮重来。
	changed := map[string]bool{}
	for _, src := range kept {
		if src.note || !strings.HasPrefix(src.path, s.root) {
			if !s.sourceUnchanged(src) {
				changed[src.docID] = true
			}
		}
	}
	s.mu.Lock()
	err = s.inTx(func(tx *sql.Tx) error {
		// 快照之后被删掉或换了内容的文档也不收。
		for _, src := range kept {
			if src.note || changed[src.docID] {
				continue
			}
			var sum string
			var removed int
			if tx.QueryRow(`SELECT sha256, removed FROM documents WHERE id = ?`, src.docID).Scan(&sum, &removed) != nil || removed != 0 || sum != src.sum {
				changed[src.docID] = true
			}
		}
		sig := ""
		if vecs != nil {
			sig = embedSig(lib.EmbedModel, vecs)
		}
		ordinal := 0
		for i, c := range chunks {
			if changed[c.docID] {
				continue
			}
			ordinal++
			id := fmt.Sprintf("chk-%s-%d-%d", libraryID, gen, ordinal)
			if _, err := tx.Exec(`INSERT INTO chunks (id, library_id, document_id, gen, page_no, ordinal, text, sha256, heading) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				id, libraryID, c.docID, gen, c.locator, ordinal, c.text, c.sum, c.heading); err != nil {
				return err
			}
			// 标题路径一起进词法索引，按章节名也能搜到正文。
			body := ftsDocument(c.heading + "\n" + c.text)
			if strings.TrimSpace(body) == "" {
				body = "empty"
			}
			if _, err := tx.Exec(`INSERT INTO chunks_fts (text, chunk_id, library_id, gen) VALUES (?, ?, ?, ?)`, body, id, libraryID, gen); err != nil {
				return err
			}
			if vecs != nil && i < len(vecs) && len(vecs[i]) > 0 {
				if _, err := tx.Exec(`INSERT OR REPLACE INTO embeddings (chunk_id, dim, sig, vec) VALUES (?, ?, ?, ?)`, id, len(vecs[i]), sig, encodeVec(vecs[i])); err != nil {
					return err
				}
			}
		}
		status := LibraryReady
		if job.dirty || len(changed) > 0 {
			status = LibraryIndexing
		}
		for _, stmt := range []string{
			`DELETE FROM chunks_fts WHERE library_id = ? AND gen != ?`,
			`DELETE FROM embeddings WHERE chunk_id IN (SELECT id FROM chunks WHERE library_id = ? AND gen != ?)`,
			`DELETE FROM chunks WHERE library_id = ? AND gen != ?`,
		} {
			if _, err := tx.Exec(stmt, libraryID, gen); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`UPDATE generations SET state = 'published', embed_sig = ?, error = ? WHERE library_id = ? AND gen = ?`, sig, embedNote, libraryID, gen); err != nil {
			return err
		}
		_, err := tx.Exec(`UPDATE libraries SET published_gen = ?, status = ?, last_error = ?, embed_dim = ?, progress_json = '', updated_at = ? WHERE id = ?`,
			gen, status, embedNote, dimOf(sig), formatTime(s.clock()), libraryID)
		return err
	})
	if err == nil {
		delete(s.vectors, libraryID)
		if len(changed) > 0 {
			job.dirty = true
		}
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("library", libraryID, "indexed")
	return nil
}

// indexSources 列出一个库现在要进索引的原文。笔记库是每篇笔记，别的库是没删的文档。调用方要持有 s.mu。
func (s *Store) indexSources(lib Library) ([]indexSource, error) {
	var out []indexSource
	if lib.Kind == NotesLibrary {
		rows, err := s.db.Query(`SELECT id, title, content FROM notes`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			src := indexSource{note: true}
			if rows.Scan(&src.docID, &src.title, &src.text) == nil {
				src.sum = hashText(src.text)
				out = append(out, src)
			}
		}
		return out, rows.Err()
	}
	rows, err := s.db.Query(`SELECT id, title, sha256, external_path, rel_path FROM documents WHERE library_id = ? AND removed = 0 ORDER BY rel_path`, lib.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var src indexSource
		var external, rel string
		if rows.Scan(&src.docID, &src.title, &src.sum, &external, &rel) != nil {
			continue
		}
		src.path = external
		if src.path == "" {
			src.path = s.rawPath(src.sum, extOf(rel))
		}
		out = append(out, src)
	}
	return out, rows.Err()
}

// extOf 返回 rel 的扩展名，带点。
func extOf(rel string) string {
	if i := strings.LastIndex(rel, "."); i >= 0 && !strings.ContainsAny(rel[i:], `/\`) {
		return rel[i:]
	}
	return ""
}

// sourceUnchanged 在一份原文的内容和快照时一样时为真。笔记比正文哈希，文件比字节哈希。
func (s *Store) sourceUnchanged(src indexSource) bool {
	if src.note {
		var content string
		s.mu.Lock()
		err := s.db.QueryRow(`SELECT content FROM notes WHERE id = ?`, src.docID).Scan(&content)
		s.mu.Unlock()
		return err == nil && hashText(content) == src.sum
	}
	sum, err := hashFile(src.path)
	return err == nil && sum == src.sum
}

// parseSource 把一份原文解析成文本单元。笔记当 Markdown 切节；文件先查解析缓存，没有再读文件，读完写回缓存。
// 文件的字节和登记时的哈希对不上时返回 source changed。读文件时不持有库锁。
func (s *Store) parseSource(src indexSource) (parsedDoc, error) {
	if src.note {
		return sectionize(markdownBlocks(normalizeText(src.text))), nil
	}
	s.mu.Lock()
	doc, ok := s.cachedParse(src.sum)
	s.mu.Unlock()
	if ok {
		return doc, nil
	}
	if _, err := os.Stat(src.path); err != nil {
		return parsedDoc{}, fmt.Errorf("the file is missing")
	}
	sum, doc, err := parseFile(src.path)
	if err != nil {
		return parsedDoc{}, err
	}
	if sum != src.sum {
		return parsedDoc{}, fmt.Errorf("source changed")
	}
	s.mu.Lock()
	err = s.storeParse(src.sum, doc)
	s.mu.Unlock()
	return doc, err
}

// recordParse 把一份文档的解析结果记在文档上：状态、单元数、单元种类、提纲和字数。err 非空时记成失败和原因。
// 整份都抽不出文字的文档记成 needs_ocr。
func (s *Store) recordParse(docID string, doc parsedDoc, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		_, _ = s.db.Exec(`UPDATE documents SET status = ?, error = ? WHERE id = ?`, DocFailed, clipRunes(err.Error(), 300), docID)
		return
	}
	status := DocParsed
	if !doc.hasText() {
		status = DocNeedsOCR
	}
	outline := doc.Outline
	if outline == nil {
		outline = []OutlineEntry{}
	}
	raw, _ := json.Marshal(outline)
	_, _ = s.db.Exec(`UPDATE documents SET status = ?, error = '', page_count = ?, unit = ?, outline_json = ?, char_count = ? WHERE id = ?`,
		status, len(doc.Units), doc.Unit, string(raw), doc.Chars, docID)
}

// embedChunks 给每个切块取向量。同一段文字在同一个模型下只嵌入一次，结果按文字哈希缓存。
// 嵌入请求分批发出，发请求时不持有库锁。返回的切片和 chunks 一一对应。
func (s *Store) embedChunks(libraryID, model string, chunks []builtChunk) ([][]float32, error) {
	out := make([][]float32, len(chunks))
	keys := make([]string, len(chunks))
	var missing []int
	s.mu.Lock()
	for i, c := range chunks {
		keys[i] = hashText(c.heading + "\n" + c.text)
		var blob []byte
		if s.db.QueryRow(`SELECT vec FROM embed_cache WHERE sig = ? AND text_sha = ?`, model, keys[i]).Scan(&blob) == nil && len(blob) > 0 {
			out[i] = decodeVec(blob)
		} else {
			missing = append(missing, i)
		}
	}
	s.mu.Unlock()
	for start := 0; start < len(missing); start += embedBatch {
		if s.stopping() {
			return nil, fmt.Errorf("interrupted")
		}
		end := start + embedBatch
		if end > len(missing) {
			end = len(missing)
		}
		batch := missing[start:end]
		s.setProgress(libraryID, "embedding", len(chunks)-len(missing)+start, len(chunks), "")
		texts := make([]string, len(batch))
		for j, i := range batch {
			texts[j] = strings.TrimSpace(chunks[i].heading + "\n" + chunks[i].text)
		}
		vecs, err := s.Embed(model, texts)
		if err != nil {
			return nil, err
		}
		if len(vecs) != len(batch) {
			return nil, fmt.Errorf("the embedding endpoint returned %d vectors for %d texts", len(vecs), len(batch))
		}
		s.mu.Lock()
		for j, i := range batch {
			out[i] = vecs[j]
			_, _ = s.db.Exec(`INSERT OR REPLACE INTO embed_cache (sig, text_sha, dim, vec) VALUES (?, ?, ?, ?)`, model, keys[i], len(vecs[j]), encodeVec(vecs[j]))
		}
		s.mu.Unlock()
	}
	return out, nil
}

// SearchLibrary 在已发布的一代里检索，最多返回 8 条。它是 SearchIn 的短写法。
func (s *Store) SearchLibrary(libraryID, query string) ([]Hit, string, error) {
	return s.SearchIn(libraryID, query, 0)
}

// SearchIn 在 libraryID 已发布的那一代里检索。词法排名和向量排名各取前 30，按倒数排名融合。
// query 去掉空白后不能空。limit 小于等于 0 时取 8，最多 20。没有命中时命中列表为空，message 说明原因。
// 取查询向量时不持有库锁；嵌入失败时只用词法结果。
func (s *Store) SearchIn(libraryID, query string, limit int) ([]Hit, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("learn store unavailable")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("query required")
	}
	if limit <= 0 {
		limit = defaultHits
	}
	if limit > maxHits {
		limit = maxHits
	}
	const none = "检索完成，没有命中"
	s.mu.Lock()
	lib, err := s.library(libraryID)
	if err != nil {
		s.mu.Unlock()
		return nil, "", err
	}
	if lib.PublishedGen == 0 {
		s.mu.Unlock()
		if lib.Status == LibraryIndexing {
			return []Hit{}, "这座文库还在建索引，稍后再搜", nil
		}
		return []Hit{}, none, nil
	}
	// 词法这一路先要每个词都出现的切块，再用任一词出现的补足。这样「sigma-token」不会被只含 token 的切块抢到前面。
	ranks := map[string]float64{}
	rank := 0
	for _, match := range []string{ftsMatchAll(query), ftsMatch(query)} {
		if match == "" || rank >= searchCandidates {
			continue
		}
		rows, err := s.db.Query(`SELECT chunk_id FROM chunks_fts WHERE chunks_fts MATCH ? AND library_id = ? AND gen = ? ORDER BY bm25(chunks_fts) LIMIT ?`,
			match, libraryID, lib.PublishedGen, searchCandidates)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id string
			if rows.Scan(&id) != nil {
				continue
			}
			if _, seen := ranks[id]; seen || rank >= searchCandidates {
				continue
			}
			rank++
			ranks[id] = 1 / (rrfK + float64(rank))
		}
		rows.Close()
	}
	var sig string
	_ = s.db.QueryRow(`SELECT embed_sig FROM generations WHERE library_id = ? AND gen = ?`, libraryID, lib.PublishedGen).Scan(&sig)
	s.mu.Unlock()

	if sig != "" && lib.EmbedModel != "" && s.Embed != nil {
		if vecs, err := s.Embed(lib.EmbedModel, []string{query}); err == nil && len(vecs) == 1 {
			for rank, id := range s.nearest(libraryID, lib.PublishedGen, vecs[0], searchCandidates) {
				ranks[id] += 1 / (rrfK + float64(rank+1))
			}
		}
	}
	if len(ranks) == 0 {
		return []Hit{}, none, nil
	}
	type pair struct {
		id    string
		score float64
	}
	pairs := make([]pair, 0, len(ranks))
	for id, score := range ranks {
		pairs = append(pairs, pair{id, score})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score != pairs[j].score {
			return pairs[i].score > pairs[j].score
		}
		return pairs[i].id < pairs[j].id
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	// 摘录和小节先按整句查询找落点，找不到再用拆出来的词。
	terms := append([]string{query}, ftsQueryTokens(query)...)
	hits := make([]Hit, 0, len(pairs))
	s.mu.Lock()
	for _, p := range pairs {
		if h, ok := s.hit(p.id, p.score, terms); ok {
			hits = append(hits, h)
		}
	}
	s.mu.Unlock()
	if len(hits) == 0 {
		return []Hit{}, none, nil
	}
	return hits, "", nil
}

// nearest 返回和查询向量 q 最接近的最多 n 个切块 id，最近的在前。向量第一次用到时读进内存，新一代发布后重读。
func (s *Store) nearest(libraryID string, gen int, q []float32, n int) []string {
	s.mu.Lock()
	set := s.vectors[libraryID]
	if set == nil || set.gen != gen {
		set = &vectorSet{gen: gen}
		rows, err := s.db.Query(`SELECT e.chunk_id, e.vec FROM embeddings e JOIN chunks c ON c.id = e.chunk_id WHERE c.library_id = ? AND c.gen = ?`, libraryID, gen)
		if err == nil {
			for rows.Next() {
				var id string
				var blob []byte
				if rows.Scan(&id, &blob) == nil {
					set.ids = append(set.ids, id)
					set.vecs = append(set.vecs, decodeVec(blob))
				}
			}
			rows.Close()
		}
		if s.vectors == nil {
			s.vectors = map[string]*vectorSet{}
		}
		s.vectors[libraryID] = set
	}
	s.mu.Unlock()
	type scored struct {
		at    int
		score float64
	}
	all := make([]scored, 0, len(set.ids))
	for i, v := range set.vecs {
		if sc := cosine(q, v); sc > 0 {
			all = append(all, scored{i, sc})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })
	if len(all) > n {
		all = all[:n]
	}
	out := make([]string, len(all))
	for i, item := range all {
		out[i] = set.ids[item.at]
	}
	return out
}

// hit 把一个切块读成命中。terms 是查询里的词，摘录围着第一个出现的词取。调用方要持有 s.mu。
func (s *Store) hit(chunkID string, score float64, terms []string) (Hit, bool) {
	h := Hit{ChunkID: chunkID, Score: score}
	err := s.db.QueryRow(`SELECT library_id, document_id, page_no, text, sha256, heading FROM chunks WHERE id = ?`, chunkID).
		Scan(&h.LibraryID, &h.DocumentID, &h.Page, &h.Text, &h.SourceHash, &h.Heading)
	if err != nil {
		return Hit{}, false
	}
	h.Excerpt = excerptAround(h.Text, terms, excerptRunes)
	// 一个切块可能跨过几个小节。命中的词前面最近的那个标题才是它真正所在的小节。
	if at := firstTerm(h.Text, terms); at > 0 {
		h.Heading = refineHeading(h.Heading, h.Text[:at])
	}
	if h.LibraryID == NotesLibrary {
		_ = s.db.QueryRow(`SELECT title FROM notes WHERE id = ?`, h.DocumentID).Scan(&h.Title)
		h.RelPath = h.DocumentID
		return h, true
	}
	_ = s.db.QueryRow(`SELECT title, rel_path FROM documents WHERE id = ?`, h.DocumentID).Scan(&h.Title, &h.RelPath)
	return h, true
}

// firstTerm 返回 terms 里排在最前面、又在 text 里出现的那个词的字节位置，不分大小写。terms 按重要程度排，整句查询在前，拆出的词在后。
// 都没出现时返回 -1。
func firstTerm(text string, terms []string) int {
	lower := strings.ToLower(text)
	for _, term := range terms {
		if term = strings.ToLower(strings.TrimSpace(term)); term == "" {
			continue
		}
		if i := strings.Index(lower, term); i >= 0 {
			return i
		}
	}
	return -1
}

// refineHeading 用命中位置之前的文字 before 修正标题路径 path：before 里最后一个 Markdown 标题是命中真正所在的小节。
// 它是第 n 级标题时，路径留下前 n-1 段再接上它。before 里没有标题时原样返回 path。
func refineHeading(path, before string) string {
	lines := strings.Split(before, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		level, title, ok := atxHeading(lines[i])
		if !ok || title == "" {
			continue
		}
		var parts []string
		if path != "" {
			parts = strings.Split(path, " > ")
		}
		if keep := level - 1; keep < len(parts) {
			parts = parts[:max(keep, 0)]
		}
		if len(parts) > 0 && parts[len(parts)-1] == title {
			return strings.Join(parts, " > ")
		}
		return clipRunes(strings.Join(append(parts, title), " > "), maxHeadingRunes)
	}
	return path
}

// excerptHeading 和 excerptMarks 去掉摘录里的 Markdown 记号：行首标题的井号、加粗和行内代码的标记。
// 摘录是给人扫一眼的，不是给渲染器的。
var (
	excerptHeading = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+`)
	excerptMarks   = strings.NewReplacer("**", "", "__", "", "`", "")
)

// excerptAround 从 text 里取一段最多 n 个符文的摘录，尽量把 terms 里排在最前面又出现了的那个词放在中间靠前的位置。
// 空白压成一个空格，Markdown 记号去掉。两头被截断时加省略号。
func excerptAround(text string, terms []string, n int) string {
	flat := strings.Join(strings.Fields(excerptMarks.Replace(excerptHeading.ReplaceAllString(text, ""))), " ")
	r := []rune(flat)
	if len(r) <= n {
		return flat
	}
	lower := strings.ToLower(flat)
	at := firstTerm(flat, terms)
	start := 0
	if at > 0 {
		start = len([]rune(lower[:at])) - n/3
		if start < 0 {
			start = 0
		}
	}
	end := start + n
	if end > len(r) {
		end = len(r)
		start = end - n
	}
	out := string(r[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(r) {
		out += "…"
	}
	return out
}

// UnitView 是读出的一个文本单元。Status 是 ok、needs_ocr 或 changed；changed 表示原文在命中之后变了，这时没有正文。
// Page 是定位号，Pages 是单元总数，Unit 是 page 或 section，Kind 是文档格式。SourceHash 是这份原文现在的哈希。
type UnitView struct {
	Text       string `json:"text"`
	Status     string `json:"status"`
	Page       int    `json:"page"`
	Pages      int    `json:"pages"`
	Unit       string `json:"unit"`
	Kind       string `json:"kind"`
	SourceHash string `json:"source_hash"`
}

// ReadUnit 读一份文档的一个单元。locator 小于 1 时读第一个。sourceHash 非空且和现在的原文对不上时返回 changed，不给正文。
// 笔记按 Markdown 切节。文件没有解析缓存时现读一遍，读文件时不持有库锁。
func (s *Store) ReadUnit(libraryID, documentID string, locator int, sourceHash string) (UnitView, error) {
	if s == nil {
		return UnitView{}, fmt.Errorf("learn store unavailable")
	}
	if locator < 1 {
		locator = 1
	}
	src, kind, err := s.readSource(libraryID, documentID)
	if err != nil {
		return UnitView{}, err
	}
	if sourceHash != "" && sourceHash != src.sum {
		return UnitView{Status: "changed", Page: locator, Kind: kind, SourceHash: src.sum}, nil
	}
	if !src.note {
		s.mu.Lock()
		unit, meta, ok := s.cachedUnit(src.sum, locator)
		s.mu.Unlock()
		if ok {
			if unit.Page == 0 {
				return UnitView{}, fmt.Errorf("page not found")
			}
			return UnitView{Text: unit.Text, Status: unit.Status, Page: locator, Pages: meta.Pages, Unit: meta.Unit, Kind: kind, SourceHash: src.sum}, nil
		}
	}
	doc, err := s.parseSource(src)
	if err != nil {
		return UnitView{}, err
	}
	if len(doc.Units) == 0 {
		return UnitView{Status: unitOK, Page: 1, Pages: 1, Unit: doc.Unit, Kind: kind, SourceHash: src.sum}, nil
	}
	if locator > len(doc.Units) {
		return UnitView{}, fmt.Errorf("page not found")
	}
	u := doc.Units[locator-1]
	return UnitView{Text: u.Text, Status: u.Status, Page: locator, Pages: len(doc.Units), Unit: doc.Unit, Kind: kind, SourceHash: src.sum}, nil
}

// readSource 找到一份文档的原文快照和格式。笔记库里是笔记正文，别的库里是文件。对不上时返回 document not found。
func (s *Store) readSource(libraryID, documentID string) (indexSource, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if libraryID == NotesLibrary {
		src := indexSource{docID: documentID, note: true}
		if err := s.db.QueryRow(`SELECT title, content FROM notes WHERE id = ?`, documentID).Scan(&src.title, &src.text); err != nil {
			return indexSource{}, "", fmt.Errorf("document not found")
		}
		src.sum = hashText(src.text)
		return src, kindNote, nil
	}
	d, err := s.document(libraryID, documentID)
	if err != nil {
		return indexSource{}, "", err
	}
	return indexSource{docID: d.ID, title: d.Title, sum: d.SHA256, path: s.docPath(libraryID, documentID)}, d.Kind, nil
}

// outlineOf 返回一份文档的提纲。笔记现切，文件读解析缓存。没有提纲时返回空数组。
func (s *Store) outlineOf(libraryID, documentID string) []OutlineEntry {
	src, _, err := s.readSource(libraryID, documentID)
	if err != nil {
		return []OutlineEntry{}
	}
	doc, err := s.parseSource(src)
	if err != nil || doc.Outline == nil {
		return []OutlineEntry{}
	}
	return doc.Outline
}

// encodeVec 把向量写成小端的 float32 字节。
func encodeVec(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

// decodeVec 把 encodeVec 写出的字节读回向量。
func decodeVec(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

// cosine 返回两个向量的余弦相似度。长度不等或有一个是零向量时返回 0。
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// embedSig 是一代索引用的嵌入签名：模型名加向量维度。换模型或维度变了，旧向量就不能和新查询比。
func embedSig(model string, vecs [][]float32) string {
	dim := 0
	for _, v := range vecs {
		if len(v) > 0 {
			dim = len(v)
			break
		}
	}
	return fmt.Sprintf("%s:%d", model, dim)
}

// dimOf 从嵌入签名里取出向量维度。签名为空时返回 0。
func dimOf(sig string) int {
	i := strings.LastIndex(sig, ":")
	if i < 0 {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(sig[i+1:], "%d", &n)
	return n
}

var keySeq atomic.Uint64

// newKey 生成一个带前缀的 id：时间、进程内序号，再加三个随机字节。
// 序号每次启动都从 1 起，没有随机字节的话，同一秒内重启会撞上旧 id。
func newKey(prefix string) string {
	n := keySeq.Add(1)
	var salt [3]byte
	_, _ = rand.Read(salt[:])
	return fmt.Sprintf("%s-%s-%d%s", prefix, time.Now().UTC().Format("20060102T150405"), n, hex.EncodeToString(salt[:]))
}
