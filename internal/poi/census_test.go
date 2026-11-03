package poi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScanCensusReadsProjectNamesNotBodies(t *testing.T) {
	dir := t.TempDir()
	desk := filepath.Join(dir, "Desktop")
	down := filepath.Join(dir, "Downloads")
	yoyo := filepath.Join(desk, "app")
	allo := filepath.Join(desk, "allo")
	if err := os.MkdirAll(yoyo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(allo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(down, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.Local)
	write := func(path, body string, at time.Time) {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(yoyo, "go.mod"), "module github.com/acme/yoyo\n\n// secret sk-do-not-read\n", now.Add(-2*time.Hour))
	write(filepath.Join(yoyo, "main.go"), "package main\nfunc main() {}\n", now.Add(-time.Hour))
	write(filepath.Join(allo, "package.json"), `{"name":"allo","scripts":{"token":"sk-secret"}}`, now.Add(-48*time.Hour))
	write(filepath.Join(allo, "blank.txt"), "", now.Add(-48*time.Hour))
	write(filepath.Join(down, "setup.exe"), "installer-bytes", now.Add(-3*time.Hour))
	write(filepath.Join(desk, "yoyo-notes.md"), "not a body we should return", now.Add(-5*time.Hour))
	write(filepath.Join(desk, "token.txt"), "sk-secret", now.Add(-time.Hour))
	if err := os.Chtimes(yoyo, now.Add(-3*time.Hour), now.Add(-3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(allo, now.Add(-72*time.Hour), now.Add(-72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i, hour := range []int{8, 9, 11, 13, 15, 17} {
		write(filepath.Join(desk, "hour-"+string(rune('a'+i))+".txt"), "x", time.Date(2026, 11, 2, hour, 15, 0, 0, time.Local))
	}

	got := ScanCensus([]string{desk, down}, now)
	if got.Projects != 2 {
		t.Fatalf("projects = %d %#v", got.Projects, got.Lines)
	}
	if len(got.Lines) == 0 || got.Lines[0].Name != "yoyo" || got.Lines[0].State != "moving" {
		t.Fatalf("lines = %#v", got.Lines)
	}
	var stalled bool
	for _, line := range got.Lines {
		if line.Name == "allo" && line.State == "stalled" && line.Evidence == "blank.txt" {
			stalled = true
		}
		if strings.Contains(line.Evidence, "sk-") || strings.Contains(line.Name, "secret") {
			t.Fatalf("secret surfaced: %#v", line)
		}
	}
	if !stalled {
		t.Fatalf("allo not stalled: %#v", got.Lines)
	}
	if len(got.Archive) != 1 || got.Archive[0].Name != "setup.exe" {
		t.Fatalf("archive = %#v", got.Archive)
	}
	if len(got.Place) != 1 || got.Place[0].Line != "yoyo" {
		t.Fatalf("place = %#v", got.Place)
	}
	if got.Hours == "" {
		t.Fatal("expected an hour span from weekday edits")
	}
	blob := got.Hours + got.Lines[0].Evidence
	if strings.Contains(blob, "sk-") || strings.Contains(blob, "do-not-read") {
		t.Fatalf("body leaked into census: %#v", got)
	}
}
