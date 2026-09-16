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

// FuzzySearch lists workspace files and directories matching q (substring,
// case-insensitive). Results are bounded; gitignored and skip dirs are omitted.
func FuzzySearch(root, q string, limit int) []FileHit {
	if limit <= 0 {
		limit = 40
	}
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	q = strings.ToLower(strings.TrimSpace(q))
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
	sort.Slice(hits, func(i, j int) bool {
		si := scoreHit(hits[i], q)
		sj := scoreHit(hits[j], q)
		if si != sj {
			return si > sj
		}
		return hits[i].Path < hits[j].Path
	})
	if len(hits) > limit {
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
