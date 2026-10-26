// 解析 Work Buddy 的 CATALOG.md，并带 6 小时缓存。

package skillmarket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	// DefaultBase 是 Work Buddy 目录的默认 raw 地址。
	DefaultBase = "https://raw.githubusercontent.com/infometa/workbuddyskills/main"
	// CacheTTL 是目录缓存有效期，6 小时。
	CacheTTL = 6 * time.Hour
	// MaxBody 是单次 HTTP 正文上限，8<<20，即 8 MiB。
	MaxBody   = 8 << 20
	cacheName = "catalog-v2.json"
)

var (
	// Base 是目录源站。测试可以改它。
	Base = DefaultBase
	// ContentsBase 是列包内文件的地址，形状同 GitHub Contents API。空字符串时从 Base 推导。
	ContentsBase = ""
	// Get 拉取一个 URL。测试可以替换它。
	Get = defaultGet
)

// Item 是目录里的一条技能。
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
	Hub           string `json:"hub,omitempty"`
	ID            string `json:"id,omitempty"`
	Owner         string `json:"owner,omitempty"`
	PageURL       string `json:"page_url,omitempty"`
	Repo          string `json:"repo,omitempty"`
	Downloads     int    `json:"downloads,omitempty"`
	Stars         int    `json:"stars,omitempty"`
}

// Catalog 是一次目录加载的结果，含条目、站、查询和是否只允许搜索。
type Catalog struct {
	Source      string    `json:"source"`
	FetchedAt   time.Time `json:"fetched_at"`
	Items       []Item    `json:"items"`
	Hub         string    `json:"hub,omitempty"`
	Hubs        []HubInfo `json:"hubs,omitempty"`
	Query       string    `json:"query,omitempty"`
	SearchOnly  bool      `json:"search_only,omitempty"`
	Warning     string    `json:"warning,omitempty"`
	Attribution string    `json:"attribution,omitempty"`
	Total       int       `json:"total,omitempty"`
	Truncated   bool      `json:"truncated,omitempty"`
}

// CatalogURL 返回 Base 下的 CATALOG.md。
func CatalogURL() string { return strings.TrimRight(Base, "/") + "/CATALOG.md" }

// SkillURL 返回 Base/skills/<slug>/SKILL.md。
func SkillURL(slug string) string {
	return strings.TrimRight(Base, "/") + "/skills/" + slug + "/SKILL.md"
}

// Load 加载 Work Buddy 目录，等价于 LoadHub 且 Hub 为 HubWorkBuddy。
// cacheDir 交给 LoadHub。
// refresh 写入 LoadOpts 的 Refresh，再交给 LoadHub。
func Load(cacheDir string, refresh bool) (Catalog, error) {
	return LoadHub(cacheDir, LoadOpts{Hub: HubWorkBuddy, Refresh: refresh})
}

// loadWorkBuddy 读 cacheDir/catalog-v2.json。未要求刷新且缓存未过期时直接返回。拉取失败但有旧条目时在 Source 后加 " (stale)"。成功后解析 CATALOG.md、补树和前言，并写回缓存。
// refresh 为假且 readCache 成功时直接返回 stampWorkBuddy 的结果；否则继续拉取。
func loadWorkBuddy(cacheDir string, refresh bool) (Catalog, error) {
	cachePath := filepath.Join(cacheDir, cacheName)
	var previous Catalog
	if prev, ok := readCache(cachePath); ok {
		previous = prev
		if !refresh {
			return stampWorkBuddy(prev), nil
		}
	}
	raw, err := Get(CatalogURL())
	if err != nil {
		if previous.Items != nil {
			previous.Source = previous.Source + " (stale)"
			return stampWorkBuddy(previous), nil
		}
		return Catalog{}, err
	}
	c := ParseCatalogMD(string(raw))
	c.Source = "infometa/workbuddyskills"
	c.FetchedAt = time.Now().UTC()
	markFeatured(c.Items)
	mergeCachedMeta(c.Items, previous.Items)
	c.Items = enrichFromTree(c.Items)
	enrichFrontmatter(c.Items)
	c = stampWorkBuddy(c)
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		if b, err := json.MarshalIndent(c, "", "  "); err == nil {
			_ = os.WriteFile(cachePath, b, 0o644)
		}
	}
	return c, nil
}

// stampWorkBuddy 把条目标成 Work Buddy，ID 用 ItemID，缺页面时指向 infometa/workbuddyskills 的 skills/<slug>。
// c 的 Hub 设为 HubWorkBuddy，每条写入 Hub、ID 和 Repo；PageURL 为空时补上 GitHub 树地址。
func stampWorkBuddy(c Catalog) Catalog {
	c.Hub = HubWorkBuddy
	for i := range c.Items {
		c.Items[i].Hub = HubWorkBuddy
		c.Items[i].ID = ItemID(HubWorkBuddy, c.Items[i].Slug)
		if c.Items[i].PageURL == "" {
			c.Items[i].PageURL = "https://github.com/infometa/workbuddyskills/tree/main/skills/" + c.Items[i].Slug
		}
		c.Items[i].Repo = "infometa/workbuddyskills"
	}
	return c
}

// ParseCatalogMD 只解析 "## 1." 到 "## 2." 之间的表格。三级标题是分类，会去掉末尾的数量括号。用途列会去掉前缀 "用途："。
// md 交给 skillsSection，再按行解析表格。
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

// SanitizeSlug 取最后一段，去掉 .md。含 ..、以点或连字符开头，或出现字母、数字、-、_、. 以外的字符时返回空串。
// s 去掉两端空白和反引号后取最后一段并去掉 .md；含 ..、结果为空、以点或连字符开头，或出现字母、数字、-、_、. 以外的字符时返回空串。
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

// githubAuthToken 先读 YOYO_GITHUB_TOKEN，再读 GITHUB_TOKEN。都空则返回空串。
func githubAuthToken() string {
	for _, k := range []string{"YOYO_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// readCache 读取目录缓存。条目为空，或 FetchedAt 距现在超过 CacheTTL，则返回 false。
// path 交给 os.ReadFile；读失败、JSON 非法、条目为空，或 FetchedAt 距现在超过 CacheTTL 时返回 false。
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

// skillsSection 截取 "## 1." 或 "## 1 " 到下一个 "## 2." 之前的正文。找不到开头时返回全文。
// md 里找 "## 1."，没有再找 "## 1 "，再截到换行加 "## 2." 之前；找不到开头时返回全文。
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

// stripCount 去掉分类名末尾的全角或半角括号加数字。
func stripCount(s string) string {
	return strings.TrimSpace(countSuffix.ReplaceAllString(s, ""))
}

// splitRow 按竖线拆表格行并去掉每格空白。
// line 去掉两端竖线后按竖线拆开，每格去掉空白。
func splitRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// headerRow 认出分隔行，以及首格为 "目录"、dir 或 slug 的表头。
// cells 拼起来含 "---" 时返回 true，否则看第一格是不是 "目录"、dir 或 slug。
func headerRow(cells []string) bool {
	joined := strings.Join(cells, "")
	if strings.Contains(joined, "---") {
		return true
	}
	first := cells[0]
	return first == "目录" || strings.EqualFold(first, "dir") || strings.EqualFold(first, "slug")
}

// cellText 去掉 Markdown 链接外壳和反引号，只留可见文字。
// s 去掉两端空白、Markdown 链接外壳和反引号。
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

// markFeatured 按 slug 命中内置关键词，最多标 10 条。一条都没有时改为标前 8 条。
// items 按 slug 小写匹配关键词，命中则把 Featured 设为 true，最多 10 条；一条都没有时把前 8 条设为 true。
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
