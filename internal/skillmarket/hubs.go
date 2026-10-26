// Package skillmarket 从多个技能站拉目录并安装源文件。市场技能写到调用方传入的 homeSkills/<slug>，旁边有 .market.json。Work Buddy 的目录来自 CATALOG.md。
package skillmarket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// HubWorkBuddy 是 Work Buddy 站的 id，值为 workbuddy。
	HubWorkBuddy = "workbuddy"
	// HubClawHub 是 ClawHub 站的 id，值为 clawhub。
	HubClawHub = "clawhub"
	// HubClawSkillHub 是 clawskillhub.com 的 id，值为 clawskillhub。
	HubClawSkillHub = "clawskillhub"
	// HubSkillsMP 是 SkillsMP 站的 id，值为 skillsmp。
	HubSkillsMP = "skillsmp"
	// HubAnthropic 是 anthropics/skills 的 id，值为 anthropic。
	HubAnthropic = "anthropic"
	// HubOsmani 是 addyosmani/agent-skills 的 id，值为 osmani。
	HubOsmani = "osmani"
	// HubClaudeSkills 是 alirezarezvani/claude-skills 的 id，值为 claude-skills。
	HubClaudeSkills = "claude-skills"
	// HubMDR 是 Markdown Registry 的 id，值为 mdr。
	HubMDR = "mdr"
	// HubTessl 是 Tessl 的 id，值为 tessl。
	HubTessl = "tessl"
	// HubVercel 是 vercel-labs/agent-skills 的 id，值为 vercel。
	HubVercel = "vercel"
	// HubSkillHubCN 是 skillhub.cn 的 id，值为 skillhubcn。
	HubSkillHubCN = "skillhubcn"
)

// HubInfo 是一个技能站的 id、显示名、主页、署名，以及能不能浏览和搜索。
type HubInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Homepage    string `json:"homepage"`
	Attribution string `json:"attribution"`
	Browse      bool   `json:"browse"`
	Search      bool   `json:"search"`
}

// LoadOpts 指定要加载的站、查询、是否刷新缓存，以及条数上限。
type LoadOpts struct {
	Hub     string
	Query   string
	Refresh bool
	Limit   int
}

// PackResult 是拉到的文件、跳过警告、页面地址、目录名和所有者。
type PackResult struct {
	Files   []PackFile
	Warn    string
	URL     string
	Catalog string
	Owner   string
}

// HubCatalog 返回内置技能站列表，顺序就是界面里的顺序。
func HubCatalog() []HubInfo {
	return []HubInfo{
		{
			ID:          HubWorkBuddy,
			Name:        "Work Buddy",
			Homepage:    "https://github.com/infometa/workbuddyskills",
			Attribution: "Work Buddy public catalog (infometa/workbuddyskills). Source files only — Yoyo does not run zip executables on install. GitHub is fetched via jsDelivr / gh-proxy when the origin is unreachable.",
			Browse:      true,
			Search:      false,
		},
		{
			ID:          HubSkillHubCN,
			Name:        "SkillHub 中国",
			Homepage:    "https://skillhub.cn",
			Attribution: "腾讯云 SkillHub（skillhub.cn）。国内节点、中文检索；只落源文件，安装时不跑 zip。",
			Browse:      true,
			Search:      true,
		},
		{
			ID:          HubClawHub,
			Name:        "ClawHub",
			Homepage:    "https://clawhub.ai",
			Attribution: "ClawHub public skill registry (clawhub.ai), with clawhubcn.com failover for mainland networks. Suspicious listings are hidden. Source files only — Yoyo does not unpack zip artifacts.",
			Browse:      true,
			Search:      true,
		},
		{
			ID:          HubClawSkillHub,
			Name:        "SkillHub",
			Homepage:    "https://clawskillhub.com",
			Attribution: "SkillHub / ClawSkillHub (clawskillhub.com). Validated OpenClaw skills; source files only.",
			Browse:      true,
			Search:      true,
		},
		{
			ID:          HubSkillsMP,
			Name:        "SkillsMP",
			Homepage:    "https://skillsmp.com",
			Attribution: "SkillsMP GitHub aggregator (skillsmp.com). Unreviewed public SKILL.md files — read before use.",
			Browse:      false,
			Search:      true,
		},
		{
			ID:          HubAnthropic,
			Name:        "Anthropic",
			Homepage:    "https://github.com/anthropics/skills",
			Attribution: "Anthropic official skills repository (anthropics/skills).",
			Browse:      true,
			Search:      false,
		},
		{
			ID:          HubOsmani,
			Name:        "Addy Osmani",
			Homepage:    "https://github.com/addyosmani/agent-skills",
			Attribution: "Addy Osmani agent-skills collection (addyosmani/agent-skills).",
			Browse:      true,
			Search:      false,
		},
		{
			ID:          HubClaudeSkills,
			Name:        "Claude Skills",
			Homepage:    "https://github.com/alirezarezvani/claude-skills",
			Attribution: "alirezarezvani/claude-skills curated library. Source files from GitHub.",
			Browse:      true,
			Search:      false,
		},
		{
			ID:          HubVercel,
			Name:        "Vercel",
			Homepage:    "https://github.com/vercel-labs/agent-skills",
			Attribution: "Vercel Labs agent-skills (skills.sh first-party set). The skills.sh REST API requires Vercel OIDC, so Yoyo lists the public GitHub tree.",
			Browse:      true,
			Search:      false,
		},
		{
			ID:          HubMDR,
			Name:        "Markdown Registry",
			Homepage:    "https://markdownregistry.com",
			Attribution: "markdownregistry.com index of public GitHub SKILL.md files, installed from the recorded repository path.",
			Browse:      false,
			Search:      true,
		},
		{
			ID:          HubTessl,
			Name:        "Tessl",
			Homepage:    "https://tessl.io",
			Attribution: "Tessl registry (api.tessl.io). Skills resolve to GitHub source files; Snyk scores live on Tessl's site.",
			Browse:      true,
			Search:      true,
		},
	}
}

