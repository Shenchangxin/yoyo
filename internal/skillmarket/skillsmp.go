// SkillsMP 只支持搜索。安装时按记录的 GitHub 路径取文件。

package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SkillsMPBase 是 SkillsMP 根地址。测试可以改它。
var SkillsMPBase = "https://skillsmp.com"

// listSkillsMP 在查询为空时返回 SearchOnly，不发请求。有查询时按每页最多 50 条翻页，最多 40 页，按 stars 排序。
// query 去掉两端空白，空则返回 SearchOnly，不再请求。
// limit 在有查询时交给 catalogLimit；收集条数达到它就停，页大小不超过 50。
func listSkillsMP(query string, limit int) (Catalog, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return Catalog{Source: "SkillsMP", SearchOnly: true}, nil
	}
	limit = catalogLimit(query, limit)
	pageSize := 50
	if limit < pageSize {
		pageSize = limit
	}
	seen := map[string]bool{}
	var items []Item
	for page := 1; page <= maxCatalogPages && len(items) < limit; page++ {
		batch, err := fetchSkillsMPPage(q, page, pageSize)
		if err != nil {
			if len(items) > 0 {
				break
			}
			return Catalog{}, err
		}
		added := 0
		for _, it := range batch {
			if it.ID == "" || seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			items = append(items, it)
			added++
			if len(items) >= limit {
				break
			}
		}
		if added == 0 || len(batch) < pageSize {
			break
		}
	}
	return Catalog{Source: "SkillsMP", Items: items, Truncated: len(items) >= limit}, nil
}

type skillsMPRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	GitHubURL   string `json:"githubUrl"`
	SkillURL    string `json:"skillUrl"`
	Stars       int    `json:"stars"`
}

// fetchSkillsMPPage 请求 /api/v1/skills/search。page 大于 1 时同时带 page 和 offset。
// query 写入查询的 q。
// pageSize 写入查询的 limit；page 大于 1 时 offset 用 (page-1) 乘 pageSize。
func fetchSkillsMPPage(query string, page, pageSize int) ([]Item, error) {
	vals := url.Values{
		"q":      {query},
		"limit":  {strconv.Itoa(pageSize)},
		"sortBy": {"stars"},
	}
	if page > 1 {
		vals.Set("page", strconv.Itoa(page))
		vals.Set("offset", strconv.Itoa((page-1)*pageSize))
	}
	u := strings.TrimRight(SkillsMPBase, "/") + "/api/v1/skills/search?" + vals.Encode()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Skills []skillsMPRow `json:"skills"`
		} `json:"data"`
	}
	if err := getJSON(u, &resp); err != nil {
		if page > 1 {
			return nil, err
		}
		return nil, err
	}
	items := make([]Item, 0, len(resp.Data.Skills))
	for _, row := range resp.Data.Skills {
		if it, ok := itemFromSkillsMP(row); ok {
			items = append(items, it)
		}
	}
	return items, nil
}

// itemFromSkillsMP 用名字做 slug。能解析 githubUrl 时 ID 带上仓库和目录。名字净化失败返回 false。
// row 的 Name 交给 SanitizeSlug，空则返回 false；GitHubURL 交给 parseGitHubURL，Description 交给 clip，第二个实参是 240。
func itemFromSkillsMP(row skillsMPRow) (Item, bool) {
	slug := SanitizeSlug(row.Name)
	if slug == "" {
		return Item{}, false
	}
	gh, ok := parseGitHubURL(row.GitHubURL)
	remote := slug
	repo := ""
	if ok {
		repo = gh.Repo
		if gh.Dir != "" {
			remote = gh.Repo + "/" + gh.Dir
		} else {
			remote = gh.Repo + "/" + slug
		}
	} else if row.ID != "" {
		remote = row.ID
	}
	name := strings.TrimSpace(row.Name)
	return Item{
		Slug:        slug,
		Name:        name,
		DisplayName: name,
		Purpose:     clip(row.Description, 240),
		Category:    "SkillsMP",
		Hub:         HubSkillsMP,
		ID:          ItemID(HubSkillsMP, remote),
		Owner:       row.Author,
		PageURL:     row.SkillURL,
		Repo:        repo,
		Stars:       row.Stars,
	}, true
}

// fetchSkillsMP 在 remote 是 GitHub 地址时直接下载。否则要求至少 owner/repo/dir 三段，ref 用 main。都不是时返回 "skillsmp: need a GitHub skill path"。
func fetchSkillsMP(remote string) (PackResult, error) {
	if gh, ok := parseGitHubURL(remote); ok {
		return fetchGitHubSkill(gh.Repo, gh.Ref, gh.Dir, remote)
	}
	// remote 形如 owner/repo/dir，或 owner/repo/skill。
	parts := strings.Split(strings.Trim(strings.ReplaceAll(remote, "\\", "/"), "/"), "/")
	if len(parts) >= 3 && parts[0] != "" && parts[1] != "" {
		repo := parts[0] + "/" + parts[1]
		dir := strings.Join(parts[2:], "/")
		return fetchGitHubSkill(repo, "main", dir, "https://github.com/"+repo+"/tree/main/"+dir)
	}
	return PackResult{}, fmt.Errorf("skillsmp: need a GitHub skill path")
}
