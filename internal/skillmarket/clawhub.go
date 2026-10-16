package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const defaultClawHub = "https://clawhub.ai"

var ClawHubBase = defaultClawHub

func clawHubBases() []string {
	if ClawHubBase != defaultClawHub {
		return []string{strings.TrimRight(ClawHubBase, "/")}
	}
	return uniqueBases([]string{defaultClawHub, "https://clawhubcn.com"})
}

func listClawHub(query string, limit int) (Catalog, error) {
	limit = catalogLimit(query, limit)
	var last error
	for _, base := range clawHubBases() {
		c, err := listClawHubAt(base, query, limit)
		if err == nil {
			return c, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("clawhub unreachable")
	}
	return Catalog{}, last
}

func listClawHubAt(base, query string, limit int) (Catalog, error) {
	src := "ClawHub"
	if host := hostOf(base); host != "" && host != "clawhub.ai" {
		src = "ClawHub (" + host + ")"
	}
	limit = catalogLimit(query, limit)
	pageSize := hubPageSize
	if limit < pageSize {
		pageSize = limit
	}
	q := strings.TrimSpace(query)
	seen := map[string]bool{}
	var items []Item
	cursor := ""
	truncated := false
	for page := 0; page < maxCatalogPages && len(items) < limit; page++ {
		vals := url.Values{
			"limit":             {strconv.Itoa(pageSize)},
			"nonSuspiciousOnly": {"true"},
		}
		if cursor != "" {
			vals.Set("cursor", cursor)
		}
		var batch []Item
		var next string
		var err error
		if q != "" {
			vals.Set("q", q)
			batch, next, err = clawHubSearchPage(base, vals)
		} else {
			vals.Set("sort", "downloads")
			batch, next, err = clawHubListPage(base, vals)
		}
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
			if q == "" && len(items) < 8 {
				it.Featured = true
			}
			items = append(items, it)
			added++
			if len(items) >= limit {
				break
			}
		}
		if next == "" || next == cursor || added == 0 {
			cursor = ""
			break
		}
		cursor = next
	}
	if len(items) >= limit && cursor != "" {
		truncated = true
	}
	return Catalog{Source: src, Items: items, Truncated: truncated}, nil
}

func clawHubListPage(base string, vals url.Values) ([]Item, string, error) {
	var resp struct {
		Items      []clawListItem `json:"items"`
		NextCursor string         `json:"nextCursor"`
	}
	if err := getJSON(strings.TrimRight(base, "/")+"/api/v1/skills?"+vals.Encode(), &resp); err != nil {
		return nil, "", err
	}
	items := make([]Item, 0, len(resp.Items))
	for _, row := range resp.Items {
		if it, ok := itemFromClawList(row); ok {
			items = append(items, it)
		}
	}
	return items, strings.TrimSpace(resp.NextCursor), nil
}

func clawHubSearchPage(base string, vals url.Values) ([]Item, string, error) {
	var resp struct {
		Results    []clawSearchHit `json:"results"`
		NextCursor string          `json:"nextCursor"`
	}
	if err := getJSON(strings.TrimRight(base, "/")+"/api/v1/search?"+vals.Encode(), &resp); err != nil {
		return nil, "", err
	}
	items := make([]Item, 0, len(resp.Results))
	for _, hit := range resp.Results {
		if it, ok := itemFromClawSearch(hit); ok {
			items = append(items, it)
		}
	}
	return items, strings.TrimSpace(resp.NextCursor), nil
}

type clawListItem struct {
	OwnerHandle   string `json:"ownerHandle"`
	Owner         *struct {
		Handle string `json:"handle"`
	} `json:"owner"`
	Slug          string `json:"slug"`
	DisplayName   string `json:"displayName"`
	Summary       string `json:"summary"`
	LatestVersion *struct {
		Version string `json:"version"`
	} `json:"latestVersion"`
	Stats struct {
		Downloads int `json:"downloads"`
		Stars     int `json:"stars"`
	} `json:"stats"`
}

