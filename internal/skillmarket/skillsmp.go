package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

var SkillsMPBase = "https://skillsmp.com"

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

func fetchSkillsMP(remote string) (PackResult, error) {
	if gh, ok := parseGitHubURL(remote); ok {
		return fetchGitHubSkill(gh.Repo, gh.Ref, gh.Dir, remote)
	}
	// remote is owner/repo/dir or owner/repo/skill
	parts := strings.Split(strings.Trim(strings.ReplaceAll(remote, "\\", "/"), "/"), "/")
	if len(parts) >= 3 && parts[0] != "" && parts[1] != "" {
		repo := parts[0] + "/" + parts[1]
		dir := strings.Join(parts[2:], "/")
		return fetchGitHubSkill(repo, "main", dir, "https://github.com/"+repo+"/tree/main/"+dir)
	}
	return PackResult{}, fmt.Errorf("skillsmp: need a GitHub skill path")
}
