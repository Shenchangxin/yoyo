// 把导入的文件收成按定位号排的文本单元：PDF 一页一个单元，md、txt、html 和 docx 按标题和段落切成节。
// 这里拥有解析入口、解析缓存、文件哈希和检索分词。切节和切块在 parse_sections.go，HTML 的读法在 parse_html.go，PDF 和 DOCX 的读法在 parse_pdf.go。
// 这里不调用模型，也不决定什么时候重新解析，那由 index.go 和 reading.go 来问。

package learn

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	// MaxImportBytes 是单文件导入上限，100 MiB。
	MaxImportBytes = 100 << 20
	// parserVersion 标在解析缓存上。切节规则或抽取方式变了就换它，旧缓存因此作废。
	parserVersion = "text-v2"

	// UnitPage 表示定位号是页码。PDF 用它。
	UnitPage = "page"
	// UnitSection 表示定位号是节号。md、txt、html、docx 和笔记用它。
	UnitSection = "section"

	// unitOK 是有文字的单元的状态。
	unitOK = "ok"
	// unitNeedsOCR 是抽不出文字的单元的状态，通常是扫描页。
	unitNeedsOCR = "needs_ocr"
)

// PageText 是解析后的一个单元。Page 是从 1 起的定位号。Status 是 ok 或 needs_ocr。
type PageText struct {
	Page   int
	Text   string
	Status string
}

// OutlineEntry 是提纲里的一条。Locator 是标题所在的单元，Offset 是标题在那个单元文字里的符文位置。
// Level 从 1 起。PDF 的条目找不到标题文字时 Offset 是 0。
type OutlineEntry struct {
	Locator int    `json:"locator"`
	Title   string `json:"title"`
	Level   int    `json:"level"`
	Offset  int    `json:"offset"`
}

// parsedDoc 是一份文档解析后的样子。Unit 是 page 或 section，Outline 按定位号和位置排好。
type parsedDoc struct {
	Unit    string
	Units   []PageText
	Outline []OutlineEntry
	Chars   int
}

// hasText 在至少一个单元有文字时为真。
func (d parsedDoc) hasText() bool {
	for _, u := range d.Units {
		if u.Status == unitOK && strings.TrimSpace(u.Text) != "" {
			return true
		}
	}
	return false
}

// parseMeta 是解析缓存里定位号为 0 的那一行：单元种类、单元数、字数和提纲。
type parseMeta struct {
	Unit    string         `json:"unit"`
	Pages   int            `json:"pages"`
	Chars   int            `json:"chars"`
	Outline []OutlineEntry `json:"outline"`
}

// ParseFile 按扩展名抽出文本单元，并返回文件字节的 sha256。path 必须是已有文件，超过 MaxImportBytes 返回错误。
func ParseFile(path string) (string, []PageText, error) {
	sum, doc, err := parseFile(path)
	return sum, doc.Units, err
}

// parseFile 解析 path 指向的文件，返回它的 sha256、文本单元和提纲。扩展名不认识或文件超过 MaxImportBytes 时返回错误。
func parseFile(path string) (string, parsedDoc, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", parsedDoc{}, err
	}
	if st.Size() > MaxImportBytes {
		return "", parsedDoc{}, fmt.Errorf("file exceeds 100MiB")
	}
	switch docKind(path) {
	case kindMarkdown, kindText, kindHTML:
		b, err := os.ReadFile(path)
		if err != nil {
			return "", parsedDoc{}, err
		}
		digest := sha256.Sum256(b)
		sum := hex.EncodeToString(digest[:])
		text := decodeText(b)
		switch docKind(path) {
		case kindMarkdown:
			return sum, sectionize(markdownBlocks(text)), nil
		case kindText:
			return sum, sectionize(textBlocks(text)), nil
		default:
			_, blocks := htmlBlocks(text)
			return sum, sectionize(plainBlocks(blocks)), nil
		}
	case kindDocx:
		sum, err := hashFile(path)
		if err != nil {
			return "", parsedDoc{}, err
		}
		blocks, err := docxBlocks(path)
		if err != nil {
			return "", parsedDoc{}, err
		}
		return sum, sectionize(blocks), nil
	case kindPDF:
		sum, err := hashFile(path)
		if err != nil {
			return "", parsedDoc{}, err
		}
		doc, err := parsePDF(path)
		if err != nil {
			return "", parsedDoc{}, err
		}
		return sum, doc, nil
	default:
		return "", parsedDoc{}, fmt.Errorf("unsupported import type %s", strings.ToLower(filepath.Ext(path)))
	}
}

