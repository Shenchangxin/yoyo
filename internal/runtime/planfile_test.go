package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
