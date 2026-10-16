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

// PackFile is one relative file inside a market skill directory.
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

// FetchPack downloads SKILL.md plus supporting files (scripts/, references/,
// assets/, icons). It never executes anything. If directory listing fails it
// falls back to SKILL.md only.
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

func downloadPackFile(slug, rel, downloadURL string) ([]byte, error) {
	if u, err := url.Parse(strings.TrimSpace(downloadURL)); err == nil && u.Scheme == "https" && u.Host != "" {
		if b, err := Get(u.String()); err == nil {
			return b, nil
		}
	}
	return Get(packFileURL(slug, rel))
}

func contentsURL(slug, rel string) string {
	base := strings.TrimRight(ContentsBase, "/")
	if base == "" {
		base = derivedContentsBase(Base)
	}
	return joinEscapedPath(base, "skills", slug, rel)
}

func packFileURL(slug, rel string) string {
	return joinEscapedPath(strings.TrimRight(Base, "/"), "skills", slug, rel)
}

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

func packHasSkillMD(files []PackFile) bool {
	for _, f := range files {
		if strings.EqualFold(path.Base(f.Rel), "SKILL.md") && !strings.Contains(f.Rel, "/") {
			return true
		}
	}
	return false
}

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

func isSkillMDRel(rel string) bool {
	rel = strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/")
	return strings.EqualFold(rel, "SKILL.md")
}

// SanitizeRelPart allows a single path segment for pack files.
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

func collectorFrom(files []PackFile) packCollector {
	c := packCollector{files: append([]PackFile(nil), files...)}
	for _, f := range files {
		c.total += len(f.Data)
	}
	return c
}

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

func packRels(files []PackFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}
