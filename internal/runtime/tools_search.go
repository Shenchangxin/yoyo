package runtime

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var skipDirNames = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	".yoyo": true, "target": true, ".next": true, "__pycache__": true,
}

func globWalk(root, pattern string, limit int) ([]string, error) {
	pattern = filepath.ToSlash(pattern)
	ign := loadIgnore(root)
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if ign.skipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if ign.skipFile(relSlash, info.Name()) {
			return nil
		}
		ok, _ := filepath.Match(pattern, relSlash)
		if !ok {
			ok, _ = filepath.Match(pattern, info.Name())
		}
		if !ok && strings.Contains(pattern, "**") {
			ok = matchStarStar(pattern, relSlash)
		}
		if !ok {
			return nil
		}
		out = append(out, relSlash)
		if len(out) >= limit {
			return errStopWalk
		}
		return nil
	})
	if err == errStopWalk {
		err = nil
	}
	return out, err
}

func matchStarStar(pattern, rel string) bool {
	pattern = strings.ReplaceAll(pattern, "**/*", "*")
	pattern = strings.ReplaceAll(pattern, "**/", "")
	pattern = strings.ReplaceAll(pattern, "**", "*")
	ok, _ := filepath.Match(pattern, rel)
	if ok {
		return true
	}
	base := rel
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		base = rel[i+1:]
	}
	ok, _ = filepath.Match(pattern, base)
	return ok
}

func grepWalk(root, ws string, re *regexp.Regexp, globPat string, limit int) (string, error) {
	var b strings.Builder
	n := 0
	ign := loadIgnore(ws)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if ign.skipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(ws, path)
		if err != nil {
			rel, _ = filepath.Rel(root, path)
		}
		relSlash := filepath.ToSlash(rel)
		if ign.skipFile(relSlash, info.Name()) {
			return nil
		}
		if globPat != "" {
			ok, _ := filepath.Match(globPat, info.Name())
			if !ok {
				ok, _ = filepath.Match(globPat, relSlash)
			}
			if !ok {
				return nil
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		head := make([]byte, 512)
		nr, _ := f.Read(head)
		if hasNull(head[:nr]) {
			return nil
		}
		if _, err := f.Seek(0, 0); err != nil {
			return nil
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			if re.MatchString(sc.Text()) {
				fmtLine(&b, relSlash, lineNo, sc.Text())
				n++
				if n >= limit {
					return errStopWalk
				}
			}
		}
		return nil
	})
	if err == errStopWalk {
		err = nil
	}
	return b.String(), err
}

func fmtLine(b *strings.Builder, rel string, n int, line string) {
	if len(line) > 240 {
		line = line[:240] + "…"
	}
	b.WriteString(rel)
	b.WriteByte(':')
	b.WriteString(itoa(n))
	b.WriteByte(':')
	b.WriteString(line)
	b.WriteByte('\n')
}

func hasNull(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

type stopWalk struct{}

func (stopWalk) Error() string { return "stop" }

var errStopWalk error = stopWalk{}
