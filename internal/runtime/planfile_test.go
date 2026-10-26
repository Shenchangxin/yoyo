// 锁住计划文件往返：正文写入 .yoyo/plans，再读回来仍含原文，父目录名是 plans。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlanFileRoundTrip 确认写入的句子能读回，并且文件位于 plans 目录。
func TestPlanFileRoundTrip(t *testing.T) {
	ws := t.TempDir()
	WritePlanFile(ws, "sess-1", "1. [pending] draft a briefing")
	got := LoadPlanFile(ws, "sess-1")
	if !strings.Contains(got, "draft a briefing") {
		t.Fatalf("%q", got)
	}
	p := PlanFilePath(ws, "sess-1")
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(p)) != "plans" {
		t.Fatalf("%s", p)
	}
}
