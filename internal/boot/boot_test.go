// boot_test.go 检查评测目录和磁盘前端的查找规则。

package boot

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEvalsDirPrefersEnv 确认 YOYO_EVALS 原样优先。
func TestEvalsDirPrefersEnv(t *testing.T) {
	t.Setenv("YOYO_EVALS", "/custom/evals")
	if got := EvalsDir(); got != "/custom/evals" {
		t.Fatalf("got %q", got)
	}
}

// TestEvalsDirFallsBackToWorkdir 确认可执行文件旁没有 evals 时用工作目录下的，并返回绝对路径。
func TestEvalsDirFallsBackToWorkdir(t *testing.T) {
	t.Setenv("YOYO_EVALS", "")
	wd := t.TempDir()
	if err := os.Mkdir(filepath.Join(wd, "evals"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(wd)
	got := EvalsDir()
	if !filepath.IsAbs(got) || filepath.Base(got) != "evals" {
		t.Fatalf("got %q", got)
	}
}

// TestEvalsDirDefault 确认哪里都没有时返回相对路径 evals。
func TestEvalsDirDefault(t *testing.T) {
	t.Setenv("YOYO_EVALS", "")
	t.Chdir(t.TempDir())
	if got := EvalsDir(); got != "evals" {
		t.Fatalf("got %q", got)
	}
}

// TestFrontendDirNeedsIndexFile 确认只有占位文件的 dist 不算前端，放进 index.html 后才算。
func TestFrontendDirNeedsIndexFile(t *testing.T) {
	wd := t.TempDir()
	dist := filepath.Join(wd, "frontend", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, ".keep"), []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(wd)
	if got := FrontendDir(); got != "" {
		t.Fatalf("stub dist accepted: %q", got)
	}
	if DiskFrontend() != nil {
		t.Fatal("stub dist served")
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := FrontendDir(); got != "frontend/dist" {
		t.Fatalf("got %q", got)
	}
	if DiskFrontend() == nil {
		t.Fatal("real dist not served")
	}
}
