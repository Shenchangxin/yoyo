// 本文件生成只有一页的 PDF，并按字面括号抽出文本。
// 页面媒体框是 0 0 612 792。正文字号 12，起点是 72 720，字体是 Helvetica。文件头是 %PDF-1.4。

package office

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// encodePDF 生成一页 PDF。标题为空时用 Document。正文和标题行依次接在标题后面，中间用换行。
// 内容流是 BT /F1 12 Tf 72 720 Td 加括号文本。目录、页面和字体共 5 个对象，根是对象 1。
// req 的 Title 为空时用 Document。Body 非空时接在标题后，Headings 再逐行接上。
func encodePDF(req CreateReq) ([]byte, error) {
	title := req.Title
	if title == "" {
		title = "Document"
	}
	var body strings.Builder
	body.WriteString(title)
	if req.Body != "" {
		body.WriteString("\n")
		body.WriteString(req.Body)
	}
	for _, h := range req.Headings {
		body.WriteString("\n")
		body.WriteString(h)
	}
	text := pdfEscape(body.String())
	stream := "BT /F1 12 Tf 72 720 Td (" + text + ") Tj ET"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		"<< /Length " + strconv.Itoa(len(stream)) + " >>\nstream\n" + stream + "\nendstream",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offs := make([]int, len(objs)+1)
	for i, obj := range objs {
		offs[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offs[i])
	}
	fmt.Fprintf(&buf, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return buf.Bytes(), nil
}

// readPDFText 依次取出文件里每一对圆括号中的文本，并做 pdfUnescape，段与段之间加换行。
// 一对括号都没有时，返回去掉首尾空白的原始文件文本。
// path 用 os.ReadFile 读取。
func readPDFText(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(raw)
	var b strings.Builder
	for {
		i := strings.Index(s, "(")
		if i < 0 {
			break
		}
		j := strings.Index(s[i+1:], ")")
		if j < 0 {
			break
		}
		b.WriteString(pdfUnescape(s[i+1 : i+1+j]))
		b.WriteByte('\n')
		s = s[i+1+j+1:]
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return strings.TrimSpace(string(raw)), nil
	}
	return out, nil
}

// replacePDF 在文件字节里替换。先把 pdfEscape 之后的 old 换成 pdfEscape 之后的 neu，再把原文 old 换成 pdfEscape 之后的 neu。
// 两种形式都不存在时返回 office: old_str not found。直接覆盖 path，不重建交叉引用表。
func replacePDF(path, old, neu string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(raw)
	if !strings.Contains(s, pdfEscape(old)) && !strings.Contains(s, old) {
		return fmt.Errorf("office: old_str not found")
	}
	s = strings.ReplaceAll(s, pdfEscape(old), pdfEscape(neu))
	s = strings.ReplaceAll(s, old, pdfEscape(neu))
	return os.WriteFile(path, []byte(s), 0o644)
}

// pdfEscape 转义反斜杠和圆括号。换行换成 ) Tj T* (，以便在同一内容流里换行。
func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	s = strings.ReplaceAll(s, "\n", `\) Tj T* (`)
	return s
}

// pdfUnescape 把 \) 和 \( 还原成括号，再把 \\ 还原成反斜杠。
func pdfUnescape(s string) string {
	s = strings.ReplaceAll(s, `\)`, `)`)
	s = strings.ReplaceAll(s, `\(`, `(`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
