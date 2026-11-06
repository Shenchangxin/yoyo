// Copyright 2026 Hypit.AI
// Translated from @hypit/atomic-file/test/replace-file.test.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileReplacementPublishesTheDestinationAndPreservesAnOpenReader(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"result.json", ".writer.json", "素材 🎬.json"} {
		destination := filepath.Join(directory, name)
		temporary := filepath.Join(directory, name+".part")
		if err := os.WriteFile(temporary, []byte("first"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ReplaceFile(temporary, destination); err != nil {
			t.Fatal(err)
		}
		if got := dirNames(t, directory); len(got) != 1 || got[0] != name {
			t.Fatalf("directory %v", got)
		}
		reader, err := openReader(destination)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(temporary, []byte("second"), 0o644); err != nil {
			reader.Close()
			t.Fatal(err)
		}
		if err := ReplaceFile(temporary, destination); err != nil {
			reader.Close()
			t.Fatal(err)
		}
		current, err := os.ReadFile(destination)
		if err != nil {
			reader.Close()
			t.Fatal(err)
		}
		if string(current) != "second" {
			reader.Close()
			t.Fatalf("destination %q", current)
		}
		opened, err := readAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(opened) != "first" {
			t.Fatalf("open reader saw %q", opened)
		}
		if got := dirNames(t, directory); len(got) != 1 || got[0] != name {
			t.Fatalf("directory %v", got)
		}
		if err := os.Remove(destination); err != nil {
			t.Fatal(err)
		}
	}
}

func dirNames(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

func readAll(file *os.File) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, info.Size())
	_, err = file.ReadAt(buf, 0)
	return buf, err
}
