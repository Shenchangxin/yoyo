package office

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Kind string

const (
	KindDocx Kind = "docx"
	KindXlsx Kind = "xlsx"
	KindPptx Kind = "pptx"
)

type CreateReq struct {
	Kind    Kind     `json:"kind"`
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	Headings []string `json:"headings,omitempty"`
	Rows    [][]string `json:"rows,omitempty"`
	Slides  []string `json:"slides,omitempty"`
}

func Create(path string, req CreateReq) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		req.Kind = KindDocx
	case ".xlsx":
		req.Kind = KindXlsx
	case ".pptx":
		req.Kind = KindPptx
	}
	var b []byte
	var err error
	switch req.Kind {
	case KindDocx:
		b, err = encodeDocx(req)
	case KindXlsx:
		b, err = encodeXlsx(req)
	case KindPptx:
		b, err = encodePptx(req)
	default:
		return fmt.Errorf("office: unknown kind %s", req.Kind)
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Query(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx":
		cells, err := ReadXlsx(path)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		for addr, v := range cells {
			fmt.Fprintf(&b, "%s=%s\n", addr, v)
		}
		return b.String(), nil
	case ".docx":
		return readDocxText(path)
	case ".pptx":
		return readPptxText(path)
	default:
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
}

func EditReplace(path, old, neu string) error {
	text, err := Query(path)
	if err != nil {
		return err
	}
	if old == "" || !strings.Contains(text, old) {
		return fmt.Errorf("office: old_str not found")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return Create(path, CreateReq{Kind: KindDocx, Title: filepath.Base(path), Body: strings.ReplaceAll(text, old, neu)})
	case ".pptx":
		slides := splitSlides(strings.ReplaceAll(text, old, neu))
		return Create(path, CreateReq{Kind: KindPptx, Title: filepath.Base(path), Slides: slides})
	case ".xlsx":
		cells, err := ReadXlsx(path)
		if err != nil {
			return err
		}
		var rows [][]string
		for r := 1; r <= 64; r++ {
			var row []string
			empty := true
			for c := 1; c <= 16; c++ {
				v := cells[cellAddr(c, r)]
				v = strings.ReplaceAll(v, old, neu)
				row = append(row, v)
				if v != "" {
					empty = false
				}
			}
			if empty {
				break
			}
			rows = append(rows, row)
		}
		return Create(path, CreateReq{Kind: KindXlsx, Title: filepath.Base(path), Rows: rows})
	default:
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := strings.ReplaceAll(string(raw), old, neu)
		return os.WriteFile(path, []byte(out), 0o644)
	}
}

func splitSlides(text string) []string {
	parts := strings.Split(text, "\n\n")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{text}
	}
	return out
}

func cellAddr(col, row int) string {
	s := ""
	for col > 0 {
		col--
		s = string(rune('A'+col%26)) + s
		col /= 26
	}
	return s + strconv.Itoa(row)
}

func zipw(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func xmlEsc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func encodeDocx(req CreateReq) ([]byte, error) {
	title := req.Title
	if title == "" {
		title = "Document"
	}
	var body strings.Builder
	body.WriteString(`<w:p><w:r><w:rPr><w:b/></w:rPr><w:t>` + xmlEsc(title) + `</w:t></w:r></w:p>`)
	for _, h := range req.Headings {
		body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>` + xmlEsc(h) + `</w:t></w:r></w:p>`)
	}
	for _, para := range strings.Split(req.Body, "\n") {
		body.WriteString(`<w:p><w:r><w:t>` + xmlEsc(para) + `</w:t></w:r></w:p>`)
	}
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + body.String() + `<w:sectPr/></w:body></w:document>`,
	}
	return zipw(files)
}

func encodeXlsx(req CreateReq) ([]byte, error) {
	rows := req.Rows
	if len(rows) == 0 {
		rows = [][]string{{req.Title}, strings.Split(req.Body, "\t")}
	}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	sheet.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		fmt.Fprintf(&sheet, `<row r="%d">`, r+1)
		for c, v := range row {
			addr := cellAddr(c+1, r+1)
			if strings.HasPrefix(v, "=") {
				fmt.Fprintf(&sheet, `<c r="%s"><f>%s</f></c>`, addr, xmlEsc(strings.TrimPrefix(v, "=")))
			} else if _, err := strconv.ParseFloat(v, 64); err == nil && v != "" {
				fmt.Fprintf(&sheet, `<c r="%s" t="n"><v>%s</v></c>`, addr, xmlEsc(v))
			} else {
				fmt.Fprintf(&sheet, `<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, addr, xmlEsc(v))
			}
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/worksheets/sheet1.xml": sheet.String(),
	}
	return zipw(files)
}

func encodePptx(req CreateReq) ([]byte, error) {
	slides := req.Slides
	if len(slides) == 0 {
		if req.Title != "" {
			slides = append(slides, req.Title)
		}
		if req.Body != "" {
			slides = append(slides, req.Body)
		}
		slides = append(slides, req.Headings...)
	}
	if len(slides) == 0 {
		slides = []string{"Slide"}
	}
	files := map[string]string{
		"[Content_Types].xml": contentTypesPPTX(len(slides)),
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`,
		"ppt/presentation.xml": presentationXML(len(slides)),
		"ppt/_rels/presentation.xml.rels": presentationRels(len(slides)),
		"ppt/slideLayouts/slideLayout1.xml": `<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" type="blank"/>`,
		"ppt/slideMasters/slideMaster1.xml": `<p:sldMaster xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"/>`,
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`,
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`,
	}
	for i, s := range slides {
		n := i + 1
		files[fmt.Sprintf("ppt/slides/slide%d.xml", n)] = slideXML(s)
		files[fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", n)] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`
	}
	return zipw(files)
}

func contentTypesPPTX(n int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>`)
	b.WriteString(`<Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>`)
	b.WriteString(`<Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i)
	}
	b.WriteString(`</Types>`)
	return b.String()
}

func presentationXML(n int) string {
	var sids strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sids, `<p:sldId id="%d" r:id="rId%d"/>`, 255+i, i)
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst>` + sids.String() + `</p:sldIdLst></p:presentation>`
}

func presentationRels(n int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i, i)
	}
	fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>`, n+1)
	b.WriteString(`</Relationships>`)
	return b.String()
}

func slideXML(text string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<p:cSld><p:spTree>
<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
<p:grpSpPr/>
<p:sp><p:nvSpPr><p:cNvPr id="2" name="Title"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>
<p:spPr/><p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>` + xmlEsc(text) + `</a:t></a:r></a:p></p:txBody></p:sp>
</p:spTree></p:cSld></p:sld>`
}
