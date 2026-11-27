// pages_test.go 检查 learn 依赖的共用规则：旧页面解码、路径清洗、截断和审阅哈希。

package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadFileDecodesCRLFPage 确认 Windows 换行的旧页面能读回标题、正文和默认值。
func TestReadFileDecodesCRLFPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.md")
	raw := "---\r\nid: p1\r\ntitle: Hello\r\n---\r\n\r\nbody line\r\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "p1" || p.Title != "Hello" || p.Content != "body line" {
		t.Fatalf("%+v", p)
	}
	if p.SpaceID != DefaultSpace || p.Revision != 1 {
		t.Fatalf("defaults not applied: space=%q rev=%d", p.SpaceID, p.Revision)
	}
}

// TestDecodeRejectsMissingFrontmatter 确认没有 YAML 头或头没闭合时报错。
func TestDecodeRejectsMissingFrontmatter(t *testing.T) {
	for _, raw := range []string{"just text", "---\nid: x\n"} {
		if _, err := decodePage([]byte(raw)); err == nil {
			t.Fatalf("%q: want error", raw)
		}
	}
}

// TestNormalizeStripsTraversal 确认空间和页面 id 都不能带出目录。
func TestNormalizeStripsTraversal(t *testing.T) {
	if got := NormalizePageID("../../etc/passwd"); strings.Contains(got, "..") || strings.ContainsAny(got, `/\`) {
		t.Fatalf("page id %q", got)
	}
	if got := NormalizePageID("....."); strings.Contains(got, "..") {
		t.Fatalf("page id %q", got)
	}
	if got := NormalizeSpace("../"); got != DefaultSpace {
		t.Fatalf("space %q", got)
	}
	if got := NormalizeSpace(" work_1 "); got != "work_1" {
		t.Fatalf("space %q", got)
	}
}

// TestClipAndPreview 确认空标题、超长标题、超长正文和预览折叠。
func TestClipAndPreview(t *testing.T) {
	if ClipTitle("   ") != "Untitled" {
		t.Fatal("empty title")
	}
	if n := len([]rune(ClipTitle(strings.Repeat("标", MaxTitle+10)))); n != MaxTitle {
		t.Fatalf("title runes %d", n)
	}
	if n := len([]rune(ClipBody(strings.Repeat("x", MaxBody+1)))); n != MaxBody {
		t.Fatalf("body runes %d", n)
	}
	if got := Preview("a\n\n b\tc"); got != "a b c" {
		t.Fatalf("preview %q", got)
	}
}

// TestHashDraftAndReviewID 确认哈希稳定、随内容变化，审阅 id 取哈希前 12 位。
func TestHashDraftAndReviewID(t *testing.T) {
	h1 := HashDraft("t", "c", "p", "home", 1)
	if h1 != HashDraft("t", "c", "p", "home", 1) {
		t.Fatal("hash not stable")
	}
	if h1 == HashDraft("t", "c2", "p", "home", 1) || h1 == HashDraft("t", "c", "p", "home", 2) {
		t.Fatal("hash ignores content or revision")
	}
	if got := ReviewID(h1); got != "rev-"+h1[:12] {
		t.Fatalf("review id %q", got)
	}
}
