// 重建写进新的一代。源哈希对得上才把 published 指针拨过去。查询只读已发布的一代。

package learn

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// Hit 是一次检索命中。Excerpt 最多 200 个符文。SourceHash 对不上当前文件时不会出现在结果里。
type Hit struct {
	LibraryID  string  `json:"library_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	RelPath    string  `json:"rel_path"`
	Page       int     `json:"page"`
	ChunkID    string  `json:"chunk_id"`
	SourceHash string  `json:"source_hash"`
	Excerpt    string  `json:"excerpt"`
	Text       string  `json:"text"`
	Score      float64 `json:"score"`
}

// StartRebuild 在后台重建。同一库已经在建时返回 indexing busy。
// libraryID 必须存在。
func (s *Store) StartRebuild(libraryID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	if s.rebuilding[libraryID] {
		s.mu.Unlock()
		return fmt.Errorf("indexing busy")
	}
	if _, err := s.library(libraryID); err != nil {
		s.mu.Unlock()
		return err
	}
	s.rebuilding[libraryID] = true
	_, _ = s.db.Exec(`UPDATE libraries SET status = 'building', last_error = '' WHERE id = ?`, libraryID)
	s.mu.Unlock()
	go func() {
		err := s.rebuild(libraryID)
		s.mu.Lock()
		delete(s.rebuilding, libraryID)
		if err != nil {
			_, _ = s.db.Exec(`UPDATE libraries SET status = 'error', last_error = ? WHERE id = ?`, clipRunes(err.Error(), 400), libraryID)
		}
		s.mu.Unlock()
	}()
	return nil
}

// RebuildSync 同步重建，测试和操作者的「试检索」之前用它。
// libraryID 必须存在。
func (s *Store) RebuildSync(libraryID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	return s.rebuild(libraryID)
}

func (s *Store) rebuild(libraryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	lib, err := s.library(libraryID)
	if err != nil {
		return err
	}
	var gen int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(gen), 0) + 1 FROM generations WHERE library_id = ?`, libraryID).Scan(&gen)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.Exec(`INSERT INTO generations (library_id, gen, state, created_at) VALUES (?, ?, 'building', ?)`, libraryID, gen, now)
	_, _ = s.db.Exec(`DELETE FROM index_journal WHERE library_id = ? AND gen = ?`, libraryID, gen)

	type src struct {
		docID, rel, title, sum, kind, external string
		text                                   string
		pages                                  []PageText
	}
	var sources []src
	if lib.Kind == "notes" {
		rows, err := s.db.Query(`SELECT id, title, content FROM notes`)
		if err != nil {
			return s.failGen(libraryID, gen, err)
		}
		for rows.Next() {
			var id, title, content string
			if err := rows.Scan(&id, &title, &content); err != nil {
				continue
			}
			sources = append(sources, src{docID: id, rel: id, title: title, sum: hashText(content), kind: "note", pages: []PageText{{Text: content, Status: "ok"}}})
		}
		rows.Close()
	} else {
		rows, err := s.db.Query(`SELECT id, rel_path, title, sha256, external_path FROM documents WHERE library_id = ? AND removed = 0`, libraryID)
		if err != nil {
			return s.failGen(libraryID, gen, err)
		}
		var docs []Document
		for rows.Next() {
			var d Document
			var external string
			if err := rows.Scan(&d.ID, &d.RelPath, &d.Title, &d.SHA256, &external); err != nil {
				continue
			}
			d.LibraryID = libraryID
			docs = append(docs, d)
			_ = external
		}
		rows.Close()
		for _, d := range docs {
			path := s.docPath(libraryID, d.ID)
			sum := d.SHA256
			pages, status, reason := s.cachedParse(path, sum)
			if status == "failed" {
				_, _ = s.db.Exec(`INSERT INTO index_journal (id, library_id, document_id, gen, status, reason) VALUES (?, ?, ?, ?, 'failed', ?)`,
					newKey("ij"), libraryID, d.ID, gen, reason)
				_, _ = s.db.Exec(`UPDATE documents SET status = 'failed', error = ? WHERE id = ?`, reason, d.ID)
				return s.failGen(libraryID, gen, fmt.Errorf("%s", reason))
			}
			sources = append(sources, src{docID: d.ID, rel: d.RelPath, title: d.Title, sum: sum, kind: "file", pages: pages})
			_, _ = s.db.Exec(`UPDATE documents SET status = 'parsed', page_count = ?, error = '' WHERE id = ?`, len(pages), d.ID)
			_, _ = s.db.Exec(`INSERT INTO index_journal (id, library_id, document_id, gen, status) VALUES (?, ?, ?, ?, 'completed')`,
				newKey("ij"), libraryID, d.ID, gen)
		}
	}

	checks := make([]srcCheck, 0, len(sources))
	for _, item := range sources {
		checks = append(checks, srcCheck{docID: item.docID, rel: item.rel, title: item.title, sum: item.sum, kind: item.kind, external: item.external, pages: item.pages})
	}
	if err := s.verifySources(libraryID, checks); err != nil {
		return s.failGen(libraryID, gen, err)
	}

	var chunkIDs []string
	var chunkText []string
	ordinal := 0
	for _, src := range sources {
		for _, part := range chunkPages(src.pages) {
			ordinal++
			id := fmt.Sprintf("chk-%s-%d", genLabel(gen), ordinal)
			_, err := s.db.Exec(`INSERT INTO chunks (id, library_id, document_id, gen, page_no, ordinal, text, sha256) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, libraryID, src.docID, gen, part.Page, ordinal, part.Text, src.sum)
			if err != nil {
				return s.failGen(libraryID, gen, err)
			}
			doc := ftsDocument(part.Text)
			if strings.TrimSpace(doc) == "" {
				doc = "empty"
			}
			_, err = s.db.Exec(`INSERT INTO chunks_fts (text, chunk_id, library_id, gen) VALUES (?, ?, ?, ?)`, doc, id, libraryID, gen)
			if err != nil {
				return s.failGen(libraryID, gen, err)
			}
			chunkIDs = append(chunkIDs, id)
			chunkText = append(chunkText, part.Text)
		}
	}

	sig := ""
	if lib.EmbedModel != "" && s.Embed != nil && len(chunkText) > 0 {
		vecs, err := s.Embed(lib.EmbedModel, chunkText)
		if err != nil {
			sig = ""
			_, _ = s.db.Exec(`UPDATE generations SET embed_sig = '', error = ? WHERE library_id = ? AND gen = ?`, "embed: "+clipRunes(err.Error(), 200), libraryID, gen)
		} else if len(vecs) == len(chunkIDs) {
			sig = embedSig(lib.EmbedModel, vecs)
			for i, id := range chunkIDs {
				_, _ = s.db.Exec(`INSERT OR REPLACE INTO embeddings (chunk_id, dim, sig, vec) VALUES (?, ?, ?, ?)`, id, len(vecs[i]), sig, encodeVec(vecs[i]))
			}
		}
	}

	_, _ = s.db.Exec(`UPDATE generations SET state = 'published', embed_sig = ? WHERE library_id = ? AND gen = ?`, sig, libraryID, gen)
	_, _ = s.db.Exec(`UPDATE libraries SET published_gen = ?, status = 'ready', last_error = '', embed_dim = ?, updated_at = ? WHERE id = ?`,
		gen, dimOf(sig), time.Now().UTC().Format(time.RFC3339Nano), libraryID)
	_, _ = s.db.Exec(`DELETE FROM chunks_fts WHERE library_id = ? AND gen != ?`, libraryID, gen)
	_, _ = s.db.Exec(`DELETE FROM embeddings WHERE chunk_id IN (SELECT id FROM chunks WHERE library_id = ? AND gen != ?)`, libraryID, gen)
	_, _ = s.db.Exec(`DELETE FROM chunks WHERE library_id = ? AND gen != ?`, libraryID, gen)
	return nil
}

func (s *Store) failGen(libraryID string, gen int, err error) error {
	_, _ = s.db.Exec(`DELETE FROM chunks_fts WHERE library_id = ? AND gen = ?`, libraryID, gen)
	_, _ = s.db.Exec(`DELETE FROM chunks WHERE library_id = ? AND gen = ?`, libraryID, gen)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	_, _ = s.db.Exec(`UPDATE generations SET state = 'failed', error = ? WHERE library_id = ? AND gen = ?`, clipRunes(msg, 400), libraryID, gen)
	_, _ = s.db.Exec(`UPDATE libraries SET status = 'error', last_error = ? WHERE id = ?`, clipRunes(msg, 400), libraryID)
	if err == nil {
		err = fmt.Errorf("index failed")
	}
	return err
}

func (s *Store) verifySources(libraryID string, sources []srcCheck) error {
	for _, src := range sources {
		if src.kind == "note" {
			var content string
			if err := s.db.QueryRow(`SELECT content FROM notes WHERE id = ?`, src.docID).Scan(&content); err != nil {
				return fmt.Errorf("source changed")
			}
			if hashText(content) != src.sum {
				return fmt.Errorf("source changed")
			}
			continue
		}
		path := s.docPath(libraryID, src.docID)
		sum, err := hashFile(path)
		if err != nil || sum != src.sum {
			return fmt.Errorf("source changed")
		}
	}
	return nil
}

type srcCheck struct {
	docID, rel, title, sum, kind, external string
	pages                                  []PageText
}

func (s *Store) docPath(libraryID, docID string) string {
	var sum, external, rel string
	_ = s.db.QueryRow(`SELECT sha256, external_path, rel_path FROM documents WHERE id = ? AND library_id = ?`, docID, libraryID).Scan(&sum, &external, &rel)
	if external != "" {
		return external
	}
	return filepath.Join(s.root, "raw", sum+strings.ToLower(filepath.Ext(rel)))
}

func (s *Store) cachedParse(path, sum string) ([]PageText, string, string) {
	rows, err := s.db.Query(`SELECT page_no, text, page_status FROM parses WHERE sha256 = ? AND parser = ? ORDER BY page_no`, sum, parserVersion)
	if err == nil {
		var pages []PageText
		for rows.Next() {
			var p PageText
			if err := rows.Scan(&p.Page, &p.Text, &p.Status); err == nil {
				pages = append(pages, p)
			}
		}
		rows.Close()
		if len(pages) > 0 {
			return pages, "parsed", ""
		}
	}
	if path == "" {
		return nil, "failed", "missing file"
	}
	if _, err := os.Stat(path); err != nil {
		return nil, "failed", "missing file"
	}
	got, pages, err := ParseFile(path)
	if err != nil {
		return nil, "failed", err.Error()
	}
	if got != sum && sum != "" {
		return nil, "failed", "source changed"
	}
	for _, p := range pages {
		_, _ = s.db.Exec(`INSERT OR REPLACE INTO parses (sha256, parser, page_no, text, page_status) VALUES (?, ?, ?, ?, ?)`,
			sum, parserVersion, p.Page, p.Text, p.Status)
	}
	return pages, "parsed", ""
}

// SearchLibrary 在已发布的一代里检索。没有命中时命中列表为空，文案是「检索完成，没有命中」。
// libraryID 限定一个库。query 去掉空白后不能空。
func (s *Store) SearchLibrary(libraryID, query string) ([]Hit, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("learn store unavailable")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("query required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lib, err := s.library(libraryID)
	if err != nil {
		return nil, "", err
	}
	if lib.PublishedGen == 0 {
		return nil, "检索完成，没有命中", nil
	}
	match := ftsMatch(query)
	ranks := map[string]float64{}
	if match != "" {
		rows, err := s.db.Query(`SELECT chunk_id, bm25(chunks_fts) FROM chunks_fts WHERE chunks_fts MATCH ? AND library_id = ? AND gen = ? ORDER BY bm25(chunks_fts) LIMIT 20`,
			match, libraryID, lib.PublishedGen)
		if err == nil {
			rank := 0
			for rows.Next() {
				var id string
				var bm float64
				if err := rows.Scan(&id, &bm); err != nil {
					continue
				}
				rank++
				ranks[id] += 1 / (60 + float64(rank))
				_ = bm
			}
			rows.Close()
		}
	}
	s.addVectorRanks(libraryID, lib.PublishedGen, query, ranks)
	if len(ranks) == 0 {
		return nil, "检索完成，没有命中", nil
	}
	type pair struct {
		id    string
		score float64
	}
	var pairs []pair
	for id, sc := range ranks {
		pairs = append(pairs, pair{id, sc})
	}
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].score > pairs[i].score {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	if len(pairs) > 8 {
		pairs = pairs[:8]
	}
	var hits []Hit
	for _, p := range pairs {
		h, ok := s.hit(p.id, p.score)
		if ok {
			hits = append(hits, h)
		}
	}
	if len(hits) == 0 {
		return nil, "检索完成，没有命中", nil
	}
	return hits, "", nil
}

func (s *Store) hit(chunkID string, score float64) (Hit, bool) {
	var h Hit
	var text, sum string
	err := s.db.QueryRow(`SELECT c.library_id, c.document_id, c.page_no, c.text, c.sha256 FROM chunks c WHERE c.id = ?`, chunkID).
		Scan(&h.LibraryID, &h.DocumentID, &h.Page, &text, &sum)
	if err != nil {
		return Hit{}, false
	}
	h.ChunkID = chunkID
	h.SourceHash = sum
	h.Text = text
	h.Excerpt = clipRunes(strings.Join(strings.Fields(text), " "), 200)
	h.Score = score
	if h.LibraryID == "notes" {
		_ = s.db.QueryRow(`SELECT title FROM notes WHERE id = ?`, h.DocumentID).Scan(&h.Title)
		h.RelPath = h.DocumentID
		return h, true
	}
	_ = s.db.QueryRow(`SELECT title, rel_path FROM documents WHERE id = ?`, h.DocumentID).Scan(&h.Title, &h.RelPath)
	return h, true
}

// DocumentPages 返回已解析的页数。笔记算 1 页。libraryID 和 documentID 对不上时返回 0。
func (s *Store) DocumentPages(libraryID, documentID string) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if libraryID == "notes" || (documentID != "" && s.noteExists(documentID) && libraryID == "notes") {
		return 1
	}
	path := s.docPath(libraryID, documentID)
	sum, err := hashFile(path)
	if err != nil {
		return 0
	}
	pages, _, _ := s.cachedParse(path, sum)
	if len(pages) == 0 {
		return 1
	}
	return len(pages)
}

// ReadPage 读一页原文。sourceHash 非空且与当前文件不符时返回 changed，不改指到别的材料。
// libraryID、documentID 定位文档。page 小于 1 时读第一页或整篇。
func (s *Store) ReadPage(libraryID, documentID, sourceHash string, page int) (string, string, error) {
	if s == nil {
		return "", "", fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if libraryID == "notes" || documentID != "" && s.noteExists(documentID) && libraryID == "notes" {
		var content string
		err := s.db.QueryRow(`SELECT content FROM notes WHERE id = ?`, documentID).Scan(&content)
		if err != nil {
			return "", "", fmt.Errorf("document not found")
		}
		sum := hashText(content)
		if sourceHash != "" && sourceHash != sum {
			return "", "changed", nil
		}
		return content, "", nil
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM documents WHERE id = ? AND library_id = ? AND removed = 0`, documentID, libraryID).Scan(&n)
	if err != nil || n == 0 {
		return "", "", fmt.Errorf("document not found")
	}
	path := s.docPath(libraryID, documentID)
	now, err := hashFile(path)
	if err != nil {
		return "", "", err
	}
	if sourceHash != "" && sourceHash != now {
		return "", "changed", nil
	}
	pages, status, reason := s.cachedParse(path, now)
	if status == "failed" {
		return "", "", fmt.Errorf("%s", reason)
	}
	if page <= 0 {
		var b strings.Builder
		for _, p := range pages {
			b.WriteString(p.Text)
			b.WriteString("\n")
		}
		return b.String(), "", nil
	}
	for _, p := range pages {
		if p.Page == page {
			return p.Text, "", nil
		}
	}
	return "", "", fmt.Errorf("page not found")
}

func (s *Store) noteExists(id string) bool {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM notes WHERE id = ?`, id).Scan(&n)
	return n > 0
}

func (s *Store) addVectorRanks(libraryID string, gen int, query string, ranks map[string]float64) {
	if s.Embed == nil {
		return
	}
	lib, err := s.library(libraryID)
	if err != nil || lib.EmbedModel == "" {
		return
	}
	vecs, err := s.Embed(lib.EmbedModel, []string{query})
	if err != nil || len(vecs) != 1 {
		return
	}
	q := vecs[0]
	rows, err := s.db.Query(`SELECT e.chunk_id, e.vec FROM embeddings e JOIN chunks c ON c.id = e.chunk_id WHERE c.library_id = ? AND c.gen = ?`, libraryID, gen)
	if err != nil {
		return
	}
	defer rows.Close()
	type scored struct {
		id    string
		score float64
	}
	var all []scored
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			continue
		}
		all = append(all, scored{id, cosine(q, decodeVec(blob))})
	}
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].score > all[i].score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	if len(all) > 20 {
		all = all[:20]
	}
	for i, item := range all {
		ranks[item.id] += 1 / (60 + float64(i+1))
	}
}

func encodeVec(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

func decodeVec(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

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

func embedSig(model string, vecs [][]float32) string {
	dim := 0
	if len(vecs) > 0 {
		dim = len(vecs[0])
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%d", model, dim)))
	return hex.EncodeToString(sum[:8]) + fmt.Sprintf(":%d", dim)
}

func dimOf(sig string) int {
	i := strings.LastIndex(sig, ":")
	if i < 0 {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(sig[i+1:], "%d", &n)
	return n
}

func genLabel(gen int) string { return fmt.Sprintf("%d", gen) }

var keySeq atomic.Uint64

func newKey(prefix string) string {
	n := keySeq.Add(1)
	return fmt.Sprintf("%s-%s-%d", prefix, time.Now().UTC().Format("20060102T150405"), n)
}
