package poi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScanAppRootsReadsNamesNotSecrets(t *testing.T) {
	root := t.TempDir()
	code := filepath.Join(root, "Cursor")
	storage := filepath.Join(code, "User", "globalStorage")
	if err := os.MkdirAll(storage, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"backupWorkspaces":{"folders":[{"folderUri":"file:///c%3A/work/yoyo"}]},"token":"sk-secret-should-not-surface"}`
	if err := os.WriteFile(filepath.Join(storage, "storage.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(code, "Cookies"), []byte("sk-secret-should-not-surface"), 0o600); err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(code, "User", "workspaceStorage", "abc")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "workspace.json"), []byte(`{"folder":"file:///c%3A/work/build"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	obsidian := filepath.Join(root, "obsidian")
	if err := os.MkdirAll(obsidian, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsidian, "obsidian.json"), []byte(`{"vaults":{"a":{"path":"C:/notes/日记"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	recent := filepath.Join(root, "Recent")
	if err := os.MkdirAll(recent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recent, "方案.docx.lnk"), []byte("do-not-read-body"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := ScanAppRoots([]AppRoot{
		{Name: "Cursor", Dir: code, Kind: "code"},
		{Name: "Obsidian", Dir: obsidian, Kind: "obsidian"},
		{Name: "最近的文件", Dir: recent, Kind: "recent"},
		{Name: "Chrome", Dir: code, Kind: "presence"},
	}, time.Now())
	joined := ""
	var kinds []string
	for _, item := range got {
		joined += item.Name + "\n"
		kinds = append(kinds, item.Kind+":"+item.Name)
		if item.Evidence != "app" && item.Evidence != "trace" {
			t.Fatalf("evidence %#v", item)
		}
	}
	for _, want := range []string{"Cursor", "yoyo", "build", "Obsidian", "日记", "方案.docx", "Chrome"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in %s", want, joined)
		}
	}
	if strings.Contains(joined, "sk-secret") || strings.Contains(joined, "do-not-read") {
		t.Fatal("secret or file body entered the scan")
	}
	if strings.Contains(strings.Join(kinds, " "), "trace:Cursor") {
		t.Fatal("app name should not also be a trace")
	}
}