type clawSearchHit struct {
	DisplayName string `json:"displayName"`
	Canonical   string `json:"canonicalUrl"`
	Featured    bool   `json:"featured"`
	Install     struct {
		Reference string `json:"reference"`
	} `json:"install"`
	Native struct {
		OwnerHandle string `json:"ownerHandle"`
		Skill       struct {
			Slug    string `json:"slug"`
			Summary string `json:"summary"`
			Stats   struct {
				Downloads int `json:"downloads"`
				Stars     int `json:"stars"`
			} `json:"stats"`
		} `json:"skill"`
	} `json:"native"`
}

func itemFromClawList(row clawListItem) (Item, bool) {
	slug := SanitizeSlug(row.Slug)
	owner := strings.TrimPrefix(strings.TrimSpace(row.OwnerHandle), "@")
	if owner == "" && row.Owner != nil {
		owner = strings.TrimPrefix(strings.TrimSpace(row.Owner.Handle), "@")
	}
	if slug == "" {
		return Item{}, false
	}
	name := strings.TrimSpace(row.DisplayName)
	if name == "" {
		name = slug
	}
	ver := ""
	if row.LatestVersion != nil {
		ver = row.LatestVersion.Version
	}
	remote := slug
	if owner != "" {
		remote = owner + "/" + slug
	}
	page := "https://clawhub.ai/skills/" + slug
	if owner != "" {
		page = "https://clawhub.ai/" + owner + "/skills/" + slug
	}
	return Item{
		Slug:        slug,
		Name:        name,
		DisplayName: name,
		Purpose:     clip(row.Summary, 240),
		Category:    "ClawHub",
		Version:     ver,
		Hub:         HubClawHub,
		ID:          ItemID(HubClawHub, remote),
		Owner:       owner,
		PageURL:     page,
		Downloads:   row.Stats.Downloads,
		Stars:       row.Stats.Stars,
	}, true
}

func itemFromClawSearch(hit clawSearchHit) (Item, bool) {
	owner := strings.TrimPrefix(strings.TrimSpace(hit.Native.OwnerHandle), "@")
	slug := SanitizeSlug(hit.Native.Skill.Slug)
	if (owner == "" || slug == "") && hit.Install.Reference != "" {
		owner, slug = splitOwnerSlug(hit.Install.Reference)
	}
	if slug == "" || owner == "" {
		return Item{}, false
	}
	name := strings.TrimSpace(hit.DisplayName)
	if name == "" {
		name = slug
	}
	remote := owner + "/" + slug
	return Item{
		Slug:        slug,
		Name:        name,
		DisplayName: name,
		Purpose:     clip(hit.Native.Skill.Summary, 240),
		Category:    "ClawHub",
		Featured:    hit.Featured,
		Hub:         HubClawHub,
		ID:          ItemID(HubClawHub, remote),
		Owner:       owner,
		PageURL:     "https://clawhub.ai/" + owner + "/skills/" + slug,
		Downloads:   hit.Native.Skill.Stats.Downloads,
		Stars:       hit.Native.Skill.Stats.Stars,
	}, true
}

