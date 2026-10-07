package runtime

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const createdSnapSuffix = ".yoyo-created"

func (t *WorkspaceTools) snapshotBeforeWrite(rel, abs string) {
	if t == nil || t.Spill == nil || t.Spill.Dir == "" || abs == "" {
		return
	}
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "/"))
	if rel == "" {
		rel = filepath.Base(abs)
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	prev, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			writeSnapFile(t.Spill.Dir, t.Spill.Mirror, stamp, rel+createdSnapSuffix, nil)
		}
		return
	}
	writeSnapFile(t.Spill.Dir, t.Spill.Mirror, stamp, rel, prev)
}

func writeSnapFile(spillDir, mirror, stamp, rel string, body []byte) {
	dest := filepath.Join(spillDir, "files", stamp, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, dest)
	if mirror == "" {
		return
	}
	m := filepath.Join(mirror, "files", stamp, filepath.FromSlash(rel))
	_ = os.MkdirAll(filepath.Dir(m), 0o755)
	_ = os.WriteFile(m, body, 0o644)
}

func RestoreFileSnapshots(spillDir, workspace string) (int, error) {
	return RestoreFileSnapshotsAfter(spillDir, workspace, time.Time{})
}

func RestoreFileSnapshotsAfter(spillDir, workspace string, after time.Time) (int, error) {
	root := filepath.Join(spillDir, "files")
	stamps, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if len(stamps) == 0 {
		return 0, nil
	}
	sortStampDirs(stamps)
	seen := map[string]bool{}
	for _, st := range stamps {
		if !st.IsDir() {
			continue
		}
		if !after.IsZero() {
			stamp, ok := ParseStampTime(st.Name())
			if !ok || stamp.Before(after) {
				continue
			}
		}
		src := filepath.Join(root, st.Name())
		err = filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil || info.IsDir() {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			key := filepath.ToSlash(rel)
			created := strings.HasSuffix(key, createdSnapSuffix)
			if created {
				key = strings.TrimSuffix(key, createdSnapSuffix)
				rel = strings.TrimSuffix(rel, createdSnapSuffix)
			}
			if seen[key] {
				return nil
			}
			seen[key] = true
			dest := filepath.Join(workspace, rel)
			if created {
				if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
					return err
				}
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			return copyFileSnap(path, dest, info.Mode())
		})
		if err != nil {
			return len(seen), err
		}
	}
	return len(seen), nil
}

func ParseStampTime(name string) (time.Time, bool) {
	name = strings.TrimSpace(name)
	for _, layout := range []string{"20060102T150405.000000000", "20060102T150405", time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, name); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func sortStampDirs(stamps []os.DirEntry) {
	for i := 0; i < len(stamps); i++ {
		for j := i + 1; j < len(stamps); j++ {
			if stamps[j].Name() < stamps[i].Name() {
				stamps[i], stamps[j] = stamps[j], stamps[i]
			}
		}
	}
}

func copyFileSnap(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func ParseRewindScope(focus string) (scope, from string) {
	s := strings.ToLower(strings.TrimSpace(focus))
	scope = "conversation"
	if strings.HasPrefix(s, "files") {
		scope = "files"
		s = strings.TrimSpace(strings.TrimPrefix(s, "files"))
	} else if strings.HasPrefix(s, "both") {
		scope = "both"
		s = strings.TrimSpace(strings.TrimPrefix(s, "both"))
	} else if strings.HasPrefix(s, "conversation") || strings.HasPrefix(s, "chat") {
		scope = "conversation"
		s = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(s, "conversation"), "chat"))
	}
	if strings.HasPrefix(s, "from ") {
		from = strings.TrimSpace(s[5:])
	} else if s != "" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(focus)), "from ") {
		from = strings.TrimSpace(s)
	}
	return scope, from
}
