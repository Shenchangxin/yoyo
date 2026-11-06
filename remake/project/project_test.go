// Copyright 2026 Hypit.AI
// Translated from @hypit/project (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindProjectRootWalksPastAnUnmarkedManifest(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(root, "package.json"), `{"hypit":{"project":true}}`)
	writeJSON(t, filepath.Join(nested, "package.json"), `{"name":"not-a-project"}`)
	got, err := FindProjectRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestResolveProjectRootRequiresAMarkedProject(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "package.json"), `{"hypit":{"project":"true"}}`)
	_, err := ResolveProjectRoot(dir, "", false)
	if err == nil || err.Error() != "No Hypit project was found; run inside a package.json with hypit.project set to true, or pass --project <directory>" {
		t.Fatalf("%v", err)
	}
	writeJSON(t, filepath.Join(dir, "package.json"), `{"hypit":[]}`)
	if _, err := FindProjectRoot(dir); err != nil {
		t.Fatal(err)
	}
	explicit := filepath.Join(dir, "missing")
	if _, err := ResolveProjectRoot(dir, explicit, true); err == nil {
		t.Fatal("missing explicit project root was accepted")
	}
}

func TestResolveProjectRootAcceptsAnExplicitDirectory(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveProjectRoot(child, "..", true)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestInvalidManifestJSONIsReturned(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "package.json"), `{`)
	_, err := FindProjectRoot(dir)
	if err == nil || !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "unexpected") {
		t.Fatal(err)
	}
}

func TestFindThroughASymlinkReturnsTheRealDirectory(t *testing.T) {
	root := t.TempDir()
	writeJSON(t, filepath.Join(root, "package.json"), `{"hypit":{"project":true}}`)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skip(err)
	}
	got, err := FindProjectRoot(link)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func writeJSON(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
