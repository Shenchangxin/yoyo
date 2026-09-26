package preview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/office"
)

func Render(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx":
		cells, err := office.ReadXlsx(path)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("# spreadsheet\n\n")
		for k, v := range cells {
			fmt.Fprintf(&b, "- %s: %s\n", k, v)
		}
		if vals, err := office.EvaluateXlsx(path); err == nil {
			b.WriteString("\n# evaluated\n")
			for k, v := range vals {
				fmt.Fprintf(&b, "- %s = %g\n", k, v)
			}
		}
		return b.String(), nil
	case ".docx", ".pptx":
		text, err := office.Query(path)
		if err != nil {
			return "", err
		}
		return text, nil
	case ".pdf":
		return office.Query(path)
	default:
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if len(raw) > 12_000 {
			raw = raw[:12_000]
		}
		return string(raw), nil
	}
}
