// 按目录列表下载 Work Buddy 技能文件。单文件 1 MiB，整包 8 MiB，最多 80 个文件、6 层深。

package skillmarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"
)

const (
	maxPackFiles = 80
	maxPackBytes = 8 << 20
	maxFileBytes = 1 << 20
	maxPackDepth = 6
)

// PackFile 是技能目录里的一个相对路径和它的字节。
type PackFile struct {
	Rel  string
	Data []byte
}

type ghEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
	Size        int    `json:"size"`
}

// FetchPack 先按目录列表下载，失败或不含根上的 SKILL.md 时再单独取 SKILL.md。不执行任何文件。slug 净化后为空返回 "invalid skill slug"。
func FetchPack(slug string) ([]PackFile, string, error) {
	slug = SanitizeSlug(slug)
	if slug == "" {
		return nil, "", fmt.Errorf("invalid skill slug")
	}
	files, warn, err := listAndDownload(slug)
	if err == nil && packHasSkillMD(files) {
		return files, warn, nil
	}
	raw, getErr := Get(SkillURL(slug))
	if getErr != nil {
		if err != nil {
			return nil, warn, err
		}
		return nil, warn, getErr
	}
	if len(files) > 0 {
		c := collectorFrom(files)
		if addErr := c.add("SKILL.md", raw); addErr != nil {
			return nil, joinWarn(warn, c.warn()), addErr
		}
		msg := "catalog tree had no SKILL.md"
		if err != nil {
			msg = "pack listing incomplete"
		}
		return c.files, joinWarn(msg, warn, c.warn()), nil
	}
	msg := "installed SKILL.md only"
	if err != nil {
		msg = "pack listing failed; " + msg
	} else {
		msg = "catalog tree had no SKILL.md; " + msg
	}
	return []PackFile{{Rel: "SKILL.md", Data: raw}}, joinWarn(warn, msg), nil
}

