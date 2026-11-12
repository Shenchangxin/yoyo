// PDF 和 DOCX 的读法。PDF 一页一个单元，抽不出文字的页标成 needs_ocr，提纲按书签标题在页面文字里找落点。
// DOCX 先收成纯文本再按段落切。这里不做 OCR，也不解析版面坐标。

package learn

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"

	"github.com/Shenchangxin/yoyo/internal/office"
)

const (
	// maxPDFPages 是一份 PDF 最多读多少页。
	maxPDFPages = 3000
	// tocTitles 是一页上出现多少个书签标题就算目录页。目录页不当标题的落点。
	tocTitles = 5
)

// parsePDF 把 path 指向的 PDF 读成按页的文档。坏文件会让底层库崩溃，这里接住并当成错误返回。
func parsePDF(path string) (doc parsedDoc, err error) {
	defer func() {
		if r := recover(); r != nil {
			doc, err = parsedDoc{}, fmt.Errorf("this pdf could not be read: %v", r)
		}
	}()
	f, r, err := pdf.Open(path)
	if err != nil {
		return parsedDoc{}, err
	}
	defer f.Close()
	n := r.NumPage()
	if n > maxPDFPages {
		n = maxPDFPages
	}
	doc = parsedDoc{Unit: UnitPage}
	for i := 1; i <= n; i++ {
		text, status := pdfPageText(r, i), unitOK
		if strings.TrimSpace(text) == "" {
			text, status = "", unitNeedsOCR
		}
		doc.Units = append(doc.Units, PageText{Page: i, Text: text, Status: status})
		doc.Chars += utf8.RuneCountInString(text)
	}
	if len(doc.Units) == 0 {
		doc.Units = []PageText{{Page: 1, Status: unitNeedsOCR}}
	}
	doc.Outline = pdfOutline(r, doc.Units)
	return doc, nil
}

// pdfPageText 抽出第 i 页的文字，一行文字一行。按行抽不出来时退回整页的纯文本。这一页读坏了就返回空串。
func pdfPageText(r *pdf.Reader, i int) (text string) {
	defer func() {
		if recover() != nil {
			text = ""
		}
	}()
	p := r.Page(i)
	if p.V.IsNull() {
		return ""
	}
	var b strings.Builder
	if rows, err := p.GetTextByRow(); err == nil {
		for _, row := range rows {
			var line strings.Builder
			for _, word := range row.Content {
				line.WriteString(word.S)
			}
			if s := strings.TrimRight(line.String(), " \t"); strings.TrimSpace(s) != "" {
				b.WriteString(s)
				b.WriteString("\n")
			}
		}
	}
	if strings.TrimSpace(b.String()) == "" {
		raw, err := p.GetPlainText(nil)
		if err != nil {
			return ""
		}
		b.Reset()
		b.WriteString(raw)
	}
	return strings.TrimSpace(normalizeText(strings.ToValidUTF8(b.String(), "")))
}

// pdfOutline 把 PDF 的书签收成提纲。每个标题从上一个标题落下的那一页起往后找，找到它出现的第一页；目录页跳过。
// 找不到落点的标题丢掉。units 是已经抽好文字的各页。书签读坏了时返回 nil。
func pdfOutline(r *pdf.Reader, units []PageText) (out []OutlineEntry) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	var flat []OutlineEntry
	var walk func(o pdf.Outline, level int)
	walk = func(o pdf.Outline, level int) {
		for _, c := range o.Child {
			if title := clipRunes(headingTitle(c.Title), maxHeadingRunes); title != "" && len(flat) < maxOutline {
				flat = append(flat, OutlineEntry{Title: title, Level: level})
			}
			walk(c, level+1)
		}
	}
	walk(r.Outline(), 1)
	if len(flat) == 0 {
		return nil
	}
	squash := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), "")) }
	pages := make([]string, len(units))
	for i, u := range units {
		pages[i] = squash(u.Text)
	}
	keys := make([]string, len(flat))
	for i, e := range flat {
		keys[i] = squash(e.Title)
	}
	toc := make([]bool, len(pages))
	for i, page := range pages {
		seen := 0
		for _, key := range keys {
			if key != "" && strings.Contains(page, key) {
				seen++
				if seen >= tocTitles {
					toc[i] = true
					break
				}
			}
		}
	}
	from := 0
	for i, e := range flat {
		if keys[i] == "" {
			continue
		}
		for at := from; at < len(pages); at++ {
			if toc[at] || !strings.Contains(pages[at], keys[i]) {
				continue
			}
			e.Locator = units[at].Page
			if pos := strings.Index(units[at].Text, e.Title); pos >= 0 {
				e.Offset = utf8.RuneCountInString(units[at].Text[:pos])
			}
			out = append(out, e)
			from = at
			break
		}
	}
	return out
}

// docxBlocks 把 path 指向的 DOCX 收成纯文本，再按空行切成段落块。
func docxBlocks(path string) ([]block, error) {
	text, err := office.Query(path)
	if err != nil {
		return nil, err
	}
	return textBlocks(normalizeText(text)), nil
}
