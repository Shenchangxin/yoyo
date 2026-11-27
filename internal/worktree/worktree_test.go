// 锁住非 git 目录下隔离副本走整树复制、持久隔离可重复调用，以及复制时跳过 skipDirs 里的目录。

package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsolateCopy 锁住 Isolate 的目标里能读到源文件，清理函数会删掉临时目录。
func TestIsolateCopy(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst, cleanup, err := Isolate(src)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "keep.txt"))
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	if string(b) != "ok" {
		cleanup()
		t.Fatalf("%q", b)
	}
	cleanup()
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("isolated dir still present after cleanup: %v", err)
	}
}

// TestPersistCopy 锁住 Persist 能复制文件，目标已存在时再次调用成功，Remove 会删掉目标。
func TestPersistCopy(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "wt")
	if err := Persist(src, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dest, "keep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ok" {
		t.Fatalf("%q", b)
	}
	if err := Persist(src, dest); err != nil {
		t.Fatal(err)
	}
	Remove(src, dest)
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("worktree still present: %v", err)
	}
}

// TestCopyTreeSkipsHeavyDirs 锁住复制时跳过 node_modules 这类目录，但源根本身即使同名也照常复制。
func TestCopyTreeSkipsHeavyDirs(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "node_modules", "pkg", "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "main.go")); err != nil {
		t.Fatalf("main.go not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "node_modules")); !os.IsNotExist(err) {
		t.Fatalf("node_modules should be skipped: %v", err)
	}
}

// TestSkipDirNamesReturnsCopy 锁住调用方改 SkipDirNames 的返回值不会影响包内的表。
func TestSkipDirNamesReturnsCopy(t *testing.T) {
	m := SkipDirNames()
	m["extra"] = true
	delete(m, ".git")
	if skipDirs["extra"] || !skipDirs[".git"] {
		t.Fatal("SkipDirNames must return an independent copy")
	}
}