// LookupHub 按小写 id 找站。workbuddyskills 归到 Work Buddy，skillhub 与 skillhub.space 归到 ClawSkillHub，skillhub.cn 与 tencent 归到 SkillHub 中国，skills.sh 归到 Vercel。空 id 返回 false。
func LookupHub(id string) (HubInfo, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	switch id {
	case "workbuddyskills", "infometa/workbuddyskills":
		id = HubWorkBuddy
	case "skillhub", "skillhub.space":
		id = HubClawSkillHub
	case "skillhubcn", "skillhub.cn", "tencent", "tencent-skillhub":
		id = HubSkillHubCN
	case "skills.sh", "skillssh":
		id = HubVercel
	}
	if id == "" {
		return HubInfo{}, false
	}
	for _, h := range HubCatalog() {
		if h.ID == id {
			return h, true
		}
	}
	return HubInfo{}, false
}

// NormalizeHub 返回 LookupHub 认的 id。认不出时返回 HubWorkBuddy。
func NormalizeHub(id string) string {
	if info, ok := LookupHub(id); ok {
		return info.ID
	}
	return HubWorkBuddy
}

// IsHub 在 LookupHub 能找到该 id 时返回 true。
func IsHub(id string) bool {
	_, ok := LookupHub(id)
	return ok
}

// ItemID 拼出 hub:remote。remote 去掉首尾斜杠和前缀 @。remote 为空时只返回站 id。
func ItemID(hub, remote string) string {
	hub = NormalizeHub(hub)
	remote = strings.TrimSpace(strings.TrimPrefix(remote, "@"))
	remote = strings.Trim(remote, "/")
	if remote == "" {
		return hub
	}
	return hub + ":" + remote
}

// ParseInstallID 解析安装 id。空串得到 HubWorkBuddy 和空 remote。hub:remote 且 hub 是已知站时拆开。否则整段当作 Work Buddy 的 remote。
func ParseInstallID(s string) (hub, remote string) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	if s == "" {
		return HubWorkBuddy, ""
	}
	if i := strings.IndexByte(s, ':'); i > 0 {
		h := strings.ToLower(s[:i])
		if IsHub(h) {
			return NormalizeHub(h), strings.TrimSpace(strings.TrimPrefix(s[i+1:], "@"))
		}
	}
	return HubWorkBuddy, s
}

// InstallSlug 取路径最后一段，再交给 SanitizeSlug。
// remote 去掉两端空白，反斜杠换成斜杠，取最后一段交给 SanitizeSlug。
func InstallSlug(remote string) string {
	remote = strings.TrimSpace(strings.ReplaceAll(remote, "\\", "/"))
	remote = strings.Trim(remote, "/")
	if i := strings.LastIndex(remote, "/"); i >= 0 {
		remote = remote[i+1:]
	}
	return SanitizeSlug(remote)
}

const (
	defaultBrowseLimit = 2000
	defaultSearchLimit = 500
	maxCatalogLimit    = 5000
	hubPageSize        = 100
	maxCatalogPages    = 40
)

// catalogLimit 在 requested 大于 0 时采用它，但不超过 5000。未指定时，有查询用 500，浏览用 2000。
// query 在 requested 不大于 0 时，去掉空白后非空则返回 defaultSearchLimit，否则返回 defaultBrowseLimit。
func catalogLimit(query string, requested int) int {
	if requested > 0 {
		if requested > maxCatalogLimit {
			return maxCatalogLimit
		}
		return requested
	}
	if strings.TrimSpace(query) != "" {
		return defaultSearchLimit
	}
	return defaultBrowseLimit
}

