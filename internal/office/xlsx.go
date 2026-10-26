// 本文件读取 xlsx 的 sheet1，并计算 SUM、AVERAGE 和四则运算。
// 也从 docx、pptx 的 XML 里抽出纯文本。SlideCount 按空行统计演示页数。

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

// xlsxCell 是 sheet XML 里的一个 c 元素。
type xlsxCell struct {
	Ref string `xml:"r,attr"`
	T   string `xml:"t,attr"`
	V   string `xml:"v"`
	F   string `xml:"f"`
	Is  *struct {
		T string `xml:"t"`
	} `xml:"is"`
}

// ReadXlsx 读取第一张名为 worksheets/sheet1.xml 的工作表。找不到时返回 office: no sheet1。
// 有内联字符串时用它。有公式时值写成等号加公式。地址为空的格子不放入结果。
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

// EvaluateXlsx 先 ReadXlsx，再把公式算成 float64。地址在计算时转成大写。
// 非公式且能解析成浮点的格子先放进数值表。公式按映射顺序计算，某个公式失败则整次返回错误。
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

// cellRe 匹配一段全大写列名加行号，供不含运算符的单元格引用使用。
var cellRe = regexp.MustCompile(`[A-Z]+\d+`)

// evalFormula 计算一条已经去掉等号、并转成大写的公式。
// SUM 支持冒号区域和逗号列表。AVERAGE 只按逗号拆，空列表返回 empty average。
// 纯数字直接解析。没有加减乘除且整段像单元格时走 cellNum。
// 四则运算只在第一个运算符处切开，除以 0 返回 div0。无法识别时返回 unsupported formula。
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

// splitBin 在第一个 +、-、* 或 / 处切开。运算符在开头时返回假，避免把负号当成二元运算。
func splitBin(expr string) (string, string, string, bool) {
	for _, op := range []string{"+", "-", "*", "/"} {
		i := strings.Index(expr, op)
		if i > 0 {
			return expr[:i], op, expr[i+1:], true
		}
	}
	return "", "", "", false
}

// cellNum 读取一个单元格的数值。地址转成大写。已有数值直接用。
// 值以等号开头时递归 evalFormula，并把结果写回 vals。不能解析成浮点时返回 cell 加地址 not numeric。
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

// expandRange 展开两个地址之间的矩形，列和行都会先排成从小到大。地址由 splitAddr 解析。
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

// splitAddr 把地址拆成列号和行号。列从 A 计为 1。行用剩下的十进制文本，解析失败时行号是 0。
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

// readDocxText 抽出 word/document.xml 里的文本。
func readDocxText(path string) (string, error) {
	return zipXMLText(path, "word/document.xml")
}

// readPptxText 把 ppt/slides/slide 下每个 xml 的文本拼起来，页与页之间空一行。
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

// zipXMLText 打开 zip 里名字完全等于 name 的成员，并去掉标签。没有这个成员时返回 os.ErrNotExist。
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

// stripXMLText 丢掉标签，标签结束处补一个空格，再把连续空白收成一个空格。
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

// SlideCount 用 readPptxText 的结果按空行计数，空段不计。读取失败时返回 0 和该错误。
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
