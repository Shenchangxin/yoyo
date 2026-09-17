package eval

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type expectFile struct {
	Path     string `toml:"path"`
	Contains string `toml:"contains"`
}

type expectDoc struct {
	File []expectFile `toml:"file"`
}

func runExpect(work, path string) (bool, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, "", err
	}
	var doc expectDoc
	if err := toml.Unmarshal(b, &doc); err != nil {
		return false, "", err
	}
	if len(doc.File) == 0 {
		return false, "", fmtNoExpect
	}
	var missing []string
	for _, f := range doc.File {
		p := f.Path
		if !filepath.IsAbs(p) {
			p = filepath.Join(work, p)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			missing = append(missing, f.Path)
			continue
		}
		if f.Contains != "" && !strings.Contains(strings.ToLower(string(raw)), strings.ToLower(f.Contains)) {
			missing = append(missing, f.Path+" (content)")
		}
	}
	if len(missing) > 0 {
		return false, "missing " + strings.Join(missing, ", "), fmtNoExpect
	}
	return true, "ok", nil
}

var fmtNoExpect = errString("expect failed")

type errString string

func (e errString) Error() string { return string(e) }
