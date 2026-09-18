package office

import (
	"path/filepath"
	"testing"
)

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
