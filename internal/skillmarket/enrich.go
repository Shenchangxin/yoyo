package skillmarket

import (
	"encoding/json"
	"path"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type gitTree struct {
	Truncated bool `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

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

func enrichFromTree(items []Item) {
	u := derivedTreeURL(Base)
	if u == "" {
		return
	}
	raw, err := Get(u)
	if err != nil {
		return
	}
	var tree gitTree
	if json.Unmarshal(raw, &tree) != nil || len(tree.Tree) == 0 {
		return
	}
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
}

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