// LoadHub 加载一个站的目录。不能搜索的站会丢掉查询。Work Buddy 走 loadWorkBuddy。只能搜索又不能浏览且查询为空时，返回 SearchOnly、不发请求。缓存未过期且没有 Refresh 时直接用缓存。拉取失败但有旧条目时，在 Source 后加 " (stale)" 并带上 Warning。
// cacheDir 在 Hub 为 HubWorkBuddy 时交给 loadWorkBuddy，否则用来拼缓存路径并创建目录。
// opts 的 Hub、Query、Refresh、Limit 分别用来选站、查询、决定是否跳过未过期缓存，以及交给 catalogLimit。
func LoadHub(cacheDir string, opts LoadOpts) (Catalog, error) {
	hub := NormalizeHub(opts.Hub)
	query := strings.TrimSpace(opts.Query)
	if info, ok := LookupHub(hub); ok && !info.Search {
		query = ""
	}
	if hub == HubWorkBuddy {
		c, err := loadWorkBuddy(cacheDir, opts.Refresh)
		return stampCatalog(c, hub, query), err
	}
	info, _ := LookupHub(hub)
	if info.Search && !info.Browse && query == "" {
		return stampCatalog(Catalog{
			Source:     info.Name,
			FetchedAt:  time.Now().UTC(),
			SearchOnly: true,
		}, hub, query), nil
	}
	cachePath := filepath.Join(cacheDir, hubCacheName(hub, query))
	var previous Catalog
	if prev, ok := readCache(cachePath); ok {
		previous = prev
		if !opts.Refresh {
			return stampCatalog(prev, hub, query), nil
		}
	}
	c, err := fetchHubCatalog(hub, query, catalogLimit(query, opts.Limit))
	if err != nil {
		if previous.Items != nil {
			previous.Source = previous.Source + " (stale)"
			previous.Warning = err.Error()
			return stampCatalog(previous, hub, query), nil
		}
		return stampCatalog(Catalog{Warning: err.Error()}, hub, query), err
	}
	c.FetchedAt = time.Now().UTC()
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		if b, err := json.MarshalIndent(c, "", "  "); err == nil {
			_ = os.WriteFile(cachePath, b, 0o644)
		}
	}
	return stampCatalog(c, hub, query), nil
}

// fetchHubCatalog 按站调用对应的列表函数。未知站返回 "unknown skill hub"。
// query 先交给 catalogLimit；ClawHub、SkillHubCN、ClawSkillHub、SkillsMP、MDR、Tessl 的列表会收到它。
// limit 先交给 catalogLimit，再传给上述列表。
func fetchHubCatalog(hub, query string, limit int) (Catalog, error) {
	limit = catalogLimit(query, limit)
	switch hub {
	case HubClawHub:
		return listClawHub(query, limit)
	case HubSkillHubCN:
		return listSkillHubCN(query, limit)
	case HubClawSkillHub:
		return listClawSkillHub(query, limit)
	case HubSkillsMP:
		return listSkillsMP(query, limit)
	case HubAnthropic:
		return listGitHubHub(hub, "anthropics/skills", "main", []string{"skills"})
	case HubOsmani:
		return listGitHubHub(hub, "addyosmani/agent-skills", "main", []string{"skills"})
	case HubClaudeSkills:
		return listGitHubHub(hub, "alirezarezvani/claude-skills", "main", []string{""})
	case HubVercel:
		return listGitHubHub(hub, "vercel-labs/agent-skills", "main", []string{""})
	case HubMDR:
		return listMarkdownRegistry(query, limit)
	case HubTessl:
		return listTessl(query, limit)
	default:
		return Catalog{}, fmt.Errorf("unknown skill hub %s", hub)
	}
}