// listAndDownload 从技能根目录往下走。深度超过 6 就跳过。SKILL.md 下载失败会返回错误，其他文件失败只记入警告。空目录返回 "empty pack"。
// slug 交给 listContents 和 downloadPackFile。
func listAndDownload(slug string) ([]PackFile, string, error) {
	c := packCollector{}
	var walk func(rel string, depth int) error
	walk = func(rel string, depth int) error {
		if depth > maxPackDepth {
			c.skip(rel, "nested too deep")
			return nil
		}
		entries, err := listContents(slug, rel)
		if err != nil {
			if rel == "" {
				return err
			}
			c.skip(rel, err.Error())
			return nil
		}
		for _, e := range entries {
			name := SanitizeRelPart(e.Name)
			if name == "" {
				if strings.TrimSpace(e.Name) != "" {
					c.skip(e.Name, "unsafe file name")
				}
				continue
			}
			child := name
			if rel != "" {
				child = rel + "/" + name
			}
			switch strings.ToLower(strings.TrimSpace(e.Type)) {
			case "dir", "tree":
				if err := walk(child, depth+1); err != nil {
					return err
				}
			case "file", "blob":
				if !packAllowed(child) {
					continue
				}
				if e.Size > maxFileBytes && !isSkillMDRel(child) {
					c.skip(child, "larger than 1MiB")
					continue
				}
				raw, err := downloadPackFile(slug, child, e.DownloadURL)
				if err != nil {
					if isSkillMDRel(child) {
						return fmt.Errorf("%s: %w", child, err)
					}
					c.skip(child, err.Error())
					continue
				}
				if err := c.add(child, raw); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk("", 0); err != nil {
		if packHasSkillMD(c.files) {
			c.skip(".", err.Error())
			return c.files, c.warn(), nil
		}
		return c.files, c.warn(), err
	}
	if len(c.files) == 0 {
		return nil, c.warn(), fmt.Errorf("empty pack")
	}
	return c.files, c.warn(), nil
}

// listContents 拉取目录 JSON。正文不是以 [ 开头时返回 "contents: expected directory listing"。
// slug 和 rel 一起交给 contentsURL。
// rel 和 slug 一起交给 contentsURL。
func listContents(slug, rel string) ([]ghEntry, error) {
	u := contentsURL(slug, rel)
	raw, err := Get(u)
	if err != nil {
		return nil, err
	}
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim[0] != '[' {
		return nil, fmt.Errorf("contents: expected directory listing")
	}
	var entries []ghEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// downloadPackFile 在 downloadURL 是带主机的 https 时先用它。失败或不是这种地址时改用 packFileURL。
// slug 在 downloadURL 不是带主机的 https 或 Get 失败时，和 rel 一起交给 packFileURL。
// rel 和 slug 一起交给 packFileURL。
func downloadPackFile(slug, rel, downloadURL string) ([]byte, error) {
	if u, err := url.Parse(strings.TrimSpace(downloadURL)); err == nil && u.Scheme == "https" && u.Host != "" {
		if b, err := Get(u.String()); err == nil {
			return b, nil
		}
	}
	return Get(packFileURL(slug, rel))
}

// contentsURL 拼出列目录的地址。ContentsBase 为空时从 Base 推导。
// slug 和 rel 接在 skills 后面，交给 joinEscapedPath。
// rel 和 slug 一起交给 joinEscapedPath。
func contentsURL(slug, rel string) string {
	base := strings.TrimRight(ContentsBase, "/")
	if base == "" {
		base = derivedContentsBase(Base)
	}
	return joinEscapedPath(base, "skills", slug, rel)
}

// packFileURL 拼出 Base/skills/<slug>/<rel>，路径段会做转义。
func packFileURL(slug, rel string) string {
	return joinEscapedPath(strings.TrimRight(Base, "/"), "skills", slug, rel)
}

// joinEscapedPath 用斜杠连接并对每一段做 PathEscape。
// base 去掉末尾斜杠后作为开头。
// segs 每段把反斜杠换成斜杠，空段跳过，其余逐段 PathEscape 后用斜杠接上。
func joinEscapedPath(base string, segs ...string) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(base, "/"))
	for _, s := range segs {
		s = strings.Trim(strings.ReplaceAll(s, "\\", "/"), "/")
		if s == "" {
			continue
		}
		for _, part := range strings.Split(s, "/") {
			if part == "" {
				continue
			}
			b.WriteByte('/')
			b.WriteString(url.PathEscape(part))
		}
	}
	return b.String()
}

// derivedContentsBase 把 raw.githubusercontent.com/<owner>/<repo>/... 换成 api.github.com 的 contents 地址。其他基址则在末尾加 /contents。
// rawBase 去掉末尾斜杠；以 https://raw.githubusercontent.com/ 开头且能拆出 owner 和 repo 时换成 api.github.com 的 contents 地址，否则在末尾加 /contents。
func derivedContentsBase(rawBase string) string {
	rawBase = strings.TrimRight(rawBase, "/")
	const prefix = "https://raw.githubusercontent.com/"
	if strings.HasPrefix(rawBase, prefix) {
		rest := strings.TrimPrefix(rawBase, prefix)
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return "https://api.github.com/repos/" + parts[0] + "/" + parts[1] + "/contents"
		}
	}
	return rawBase + "/contents"
}

// derivedTreeURL 从 raw 地址推出递归 git tree。认不出 owner、repo、ref 时返回空串。
// rawBase 去掉末尾斜杠；以 https://raw.githubusercontent.com/ 开头且至少有三段时拼出递归 git tree，否则返回空串。
func derivedTreeURL(rawBase string) string {
	rawBase = strings.TrimRight(rawBase, "/")
	const prefix = "https://raw.githubusercontent.com/"
	if strings.HasPrefix(rawBase, prefix) {
		rest := strings.TrimPrefix(rawBase, prefix)
		parts := strings.Split(rest, "/")
		if len(parts) >= 3 && parts[0] != "" && parts[1] != "" {
			ref := parts[2]
			return "https://api.github.com/repos/" + parts[0] + "/" + parts[1] + "/git/trees/" + ref + "?recursive=1"
		}
	}
	return ""
}

// packHasSkillMD 在某个文件的基名是 SKILL.md 且相对路径里没有斜杠时返回 true。
// files 里某个 Rel 的基名是 SKILL.md 且路径不含斜杠时返回 true。
func packHasSkillMD(files []PackFile) bool {
	for _, f := range files {
		if strings.EqualFold(path.Base(f.Rel), "SKILL.md") && !strings.Contains(f.Rel, "/") {
			return true
		}
	}
	return false
}

// packAllowed 拒绝空路径、..、绝对路径、点开头的文件名，以及 exe、dll、so、dylib、bin、压缩包、bat、cmd、msi、scr、com、wasm。无扩展名时允许 scripts/ 下的文件、license、notice、readme、makefile、dockerfile、gemfile、procfile 和 skill.md。
// rel 转成小写并把反斜杠换成斜杠，空、含 ..、以 / 开头或以点开头的文件名返回 false。
func packAllowed(rel string) bool {
	rel = strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return false
	}
	base := path.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	ext := path.Ext(rel)
	switch ext {
	case ".exe", ".dll", ".so", ".dylib", ".bin", ".zip", ".tar", ".gz", ".tgz", ".7z", ".rar", ".bat", ".cmd", ".msi", ".scr", ".com", ".wasm":
		return false
	}
	if ext == "" {
		if strings.HasPrefix(rel, "scripts/") {
			return true
		}
		switch base {
		case "license", "notice", "readme", "makefile", "dockerfile", "gemfile", "procfile":
			return true
		}
		return strings.EqualFold(base, "skill.md")
	}
	switch ext {
	case ".md", ".txt", ".py", ".js", ".ts", ".tsx", ".jsx", ".mjs", ".cjs", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".css",
		".sh", ".ps1", ".rb", ".pl", ".r", ".sql", ".csv", ".tsv",
		".go", ".rs", ".java", ".kt", ".swift", ".cs", ".vue",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
		".jinja", ".j2", ".template":
		return true
	default:
		return false
	}
}

