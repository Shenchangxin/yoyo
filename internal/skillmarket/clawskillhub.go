package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var ClawSkillHubBase = "https://clawskillhub.com"

type cshOwner struct {
	Handle    string `json:"handle"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type cshSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Downloads   int      `json:"downloads"`
	Stars       int      `json:"stars"`
	Tags        []string `json:"tags"`
	Score       int      `json:"validationScore"`
	Owner       cshOwner `json:"owner"`
	Versions    []struct {
		Version string `json:"version"`
		SkillMD string `json:"skillMd"`
	} `json:"versions"`
}

func listClawSkillHub(query string, limit int) (Catalog, error) {
	limit = catalogLimit(query, limit)
	base := strings.TrimRight(ClawSkillHubBase, "/")
	q := strings.TrimSpace(query)
	pageSize := hubPageSize
	if limit < pageSize {
		pageSize = limit
	}
	first, err := fetchClawSkillHubPage(base, q, pageSize, 0)
	if err != nil {
		return Catalog{}, err
	}
	rows := first
	if len(first) >= pageSize && limit > len(first) {
		second, err := fetchClawSkillHubPage(base, q, pageSize, pageSize)
		if err == nil && clawSkillHubPageAdds(first, second) {
			rows = append(rows, second...)
			need := limit - len(rows)
			extra := (need + pageSize - 1) / pageSize
			if extra > maxCatalogPages-2 {
				extra = maxCatalogPages - 2
			}
			more := make([][]cshSkill, extra)
			var wg sync.WaitGroup
			sem := make(chan struct{}, 4)
			for i := 0; i < extra; i++ {
				offset := (i + 2) * pageSize
				wg.Add(1)
				go func(i, offset int) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					batch, err := fetchClawSkillHubPage(base, q, pageSize, offset)
					if err != nil {
						return
					}
					more[i] = batch
				}(i, offset)
			}
			wg.Wait()
			for _, batch := range more {
				if len(batch) == 0 {
					break
				}
				rows = append(rows, batch...)
				if len(batch) < pageSize {
					break
				}
			}
		}
	}
	seen := map[string]bool{}
	var out []Item
	for _, row := range rows {
		it, ok := itemFromClawSkillHub(row)
		if !ok || seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		if q == "" && len(out) < 8 {
			it.Featured = true
		}
		if row.Score >= 85 {
			it.Featured = true
		}
		out = append(out, it)
		if len(out) >= limit {
			break
		}
	}
	return Catalog{Source: "SkillHub", Items: out, Truncated: len(out) >= limit}, nil
}

func clawSkillHubPageAdds(prev, next []cshSkill) bool {
	if len(next) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, row := range prev {
		if it, ok := itemFromClawSkillHub(row); ok {
			seen[it.ID] = true
		}
	}
	for _, row := range next {
		if it, ok := itemFromClawSkillHub(row); ok && !seen[it.ID] {
			return true
		}
	}
	return false
}

func fetchClawSkillHubPage(base, query string, limit, offset int) ([]cshSkill, error) {
	vals := url.Values{
		"sort":   {"downloads"},
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}
	if query != "" {
		u := base + "/api/cli/search?" + url.Values{"q": {query}, "limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}}.Encode()
		var hits []struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
			Stars       int    `json:"stars"`
			Downloads   int    `json:"downloads"`
			Owner       string `json:"owner"`
		}
		if err := getJSON(u, &hits); err == nil {
			items := make([]cshSkill, 0, len(hits))
			for _, h := range hits {
				items = append(items, cshSkill{
					Name:        h.Name,
					Slug:        h.Slug,
					Description: h.Description,
					Stars:       h.Stars,
					Downloads:   h.Downloads,
					Owner:       cshOwner{Handle: h.Owner},
				})
			}
			return items, nil
		}
		vals.Set("search", query)
	}
	var items []cshSkill
	if err := getJSON(base+"/api/skills?"+vals.Encode(), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func itemFromClawSkillHub(row cshSkill) (Item, bool) {
	slug := SanitizeSlug(row.Slug)
	if slug == "" {
		slug = SanitizeSlug(row.Name)
	}
	owner := strings.TrimSpace(row.Owner.Handle)
	if owner == "" {
		owner = "skillhub"
	}
	if slug == "" {
		return Item{}, false
	}
	name := strings.TrimSpace(row.Name)
	if name == "" {
		name = slug
	}
	cat := "SkillHub"
	if len(row.Tags) > 0 && strings.TrimSpace(row.Tags[0]) != "" {
		cat = row.Tags[0]
	}
	remote := owner + "/" + slug
	return Item{
		Slug:        slug,
		Name:        name,
		DisplayName: name,
		Purpose:     clip(row.Description, 240),
		Category:    cat,
		Hub:         HubClawSkillHub,
		ID:          ItemID(HubClawSkillHub, remote),
		Owner:       owner,
		PageURL:     "https://clawskillhub.com/skills/" + owner + "/" + slug,
		Downloads:   row.Downloads,
		Stars:       row.Stars,
	}, true
}

func fetchClawSkillHub(remote string) (PackResult, error) {
	owner, slug := splitOwnerSlug(remote)
	if owner == "" || slug == "" {
		return PackResult{}, fmt.Errorf("skillhub id must be owner/slug")
	}
	base := strings.TrimRight(ClawSkillHubBase, "/")
	var info cshSkill
	if err := getJSON(base+"/api/skills/"+url.PathEscape(owner)+"/"+url.PathEscape(slug), &info); err != nil {
		return PackResult{}, err
	}
	version := "1.0.0"
	if len(info.Versions) > 0 && strings.TrimSpace(info.Versions[0].Version) != "" {
		version = info.Versions[0].Version
	}
	var inst struct {
		Skill struct {
			Name  string `json:"name"`
			Slug  string `json:"slug"`
			Owner string `json:"owner"`
		} `json:"skill"`
		Version string `json:"version"`
		SkillMD string `json:"skillMd"`
		Readme  string `json:"readme"`
		Files   []struct {
			Path     string `json:"path"`
			Content  string `json:"content"`
			IsBinary bool   `json:"isBinary"`
		} `json:"files"`
	}
	installURL := base + "/api/cli/skills/" + url.PathEscape(owner) + "/" + url.PathEscape(slug) + "/install?" + url.Values{"version": {version}}.Encode()
	if err := getJSON(installURL, &inst); err != nil {
		if len(info.Versions) > 0 && strings.TrimSpace(info.Versions[0].SkillMD) != "" {
			return PackResult{
				Files:   []PackFile{{Rel: "SKILL.md", Data: []byte(info.Versions[0].SkillMD)}},
				Warn:    "installed SKILL.md only",
				URL:     "https://clawskillhub.com/skills/" + owner + "/" + slug,
				Catalog: "clawskillhub.com",
				Owner:   owner,
			}, nil
		}
		return PackResult{}, err
	}
	c := packCollector{}
	if strings.TrimSpace(inst.SkillMD) != "" {
		if err := c.add("SKILL.md", []byte(inst.SkillMD)); err != nil {
			return PackResult{}, err
		}
	}
	if strings.TrimSpace(inst.Readme) != "" {
		_ = c.add("README.md", []byte(inst.Readme))
	}
	for _, f := range inst.Files {
		if f.IsBinary {
			continue
		}
		_ = c.add(f.Path, []byte(f.Content))
	}
	if !packHasSkillMD(c.files) {
		return PackResult{}, fmt.Errorf("pack missing SKILL.md")
	}
	return PackResult{
		Files:   c.files,
		Warn:    c.warn(),
		URL:     "https://clawskillhub.com/skills/" + owner + "/" + slug,
		Catalog: "clawskillhub.com",
		Owner:   owner,
	}, nil
}