// FetchFrom 按站下载一份技能的源文件。remote 为空返回 "invalid skill id"。未知站返回 "unknown skill hub"。
func FetchFrom(hub, remote string) (PackResult, error) {
	hub = NormalizeHub(hub)
	remote = strings.TrimSpace(strings.TrimPrefix(remote, "@"))
	if remote == "" {
		return PackResult{}, fmt.Errorf("invalid skill id")
	}
	switch hub {
	case HubWorkBuddy:
		files, warn, err := FetchPack(InstallSlug(remote))
		return PackResult{Files: files, Warn: warn, URL: SkillURL(InstallSlug(remote)), Catalog: "infometa/workbuddyskills"}, err
	case HubClawHub:
		return fetchClawHub(remote)
	case HubSkillHubCN:
		return fetchSkillHubCN(remote)
	case HubClawSkillHub:
		return fetchClawSkillHub(remote)
	case HubSkillsMP:
		return fetchSkillsMP(remote)
	case HubAnthropic:
		dir := githubSkillDir("skills", remote)
		return fetchGitHubSkill("anthropics/skills", "main", dir, "https://github.com/anthropics/skills/tree/main/"+dir)
	case HubOsmani:
		dir := githubSkillDir("skills", remote)
		return fetchGitHubSkill("addyosmani/agent-skills", "main", dir, "https://github.com/addyosmani/agent-skills/tree/main/"+dir)
	case HubClaudeSkills:
		return fetchClaudeSkills(remote)
	case HubVercel:
		return fetchVercelSkill(remote)
	case HubMDR:
		return fetchMarkdownRegistry(remote)
	case HubTessl:
		return fetchTessl(remote)
	default:
		return PackResult{}, fmt.Errorf("unknown skill hub %s", hub)
	}
}

// stampCatalog 填上站 id、全部站列表、查询和署名。只能搜索又不能浏览且查询为空时把 SearchOnly 设为真。条目缺 ID 时用 ItemID 补上。
// hub 交给 LookupHub，写入 Hub；条目缺 Hub 时也写上。
// query 写入 Query；只能搜索又不能浏览且 query 为空时把 SearchOnly 设为真。
func stampCatalog(c Catalog, hub, query string) Catalog {
	info, _ := LookupHub(hub)
	c.Hub = hub
	c.Hubs = HubCatalog()
	c.Query = query
	if c.Source == "" {
		c.Source = info.Name
	}
	if c.Attribution == "" {
		c.Attribution = info.Attribution
	}
	if info.Search && !info.Browse && query == "" {
		c.SearchOnly = true
	}
	for i := range c.Items {
		if c.Items[i].Hub == "" {
			c.Items[i].Hub = hub
		}
		if c.Items[i].ID == "" {
			remote := c.Items[i].Owner
			if remote != "" && c.Items[i].Slug != "" {
				remote += "/" + c.Items[i].Slug
			} else {
				remote = c.Items[i].Slug
			}
			c.Items[i].ID = ItemID(c.Items[i].Hub, remote)
		}
	}
	return c
}

// hubCacheName 在查询为空时返回 catalog-v4-<hub>.json。否则把站和查询的 SHA-256 前 6 字节编进文件名。
// query 转成小写并去掉两端空白，空则文件名不含摘要；否则和 hub 一起做 SHA-256，取前 6 字节的十六进制。
func hubCacheName(hub, query string) string {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return "catalog-v4-" + hub + ".json"
	}
	sum := sha256.Sum256([]byte(hub + "\n" + q))
	return "catalog-v4-" + hub + "-" + hex.EncodeToString(sum[:6]) + ".json"
}

// getJSON 用 Get 拉正文并解进 dest。空正文返回 "empty response"。以 < 开头返回 "expected JSON, got HTML"。
// rawURL 交给 Get。
func getJSON(rawURL string, dest any) error {
	raw, err := Get(rawURL)
	if err != nil {
		return err
	}
	trim := strings.TrimSpace(string(raw))
	if trim == "" {
		return fmt.Errorf("empty response")
	}
	if trim[0] == '<' {
		return fmt.Errorf("expected JSON, got HTML")
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("json: %w", err)
	}
	return nil
}

// joinURL 用斜杠接上路径段，去掉每段两侧的斜杠。
// base 去掉末尾斜杠后作为开头。
// parts 每段把反斜杠换成斜杠并去掉两端斜杠，空段跳过，其余用斜杠接上。
func joinURL(base string, parts ...string) string {
	u := strings.TrimRight(base, "/")
	for _, p := range parts {
		p = strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
		if p == "" {
			continue
		}
		u += "/" + p
	}
	return u
}

// clip 去掉首尾空白。n 小于等于 0 或字符串不长于 n 时原样返回，否则截到 n 并加上省略号。
// s 先去掉首尾空白，再按 n 决定是否截断。
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

// splitOwnerSlug 把 owner/slug 拆开。没有斜杠时 owner 为空，slug 经过 SanitizeSlug。
// id 去掉前缀 @、两端空白和两端斜杠；没有斜杠时 owner 为空，slug 交给 SanitizeSlug。
func splitOwnerSlug(id string) (owner, slug string) {
	id = strings.TrimSpace(strings.TrimPrefix(id, "@"))
	id = strings.Trim(strings.ReplaceAll(id, "\\", "/"), "/")
	if id == "" {
		return "", ""
	}
	if i := strings.IndexByte(id, '/'); i > 0 {
		return strings.TrimPrefix(id[:i], "@"), SanitizeSlug(id[i+1:])
	}
	return "", SanitizeSlug(id)
}
