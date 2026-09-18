package skillmarket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Record struct {
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Catalog   string    `json:"catalog"`
	URL       string    `json:"url"`
	Installed time.Time `json:"installed_at"`
	Scan      Scan      `json:"scan"`
}

func Install(homeSkills, slug string, scan Scan, raw []byte) (string, error) {
	slug = SanitizeSlug(slug)
	if slug == "" {
		return "", fmt.Errorf("invalid skill slug")
	}
	if !scan.OK {
		return "", fmt.Errorf("scan blocked: %s", stringsJoin(scan.Reasons))
	}
	dir := filepath.Join(homeSkills, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), raw, 0o644); err != nil {
		return "", err
	}
	rec := Record{
		Slug:      slug,
		Name:      scan.Name,
		Source:    "workbuddy",
		Catalog:   "infometa/workbuddyskills",
		URL:       SkillURL(slug),
		Installed: time.Now().UTC(),
		Scan:      scan,
	}
	b, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, ".market.json"), b, 0o644); err != nil {
		return dir, err
	}
	return dir, nil
}

func Uninstall(homeSkills, slug string) error {
	slug = SanitizeSlug(slug)
	if slug == "" {
		return fmt.Errorf("invalid skill slug")
	}
	dir := filepath.Join(homeSkills, slug)
	if _, err := os.Stat(filepath.Join(dir, ".market.json")); err != nil {
		return fmt.Errorf("not a market install")
	}
	return os.RemoveAll(dir)
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
