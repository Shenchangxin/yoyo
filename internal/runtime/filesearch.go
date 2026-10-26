// 工作区文件和目录的子串检索，给补全和 @ 提及用。忽略规则走 loadIgnore。路径关进工作区用 mentions.go 的 jailPath。

package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileHit 是一条相对路径。Kind 为 file 或 dir。
type FileHit struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// dirListingCap 是钻进一个目录时列出的子项上限，值为 200。整树补全仍用调用方的 limit（常见是 40）；带斜杠且前缀是真实目录时，列表上限至少抬到这个值。
const dirListingCap = 200

// FuzzySearch 按子串、不区分大小写列出工作区里的文件和目录。limit 小于等于 0 时用 40。查询含斜杠且最后一个斜杠前的前缀是真实目录时，改为列出该目录的子项，不再走有上限的整树遍历。
func FuzzySearch(root, q string, limit int) []FileHit {
	if limit <= 0 {
		limit = 40
	}
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	q = strings.TrimSpace(q)
	q = filepath.ToSlash(q)
	if dir, rest, ok := listingPrefix(root, q); ok {
		cap := limit
		if cap < dirListingCap {
			cap = dirListingCap
		}
		return listDirChildren(root, dir, rest, cap)
	}
	q = strings.ToLower(q)
	ign := loadIgnore(root)
	var hits []FileHit
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if info.IsDir() {
			if ign.skipDir(info.Name()) {
				return filepath.SkipDir
			}
			if q == "" || strings.Contains(strings.ToLower(relSlash), q) || strings.Contains(strings.ToLower(info.Name()), q) {
				hits = append(hits, FileHit{Path: relSlash, Kind: "dir"})
			}
			return nil
		}
		if ign.skipFile(relSlash, info.Name()) {
			return nil
		}
		if q == "" || strings.Contains(strings.ToLower(relSlash), q) || strings.Contains(strings.ToLower(info.Name()), q) {
			hits = append(hits, FileHit{Path: relSlash, Kind: "file"})
		}
		return nil
	})
	return rankHits(hits, q, limit)
}

// listingPrefix 在查询含斜杠、前缀非空、且 jailPath 之后 Stat 到目录时，拆出目录和剩余过滤串。
func listingPrefix(root, q string) (dir, rest string, ok bool) {
	if q == "" || !strings.Contains(q, "/") {
		return "", "", false
	}
	slash := strings.LastIndex(q, "/")
	dir = strings.Trim(q[:slash], "/")
	rest = q[slash+1:]
	if dir == "" {
		return "", "", false
	}
	p, err := jailPath(root, dir)
	if err != nil {
		return "", "", false
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		return "", "", false
	}
	return dir, rest, true
}

// listDirChildren 只读该目录的直接子项，跳过忽略项，并用剩余串过滤文件名，再按 rankHits 截断。
func listDirChildren(root, dir, rest string, limit int) []FileHit {
	p, err := jailPath(root, dir)
	if err != nil {
		return nil
	}
	ents, err := os.ReadDir(p)
	if err != nil {
		return nil
	}
	ign := loadIgnore(root)
	fold := strings.ToLower(strings.TrimSpace(rest))
	hits := make([]FileHit, 0, len(ents))
	for _, e := range ents {
		name := e.Name()
		rel := filepath.ToSlash(filepath.Join(dir, name))
		if e.IsDir() {
			if ign.skipDir(name) {
				continue
			}
		} else if ign.skipFile(rel, name) {
			continue
		}
		if fold != "" && !strings.Contains(strings.ToLower(name), fold) {
			continue
		}
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		hits = append(hits, FileHit{Path: rel, Kind: kind})
	}
	return rankHits(hits, fold, limit)
}

// ListTree 用空查询调用 FuzzySearch。limit 小于等于 0 时用 4000。
func ListTree(root string, limit int) []FileHit {
	if limit <= 0 {
		limit = 4000
	}
	return FuzzySearch(root, "", limit)
}

// rankHits 按 scoreHit 从高到低排序，同分按路径。limit 大于 0 时截断。
func rankHits(hits []FileHit, q string, limit int) []FileHit {
	sort.Slice(hits, func(i, j int) bool {
		si := scoreHit(hits[i], q)
		sj := scoreHit(hits[j], q)
		if si != sj {
			return si > sj
		}
		return hits[i].Path < hits[j].Path
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// scoreHit 在查询为空时返回 0。基名完全相等为 3，前缀为 2，包含为 1，否则为 0。
func scoreHit(h FileHit, q string) int {
	base := strings.ToLower(filepath.Base(h.Path))
	if q == "" {
		return 0
	}
	if base == q {
		return 3
	}
	if strings.HasPrefix(base, q) {
		return 2
	}
	if strings.Contains(base, q) {
		return 1
	}
	return 0
}
