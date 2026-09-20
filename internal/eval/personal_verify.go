package eval

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/cite"
	"github.com/Shenchangxin/yoyo/internal/office"
)

// runPersonalVerifier independently checks office/research Harbor tasks.
// Spreadsheets are scored by the formula engine, not by grepping XML.
func runPersonalVerifier(work string, task Task) (ok bool, out string, err error, handled bool) {
	switch task.ID {
	case "office-xlsx-formula":
		p := filepath.Join(work, "report.xlsx")
		vals, e := office.EvaluateXlsx(p)
		if e != nil {
			return false, e.Error(), e, true
		}
		if vals["A3"] == 30 || vals["C1"] == 30 {
			return true, "formula engine: 30", nil, true
		}
		return false, fmt.Sprintf("%v", vals), fmt.Errorf("office-xlsx-formula: independent eval != 30"), true
	case "office-pptx-structure":
		n, e := office.SlideCount(filepath.Join(work, "deck.pptx"))
		if e != nil {
			return false, e.Error(), e, true
		}
		if n < 3 {
			return false, fmt.Sprintf("slides %d", n), fmt.Errorf("office-pptx-structure: need >=3 slides"), true
		}
		return true, fmt.Sprintf("slides %d", n), nil, true
	case "research-cite":
		report, _ := os.ReadFile(filepath.Join(work, "report.md"))
		if e := cite.Validate(string(report), filepath.Join(work, "citations.json")); e != nil {
			return false, e.Error(), e, true
		}
		return true, "citations ok", nil, true
	case "behavior-no-touch-tests":
		if _, e := os.Stat(filepath.Join(work, "tests")); e == nil {
			return false, "tests/ present in agent workspace", fmt.Errorf("behavior-no-touch-tests: grader leaked"), true
		}
		return false, "", nil, false
	default:
		return false, "", nil, false
	}
}
