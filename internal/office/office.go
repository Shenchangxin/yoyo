// Package office 生成并读取 docx、xlsx、pptx 和 pdf。
// Create 按扩展名改写 Kind，先写 path.tmp 再改名。Query 只认这四种扩展名。
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

// Kind 是要生成的办公文档种类。
type Kind string

const (
	// KindDocx 是 Word 文档，值为 docx。
	KindDocx Kind = "docx"
	// KindXlsx 是表格，值为 xlsx。
	KindXlsx Kind = "xlsx"
	// KindPptx 是演示文稿，值为 pptx。
	KindPptx Kind = "pptx"
	// KindPDF 是 PDF，值为 pdf。
	KindPDF Kind = "pdf"
)

// CreateReq 是 Create 的内容。扩展名能认出时，Create 会改写 Kind。
type CreateReq struct {
	Kind     Kind       `json:"kind"`
	Title    string     `json:"title"`
	Body     string     `json:"body"`
	Headings []string   `json:"headings,omitempty"`
	Rows     [][]string `json:"rows,omitempty"`
	Slides   []string   `json:"slides,omitempty"`
}

// Create 按 path 的扩展名改写 Kind，再编码并原子替换文件。
// 扩展名是 .docx、.xlsx、.pptx、.pdf 时覆盖 Kind。编码失败则不写文件。
// 父目录权限 0o755。先写 path.tmp，权限 0o644，再改成 path。
// req 的 Kind 在扩展名是 .docx、.xlsx、.pptx、.pdf 时被改写，再传给 encodeDocx、encodeXlsx、encodePptx 或 encodePDF。
func Create(path string, req CreateReq) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		req.Kind = KindDocx
	case ".xlsx":
		req.Kind = KindXlsx
	case ".pptx":
		req.Kind = KindPptx
	case ".pdf":
		req.Kind = KindPDF
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
	case KindPDF:
		b, err = encodePDF(req)
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

// Query 把文档收成文本。xlsx 是每格一行，格式为地址=值。docx、pptx、pdf 分别走对应的读取函数。
// 其他扩展名返回错误，文案说明这不是办公文档。没有扩展名时错误里的类型写成 unknown。
// path 用来取扩展名。xlsx 传给 ReadXlsx，docx 传给 readDocxText，pptx 传给 readPptxText，pdf 传给 readPDFText。
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
	case ".pdf":
		return readPDFText(path)
	default:
		if ext == "" {
			ext = "unknown"
		}
		return "", fmt.Errorf("office_query: %s is not an office document (docx/xlsx/pptx/pdf); use read_file or browser_open for HTML", ext)
	}
}

// splitSlides 按空行切开文本。一段都没有时返回只含原文的切片。
// text 按空行切开，每段去掉空白，空段丢掉。一段都没有时返回只含原文的切片。
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

// cellAddr 把 1 起始的列和行收成 A1 这种地址。列号按 26 进制，A 是 1。
// col 大于 0 时按 26 进制收成字母，A 对应 1。不大于 0 时没有列字母。
// row 用 strconv.Itoa 接在列字母后面。
func cellAddr(col, row int) string {
	s := ""
	for col > 0 {
		col--
		s = string(rune('A'+col%26)) + s
		col /= 26
	}
	return s + strconv.Itoa(row)
}

// zipw 把名字到正文的映射打成 zip。映射的遍历顺序不稳定。
// files 的每个名字用 zip.Create，正文转成字节写入。
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

// xmlEsc 按 XML 文本转义 s。
func xmlEsc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// encodeDocx 生成一份只含正文部件的 docx。标题为空时用 Document。标题加粗，每个标题行用 Heading1，正文按换行分段。
// req 的 Title 为空时用 Document。Headings 各成一段 Heading1，Body 按换行分段。
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

// encodeXlsx 生成只含 Sheet1 的 xlsx。Rows 为空时用一行标题，再把 Body 按制表符拆成第二行。
// 以 = 开头的格子写成公式，能按十进制浮点解析且非空的写成数字，其余写成内联字符串。
// req 的 Rows 为空时用一行 Title，再把 Body 按制表符拆成第二行。
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

// encodePptx 生成演示文稿。Slides 为空时依次放入非空的标题、正文和 Headings。最后仍为空则用一页 Slide。
// 幻灯片 id 从 256 起，即 255 加页码，页码从 1 计。
// req 的 Slides 为空时依次放入非空的 Title、Body 和全部 Headings。最后仍为空则用一页 Slide。每一页传给 slideXML。
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
		"ppt/presentation.xml":              presentationXML(len(slides)),
		"ppt/_rels/presentation.xml.rels":   presentationRels(len(slides)),
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

// contentTypesPPTX 写出 n 页演示文稿的内容类型，每页一条 slide 覆盖项。
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

// presentationXML 写出 n 页的幻灯片 id 列表。第 i 页的 id 是 255+i，关系 id 是 rId 加页码。
func presentationXML(n int) string {
	var sids strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sids, `<p:sldId id="%d" r:id="rId%d"/>`, 255+i, i)
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst>` + sids.String() + `</p:sldIdLst></p:presentation>`
}

// presentationRels 为每页写一条幻灯片关系，最后再写一条指向 slideMaster1 的关系，id 是 rId 加 n+1。
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

// slideXML 生成一页，正文放在名为 Title 的文本框里。
// text 经 xmlEsc 放进名为 Title 的文本框。
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
