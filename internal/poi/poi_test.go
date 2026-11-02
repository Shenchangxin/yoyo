package poi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRefreshKeepsNamesAndLinksWithoutFileBodies(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(secret, []byte("token sk-secret-should-not-surface"), 0o600); err != nil {
		t.Fatal(err)
	}
	desk := ScanRoots([]string{dir}, time.Now())
	if len(desk) != 1 || desk[0].Name != "notes" {
		t.Fatalf("scan = %#v", desk)
	}
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)
	first := store.Refresh([]Observation{
		{Name: "Agent 工程", Kind: "profile", Evidence: "memory"},
		{Name: "yoyo", Kind: "project", Evidence: "memory"},
		desk[0],
	}, []string{"在做 Agent 工程，仓库是 yoyo"}, now)
	if !first.Pending || len(first.Added) != 3 {
		t.Fatalf("first = %#v", first)
	}
	var linked bool
	for _, pt := range first.Points {
		if pt.Name == "Agent 工程" && len(pt.Links) > 0 {
			linked = true
		}
		if pt.Evidence != "memory" && pt.Evidence != "desk" {
			t.Fatalf("evidence leaked: %#v", pt)
		}
	}
	if !linked {
		t.Fatal("expected a link between names in the same memory")
	}
	raw, err := os.ReadFile(filepath.Join(store.path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-secret") {
		t.Fatal("file body entered the poi store")
	}
	if !store.Ack(first.Sig) {
		t.Fatal("ack")
	}
	second := store.Refresh([]Observation{{Name: "Agent 工程", Kind: "profile", Evidence: "memory"}}, nil, now)
	if second.Pending {
		t.Fatalf("same day unseen should not drop yet: %#v", second)
	}
	nextDay := store.Refresh([]Observation{{Name: "Agent 工程", Kind: "profile", Evidence: "memory"}}, nil, now.Add(24*time.Hour))
	if !nextDay.Pending {
		t.Fatal("expected dropped names after a day")
	}
}
