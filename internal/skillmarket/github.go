package skillmarket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
)

var (
	GitHubAPI = "https://api.github.com"
	GitHubRaw = "https://raw.githubusercontent.com"
)

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

func jsonUnmarshal(raw []byte, dest any) error {
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim[0] == '<' {
		return fmt.Errorf("expected JSON")
	}
	return json.Unmarshal(raw, dest)
}
