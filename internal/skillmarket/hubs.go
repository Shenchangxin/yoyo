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
	HubWorkBuddy    = "workbuddy"
	HubClawHub      = "clawhub"
	HubClawSkillHub = "clawskillhub"
	HubSkillsMP     = "skillsmp"
	HubAnthropic    = "anthropic"
	HubOsmani       = "osmani"
	HubClaudeSkills = "claude-skills"
	HubMDR          = "mdr"
	HubTessl        = "tessl"
	HubVercel       = "vercel"
	HubSkillHubCN   = "skillhubcn"
)

type HubInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Homepage    string `json:"homepage"`
	Attribution string `json:"attribution"`
	Browse      bool   `json:"browse"`
	Search      bool   `json:"search"`
}

type LoadOpts struct {
	Hub     string
	Query   string
	Refresh bool
	Limit   int
}

type PackResult struct {
	Files   []PackFile
	Warn    string
	URL     string
	Catalog string
	Owner   string
}

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

func NormalizeHub(id string) string {
	if info, ok := LookupHub(id); ok {
		return info.ID
	}
	return HubWorkBuddy
}

func IsHub(id string) bool {
	_, ok := LookupHub(id)
	return ok
}

func ItemID(hub, remote string) string {
	hub = NormalizeHub(hub)
	remote = strings.TrimSpace(strings.TrimPrefix(remote, "@"))
	remote = strings.Trim(remote, "/")
	if remote == "" {
		return hub
	}
	return hub + ":" + remote
}

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

func hubCacheName(hub, query string) string {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return "catalog-v4-" + hub + ".json"
	}
	sum := sha256.Sum256([]byte(hub + "\n" + q))
	return "catalog-v4-" + hub + "-" + hex.EncodeToString(sum[:6]) + ".json"
}

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

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

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
