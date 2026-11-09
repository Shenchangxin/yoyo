// disk.go 拥有 Buddy 的磁盘占用统计和缓存清理：每项托管安装、托管 Node 和 uv、下载缓存和探测目录各占多少。
// 偏好页打开时统计一次；清理只删能重新下载的东西（npm 和 uv 的缓存、探测目录、残留的下载），不动已装的 agent。

package buddy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// DiskUsage 是 Buddy 占的磁盘（字节）。Items 按 agent 编号列托管安装，Runtime 是托管 Node 和 uv，Cache 是能清掉的部分。
type DiskUsage struct {
	Total   int64            `json:"total"`
	Runtime int64            `json:"runtime"`
	Cache   int64            `json:"cache"`
	Items   map[string]int64 `json:"items"`
}

// cacheDirs 是能清掉的目录：npm 和 uv 的下载缓存、探测目录、各项残留的下载。
func (m *Manager) cacheDirs() []string {
	root := m.opts.Root
	dirs := []string{filepath.Join(root, ".npm-cache"), filepath.Join(root, ".uv-cache"), filepath.Join(root, ".probe")}
	for _, id := range m.itemDirs() {
		dirs = append(dirs, filepath.Join(root, id, ".download"))
	}
	return dirs
}

// itemDirs 列出 buddies 下每项 agent 的目录名，跳过点开头的缓存和图标目录。
func (m *Manager) itemDirs() []string {
	entries, err := os.ReadDir(m.opts.Root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && e.Name() != "icons" {
			out = append(out, e.Name())
		}
	}
	return out
}

// Disk 统计磁盘占用。要把目录走一遍，只在偏好页打开时调用。
func (m *Manager) Disk() DiskUsage {
	u := DiskUsage{Items: map[string]int64{}}
	if m.opts.Root == "" {
		return u
	}
	for _, d := range m.cacheDirs() {
		u.Cache += dirSize(d)
	}
	if m.opts.RuntimeDir != "" {
		u.Runtime = dirSize(filepath.Join(m.opts.RuntimeDir, "node")) + dirSize(filepath.Join(m.opts.RuntimeDir, "uv"))
	}
	for _, id := range m.itemDirs() {
		if n := dirSize(filepath.Join(m.opts.Root, id)) - dirSize(filepath.Join(m.opts.Root, id, ".download")); n > 0 {
			u.Items[id] = n
		}
	}
	u.Total = u.Runtime + u.Cache
	for _, n := range u.Items {
		u.Total += n
	}
	return u
}

// ClearCache 删掉缓存目录，返回释放的字节数。有安装或探测在跑时拒绝，免得删掉正在用的文件；删不掉的目录跳过。
func (m *Manager) ClearCache(ctx context.Context) (int64, error) {
	if m.opts.Root == "" {
		return 0, nil
	}
	m.mu.Lock()
	busy := len(m.jobs) > 0 || len(m.probing) > 0
	m.mu.Unlock()
	if busy {
		return 0, errors.New(m.say("有安装或检测在进行，稍后再清理", "an install or check is running; try again later"))
	}
	var freed int64
	for _, d := range m.cacheDirs() {
		n := dirSize(d)
		if n == 0 {
			continue
		}
		if removeAllRetry(ctx, d) == nil {
			freed += n
		}
	}
	return freed, nil
}
