package office

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type xlsxCell struct {
	Ref string `xml:"r,attr"`
	T   string `xml:"t,attr"`
	V   string `xml:"v"`
	F   string `xml:"f"`
	Is  *struct {
		T string `xml:"t"`
	} `xml:"is"`
}

func ReadXlsx(path string) (map[string]string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var sheet []byte
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "worksheets/sheet1.xml") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			sheet, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if sheet == nil {
		return nil, fmt.Errorf("office: no sheet1")
	}
	type row struct {
		C []xlsxCell `xml:"c"`
	}
	var doc struct {
		SheetData struct {
			Row []row `xml:"row"`
		} `xml:"sheetData"`
	}
	if err := xml.Unmarshal(sheet, &doc); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, row := range doc.SheetData.Row {
		for _, c := range row.C {
			v := strings.TrimSpace(c.V)
			if c.Is != nil && c.Is.T != "" {
				v = c.Is.T
			}
			if c.F != "" {
				v = "=" + c.F
			}
			if c.Ref != "" {
				out[c.Ref] = v
			}
		}
	}
	return out, nil
}

func EvaluateXlsx(path string) (map[string]float64, error) {
	cells, err := ReadXlsx(path)
	if err != nil {
		return nil, err
	}
	vals := map[string]float64{}
	raw := map[string]string{}
	for k, v := range cells {
		raw[strings.ToUpper(k)] = v
		if !strings.HasPrefix(v, "=") {
			if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				vals[strings.ToUpper(k)] = n
			}
		}
	}
	for k, v := range raw {
		if strings.HasPrefix(v, "=") {
			n, err := evalFormula(strings.TrimPrefix(v, "="), raw, vals)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			vals[k] = n
		}
	}
	return vals, nil
}

var cellRe = regexp.MustCompile(`[A-Z]+\d+`)

func evalFormula(expr string, raw map[string]string, vals map[string]float64) (float64, error) {
	expr = strings.TrimSpace(strings.ToUpper(expr))
	if strings.HasPrefix(expr, "SUM(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(expr, "SUM("), ")")
		if strings.Contains(inner, ":") {
			parts := strings.SplitN(inner, ":", 2)
			addrs := expandRange(parts[0], parts[1])
			sum := 0.0
			for _, a := range addrs {
				n, err := cellNum(a, raw, vals)
				if err != nil {
					return 0, err
				}
				sum += n
			}
			return sum, nil
		}
		sum := 0.0
		for _, a := range strings.Split(inner, ",") {
			n, err := cellNum(strings.TrimSpace(a), raw, vals)
			if err != nil {
				return 0, err
			}
			sum += n
		}
		return sum, nil
	}
	if strings.HasPrefix(expr, "AVERAGE(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(expr, "AVERAGE("), ")")
		parts := strings.Split(inner, ",")
		if len(parts) == 0 {
			return 0, fmt.Errorf("empty average")
		}
		sum := 0.0
		for _, a := range parts {
			n, err := cellNum(strings.TrimSpace(a), raw, vals)
			if err != nil {
				return 0, err
			}
			sum += n
		}
		return sum / float64(len(parts)), nil
	}
	if n, err := strconv.ParseFloat(expr, 64); err == nil {
		return n, nil
	}
	if cellRe.MatchString(expr) && !strings.ContainsAny(expr, "+-*/") {
		return cellNum(expr, raw, vals)
	}
	left, op, right, ok := splitBin(expr)
	if !ok {
		return 0, fmt.Errorf("unsupported formula %s", expr)
	}
	a, err := evalFormula(left, raw, vals)
	if err != nil {
		return 0, err
	}
	b, err := evalFormula(right, raw, vals)
	if err != nil {
		return 0, err
	}
	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("div0")
		}
		return a / b, nil
	}
	return 0, fmt.Errorf("op")
}

func splitBin(expr string) (string, string, string, bool) {
	for _, op := range []string{"+", "-", "*", "/"} {
		i := strings.Index(expr, op)
		if i > 0 {
			return expr[:i], op, expr[i+1:], true
		}
	}
	return "", "", "", false
}

func cellNum(addr string, raw map[string]string, vals map[string]float64) (float64, error) {
	addr = strings.ToUpper(strings.TrimSpace(addr))
	if n, ok := vals[addr]; ok {
		return n, nil
	}
	v := raw[addr]
	if strings.HasPrefix(v, "=") {
		n, err := evalFormula(strings.TrimPrefix(v, "="), raw, vals)
		if err != nil {
			return 0, err
		}
		vals[addr] = n
		return n, nil
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0, fmt.Errorf("cell %s not numeric", addr)
	}
	vals[addr] = n
	return n, nil
}

func expandRange(a, b string) []string {
	c1, r1 := splitAddr(a)
	c2, r2 := splitAddr(b)
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	var out []string
	for r := r1; r <= r2; r++ {
		for c := c1; c <= c2; c++ {
			out = append(out, cellAddr(c, r))
		}
	}
	return out
}

func splitAddr(s string) (col, row int) {
	s = strings.ToUpper(strings.TrimSpace(s))
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		col = col*26 + int(s[i]-'A'+1)
		i++
	}
	row, _ = strconv.Atoi(s[i:])
	return col, row
}

func readDocxText(path string) (string, error) {
	return zipXMLText(path, "word/document.xml")
}

func readPptxText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	var b strings.Builder
	for _, f := range r.File {
		if !strings.Contains(f.Name, "ppt/slides/slide") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		raw, _ := io.ReadAll(rc)
		rc.Close()
		b.WriteString(stripXMLText(string(raw)))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func zipXMLText(path, name string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		return stripXMLText(string(raw)), nil
	}
	return "", os.ErrNotExist
}

func stripXMLText(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch r {
		case '<':
			in = true
		case '>':
			in = false
			b.WriteByte(' ')
		default:
			if !in {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func SlideCount(path string) (int, error) {
	text, err := readPptxText(path)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range strings.Split(text, "\n\n") {
		if strings.TrimSpace(p) != "" {
			n++
		}
	}
	return n, nil
}