// 文档的格式。Document.Kind 用这些值，阅读器据此决定怎么渲染。
const (
	kindPDF      = "pdf"
	kindMarkdown = "markdown"
	kindText     = "text"
	kindHTML     = "html"
	kindDocx     = "docx"
	kindNote     = "note"
)

// docKind 按 path 的扩展名返回文档格式。不支持的扩展名返回空串。
func docKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return kindPDF
	case ".md", ".markdown":
		return kindMarkdown
	case ".txt":
		return kindText
	case ".html", ".htm":
		return kindHTML
	case ".docx":
		return kindDocx
	default:
		return ""
	}
}

// supportedExt 在 path 的扩展名可以导入时为真。
func supportedExt(path string) bool {
	return docKind(path) != ""
}

// decodeText 把文件字节 b 读成 UTF-8 文本，并把换行统一成 \n。带 BOM 时按 BOM 解。
// 不是合法 UTF-8 时按 GB18030 试一次，仍不行就把坏字节换成替换符。
func decodeText(b []byte) string {
	var text string
	switch {
	case len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF:
		text = strings.ToValidUTF8(string(b[3:]), "\uFFFD")
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE:
		text = decodeUTF16(b[2:], false)
	case len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF:
		text = decodeUTF16(b[2:], true)
	case utf8.Valid(b):
		text = string(b)
	default:
		if out, err := simplifiedchinese.GB18030.NewDecoder().Bytes(b); err == nil && utf8.Valid(out) {
			text = string(out)
		} else {
			text = strings.ToValidUTF8(string(b), "\uFFFD")
		}
	}
	return normalizeText(text)
}

// decodeUTF16 把去掉 BOM 的 UTF-16 字节 b 解成字符串。bigEndian 决定字节序，落单的最后一个字节丢掉。
func decodeUTF16(b []byte, bigEndian bool) string {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		if bigEndian {
			units = append(units, uint16(b[i])<<8|uint16(b[i+1]))
		} else {
			units = append(units, uint16(b[i+1])<<8|uint16(b[i]))
		}
	}
	return string(utf16.Decode(units))
}

// normalizeText 把 text 里的 \r\n 和 \r 换成 \n，并去掉 NUL。批注的位置按这份文字算，所以所有来源都先过这一步。
func normalizeText(text string) string {
	if strings.ContainsRune(text, '\r') {
		text = strings.ReplaceAll(text, "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
	}
	if strings.ContainsRune(text, 0) {
		text = strings.ReplaceAll(text, "\x00", "")
	}
	return text
}

// hashFile 流式算出 path 的 sha256，不把整个文件读进内存。
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashText 返回 s 的 sha256。笔记的来源哈希和嵌入缓存的键用它。
func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// clipRunes 把 s 截到最多 n 个符文。
func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// cachedParse 读出 sum 这份字节在当前解析版本下的缓存。没有缓存或缓存不完整时 ok 为假。调用方要持有 s.mu。
func (s *Store) cachedParse(sum string) (parsedDoc, bool) {
	rows, err := s.db.Query(`SELECT page_no, text, page_status FROM parses WHERE sha256 = ? AND parser = ? ORDER BY page_no`, sum, parserVersion)
	if err != nil {
		return parsedDoc{}, false
	}
	var doc parsedDoc
	var meta string
	for rows.Next() {
		var p PageText
		if err := rows.Scan(&p.Page, &p.Text, &p.Status); err != nil {
			continue
		}
		if p.Page == 0 {
			meta = p.Text
			continue
		}
		doc.Units = append(doc.Units, p)
	}
	rows.Close()
	var m parseMeta
	if meta == "" || json.Unmarshal([]byte(meta), &m) != nil || m.Pages != len(doc.Units) {
		return parsedDoc{}, false
	}
	doc.Unit, doc.Outline, doc.Chars = m.Unit, m.Outline, m.Chars
	return doc, true
}

// cachedUnit 只读出 sum 的一个单元和整份文档的概要，翻页时不必把整本书读出来。
// locator 是从 1 起的定位号。没有缓存时 ok 为假；定位号超出范围时 ok 为真而单元的 Page 是 0。调用方要持有 s.mu。
func (s *Store) cachedUnit(sum string, locator int) (PageText, parseMeta, bool) {
	var raw string
	if err := s.db.QueryRow(`SELECT text FROM parses WHERE sha256 = ? AND parser = ? AND page_no = 0`, sum, parserVersion).Scan(&raw); err != nil {
		return PageText{}, parseMeta{}, false
	}
	var meta parseMeta
	if json.Unmarshal([]byte(raw), &meta) != nil {
		return PageText{}, parseMeta{}, false
	}
	var unit PageText
	err := s.db.QueryRow(`SELECT page_no, text, page_status FROM parses WHERE sha256 = ? AND parser = ? AND page_no = ?`, sum, parserVersion, locator).
		Scan(&unit.Page, &unit.Text, &unit.Status)
	if err != nil {
		return PageText{}, meta, true
	}
	return unit, meta, true
}

// storeParse 把 doc 写成 sum 的解析缓存，并删掉这份字节在旧解析版本下的缓存。调用方要持有 s.mu。
func (s *Store) storeParse(sum string, doc parsedDoc) error {
	outline := doc.Outline
	if outline == nil {
		outline = []OutlineEntry{}
	}
	meta, err := json.Marshal(parseMeta{Unit: doc.Unit, Pages: len(doc.Units), Chars: doc.Chars, Outline: outline})
	if err != nil {
		return err
	}
	return s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM parses WHERE sha256 = ?`, sum); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO parses (sha256, parser, page_no, text, page_status) VALUES (?, ?, 0, ?, 'meta')`, sum, parserVersion, string(meta)); err != nil {
			return err
		}
		for _, u := range doc.Units {
			if _, err := tx.Exec(`INSERT INTO parses (sha256, parser, page_no, text, page_status) VALUES (?, ?, ?, ?, ?)`,
				sum, parserVersion, u.Page, u.Text, u.Status); err != nil {
				return err
			}
		}
		return nil
	})
}

