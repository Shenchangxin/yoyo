// 从 GitHub 树列出并下载带 SKILL.md 的目录。

package skillmarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
)

var (
	// GitHubAPI 是 GitHub API 根地址。测试可以改它。
	GitHubAPI = "https://api.github.com"
	// GitHubRaw 是 raw 文件根地址。测试可以改它。
	GitHubRaw = "https://raw.githubusercontent.com"
)

// listGitHubHub 列出仓库里符合前缀的 SKILL.md，并把 ID 改成 ItemID。条数达到 5000 时 Truncated 为真。
func listGitHubHub(hub, repo, ref string, prefixes []string) (Catalog, error) {
	info, _ := LookupHub(hub)
	items, err := listGitHubSkills(repo, ref, prefixes)
	if err != nil {
		return Catalog{}, err
	}
	for i := range items {
		items[i].Hub = hub
		dir := items[i].ID
		if dir == "" {
			dir = items[i].Slug
		}
		items[i].ID = ItemID(hub, dir)
		if items[i].Repo == "" {
			items[i].Repo = repo
		}
	}
	markFeatured(items)
	return Catalog{
		Source:    info.Name,
		Items:     items,
		Total:     len(items),
		Truncated: len(items) >= maxCatalogLimit,
	}, nil
}

// listGitHubSkills 走递归树，只收目录下的 SKILL.md。repo 必须是 owner/name。ref 空则用 main。跳过 node_modules 和 .git。同目录只收一次。没有 SKILL.md 返回 "no SKILL.md files"。最多 5000 条。
func listGitHubSkills(repo, ref string, prefixes []string) ([]Item, error) {
	repo = strings.Trim(strings.ReplaceAll(repo, "\\", "/"), "/")
	if repo == "" || strings.Count(repo, "/") != 1 {
		return nil, fmt.Errorf("invalid github repo")
	}
	if strings.TrimSpace(ref) == "" {
		ref = "main"
	}
	tree, err := loadGitTree(repo, ref)
	if err != nil {
		return nil, err
	}
	want := normalizePrefixes(prefixes)
	seenDir := map[string]bool{}
	var items []Item
	for _, n := range tree.Tree {
		if n.Type != "blob" {
			continue
		}
		p := strings.Trim(strings.ReplaceAll(n.Path, "\\", "/"), "/")
		if !strings.EqualFold(path.Base(p), "SKILL.md") {
			continue
		}
		dir := strings.TrimSuffix(p, "/"+path.Base(p))
		if dir == "" || dir == p {
			continue
		}
		if !matchPrefix(dir, want) {
			continue
		}
		if strings.Contains(dir, "node_modules/") || strings.Contains(dir, ".git/") {
			continue
		}
		slug := SanitizeSlug(path.Base(dir))
		if slug == "" || seenDir[dir] {
			continue
		}
		seenDir[dir] = true
		cat := githubCategory(dir, want)
		page := "https://github.com/" + repo + "/tree/" + ref + "/" + dir
		items = append(items, Item{
			Slug:     slug,
			Name:     slug,
			Purpose:  "",
			Category: cat,
			PageURL:  page,
			Repo:     repo,
			ID:       dir,
		})
		if len(items) >= maxCatalogLimit {
			break
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no SKILL.md files in %s", repo)
	}
	return items, nil
}

// fetchGitHubSkill 下载 dir 下允许的文件，页面地址用 pageURL，Catalog 记成仓库名。
func fetchGitHubSkill(repo, ref, dir, pageURL string) (PackResult, error) {
	files, warn, err := fetchGitHubDir(repo, ref, dir)
	if err != nil {
		return PackResult{}, err
	}
	return PackResult{
		Files:   files,
		Warn:    warn,
		URL:     pageURL,
		Catalog: repo,
	}, nil
}

// fetchGitHubDir 从树里取出 dir 下的 blob。非 SKILL.md 且声明大小超过 1 MiB 的文件跳过。根上没有 SKILL.md 返回 "pack missing SKILL.md"。
func fetchGitHubDir(repo, ref, dir string) ([]PackFile, string, error) {
	repo = strings.Trim(strings.ReplaceAll(repo, "\\", "/"), "/")
	if repo == "" || strings.Count(repo, "/") != 1 {
		return nil, "", fmt.Errorf("invalid github repo")
	}
	if strings.TrimSpace(ref) == "" {
		ref = "main"
	}
	dir = strings.Trim(strings.ReplaceAll(dir, "\\", "/"), "/")
	tree, err := loadGitTree(repo, ref)
	if err != nil {
		return nil, "", err
	}
	prefix := ""
	if dir != "" {
		prefix = dir + "/"
	}
	c := packCollector{}
	for _, n := range tree.Tree {
		if n.Type != "blob" {
			continue
		}
		p := strings.ReplaceAll(n.Path, "\\", "/")
		var rel string
		switch {
		case dir != "" && p == dir:
			continue
		case dir != "" && strings.HasPrefix(p, prefix):
			rel = strings.TrimPrefix(p, prefix)
		case dir == "":
			rel = p
		default:
			continue
		}
		if !packAllowed(rel) {
			continue
		}
		if n.Size > maxFileBytes && !isSkillMDRel(rel) {
			c.skip(rel, "larger than 1MiB")
			continue
		}
		data, err := Get(joinEscapedPath(GitHubRaw, repo, ref, p))
		if err != nil {
			if isSkillMDRel(rel) {
				return nil, c.warn(), fmt.Errorf("%s: %w", rel, err)
			}
			c.skip(rel, err.Error())
			continue
		}
		if err := c.add(rel, data); err != nil {
			return nil, c.warn(), err
		}
	}
	if !packHasSkillMD(c.files) {
		return nil, c.warn(), fmt.Errorf("pack missing SKILL.md")
	}
	return c.files, c.warn(), nil
}

// loadGitTree 先要 GitHub 递归树。树被截断或请求失败时改要 jsDelivr 的扁平列表。两边都空则返回 "empty github tree"。
func loadGitTree(repo, ref string) (gitTree, error) {
	raw, err := Get(fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=1", strings.TrimRight(GitHubAPI, "/"), repo, url.PathEscape(ref)))
	var ghTree gitTree
	ghOK := false
	if err == nil && jsonUnmarshal(raw, &ghTree) == nil && len(ghTree.Tree) > 0 {
		if !ghTree.Truncated {
			return ghTree, nil
		}
		ghOK = true
	}
	raw, err2 := Get(jsdelivrTreeURL(repo, ref))
	if err2 == nil {
		tree, err3 := parseJSDelivrTree(raw)
		if err3 == nil && len(tree.Tree) > 0 {
			return tree, nil
		}
	}
	if ghOK {
		return ghTree, nil
	}
	if err != nil {
		return gitTree{}, err
	}
	if err2 != nil {
		return gitTree{}, err2
	}
	return gitTree{}, fmt.Errorf("empty github tree")
}

// parseJSDelivrTree 把 jsDelivr 的 files 列表转成 type 为 blob 的树。
func parseJSDelivrTree(raw []byte) (gitTree, error) {
	var wrap struct {
		Files []struct {
			Name string `json:"name"`
			Size int    `json:"size"`
		} `json:"files"`
	}
	if err := jsonUnmarshal(raw, &wrap); err != nil {
		return gitTree{}, fmt.Errorf("jsdelivr tree: %w", err)
	}
	var tree gitTree
	for _, f := range wrap.Files {
		p := strings.TrimPrefix(strings.ReplaceAll(f.Name, "\\", "/"), "/")
		if p == "" {
			continue
		}
		tree.Tree = append(tree.Tree, struct {
			Path string `json:"path"`
			Type string `json:"type"`
			Size int    `json:"size"`
		}{Path: p, Type: "blob", Size: f.Size})
	}
	return tree, nil
}

// githubSkillDir 在 remote 已含斜杠时原样返回。否则接上前缀和安装 slug。前缀为空时只返回 slug。
func githubSkillDir(prefix, remote string) string {
	remote = strings.Trim(strings.ReplaceAll(remote, "\\", "/"), "/")
	if strings.Contains(remote, "/") {
		return remote
	}
	slug := InstallSlug(remote)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return slug
	}
	return prefix + "/" + slug
}

// fetchClaudeSkills 从 alirezarezvani/claude-skills 的 main 取文件。remote 含斜杠时按该路径；否则依次试 .claude/skills/<slug> 和 skills/<slug>。都失败返回 "claude-skills: <slug> not found"。
func fetchClaudeSkills(remote string) (PackResult, error) {
	if strings.Contains(strings.ReplaceAll(remote, "\\", "/"), "/") {
		dir := githubSkillDir("", remote)
		res, err := fetchGitHubSkill("alirezarezvani/claude-skills", "main", dir, "https://github.com/alirezarezvani/claude-skills/tree/main/"+dir)
		if err == nil {
			return res, nil
		}
	}
	slug := InstallSlug(remote)
	for _, prefix := range []string{".claude/skills/", "skills/"} {
		res, err := fetchGitHubSkill("alirezarezvani/claude-skills", "main", prefix+slug, "https://github.com/alirezarezvani/claude-skills/tree/main/"+prefix+slug)
		if err == nil {
			return res, nil
		}
	}
	return PackResult{}, fmt.Errorf("claude-skills: %s not found", slug)
}

// fetchVercelSkill 从 vercel-labs/agent-skills 的 main 取文件。含斜杠时按该路径；否则先试 skills/<slug>，再试仓库根下的 slug。都失败返回 "vercel skill <slug> not found"。
func fetchVercelSkill(remote string) (PackResult, error) {
	if strings.Contains(strings.ReplaceAll(remote, "\\", "/"), "/") {
		dir := githubSkillDir("", remote)
		res, err := fetchGitHubSkill("vercel-labs/agent-skills", "main", dir, "https://github.com/vercel-labs/agent-skills/tree/main/"+dir)
		if err == nil {
			return res, nil
		}
	}
	slug := InstallSlug(remote)
	for _, prefix := range []string{"skills/", ""} {
		dir := slug
		if prefix != "" {
			dir = prefix + slug
		}
		res, err := fetchGitHubSkill("vercel-labs/agent-skills", "main", dir, "https://github.com/vercel-labs/agent-skills/tree/main/"+dir)
		if err == nil {
			return res, nil
		}
	}
	return PackResult{}, fmt.Errorf("vercel skill %s not found", slug)
}

type ghRepoRef struct {
	Repo string
	Ref  string
	Dir  string
}

// parseGitHubURL 只接受主机为 github.com 的地址。tree 或 blob 的第四段是 ref，后面是目录。blob 以 SKILL.md 结尾时目录去掉这一段。默认 ref 是 main。
func parseGitHubURL(raw string) (ghRepoRef, bool) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host != "github.com" {
		return ghRepoRef{}, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ghRepoRef{}, false
	}
	out := ghRepoRef{Repo: parts[0] + "/" + parts[1], Ref: "main"}
	if len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob") {
		out.Ref = parts[3]
		rest := strings.Join(parts[4:], "/")
		if parts[2] == "blob" && strings.EqualFold(path.Base(rest), "SKILL.md") {
			rest = strings.TrimSuffix(rest, "/"+path.Base(rest))
		}
		out.Dir = strings.Trim(rest, "/")
	}
	return out, true
}

