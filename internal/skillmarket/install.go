package skillmarket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Record struct {
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Catalog   string    `json:"catalog"`
	URL       string    `json:"url"`
	Icon      string    `json:"icon,omitempty"`
	Files     []string  `json:"files,omitempty"`
	Installed time.Time `json:"installed_at"`
	Scan      Scan      `json:"scan"`
}

func Install(homeSkills, slug string, scan Scan, files []PackFile) (string, error) {
	slug = SanitizeSlug(slug)
	if slug == "" {
		return "", fmt.Errorf("invalid skill slug")
	}
	if !scan.OK {
		return "", fmt.Errorf("scan blocked: %s", stringsJoin(scan.Reasons))
	}
	if !packHasSkillMD(files) {
		return "", fmt.Errorf("pack missing SKILL.md")
	}
	dir := filepath.Join(homeSkills, slug)
	tmp := dir + ".install-tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	for _, f := range files {
		rel := filepath.FromSlash(f.Rel)
		if !validPackRel(rel) {
			return "", fmt.Errorf("invalid pack path %s", f.Rel)
		}
		dst := filepath.Join(tmp, rel)
		if !withinDir(tmp, dst) {
			return "", fmt.Errorf("pack path escapes skill directory")
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		mode := os.FileMode(0o644)
		if isScriptRel(f.Rel) {
			mode = 0o755
		}
		if err := os.WriteFile(dst, f.Data, mode); err != nil {
			return "", err
		}
	}
	rec := Record{
		Slug:      slug,
		Name:      scan.Name,
		Source:    "workbuddy",
		Catalog:   "infometa/workbuddyskills",
		URL:       SkillURL(slug),
		Icon:      scan.Icon,
		Files:     packRels(files),
		Installed: time.Now().UTC(),
		Scan:      scan,
	}
	b, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, ".market.json"), b, 0o644); err != nil {
		return "", err
	}
	_ = os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		return "", err
	}
	return dir, nil
}

func Uninstall(homeSkills, name string) error {
	dir, err := ResolveInstall(homeSkills, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".market.json")); err != nil {
		return fmt.Errorf("not a market install")
	}
	return os.RemoveAll(dir)
}

// ResolveInstall finds a market pack by directory slug or frontmatter name.
func ResolveInstall(homeSkills, name string) (string, error) {
	slug := SanitizeSlug(name)
	if slug != "" {
		dir := filepath.Join(homeSkills, slug)
		if _, err := os.Stat(filepath.Join(dir, ".market.json")); err == nil {
			return dir, nil
		}
	}
	want := strings.TrimSpace(name)
	if want == "" {
		return "", fmt.Errorf("invalid skill slug")
	}
	ents, err := os.ReadDir(homeSkills)
	if err != nil {
		return "", fmt.Errorf("not a market install")
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(homeSkills, e.Name())
		b, err := os.ReadFile(filepath.Join(dir, ".market.json"))
		if err != nil {
			continue
		}
		var rec Record
		if json.Unmarshal(b, &rec) != nil {
			continue
		}
		if rec.Name == want || rec.Slug == want {
			return dir, nil
		}
	}
	return "", fmt.Errorf("not a market install")
}

func ReadRecord(dir string) (Record, bool) {
	var rec Record
	b, err := os.ReadFile(filepath.Join(dir, ".market.json"))
	if err != nil || json.Unmarshal(b, &rec) != nil {
		return Record{}, false
	}
	return rec, true
}

func PackIncomplete(dir, body string) bool {
	if dir == "" || !MentionsScripts(body) {
		return false
	}
	ents, err := os.ReadDir(filepath.Join(dir, "scripts"))
	if err != nil {
		return true
	}
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		return false
	}
	return true
}

func validPackRel(rel string) bool {
	if rel == "" || filepath.IsAbs(rel) {
		return false
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return false
	}
	return packAllowed(filepath.ToSlash(rel))
}

func withinDir(root, p string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isScriptRel(rel string) bool {
	rel = strings.ToLower(filepath.ToSlash(rel))
	switch filepath.Ext(rel) {
	case ".sh", ".py", ".ps1", ".rb", ".js", ".mjs":
		return true
	default:
		return false
	}
}

func stringsJoin(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += "; "
		}
		out += v
	}
	return out
}
