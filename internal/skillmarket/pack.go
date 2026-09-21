package skillmarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
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
	files, err := listAndDownload(slug)
	if err == nil && packHasSkillMD(files) {
		return files, "", nil
	}
	raw, getErr := Get(SkillURL(slug))
	if getErr != nil {
		if err != nil {
			return nil, "", fmt.Errorf("%w", err)
		}
		return nil, "", getErr
	}
	warn := "installed SKILL.md only"
	if err != nil {
		warn = "pack listing failed; " + warn
	} else {
		warn = "catalog tree had no SKILL.md; " + warn
	}
	return []PackFile{{Rel: "SKILL.md", Data: raw}}, warn, nil
}

func listAndDownload(slug string) ([]PackFile, error) {
	var files []PackFile
	var total int
	var walk func(rel string, depth int) error
	walk = func(rel string, depth int) error {
		if depth > maxPackDepth {
			return fmt.Errorf("pack nested too deep")
		}
		entries, err := listContents(slug, rel)
		if err != nil {
			return err
		}
		for _, e := range entries {
			name := SanitizeRelPart(e.Name)
			if name == "" {
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
				if e.Size > maxFileBytes {
					return fmt.Errorf("%s larger than 1MiB", child)
				}
				raw, err := downloadPackFile(slug, child, e.DownloadURL)
				if err != nil {
					return fmt.Errorf("%s: %w", child, err)
				}
				if len(raw) > maxFileBytes {
					return fmt.Errorf("%s larger than 1MiB", child)
				}
				if err := scanPackFile(child, raw); err != nil {
					return fmt.Errorf("%s: %w", child, err)
				}
				total += len(raw)
				if total > maxPackBytes {
					return fmt.Errorf("pack larger than 8MiB")
				}
				if len(files) >= maxPackFiles {
					return fmt.Errorf("pack has more than %d files", maxPackFiles)
				}
				files = append(files, PackFile{Rel: child, Data: raw})
			}
		}
		return nil
	}
	if err := walk("", 0); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("empty pack")
	}
	return files, nil
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
	u := base + "/skills/" + slug
	rel = strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/")
	if rel != "" {
		u += "/" + rel
	}
	return u
}

func packFileURL(slug, rel string) string {
	rel = strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/")
	return strings.TrimRight(Base, "/") + "/skills/" + slug + "/" + rel
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
		return base == "license" || base == "notice" || base == "readme" || base == "makefile" || strings.EqualFold(base, "skill.md")
	}
	switch ext {
	case ".md", ".txt", ".py", ".js", ".ts", ".mjs", ".cjs", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".css",
		".sh", ".ps1", ".rb", ".pl", ".r", ".sql", ".csv", ".tsv",
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
		".jinja", ".j2", ".template":
		return true
	default:
		return false
	}
}

// SanitizeRelPart allows a single path segment for pack files.
func SanitizeRelPart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/\\")
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\") {
		return ""
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return ""
	}
	if s[0] == '.' {
		return ""
	}
	return s
}

func packRels(files []PackFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}
