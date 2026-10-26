// 从 api.skillhub.cn 分页列出并下载技能。

package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// SkillHubCNBase 是 SkillHub 中国的 API 根地址。测试可以改它。
var SkillHubCNBase = "https://api.skillhub.cn"

// listSkillHubCN 从第 1 页起拉，后续页最多再要 39 页，并发数是 4。无查询时前 8 条标成 Featured。total 大于实际条数时 Truncated 为真。
// query 先交给 catalogLimit，并传给 fetchSkillHubCNPage；为空串时把前 8 条标成 Featured。
// limit 先交给 catalogLimit；收集条数达到它就停。
func listSkillHubCN(query string, limit int) (Catalog, error) {
	limit = catalogLimit(query, limit)
	pageSize := hubPageSize
	if limit < pageSize {
		pageSize = limit
	}
	rows, total, err := fetchSkillHubCNPage(query, 1, pageSize)
	if err != nil {
		return Catalog{}, err
	}
	if pageSize > 0 && limit > len(rows) {
		need := limit - len(rows)
		extra := (need + pageSize - 1) / pageSize
		if extra > maxCatalogPages-1 {
			extra = maxCatalogPages - 1
		}
		if total > 0 {
			last := (total + pageSize - 1) / pageSize
			if 1+extra > last {
				extra = last - 1
			}
		}
		if extra < 0 {
			extra = 0
		}
		if extra > 0 && len(rows) >= pageSize {
			more := make([][]skillHubCNRow, extra)
			var wg sync.WaitGroup
			sem := make(chan struct{}, 4)
			var mu sync.Mutex
			var firstErr error
			for i := 0; i < extra; i++ {
				page := i + 2
				wg.Add(1)
				go func(i, page int) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					batch, _, err := fetchSkillHubCNPage(query, page, pageSize)
					if err != nil {
						mu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						mu.Unlock()
						return
					}
					more[i] = batch
				}(i, page)
			}
			wg.Wait()
			if firstErr != nil && len(rows) == 0 {
				return Catalog{}, firstErr
			}
			for _, batch := range more {
				rows = append(rows, batch...)
			}
		}
	}
	seen := map[string]bool{}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		it, ok := itemFromSkillHubCN(row)
		if !ok || seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		if query == "" && len(out) < 8 {
			it.Featured = true
		}
		out = append(out, it)
		if len(out) >= limit {
			break
		}
	}
	truncated := total > len(out)
	return Catalog{Source: "SkillHub 中国", Items: out, Total: total, Truncated: truncated}, nil
}

type skillHubCNList struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Skills []skillHubCNRow `json:"skills"`
		Total  int             `json:"total"`
	} `json:"data"`
}

// fetchSkillHubCNPage 请求 /api/skills。page 小于 1 时按 1。pageSize 非法时用 100。有关键词时 sortBy=score，否则按 downloads 降序。code 不是 0 时返回 "skillhub.cn: " 加 message。
// query 去掉两端空白后非空时写入 keyword，并把 sortBy 改成 score。
func fetchSkillHubCNPage(query string, page, pageSize int) ([]skillHubCNRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > hubPageSize {
		pageSize = hubPageSize
	}
	vals := url.Values{
		"sortBy":   {"downloads"},
		"order":    {"desc"},
		"pageSize": {strconv.Itoa(pageSize)},
		"page":     {strconv.Itoa(page)},
	}
	if q := strings.TrimSpace(query); q != "" {
		vals.Set("keyword", q)
		vals.Set("sortBy", "score")
	}
	u := strings.TrimRight(SkillHubCNBase, "/") + "/api/skills?" + vals.Encode()
	var resp skillHubCNList
	if err := getJSON(u, &resp); err != nil {
		return nil, 0, err
	}
	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("skillhub.cn: %s", strings.TrimSpace(resp.Message))
	}
	return resp.Data.Skills, resp.Data.Total, nil
}

type skillHubCNRow struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Description   string `json:"description"`
	DescriptionZh string `json:"description_zh"`
	Category      string `json:"category"`
	Downloads     int    `json:"downloads"`
	Stars         int    `json:"stars"`
	Version       string `json:"version"`
	IconURL       string `json:"iconUrl"`
	Namespace     struct {
		Handle     string `json:"handle"`
		PublicSlug string `json:"publicSlug"`
	} `json:"namespace"`
	SubCategories []struct {
		Name string `json:"name"`
	} `json:"subCategories"`
}

