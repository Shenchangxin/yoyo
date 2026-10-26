// glob 和 grep 的工作区遍历。跳过 skipDirNames 和 loadIgnore 命中的路径。含 NUL 的文件不参与 grep。

package runtime

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// skipDirNames 是遍历时跳过的目录名：.git、node_modules、vendor、dist、.yoyo、target、.next、__pycache__。copyTree 也用它。
var skipDirNames = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	".yoyo": true, "target": true, ".next": true, "__pycache__": true,
}

// globWalk 按 filepath.Match 匹配相对路径或文件名。模式含 ** 时再试 matchStarStar。达到 limit 就停止。忽略目录和文件被跳过。
// root 交给 loadIgnore 和 filepath.Walk，并用来算相对路径。
// pattern 先换成斜杠，再对相对路径和文件名做 filepath.Match；含 ** 时再交给 matchStarStar。
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

// matchStarStar 把 **/*、**/ 和 ** 收成 * 后再做 filepath.Match，先对相对路径，再对基名。
// pattern 把 **/*、**/ 和 ** 收成 * 后再做 filepath.Match。
// rel 先整段匹配，不中再取最后一个斜杠后面的基名匹配。
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

// grepWalk 逐行匹配。globPat 非空时先按文件名或相对路径过滤。文件头 512 字节含 NUL 则跳过。扫描缓冲最大 1MiB。单行超过 240 字节会截断。达到 limit 就停止。
// root 交给 filepath.Walk；相对路径先对 ws 取，失败再对 root 取。
// ws 交给 loadIgnore，并作为 filepath.Rel 的第一选择。
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

// fmtLine 写成 相对路径:行号:内容。内容长于 240 字节时截断并加省略号。
// b 写入相对路径、行号和内容，并加换行。
// rel 写在行首。
// line 长于 240 字节时截到 240 再写入。
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

// hasNull 在字节里出现 0 时返回真。
// b 逐字节看有没有 0。
func hasNull(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

// stopWalk 是提前结束 Walk 的错误，Error 返回 stop。
type stopWalk struct{}

// Error 返回 stop，供 Walk 识别提前结束。
func (stopWalk) Error() string { return "stop" }

// errStopWalk 是 stopWalk 的实例。globWalk 和 grepWalk 遇到它会当成正常停止。
var errStopWalk error = stopWalk{}
