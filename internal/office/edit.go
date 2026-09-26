package office

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func EditReplace(path, old, neu string) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx", ".pptx", ".xlsx":
		if err := replaceInZip(path, old, neu); err == nil {
			return nil
		}
		return rewriteOffice(path, old, neu)
	case ".pdf":
		return replacePDF(path, old, neu)
	default:
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if old == "" || !strings.Contains(string(raw), old) {
			return fmt.Errorf("office: old_str not found")
		}
		out := strings.ReplaceAll(string(raw), old, neu)
		return os.WriteFile(path, []byte(out), 0o644)
	}
}

func rewriteOffice(path, old, neu string) error {
	text, err := Query(path)
	if err != nil {
		return err
	}
	if old == "" || !strings.Contains(text, old) {
		return fmt.Errorf("office: old_str not found")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return Create(path, CreateReq{Kind: KindDocx, Title: filepath.Base(path), Body: strings.ReplaceAll(text, old, neu)})
	case ".pptx":
		slides := splitSlides(strings.ReplaceAll(text, old, neu))
		return Create(path, CreateReq{Kind: KindPptx, Title: filepath.Base(path), Slides: slides})
	case ".xlsx":
		cells, err := ReadXlsx(path)
		if err != nil {
			return err
		}
		var rows [][]string
		for r := 1; r <= 64; r++ {
			var row []string
			empty := true
			for c := 1; c <= 16; c++ {
				v := cells[cellAddr(c, r)]
				v = strings.ReplaceAll(v, old, neu)
				row = append(row, v)
				if v != "" {
					empty = false
				}
			}
			if empty {
				break
			}
			rows = append(rows, row)
		}
		return Create(path, CreateReq{Kind: KindXlsx, Title: filepath.Base(path), Rows: rows})
	default:
		return fmt.Errorf("office: unknown kind")
	}
}

func replaceInZip(path, old, neu string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	files := map[string][]byte{}
	hit := false
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if strings.HasSuffix(f.Name, ".xml") && strings.Contains(string(raw), old) {
			raw = []byte(strings.ReplaceAll(string(raw), old, xmlEsc(neu)))
			if neu != xmlEsc(neu) && strings.Contains(string(raw), xmlEsc(old)) {
				raw = []byte(strings.ReplaceAll(string(raw), xmlEsc(old), xmlEsc(neu)))
			}
			hit = true
		}
		files[f.Name] = raw
	}
	if !hit {
		return fmt.Errorf("office: old_str not found")
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(body); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