func fetchClawHub(remote string) (PackResult, error) {
	owner, slug := splitOwnerSlug(remote)
	if slug == "" {
		slug = SanitizeSlug(remote)
	}
	if slug == "" {
		return PackResult{}, fmt.Errorf("clawhub id must be owner/slug")
	}
	var last error
	for _, root := range clawHubBases() {
		pack, err := fetchClawHubAt(root, owner, slug)
		if err == nil {
			return pack, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("clawhub unreachable")
	}
	return PackResult{}, last
}

func fetchClawHubAt(root, owner, slug string) (PackResult, error) {
	q := url.Values{}
	if owner != "" {
		q.Set("ownerHandle", owner)
	}
	base := strings.TrimRight(root, "/") + "/api/v1/skills/" + url.PathEscape(slug)
	suffix := ""
	if encoded := q.Encode(); encoded != "" {
		suffix = "?" + encoded
	}
	var detail struct {
		Skill struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"displayName"`
		} `json:"skill"`
		LatestVersion *struct {
			Version string `json:"version"`
		} `json:"latestVersion"`
		Moderation struct {
			IsMalwareBlocked bool   `json:"isMalwareBlocked"`
			Verdict          string `json:"verdict"`
		} `json:"moderation"`
	}
	if err := getJSON(base+suffix, &detail); err != nil {
		return PackResult{}, err
	}
	if detail.Moderation.IsMalwareBlocked || strings.EqualFold(detail.Moderation.Verdict, "malicious") {
		return PackResult{}, fmt.Errorf("clawhub blocked this skill (%s)", detail.Moderation.Verdict)
	}
	version := ""
	if detail.LatestVersion != nil {
		version = strings.TrimSpace(detail.LatestVersion.Version)
	}
	if version == "" {
		var vers struct {
			Items []struct {
				Version string `json:"version"`
			} `json:"items"`
		}
		if err := getJSON(base+"/versions"+suffix, &vers); err != nil {
			return PackResult{}, err
		}
		if len(vers.Items) == 0 {
			return PackResult{}, fmt.Errorf("clawhub skill has no versions")
		}
		version = vers.Items[0].Version
	}
	var ver struct {
		Version struct {
			Version string `json:"version"`
			Files   []struct {
				Path        string `json:"path"`
				Size        int    `json:"size"`
				ContentType string `json:"contentType"`
			} `json:"files"`
			Security struct {
				Status string `json:"status"`
			} `json:"security"`
		} `json:"version"`
	}
	if err := getJSON(base+"/versions/"+url.PathEscape(version)+suffix, &ver); err != nil {
		return PackResult{}, err
	}
	status := strings.ToLower(strings.TrimSpace(ver.Version.Security.Status))
	if status == "malicious" {
		return PackResult{}, fmt.Errorf("clawhub scan marked this skill malicious")
	}
	c := packCollector{}
	for _, f := range ver.Version.Files {
		rel := strings.Trim(strings.ReplaceAll(f.Path, "\\", "/"), "/")
		if !packAllowed(rel) {
			continue
		}
		if strings.Contains(strings.ToLower(f.ContentType), "zip") {
			continue
		}
		if f.Size > maxFileBytes && !isSkillMDRel(rel) {
			c.skip(rel, "larger than 1MiB")
			continue
		}
		vals := url.Values{"path": {rel}, "version": {version}}
		if owner != "" {
			vals.Set("ownerHandle", owner)
		}
		fileURL := base + "/file?" + vals.Encode()
		data, err := Get(fileURL)
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
		vals := url.Values{"path": {"SKILL.md"}, "version": {version}}
		if owner != "" {
			vals.Set("ownerHandle", owner)
		}
		data, err := Get(base + "/file?" + vals.Encode())
		if err != nil {
			return PackResult{}, fmt.Errorf("pack missing SKILL.md")
		}
		if err := c.add("SKILL.md", data); err != nil {
			return PackResult{}, err
		}
		return clawPack(root, owner, slug, c.files, joinWarn("installed SKILL.md only", c.warn())), nil
	}
	return clawPack(root, owner, slug, c.files, c.warn()), nil
}

func clawPack(root, owner, slug string, files []PackFile, warn string) PackResult {
	host := hostOf(root)
	page := "https://clawhub.ai/skills/" + slug
	if owner != "" {
		page = "https://clawhub.ai/" + owner + "/skills/" + slug
	}
	if host != "" && host != "clawhub.ai" {
		page = strings.TrimRight(root, "/") + "/" + owner + "/skills/" + slug
		if owner == "" {
			page = strings.TrimRight(root, "/") + "/skills/" + slug
		}
	}
	catalog := "clawhub.ai"
	if host != "" && host != "clawhub.ai" {
		catalog = host
	}
	return PackResult{Files: files, Warn: warn, URL: page, Catalog: catalog, Owner: owner}
}
