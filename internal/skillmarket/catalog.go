package skillmarket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	DefaultBase = "https://raw.githubusercontent.com/infometa/workbuddyskills/main"
	CacheTTL    = 6 * time.Hour
	MaxBody     = 8 << 20
	cacheName   = "catalog-v2.json"
)

var (
	// Base is the catalog origin. Tests may override.
	Base = DefaultBase
	// ContentsBase lists pack files (GitHub Contents API shape). Empty derives from Base.
	ContentsBase = ""
	Get          = defaultGet
)

type Item struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Purpose       string `json:"purpose"`
	Prerequisites string `json:"prerequisites"`
	Category      string `json:"category"`
	Featured      bool   `json:"featured"`
	Icon          string `json:"icon,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	Version       string `json:"version,omitempty"`
	HasScripts    bool   `json:"has_scripts,omitempty"`
}

type Catalog struct {
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
	Items     []Item    `json:"items"`
}

func CatalogURL() string { return strings.TrimRight(Base, "/") + "/CATALOG.md" }

func SkillURL(slug string) string {
	return strings.TrimRight(Base, "/") + "/skills/" + slug + "/SKILL.md"
}

func Load(cacheDir string, refresh bool) (Catalog, error) {
	cachePath := filepath.Join(cacheDir, cacheName)
	var previous Catalog
	if prev, ok := readCache(cachePath); ok {
		previous = prev
		if !refresh {
			return prev, nil
		}
	}
	raw, err := Get(CatalogURL())
	if err != nil {
		if previous.Items != nil {
			previous.Source = previous.Source + " (stale)"
			return previous, nil
		}
		return Catalog{}, err
	}
	c := ParseCatalogMD(string(raw))
	c.Source = "infometa/workbuddyskills"
	c.FetchedAt = time.Now().UTC()
	markFeatured(c.Items)
	mergeCachedMeta(c.Items, previous.Items)
	enrichFromTree(c.Items)
	enrichFrontmatter(c.Items)
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		if b, err := json.MarshalIndent(c, "", "  "); err == nil {
			_ = os.WriteFile(cachePath, b, 0o644)
		}
	}
	return c, nil
}

func ParseCatalogMD(md string) Catalog {
	body := skillsSection(md)
	var items []Item
	seen := map[string]bool{}
	category := "Skills"
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "### ") {
			category = strings.TrimSpace(strings.TrimPrefix(trim, "### "))
			category = stripCount(category)
			continue
		}
		if !strings.HasPrefix(trim, "|") {
			continue
		}
		cells := splitRow(trim)
		if len(cells) < 3 {
			continue
		}
		if headerRow(cells) {
			continue
		}
		slug := SanitizeSlug(cellText(cells[0]))
		if slug == "" || seen[slug] {
			continue
		}
		name := cellText(cells[1])
		if name == "" {
			name = slug
		}
		purpose := strings.TrimSpace(strings.TrimPrefix(cellText(cells[2]), "用途："))
		purpose = strings.TrimSpace(strings.TrimPrefix(purpose, "用途:"))
		prereq := ""
		if len(cells) > 3 {
			prereq = cellText(cells[3])
		}
		seen[slug] = true
		items = append(items, Item{
			Slug:          slug,
			Name:          name,
			Purpose:       purpose,
			Prerequisites: prereq,
			Category:      category,
		})
	}
	return Catalog{Items: items}
}

func SanitizeSlug(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`")
	if strings.Contains(s, "..") {
		return ""
	}
	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.Trim(s, "/")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimSuffix(s, ".md")
	if s == "" {
		return ""
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return ""
	}
	if s[0] == '.' || s[0] == '-' {
		return ""
	}
	return s
}

func defaultGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Yoyo-Workstation")
	if strings.Contains(url, "api.github.com") {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	} else {
		req.Header.Set("Accept", "text/plain, text/markdown, application/json, */*")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skill market: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxBody))
}

func readCache(path string) (Catalog, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, false
	}
	var c Catalog
	if json.Unmarshal(b, &c) != nil || len(c.Items) == 0 {
		return Catalog{}, false
	}
	if time.Since(c.FetchedAt) > CacheTTL {
		return Catalog{}, false
	}
	return c, true
}

func skillsSection(md string) string {
	start := strings.Index(md, "## 1.")
	if start < 0 {
		start = strings.Index(md, "## 1 ")
	}
	if start < 0 {
		return md
	}
	rest := md[start:]
	if i := strings.Index(rest[4:], "\n## 2."); i >= 0 {
		return rest[:i+4]
	}
	return rest
}

var countSuffix = regexp.MustCompile(`[（(]\d+[）)]$`)

func stripCount(s string) string {
	return strings.TrimSpace(countSuffix.ReplaceAllString(s, ""))
}

func splitRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func headerRow(cells []string) bool {
	joined := strings.Join(cells, "")
	if strings.Contains(joined, "---") {
		return true
	}
	first := cells[0]
	return first == "目录" || strings.EqualFold(first, "dir") || strings.EqualFold(first, "slug")
}

func cellText(s string) string {
	s = strings.TrimSpace(s)
	for {
		i := strings.Index(s, "[")
		j := strings.Index(s, "](")
		if i < 0 || j < 0 || j < i {
			break
		}
		k := strings.Index(s[j:], ")")
		if k < 0 {
			break
		}
		k += j
		s = s[:i] + s[i+1:j] + s[k+1:]
	}
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}

var featuredHints = []string{
	"browser", "github", "git", "pdf", "xlsx", "excel", "docx", "search",
	"playwright", "deep-research", "arxiv", "stock", "diagnose", "handoff",
}

func markFeatured(items []Item) {
	n := 0
	for i := range items {
		slug := strings.ToLower(items[i].Slug)
		for _, h := range featuredHints {
			if slug == h || strings.Contains(slug, h) {
				items[i].Featured = true
				n++
				break
			}
		}
		if n >= 10 {
			break
		}
	}
	if n > 0 {
		return
	}
	for i := range items {
		if i >= 8 {
			break
		}
		items[i].Featured = true
	}
}