// isCJK 在 r 是汉字、假名或谚文时为真。这些文字没有空格分词，检索时切成二字片段。
func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

// ftsTokens 把 s 切成检索用的词：中日韩文字切成二字片段，其余字母和数字连成小写的词。索引和查询共用。
// 单独一个汉字自己成词。
func ftsTokens(s string) []string {
	var out []string
	var word, cjk []rune
	flushWord := func() {
		if len(word) > 0 {
			out = append(out, string(word))
			word = word[:0]
		}
	}
	flushCJK := func() {
		if len(cjk) == 1 {
			out = append(out, string(cjk))
		}
		for i := 0; i+1 < len(cjk); i++ {
			out = append(out, string(cjk[i:i+2]))
		}
		cjk = cjk[:0]
	}
	for _, r := range s {
		switch {
		case isCJK(r):
			flushWord()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCJK()
			word = append(word, unicode.ToLower(r))
		default:
			flushWord()
			flushCJK()
		}
	}
	flushWord()
	flushCJK()
	return out
}

// ftsDocument 把 s 写成给 FTS5 索引的一行：分好的词用空格隔开。没有任何词时写成 empty，免得索引行是空的。
func ftsDocument(s string) string {
	doc := strings.Join(ftsTokens(s), " ")
	if doc == "" {
		return "empty"
	}
	return doc
}

// ftsQueryTokens 返回查询 s 去重后的词，最多 64 个。
func ftsQueryTokens(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range ftsTokens(s) {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
		if len(out) == 64 {
			break
		}
	}
	return out
}

// ftsMatch 把查询 s 写成 FTS5 的 MATCH 表达式：各个词之间是 OR。单独一个汉字按前缀匹配，这样它能命中以它开头的二字片段。
// 没有任何词时返回空串。
func ftsMatch(s string) string {
	return ftsMatchWith(s, " OR ")
}

// ftsMatchAll 和 ftsMatch 一样，但各个词之间是 AND：每个词都要出现。只有一个词时和 ftsMatch 相同，这时返回空串，免得同一个查询跑两遍。
func ftsMatchAll(s string) string {
	if len(ftsQueryTokens(s)) < 2 {
		return ""
	}
	return ftsMatchWith(s, " AND ")
}

// ftsMatchWith 把查询 s 的各个词用 op 连成 MATCH 表达式。op 是 " OR " 或 " AND "。
func ftsMatchWith(s, op string) string {
	toks := ftsQueryTokens(s)
	if len(toks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(toks))
	for _, t := range toks {
		quoted := `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
		if r := []rune(t); len(r) == 1 && isCJK(r[0]) {
			quoted += "*"
		}
		parts = append(parts, quoted)
	}
	return strings.Join(parts, op)
}
