// Markdown Registry 只支持搜索，文件从登记的 GitHub 路径下载。

package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MDRBase 是 Markdown Registry 根地址。测试可以改它。
var MDRBase = "https://markdownregistry.com"

// listMarkdownRegistry 在查询为空时返回 SearchOnly。有查询时按页搜索 kind=skill，最多 40 页。
// query 去掉两端空白，空则返回 SearchOnly，不再请求。
// limit 在有查询时交给 catalogLimit；收集条数达到它就停。
func listMarkdownRegistry(query string, limit int) (Catalog, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return Catalog{Source: "Markdown Registry", SearchOnly: true}, nil
	}
	limit = catalogLimit(query, limit)
	pageSize := hubPageSize
	if limit < pageSize {
		pageSize = limit
	}
	seen := map[string]bool{}
	var items []Item
	for page := 1; page <= maxCatalogPages && len(items) < limit; page++ {
		batch, err := fetchMDRPage(q, page, pageSize)
		if err != nil {
			if len(items) > 0 {
				break
			}
			return Catalog{}, err
		}
		added := 0
		for _, row := range batch {
			it, ok := itemFromMDR(row)
			if !ok || seen[it.ID] {
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
	return Catalog{Source: "Markdown Registry", Items: items, Truncated: len(items) >= limit}, nil
}

// fetchMDRPage 请求 /api/v1/search。page 大于 1 时同时带 page 和 offset。
// query 写入查询的 q。
// pageSize 写入查询的 limit；page 大于 1 时 offset 用 (page-1) 乘 pageSize。
func fetchMDRPage(query string, page, pageSize int) ([]mdrArtifact, error) {
	vals := url.Values{
		"q":     {query},
		"kind":  {"skill"},
		"limit": {strconv.Itoa(pageSize)},
	}
	if page > 1 {
		vals.Set("page", strconv.Itoa(page))
		vals.Set("offset", strconv.Itoa((page-1)*pageSize))
	}
	u := strings.TrimRight(MDRBase, "/") + "/api/v1/search?" + vals.Encode()
	var resp struct {
		Artifacts []mdrArtifact `json:"artifacts"`
	}
	if err := getJSON(u, &resp); err != nil {
		return nil, err
	}
	return resp.Artifacts, nil
}

type mdrArtifact struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Stars       int    `json:"stars"`
}

// itemFromMDR 只收 kind 为空或 skill 的行。名字净化失败返回 false。
// row 的 Kind 不是空也不是 skill 时返回 false；Name 交给 SanitizeSlug，Description 交给 clip，第二个实参是 240。
func itemFromMDR(row mdrArtifact) (Item, bool) {
	if row.Kind != "" && !strings.EqualFold(row.Kind, "skill") {
		return Item{}, false
	}
	slug := SanitizeSlug(row.Name)
	if slug == "" {
		return Item{}, false
	}
	repo := strings.Trim(row.Owner+"/"+row.Repo, "/")
	remote := row.Owner + "/" + row.Repo + "/" + slug
	page := strings.TrimRight(MDRBase, "/") + "/s/" + row.ID
	if row.ID == "" {
		page = "https://github.com/" + repo
	}
	return Item{
		Slug:        slug,
		Name:        slug,
		DisplayName: slug,
		Purpose:     clip(row.Description, 240),
		Category:    "Markdown Registry",
		Hub:         HubMDR,
		ID:          ItemID(HubMDR, remote),
		Owner:       row.Owner,
		PageURL:     page,
		Repo:        repo,
		Stars:       row.Stars,
	}, true
}

// fetchMarkdownRegistry 用 /api/v1/resolve 取第一条记录，再从该 GitHub 仓库的 main 下载。找不到、仓库不像 owner/name，或没有技能路径时返回错误。
// remote 把反斜杠换成斜杠并去掉两端斜杠后，作为 resolve 的 ref。
func fetchMarkdownRegistry(remote string) (PackResult, error) {
	remote = strings.Trim(strings.ReplaceAll(remote, "\\", "/"), "/")
	u := strings.TrimRight(MDRBase, "/") + "/api/v1/resolve?" + url.Values{"ref": {remote}}.Encode()
	var resp struct {
		Artifacts []mdrArtifact `json:"artifacts"`
	}
	if err := getJSON(u, &resp); err != nil {
		return PackResult{}, err
	}
	if len(resp.Artifacts) == 0 {
		return PackResult{}, fmt.Errorf("markdownregistry: not found")
	}
	row := resp.Artifacts[0]
	repo := strings.Trim(row.Owner+"/"+row.Repo, "/")
	if repo == "/" || strings.Count(repo, "/") != 1 {
		return PackResult{}, fmt.Errorf("markdownregistry: missing repo")
	}
	dir := strings.Trim(strings.ReplaceAll(row.Path, "\\", "/"), "/")
	if strings.EqualFold(pathBase(dir), "skill.md") {
		dir = strings.TrimSuffix(dir, "/"+pathBase(dir))
	}
	if dir == "" {
		return PackResult{}, fmt.Errorf("markdownregistry: missing skill path")
	}
	return fetchGitHubSkill(repo, "main", dir, "https://github.com/"+repo+"/tree/main/"+dir)
}

// pathBase 返回相对路径的最后一段。
// rel 把反斜杠换成斜杠后取最后一段。
func pathBase(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}
