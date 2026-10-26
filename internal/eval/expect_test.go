// 锁住 expect.toml：子串对上则通过，文件缺失则失败。比较忽略大小写由 runExpect 实现，这里用原样小写锁住通过路径。

package eval

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunExpectFileContains 要求工作区文件含指定子串时 runExpect 返回真且没有错误。
func TestRunExpectFileContains(t *testing.T) {
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "alpha.txt"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect := filepath.Join(t.TempDir(), "expect.toml")
	if err := os.WriteFile(expect, []byte("[[file]]\npath = \"alpha.txt\"\ncontains = \"alpha\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, _, err := runExpect(work, expect)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

// TestRunExpectMissingFile 要求 expect 指向不存在的文件时返回假且错误非空。
func TestRunExpectMissingFile(t *testing.T) {
	expect := filepath.Join(t.TempDir(), "expect.toml")
	if err := os.WriteFile(expect, []byte("[[file]]\npath = \"missing.txt\"\ncontains = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, _, err := runExpect(t.TempDir(), expect)
	if ok || err == nil {
		t.Fatal("expected failure")
	}
}