// normalizePrefixes 去掉前缀两侧斜杠。传入空切片时返回只含空串的切片，表示不限制前缀。
func normalizePrefixes(prefixes []string) []string {
	if len(prefixes) == 0 {
		return []string{""}
	}
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		p = strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
		out = append(out, p)
	}
	return out
}

// matchPrefix 在任一前缀为空，或目录等于该前缀、或以该前缀加斜杠开头时返回 true。
func matchPrefix(dir string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			return true
		}
		if dir == p || strings.HasPrefix(dir, p+"/") {
			return true
		}
	}
	return false
}

// githubCategory 取去掉前缀后的第一段当分类。对不上时返回 "Skills"。
func githubCategory(dir string, prefixes []string) string {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		rest := dir
		if dir == p {
			return p
		}
		if strings.HasPrefix(dir, p+"/") {
			rest = strings.TrimPrefix(dir, p+"/")
		}
		if i := strings.IndexByte(rest, '/'); i > 0 {
			return rest[:i]
		}
	}
	return "Skills"
}

// jsonUnmarshal 拒绝空白正文和以 < 开头的正文，其余按 JSON 解析。
func jsonUnmarshal(raw []byte, dest any) error {
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim[0] == '<' {
		return fmt.Errorf("expected JSON")
	}
	return json.Unmarshal(raw, dest)
}
