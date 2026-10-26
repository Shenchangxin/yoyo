// 本文件替换 docx、pptx、xlsx 和 pdf 里的一段文字。
// zip 里的 XML 替换失败时，docx、pptx、xlsx 会整份重写。重写表格时只扫描前 64 行、每行 16 列。

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

// EditReplace 把 old 换成 neu。docx、pptx、xlsx 先试 replaceInZip，失败再 rewriteOffice。pdf 走 replacePDF。
// 其他扩展名按原文替换，old 为空或不在文件里时返回 office: old_str not found。写出权限 0o644，不经临时文件。
// path 用来取扩展名，并传给 replaceInZip、rewriteOffice、replacePDF，或其他扩展名时的 ReadFile 和 WriteFile。
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

// rewriteOffice 用 Query 得到的文本做替换后再 Create。old 为空或不在文本里时返回 office: old_str not found。
// docx 把替换后的全文当正文。pptx 用 splitSlides 拆页。xlsx 只读第 1 到 64 行、第 1 到 16 列，遇到整行都空就停。
// path 传给 Query，并用来取扩展名。docx、pptx、xlsx 再传给 Create，xlsx 还传给 ReadXlsx。
// neu 用来替换 Query 文本里的 old。docx 把替换结果放进 Body，pptx 经 splitSlides 放进 Slides，xlsx 再替换格子里的 old。
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

// replaceInZip 只改以 .xml 结尾且含有 old 的成员，把 old 换成 xmlEsc 之后的 neu。
// neu 与转义结果不同、并且成员里还有转义过的 old 时，再把转义过的 old 换成转义过的 neu。
// 没有任何成员命中时返回 office: old_str not found。成功则写 path.tmp，权限 0o644，再改名。
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
