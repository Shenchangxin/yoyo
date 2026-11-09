// disk_test.go 锁住磁盘统计分得清安装、运行时和缓存，清理只删缓存、不动已装的 agent，有安装在跑时拒绝清理。

package buddy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeSized 写一个 n 字节的文件，目录不存在时建好。
func writeSized(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDiskAndClearCache 锁住统计的各项字节数，以及清理后缓存没了、安装还在、释放数等于缓存数。
func TestDiskAndClearCache(t *testing.T) {
	f := newFixture(t, nil)
	root, rt := f.m.opts.Root, f.m.opts.RuntimeDir
	writeSized(t, filepath.Join(root, ".npm-cache", "a"), 100)
	writeSized(t, filepath.Join(root, ".probe", "gemini", "b"), 10)
	writeSized(t, filepath.Join(root, "gemini", ".download", "c.zip"), 50)
	writeSized(t, filepath.Join(root, "gemini", "pkg-1", "d.js"), 1000)
	writeSized(t, filepath.Join(root, "icons", "gemini.svg"), 7)
	writeSized(t, filepath.Join(rt, "node", "v1", "node"), 400)
	writeSized(t, filepath.Join(rt, "pwsh", "pwsh.exe"), 9999)
	u := f.m.Disk()
	if u.Cache != 160 || u.Runtime != 400 || u.Items["gemini"] != 1000 || len(u.Items) != 1 || u.Total != 1560 {
		t.Fatalf("usage %+v", u)
	}
	f.m.mu.Lock()
	f.m.jobs["opencode"] = &job{cancel: func() {}}
	f.m.mu.Unlock()
	if _, err := f.m.ClearCache(context.Background()); err == nil {
		t.Fatal("cleared while installing")
	}
	f.m.mu.Lock()
	delete(f.m.jobs, "opencode")
	f.m.mu.Unlock()
	freed, err := f.m.ClearCache(context.Background())
	if err != nil || freed != 160 {
		t.Fatalf("freed %d %v", freed, err)
	}
	for _, gone := range []string{".npm-cache", ".probe", filepath.Join("gemini", ".download")} {
		if _, err := os.Stat(filepath.Join(root, gone)); !os.IsNotExist(err) {
			t.Fatalf("%s left: %v", gone, err)
		}
	}
	for _, kept := range []string{filepath.Join(root, "gemini", "pkg-1", "d.js"), filepath.Join(root, "icons", "gemini.svg"), filepath.Join(rt, "node", "v1", "node")} {
		if _, err := os.Stat(kept); err != nil {
			t.Fatalf("%s removed: %v", kept, err)
		}
	}
}