// itemFromSkillHubCN 优先用中文描述。slug 净化失败返回 false。分类空时用 "SkillHub 中国"。
// row 的 Slug 或 namespace 的 PublicSlug 交给 SanitizeSlug，都空则返回 false；描述优先用 DescriptionZh，再交给 clip，第二个实参是 240。
func itemFromSkillHubCN(row skillHubCNRow) (Item, bool) {
	slug := SanitizeSlug(row.Slug)
	if slug == "" {
		slug = SanitizeSlug(row.Namespace.PublicSlug)
	}
	if slug == "" {
		return Item{}, false
	}
	name := strings.TrimSpace(row.Name)
	if name == "" {
		name = slug
	}
	purpose := strings.TrimSpace(row.DescriptionZh)
	if purpose == "" {
		purpose = strings.TrimSpace(row.Description)
	}
	cat := strings.TrimSpace(row.Category)
	if len(row.SubCategories) > 0 && strings.TrimSpace(row.SubCategories[0].Name) != "" {
		cat = row.SubCategories[0].Name
	}
	if cat == "" {
		cat = "SkillHub 中国"
	}
	owner := strings.TrimSpace(row.Namespace.Handle)
	remote := slug
	if owner != "" {
		remote = owner + "/" + slug
	}
	return Item{
		Slug:        slug,
		Name:        name,
		DisplayName: name,
		Purpose:     clip(purpose, 240),
		Category:    cat,
		Version:     row.Version,
		Icon:        row.IconURL,
		Hub:         HubSkillHubCN,
		ID:          ItemID(HubSkillHubCN, remote),
		Owner:       owner,
		PageURL:     "https://skillhub.cn/skills/" + slug,
		Downloads:   row.Downloads,
		Stars:       row.Stars,
	}, true
}

// fetchSkillHubCN 用 slug 调 /api/v1/skills/<slug>。keen 状态为 malicious、virus 或 trojan 时拒绝。再按版本列出文件并下载。没有 SKILL.md 返回 "pack missing SKILL.md"。
// remote 交给 InstallSlug，空则返回错误，否则经 PathEscape 放进 /api/v1/skills/。
func fetchSkillHubCN(remote string) (PackResult, error) {
	slug := InstallSlug(remote)
	if slug == "" {
		return PackResult{}, fmt.Errorf("skillhub.cn: invalid skill id")
	}
	base := strings.TrimRight(SkillHubCNBase, "/") + "/api/v1/skills/" + url.PathEscape(slug)
	var detail struct {
		Slug  string `json:"slug"`
		Skill struct {
			Slug    string `json:"slug"`
			Summary string `json:"summary"`
		} `json:"skill"`
		LatestVersion *struct {
			Version string `json:"version"`
		} `json:"latestVersion"`
		SecurityReports struct {
			Keen struct {
				Status string `json:"status"`
			} `json:"keen"`
		} `json:"securityReports"`
	}
	if err := getJSON(base, &detail); err != nil {
		return PackResult{}, err
	}
	status := strings.ToLower(strings.TrimSpace(detail.SecurityReports.Keen.Status))
	if status == "malicious" || status == "virus" || status == "trojan" {
		return PackResult{}, fmt.Errorf("skillhub.cn blocked this skill (%s)", status)
	}
	version := ""
	if detail.LatestVersion != nil {
		version = strings.TrimSpace(detail.LatestVersion.Version)
	}
	var listing struct {
		Files []struct {
			Path string `json:"path"`
			Size int    `json:"size"`
		} `json:"files"`
		Version string `json:"version"`
	}
	filesURL := base + "/files"
	if version != "" {
		filesURL += "?" + url.Values{"version": {version}}.Encode()
	}
	if err := getJSON(filesURL, &listing); err != nil {
		return PackResult{}, err
	}
	if listing.Version != "" {
		version = listing.Version
	}
	c := packCollector{}
	for _, f := range listing.Files {
		rel := strings.Trim(strings.ReplaceAll(f.Path, "\\", "/"), "/")
		if !packAllowed(rel) {
			continue
		}
		if f.Size > maxFileBytes && !isSkillMDRel(rel) {
			c.skip(rel, "larger than 1MiB")
			continue
		}
		vals := url.Values{"path": {rel}}
		if version != "" {
			vals.Set("version", version)
		}
		data, err := Get(base + "/file?" + vals.Encode())
		if err != nil {
			if isSkillMDRel(rel) {
				return PackResult{}, fmt.Errorf("%s: %w", rel, err)
			}
			c.skip(rel, err.Error())
			continue
		}
		if err := c.add(rel, data); err != nil {
			return PackResult{}, err
		}
	}
	if !packHasSkillMD(c.files) {
		return PackResult{}, fmt.Errorf("pack missing SKILL.md")
	}
	return PackResult{
		Files:   c.files,
		Warn:    c.warn(),
		URL:     "https://skillhub.cn/skills/" + slug,
		Catalog: "skillhub.cn",
	}, nil
}
