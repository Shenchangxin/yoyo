// 写文件前把旧内容按时间戳存进 Spill 的 files 目录，供回退恢复。同一相对路径先遇到的快照生效，所以恢复的是最早那一份。不负责 git。

package runtime

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// createdSnapSuffix 是 .yoyo-created。写入前文件不存在时用这个后缀记下「当时还没有这个文件」，恢复时删除它。
const createdSnapSuffix = ".yoyo-created"

// snapshotBeforeWrite 在 Spill 可用时保存写入前的字节。文件不存在则写入空正文加 createdSnapSuffix。读失败且不是不存在则不记。
// rel 去掉前导斜杠后转成斜杠，为空时改用 abs 的文件名。
// abs 为空时返回，否则读取写入前的字节。
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

// writeSnapFile 把正文写到 spillDir/files/stamp 下，先写 .tmp 再改名。mirror 非空时再写一份到镜像目录，镜像写入失败被忽略。
// rel 转成系统路径后接在时间戳目录下。
// body 写入临时文件，再随改名落到目标。
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

// RestoreFileSnapshots 调用 RestoreFileSnapshotsAfter，after 为零值，因此不按时间跳过。
// spillDir 传给 RestoreFileSnapshotsAfter。
// workspace 传给 RestoreFileSnapshotsAfter。
func RestoreFileSnapshots(spillDir, workspace string) (int, error) {
	return RestoreFileSnapshotsAfter(spillDir, workspace, time.Time{})
}

// RestoreFileSnapshotsAfter 按时间戳目录名从早到晚走。after 非零时，解析失败或早于 after 的目录跳过。同一相对路径只处理第一次遇到的快照。带 createdSnapSuffix 的项会删除工作区里的对应文件，其余项覆盖写回。
// spillDir 与 files 拼成时间戳目录的根。
// workspace 与相对路径拼成恢复目标。
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

// ParseStampTime 按带纳秒的紧凑时间、不带小数的紧凑时间、RFC3339Nano、RFC3339 依次解析，成功则返回 UTC。
// name 去掉空白后按这四种布局解析。
func ParseStampTime(name string) (time.Time, bool) {
	name = strings.TrimSpace(name)
	for _, layout := range []string{"20060102T150405.000000000", "20060102T150405", time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, name); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// sortStampDirs 按目录名升序排序，较早的时间戳在前。
// stamps 按 Name 升序就地排序。
func sortStampDirs(stamps []os.DirEntry) {
	for i := 0; i < len(stamps); i++ {
		for j := i + 1; j < len(stamps); j++ {
			if stamps[j].Name() < stamps[i].Name() {
				stamps[i], stamps[j] = stamps[j], stamps[i]
			}
		}
	}
}

// copyFileSnap 按给定模式覆盖写入目标文件。
// src 打开来读。
// dest 按 mode 创建并覆盖写入。
// mode 传给 OpenFile。
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

// ParseRewindScope 把 focus 收成小写。以 files、both、conversation 或 chat 开头时设定 scope，其余默认 conversation。from 之后的文字是起点；没有这些前缀时，剩余文字也当作 from。
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
