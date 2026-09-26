package office

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	s = strings.ReplaceAll(s, "\n", `\) Tj T* (`)
	return s
}

func pdfUnescape(s string) string {
	s = strings.ReplaceAll(s, `\)`, `)`)
	s = strings.ReplaceAll(s, `\(`, `(`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
