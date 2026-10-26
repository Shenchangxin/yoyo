// 本文件锁住表格公式求值、演示文稿页数、PDF 与 docx 的往返，以及 Query 拒绝 HTML。
// 不调用外部办公软件。由 go test 调用。

package office

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestXlsxFormulaEvaluatesIndependently 要求 SUM(A1:B1) 在 C1 上得到 30。
func TestXlsxFormulaEvaluatesIndependently(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "report.xlsx")
	if err := Create(p, CreateReq{Kind: KindXlsx, Rows: [][]string{{"10", "20", "=SUM(A1:B1)"}}}); err != nil {
		t.Fatal(err)
	}
	vals, err := EvaluateXlsx(p)
	if err != nil {
		t.Fatal(err)
	}
	if vals["C1"] != 30 {
		t.Fatalf("C1=%v", vals)
	}
}

// TestPptxHasSlides 要求三页标题生成的文稿，SlideCount 至少是 3。
func TestPptxHasSlides(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "deck.pptx")
	if err := Create(p, CreateReq{Kind: KindPptx, Slides: []string{"Title", "Findings", "Next"}}); err != nil {
		t.Fatal(err)
	}
	n, err := SlideCount(p)
	if err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Fatalf("slides %d", n)
	}
}

// TestPDFRoundTrip 要求 Query 能读到标题和正文，并且 EditReplace 把 Friday 换成 Monday 后仍能读到。
func TestPDFRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "brief.pdf")
	if err := Create(p, CreateReq{Kind: KindPDF, Title: "Brief", Body: "Ship Friday"}); err != nil {
		t.Fatal(err)
	}
	text, err := Query(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Brief") || !strings.Contains(text, "Ship Friday") {
		t.Fatalf("%q", text)
	}
	if err := EditReplace(p, "Friday", "Monday"); err != nil {
		t.Fatal(err)
	}
	text, err = Query(p)
	if err != nil || !strings.Contains(text, "Monday") {
		t.Fatalf("%q %v", text, err)
	}
}

// TestQueryRejectsHTML 要求对 html 文件返回的错误里含有 not an office document。
func TestQueryRejectsHTML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "game.html")
	if err := os.WriteFile(p, []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Query(p)
	if err == nil || !strings.Contains(err.Error(), "not an office document") {
		t.Fatalf("%v", err)
	}
}

// TestDocxRoundTrip 要求带标题和正文的 docx 经 Query 后不是空串。
func TestDocxRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "n.docx")
	if err := Create(p, CreateReq{Kind: KindDocx, Title: "Minutes", Body: "Decide ship", Headings: []string{"Decisions"}}); err != nil {
		t.Fatal(err)
	}
	text, err := Query(p)
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("empty docx")
	}
}
