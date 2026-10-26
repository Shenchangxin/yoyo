// ClawHub 列表和下载。默认站不可达时再试 clawhubcn.com。

package skillmarket

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const defaultClawHub = "https://clawhub.ai"

// ClawHubBase 是 ClawHub 根地址，默认 https://clawhub.ai。测试可以改它。改过之后不再自动尝试 clawhubcn.com。
var ClawHubBase = defaultClawHub

// clawHubBases 在 ClawHubBase 仍是默认值时返回 clawhub.ai 和 clawhubcn.com。否则只返回当前基址。
func clawHubBases() []string {
	if ClawHubBase != defaultClawHub {
		return []string{strings.TrimRight(ClawHubBase, "/")}
	}
	return uniqueBases([]string{defaultClawHub, "https://clawhubcn.com"})
}

// listClawHub 按 clawHubBases 的顺序试列表。都失败且没有错误时返回 "clawhub unreachable"。
// query 先交给 catalogLimit，再传给 listClawHubAt。
// limit 先交给 catalogLimit，再传给 listClawHubAt。
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

// listClawHubAt 按 cursor 翻页，最多 40 页。无查询时 sort=downloads 且 nonSuspiciousOnly=true，有查询时走搜索。浏览结果的前 8 条标成 Featured。
// base 交给 hostOf 决定 Source，并传给 clawHubSearchPage 或 clawHubListPage。
// query 去掉两端空白，非空时走搜索并写入 q；空则列表 sort 为 downloads，并把前 8 条标成 Featured。
// limit 先交给 catalogLimit；已收集条数达到它就停，页大小不超过它。
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

// clawHubListPage 请求 /api/v1/skills，并返回下一页 cursor。
// base 去掉末尾斜杠后拼上 /api/v1/skills。
// vals 经 Encode 接在查询串上，整段交给 getJSON。
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

// clawHubSearchPage 请求 /api/v1/search，并返回下一页 cursor。
// base 去掉末尾斜杠后拼上 /api/v1/search。
// vals 经 Encode 接在查询串上，整段交给 getJSON。
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
	OwnerHandle string `json:"ownerHandle"`
	Owner       *struct {
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

// itemFromClawList 把列表行变成条目。slug 为空则返回 false。没有 owner 时 ID 只有 slug。摘要截到 240 字。
// row 的 Slug 交给 SanitizeSlug，空则返回 false；Summary 交给 clip，第二个实参是 240。
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

// itemFromClawSearch 从搜索命中取出 owner 和 slug。缺一则返回 false。
// hit 取出 owner 和 slug，缺一则返回 false；Summary 交给 clip，第二个实参是 240。
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

// fetchClawHub 按基址顺序下载。slug 为空返回 "clawhub id must be owner/slug"。都失败时返回 "clawhub unreachable"。
// remote 交给 splitOwnerSlug；得到的 slug 为空时再把 remote 交给 SanitizeSlug，仍空则返回错误。
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

// fetchClawHubAt 读技能详情和版本文件。isMalwareBlocked 或 verdict 为 malicious，或版本 security.status 为 malicious 时拒绝。跳过 contentType 含 zip 的文件。根上没有 SKILL.md 时再单独要一次。
// root 去掉末尾斜杠后拼 API 路径，最后交给 clawPack。
// owner 非空时写入查询的 ownerHandle，并传给 clawPack。
// slug 经 PathEscape 放进 /api/v1/skills/ 路径，并传给 clawPack。
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

// clawPack 组装结果。主机不是 clawhub.ai 时，页面和 Catalog 改用这次实际的根地址。
// root 交给 hostOf；主机不是 clawhub.ai 时页面和 Catalog 改用这次的根地址。
// owner 非空时页面路径带上它，并写入结果的 Owner。
// slug 拼进页面路径。
// files 写入结果的 Files。
// warn 写入结果的 Warn。
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
