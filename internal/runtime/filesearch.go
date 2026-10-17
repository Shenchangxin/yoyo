package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileHit struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// dirListingCap is the overlay budget for one directory's children. Typeahead
// across the tree stays at the caller limit (typically 40); a drilled folder
// must list its own entries or Tab into src/ is useless.
const dirListingCap = 200

// FuzzySearch lists workspace files and directories matching q (substring,
// case-insensitive). A query with a slash whose prefix is a real directory
// lists that directory's children instead of a capped whole-tree walk.
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

func ListTree(root string, limit int) []FileHit {
	if limit <= 0 {
		limit = 4000
	}
	return FuzzySearch(root, "", limit)
}

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
