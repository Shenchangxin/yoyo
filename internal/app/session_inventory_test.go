// 上下文清单里能看到计划和 Today。

package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// TestContextInventoryIncludesPlanAndToday 确认清单里有计划文本和钉住的文件，并且计划文件写到了工作区。
func TestContextInventoryIncludesPlanAndToday(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	sess, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetSessionPlan(sess.ID, "1. [pending] brief the week"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ContextPin(sess.ID, "file", "docs/week.md", true); err != nil {
		t.Fatal(err)
	}
	inv := a.ContextInventory(sess.ID)
	if inv.Plan == "" {
		t.Fatal("missing plan")
	}
	foundFile := false
	for _, o := range inv.Objects {
		if o.Kind == "file" && o.Path == "docs/week.md" && o.Pinned {
			foundFile = true
		}
	}
	if !foundFile {
		t.Fatalf("objects %+v", inv.Objects)
	}
	if runtime.LoadPlanFile(ws, sess.ID) == "" {
		t.Fatal("plan file missing")
	}
	if _, err := os.Stat(runtime.PlanFilePath(ws, sess.ID)); err != nil {
		t.Fatal(err)
	}
}
