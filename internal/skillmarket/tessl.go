// 从 api.tessl.io 列技能，文件仍从记录的 GitHub 路径下载。

package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// TesslBase 是 Tessl API 根地址。测试可以改它。
var TesslBase = "https://api.tessl.io"

type tesslSkill struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SourceURL   string `json:"sourceUrl"`
		SourceRef   string `json:"sourceRef"`
		Path        string `json:"path"`
		SkillMDBody string `json:"skillMdBody"`
	} `json:"attributes"`
}

// listTessl 按页收集，最多 40 页。无查询时浏览接口失败会返回 SearchOnly 和 Warning，而不是错误。前 8 条浏览结果标成 Featured。
func listTessl(query string, limit int) (Catalog, error) {
	limit = catalogLimit(query, limit)
	q := strings.TrimSpace(query)
	pageSize := hubPageSize
	if limit < pageSize {
		pageSize = limit
	}
	seen := map[string]bool{}
	var items []Item
	browse := q == ""
	for page := 1; page <= maxCatalogPages && len(items) < limit; page++ {
		batch, err := fetchTesslPage(q, page, pageSize, browse)
		if err != nil {
			if len(items) > 0 {
				break
			}
			if browse {
				return Catalog{Source: "Tessl", SearchOnly: true, Warning: err.Error()}, nil
			}
			return Catalog{}, err
		}
		added := 0
		for _, row := range batch {
			it, ok := itemFromTessl(row)
			if !ok || seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			if q == "" && len(items) < 8 {
				it.Featured = true
			}
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
	return Catalog{Source: "Tessl", Items: items, Truncated: len(items) >= limit}, nil
}

// fetchTesslPage 无查询的浏览走 /experimental/skills，失败再以 q=skill 打 /experimental/search。有查询时直接搜索。过滤 type 等于 skill。
func fetchTesslPage(query string, page, pageSize int, browse bool) ([]tesslSkill, error) {
	vals := url.Values{
		"filter[type][eq]": {"skill"},
		"page[size]":       {strconv.Itoa(pageSize)},
		"page[number]":     {strconv.Itoa(page)},
	}
	endpoint := "/experimental/search"
	if browse && query == "" {
		endpoint = "/experimental/skills"
	}
	if query != "" {
		vals.Set("q", query)
	}
	u := strings.TrimRight(TesslBase, "/") + endpoint + "?" + vals.Encode()
	var resp struct {
		Data []tesslSkill `json:"data"`
	}
	if err := getJSON(u, &resp); err != nil && browse && query == "" {
		vals.Set("q", "skill")
		u = strings.TrimRight(TesslBase, "/") + "/experimental/search?" + vals.Encode()
		if err2 := getJSON(u, &resp); err2 != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// itemFromTessl 用名字做 slug。能解析 sourceUrl 时，ID 带上仓库和目录。名字净化失败返回 false。
func itemFromTessl(row tesslSkill) (Item, bool) {
	slug := SanitizeSlug(row.Attributes.Name)
	if slug == "" {
		return Item{}, false
	}
	gh, ok := parseGitHubURL(row.Attributes.SourceURL)
	remote := slug
	repo := ""
	if ok {
		repo = gh.Repo
		dir := gh.Dir
		if dir == "" {
			p := strings.Trim(strings.ReplaceAll(row.Attributes.Path, "\\", "/"), "/")
			if strings.EqualFold(pathBase(p), "skill.md") {
				p = strings.TrimSuffix(p, "/"+pathBase(p))
			}
			dir = p
		}
		if dir != "" {
			remote = repo + "/" + dir
		} else {
			remote = repo + "/" + slug
		}
	}
	name := strings.TrimSpace(row.Attributes.Name)
	return Item{
		Slug:        slug,
		Name:        name,
		DisplayName: name,
		Purpose:     clip(row.Attributes.Description, 240),
		Category:    "Tessl",
		Hub:         HubTessl,
		ID:          ItemID(HubTessl, remote),
		PageURL:     row.Attributes.SourceURL,
		Repo:        repo,
	}, true
}

// fetchTessl 若 remote 是 GitHub 地址就直接下载该目录。否则按 owner/repo/余下路径取详情里的 ref 和 path。GitHub 下载失败且详情里有 skillMdBody 时，只装 SKILL.md。段数不足返回 "tessl: need owner/repo/skill"。
func fetchTessl(remote string) (PackResult, error) {
	remote = strings.Trim(strings.ReplaceAll(remote, "\\", "/"), "/")
	if gh, ok := parseGitHubURL(remote); ok {
		return fetchGitHubSkill(gh.Repo, gh.Ref, gh.Dir, remote)
	}
	parts := strings.Split(remote, "/")
	if len(parts) >= 3 {
		repo := parts[0] + "/" + parts[1]
		dir := strings.Join(parts[2:], "/")
		ref := "main"
		detailURL := strings.TrimRight(TesslBase, "/") + "/experimental/skills/github/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/" + url.PathEscape(InstallSlug(remote))
		var wrap struct {
			Data tesslSkill `json:"data"`
		}
		if err := getJSON(detailURL, &wrap); err == nil {
			if wrap.Data.Attributes.SourceRef != "" {
				ref = wrap.Data.Attributes.SourceRef
			}
			p := strings.Trim(strings.ReplaceAll(wrap.Data.Attributes.Path, "\\", "/"), "/")
			if strings.EqualFold(pathBase(p), "skill.md") {
				p = strings.TrimSuffix(p, "/"+pathBase(p))
			}
			if p != "" {
				dir = p
			}
			res, err := fetchGitHubSkill(repo, ref, dir, wrap.Data.Attributes.SourceURL)
			if err == nil {
				return res, nil
			}
			if body := strings.TrimSpace(wrap.Data.Attributes.SkillMDBody); body != "" {
				md := body
				if !strings.HasPrefix(md, "---") {
					md = "---\nname: " + InstallSlug(remote) + "\ndescription: " + clip(wrap.Data.Attributes.Description, 200) + "\n---\n" + body
				}
				return PackResult{
					Files:   []PackFile{{Rel: "SKILL.md", Data: []byte(md)}},
					Warn:    "installed SKILL.md only",
					URL:     wrap.Data.Attributes.SourceURL,
					Catalog: "tessl.io",
				}, nil
			}
		}
		return fetchGitHubSkill(repo, "main", dir, "https://github.com/"+repo+"/tree/main/"+dir)
	}
	return PackResult{}, fmt.Errorf("tessl: need owner/repo/skill")
}
