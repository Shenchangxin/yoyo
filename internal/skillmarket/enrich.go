// 用 Git 树和少量 SKILL.md 前言补全 Work Buddy 条目的图标和脚本标记。

package skillmarket

import (
	"encoding/json"
	"path"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type gitTree struct {
	SHA       string `json:"sha"`
	Truncated bool   `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tree"`
}

// mergeCachedMeta 按 slug 把旧缓存里的图标、显示名、版本和脚本标记补到新条目上。
// items 按 slug 补上 previous 里已有而自己为空的图标、显示名和版本，脚本标记按或合并。
// previous 长度为 0 时直接返回，否则按 slug 建表。
func mergeCachedMeta(items, previous []Item) {
	if len(previous) == 0 {
		return
	}
	bySlug := make(map[string]Item, len(previous))
	for _, it := range previous {
		bySlug[it.Slug] = it
	}
	for i := range items {
		old, ok := bySlug[items[i].Slug]
		if !ok {
			continue
		}
		if items[i].Icon == "" {
			items[i].Icon = old.Icon
		}
		if items[i].DisplayName == "" {
			items[i].DisplayName = old.DisplayName
		}
		if items[i].Version == "" {
			items[i].Version = old.Version
		}
		items[i].HasScripts = items[i].HasScripts || old.HasScripts
	}
}

// enrichFromTree 拉仓库树，补上目录里有而表格没有的技能，并标记 scripts/ 和常见图标文件。
// items 在树为空时原样返回；否则先交给 appendTreeSkills，再按树路径标脚本和图标。
func enrichFromTree(items []Item) []Item {
	var tree gitTree
	if u := derivedTreeURL(Base); u != "" {
		raw, err := Get(u)
		if err == nil {
			_ = json.Unmarshal(raw, &tree)
		}
	}
	if len(tree.Tree) == 0 && strings.Contains(Base, "infometa/workbuddyskills") {
		if t, err := loadGitTree("infometa/workbuddyskills", "main"); err == nil {
			tree = t
		}
	}
	if len(tree.Tree) == 0 {
		return items
	}
	items = appendTreeSkills(items, tree)
	idx := map[string]int{}
	for i := range items {
		idx[items[i].Slug] = i
	}
	for _, node := range tree.Tree {
		if node.Type != "blob" {
			continue
		}
		slug, rest, ok := skillRel(node.Path)
		if !ok {
			continue
		}
		i, known := idx[slug]
		if !known {
			continue
		}
		lower := strings.ToLower(rest)
		if strings.HasPrefix(lower, "scripts/") {
			items[i].HasScripts = true
		}
		if items[i].Icon != "" {
			continue
		}
		base := strings.ToLower(path.Base(rest))
		if base == "icon.png" || base == "icon.svg" || base == "icon.webp" || base == "icon.jpg" || base == "avatar.png" {
			items[i].Icon = packFileURL(slug, rest)
		}
	}
	return items
}

// appendTreeSkills 把树里 skills/<slug>/SKILL.md 对应、但条目里还没有的 slug 追加进去。
// items 里已有的 slug 不再追加。
// tree 里类型为 blob、余下路径等于 SKILL.md 且 slug 还没有的，追加一条条目。
func appendTreeSkills(items []Item, tree gitTree) []Item {
	idx := map[string]bool{}
	for _, it := range items {
		idx[it.Slug] = true
	}
	for _, node := range tree.Tree {
		if node.Type != "blob" {
			continue
		}
		slug, rest, ok := skillRel(node.Path)
		if !ok || !strings.EqualFold(rest, "SKILL.md") {
			continue
		}
		if idx[slug] {
			continue
		}
		idx[slug] = true
		items = append(items, Item{Slug: slug, Name: slug, Category: "Skills"})
	}
	return items
}

// skillRel 要求路径以 skills/ 开头，并拆出 slug 和余下部分。slug 净化失败返回 false。
// p 把反斜杠换成斜杠并去掉两端斜杠，不以 skills/ 开头或净化后 slug 为空则返回 false。
func skillRel(p string) (slug, rest string, ok bool) {
	p = strings.Trim(strings.ReplaceAll(p, "\\", "/"), "/")
	if !strings.HasPrefix(p, "skills/") {
		return "", "", false
	}
	p = strings.TrimPrefix(p, "skills/")
	slug, rest, ok = strings.Cut(p, "/")
	if !ok {
		return "", "", false
	}
	slug = SanitizeSlug(slug)
	if slug == "" {
		return "", "", false
	}
	return slug, rest, true
}

// enrichFrontmatter 优先拉精选条目的 SKILL.md，最多 24 个，并发数是 8。已有图标和显示名的跳过。
// items 先排精选再排其余；图标和显示名都已有的跳过，最多处理 24 个。
func enrichFrontmatter(items []Item) {
	order := make([]int, 0, len(items))
	for i := range items {
		if items[i].Featured {
			order = append(order, i)
		}
	}
	for i := range items {
		if !items[i].Featured {
			order = append(order, i)
		}
	}
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	n := 0
	for _, i := range order {
		if items[i].Icon != "" && items[i].DisplayName != "" {
			continue
		}
		if n >= 24 {
			break
		}
		n++
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			raw, err := Get(SkillURL(items[i].Slug))
			if err != nil {
				return
			}
			peek := PeekMeta(raw)
			if items[i].Icon == "" {
				items[i].Icon = artifact.SafeIconURL(peek.Icon)
			}
			if items[i].DisplayName == "" {
				items[i].DisplayName = peek.DisplayName
			}
			if items[i].Version == "" {
				items[i].Version = peek.Version
			}
			if MentionsScripts(string(raw)) {
				items[i].HasScripts = true
			}
		}(i)
	}
	wg.Wait()
}
