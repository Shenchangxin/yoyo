package eval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/cite"
	"github.com/Shenchangxin/yoyo/internal/office"
)

func TestOfficeXlsxVerifierUsesFormulaEngine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "report.xlsx")
	if err := office.Create(p, office.CreateReq{
		Kind: office.KindXlsx,
		Rows: [][]string{{"10"}, {"20"}, {"=SUM(A1:A2)"}},
	}); err != nil {
		t.Fatal(err)
	}
	ok, out, err, handled := runPersonalVerifier(dir, Task{ID: "office-xlsx-formula"})
	if !handled || err != nil || !ok {
		t.Fatalf("handled=%v ok=%v err=%v out=%s", handled, ok, err, out)
	}
}

func TestOfficeXlsxVerifierRejectsStringOnlySheet(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "report.xlsx")
	if err := office.Create(p, office.CreateReq{
		Kind: office.KindXlsx,
		Rows: [][]string{{"SUM", "looks like a table"}},
	}); err != nil {
		t.Fatal(err)
	}
	ok, _, err, handled := runPersonalVerifier(dir, Task{ID: "office-xlsx-formula"})
	if !handled || ok || err == nil {
		t.Fatal("string-only sheet must fail independent eval")
	}
}

func TestResearchCiteVerifier(t *testing.T) {
	dir := t.TempDir()
	src := cite.HashExcerpt("https://example.com/a", "water boils at 100")
	if err := cite.Write(filepath.Join(dir, "citations.json"), []cite.Source{src}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte("water boils at 100 C. https://example.com/a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, out, err, handled := runPersonalVerifier(dir, Task{ID: "research-cite"})
	if !handled || !ok || err != nil {
		t.Fatalf("ok=%v err=%v out=%s", ok, err, out)
	}
}
