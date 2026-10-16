package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var MDRBase = "https://markdownregistry.com"

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

func pathBase(rel string) string {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}
