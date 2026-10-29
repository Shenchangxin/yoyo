// 把 md、txt、html、docx 和 pdf 收成按页的纯文本。解析结果按文件字节哈希缓存。不调用模型。

package learn

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"

	"github.com/Shenchangxin/yoyo/internal/office"
)

const (
	// MaxImportBytes 是单文件导入上限，100 MiB。
	MaxImportBytes = 100 << 20
	parserVersion  = "text-v1"
)

// PageText 是解析后的一页。Page 从 1 起。笔记和 docx 用 0 表示没有页码。
type PageText struct {
	Page   int
	Text   string
	Status string
}

var tagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// ParseFile 按扩展名抽出文本。path 必须是已有文件。超过 MaxImportBytes 返回错误。
func ParseFile(path string) (string, []PageText, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	if st.Size() > MaxImportBytes {
		return "", nil, fmt.Errorf("file exceeds 100MiB")
	}
	sum, err := hashFile(path)
	if err != nil {
		return "", nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var pages []PageText
	switch ext {
	case ".md", ".txt":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		pages = []PageText{{Page: 0, Text: string(b), Status: "ok"}}
	case ".html", ".htm":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		plain := html.UnescapeString(tagRe.ReplaceAllString(string(b), " "))
		plain = strings.Join(strings.Fields(plain), " ")
		pages = []PageText{{Page: 0, Text: plain, Status: "ok"}}
	case ".docx":
		text, err := office.Query(path)
		if err != nil {
			return "", nil, err
		}
		pages = []PageText{{Page: 0, Text: text, Status: "unpaged"}}
	case ".pdf":
		pages, err = parsePDF(path)
		if err != nil {
			return "", nil, err
		}
	default:
		return "", nil, fmt.Errorf("unsupported import type %s", ext)
	}
	return sum, pages, nil
}

func parsePDF(path string) ([]PageText, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	n := r.NumPage()
	if n > 2000 {
		n = 2000
	}
	out := make([]PageText, 0, n)
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			out = append(out, PageText{Page: i, Status: "needs_ocr"})
			continue
		}
		text, err := p.GetPlainText(nil)
		status := "ok"
		if err != nil || strings.TrimSpace(text) == "" {
			status = "needs_ocr"
			text = ""
		}
		out = append(out, PageText{Page: i, Text: text, Status: status})
	}
	if len(out) == 0 {
		return []PageText{{Page: 1, Status: "needs_ocr"}}, nil
	}
	return out, nil
}

func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// chunkPages 按页切开。每块大约 1200 个符文，重叠 120。单块不超过 4000。
func chunkPages(pages []PageText) []PageText {
	var out []PageText
	for _, p := range pages {
		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}
		runes := []rune(text)
		if len(runes) > 4000 && len(runes) <= 1200 {
			runes = runes[:4000]
		}
		if len(runes) <= 1200 {
			out = append(out, PageText{Page: p.Page, Text: string(runes), Status: p.Status})
			continue
		}
		step := 1200 - 120
		for i := 0; i < len(runes); i += step {
			end := i + 1200
			if end > len(runes) {
				end = len(runes)
			}
			if end-i > 4000 {
				end = i + 4000
			}
			out = append(out, PageText{Page: p.Page, Text: string(runes[i:end]), Status: p.Status})
			if end == len(runes) {
				break
			}
		}
	}
	return out
}

// ftsTokens 把中文切成二字片段，英文收成小写词，供 FTS5 索引和查询共用。
func ftsTokens(s string) []string {
	var out []string
	var latin []rune
	flush := func() {
		if len(latin) == 0 {
			return
		}
		out = append(out, strings.ToLower(string(latin)))
		latin = latin[:0]
	}
	var cjk []rune
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
		case r >= 0x4E00 && r <= 0x9FFF:
			flush()
			cjk = append(cjk, r)
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			flushCJK()
			latin = append(latin, r)
		default:
			flush()
			flushCJK()
		}
	}
	flush()
	flushCJK()
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

func ftsDocument(s string) string {
	return strings.Join(ftsTokens(s), " ")
}

func ftsMatch(s string) string {
	toks := ftsTokens(s)
	if len(toks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(toks))
	for _, t := range toks {
		parts = append(parts, `"`+strings.ReplaceAll(t, `"`, `""`)+`"`)
	}
	return strings.Join(parts, " OR ")
}

func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