// isSkillMDRel 在去掉两侧斜杠后等于 SKILL.md（不区分大小写）时返回 true。
// rel 把反斜杠换成斜杠并去掉两端斜杠后，与 SKILL.md 不区分大小写比较。
func isSkillMDRel(rel string) bool {
	rel = strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/")
	return strings.EqualFold(rel, "SKILL.md")
}

// SanitizeRelPart 只接受一段路径。允许字母、数字、空格和 - _ . + ( ) [ ]，不能以点开头，也不能含斜杠或 Windows 非法字符。
func SanitizeRelPart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/\\")
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, `/\:*?"<>|`) {
		return ""
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' || r == ' ' || r == '+' || r == '(' || r == ')' || r == '[' || r == ']' {
			continue
		}
		return ""
	}
	if s[0] == '.' {
		return ""
	}
	return s
}

type packCollector struct {
	files []PackFile
	total int
	skips []string
}

// collectorFrom 用已有文件做收集器，并按字节数累加总量。
// files 复制进收集器，并把每个文件的字节数加进总量。
func collectorFrom(files []PackFile) packCollector {
	c := packCollector{files: append([]PackFile(nil), files...)}
	for _, f := range files {
		c.total += len(f.Data)
	}
	return c
}

// skip 记下被跳过的相对路径和原因。路径为空白时不记。
// rel 去掉两端空白，空则不记。
// reason 去掉两端空白后非空时接在路径后面。
func (c *packCollector) skip(rel, reason string) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return
	}
	msg := rel
	if reason = strings.TrimSpace(reason); reason != "" {
		msg += ": " + reason
	}
	c.skips = append(c.skips, msg)
}

// add 收入一个文件。单文件超过 1 MiB、整包超过 8 MiB 或文件数达到 80 时，SKILL.md 返回错误，其他文件只跳过。同名（不区分大小写）会替换。
// rel 把反斜杠换成斜杠并去掉两端斜杠，再交给 packAllowed 和 scanPackFile。
// data 的长度超过 maxFileBytes，或加上已有总量超过 maxPackBytes 时，根上的 SKILL.md 返回错误，其他文件跳过。
func (c *packCollector) add(rel string, data []byte) error {
	rel = strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/")
	required := isSkillMDRel(rel)
	if !packAllowed(rel) {
		if required {
			return fmt.Errorf("%s not allowed", rel)
		}
		return nil
	}
	if len(data) > maxFileBytes {
		if required {
			return fmt.Errorf("%s larger than 1MiB", rel)
		}
		c.skip(rel, "larger than 1MiB")
		return nil
	}
	if err := scanPackFile(rel, data); err != nil {
		if required {
			return fmt.Errorf("%s: %w", rel, err)
		}
		c.skip(rel, err.Error())
		return nil
	}
	if c.total+len(data) > maxPackBytes || len(c.files) >= maxPackFiles {
		if required {
			return fmt.Errorf("pack too large")
		}
		c.skip(rel, "pack budget")
		return nil
	}
	for i, f := range c.files {
		if strings.EqualFold(f.Rel, rel) {
			c.total += len(data) - len(f.Data)
			c.files[i] = PackFile{Rel: rel, Data: data}
			return nil
		}
	}
	c.files = append(c.files, PackFile{Rel: rel, Data: data})
	c.total += len(data)
	return nil
}

// warn 汇总跳过的文件。超过 4 条时只列出前 4 条并加上省略号。没有跳过则返回空串。
func (c *packCollector) warn() string {
	if len(c.skips) == 0 {
		return ""
	}
	shown := c.skips
	extra := ""
	if len(shown) > 4 {
		shown = shown[:4]
		extra = "; …"
	}
	msg := "skipped " + strconv.Itoa(len(c.skips)) + " file"
	if len(c.skips) != 1 {
		msg += "s"
	}
	return msg + " (" + strings.Join(shown, "; ") + extra + ")"
}

// joinWarn 用 "; " 接上非空且不重复的警告。
// parts 去掉两端空白，空的和重复的丢掉，其余用 "; " 接上。
func joinWarn(parts ...string) string {
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return strings.Join(out, "; ")
}

// packRels 返回每个文件的相对路径。
// files 里每个 Rel 按原顺序放进结果。
func packRels(files []PackFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}
