// Copyright 2026 Hypit.AI
// Translated from @hypit/workspace/test/node/workspace.test.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).

package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Shenchangxin/yoyo/remake/source"
)

func TestAnInRootSourceWhoseBasenameStartsWithDotDotIsAdmitted(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(root, "..keep.svml")
	if err := os.WriteFile(entryPath, []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.svs"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	session := openWorkspace(t, NodeFilesystemWorkspaceOptions{Root: &root, SourceAdapter: rawSourceAdapter}, entryPath)
	if source.DecodeSourceText(session.Entry().Unit) != "inside" {
		t.Fatal(source.DecodeSourceText(session.Entry().Unit))
	}
	err := recoverWorkspace(func() {
		_, callErr := session.ResolveSource(session.Entry().Unit, source.SourceImportRequest{From: "../outside.svs", Alias: "escaped"})
		if callErr != nil {
			panic(callErr)
		}
	})
	if err == nil || err.Code != "SOURCE_OUTSIDE_ROOT" {
		t.Fatalf("%v", err)
	}
	link := filepath.Join(root, "..linked")
	command := exec.Command("cmd", "/c", "mklink", "/J", link, parent)
	if output, callErr := command.CombinedOutput(); callErr != nil {
		t.Fatalf("%v %s", callErr, output)
	}
	err = recoverWorkspace(func() {
		_, callErr := session.ResolveSource(session.Entry().Unit, source.SourceImportRequest{From: "./..linked/outside.svs", Alias: "linked"})
		if callErr != nil {
			panic(callErr)
		}
	})
	if err == nil || err.Code != "SOURCE_OUTSIDE_ROOT" {
		t.Fatalf("%v", err)
	}
}

func TestAHostResolvedPackageAssetStaysInsideTheResolversExplicitRoot(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "project")
	installed := filepath.Join(parent, "installed-font")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(installed, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(project, "main.svml")
	fontPath := filepath.Join(installed, "files", "brand.woff2")
	if err := os.WriteFile(entryPath, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fontPath, []byte("font bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	session := openWorkspace(t, NodeFilesystemWorkspaceOptions{
		Root:          &project,
		SourceAdapter: rawSourceAdapter,
		ExternalAssetResolver: func(_ source.SourceUnit, request source.SourceAssetRequest) (NodeFilesystemExternalAsset, error) {
			if request.From != "package:@acme/brand-fonts/files/brand.woff2" {
				t.Fatalf("from %q", request.From)
			}
			return NodeFilesystemExternalAsset{Root: installed, Asset: fontPath}, nil
		},
	}, entryPath)
	resolved, err := session.ResolveAsset(session.Entry().Unit, source.SourceAssetRequest{
		From:      "package:@acme/brand-fonts/files/brand.woff2",
		MediaType: "font/woff2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Artifact.MediaType != "font/woff2" || len(session.Attachments()) != 1 {
		t.Fatalf("%+v %d", resolved.Artifact, len(session.Attachments()))
	}
}

func TestDefaultAdapterKeepsTheSelfDescribedFrontend(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.svml")
	body := "<?svml using=\"@hypit/markup@1\"?>\n<svml/>\n"
	if err := os.WriteFile(entry, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	session := openWorkspace(t, NodeFilesystemWorkspaceOptions{}, entry)
	if session.Entry().Frontend != "@hypit/markup@1" {
		t.Fatal(session.Entry().Frontend)
	}
}

func TestPathToFileURLMatchesNode(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Node sample was captured on Windows")
	}
	cases := []struct{ path, href string }{
		{`C:\Users\scx\AppData\Local\Temp\a b.txt`, "file:///C:/Users/scx/AppData/Local/Temp/a%20b.txt"},
		{`C:\temp\素材.json`, "file:///C:/temp/%E7%B4%A0%E6%9D%90.json"},
		{`C:\temp\a#b.txt`, "file:///C:/temp/a%23b.txt"},
		{`C:\temp\a?b.txt`, "file:///C:/temp/a%3Fb.txt"},
	}
	for _, tc := range cases {
		if got := pathToFileURL(tc.path); got != tc.href {
			t.Fatalf("%s: %s", tc.path, got)
		}
	}
}

func rawSourceAdapter(unit source.SourceUnit) source.ResolvedSource {
	return source.ResolvedSource{Unit: unit, Frontend: "test.frontend@1"}
}

func openWorkspace(t *testing.T, options NodeFilesystemWorkspaceOptions, entry string) *NodeFilesystemWorkspaceSession {
	t.Helper()
	opened, err := NewNodeFilesystemWorkspace(options).Open(entry)
	if err != nil {
		t.Fatal(err)
	}
	session, ok := opened.(*NodeFilesystemWorkspaceSession)
	if !ok {
		t.Fatalf("%T", opened)
	}
	return session
}

func recoverWorkspace(fn func()) *WorkspaceError {
	var got *WorkspaceError
	func() {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			err, ok := rec.(*WorkspaceError)
			if !ok {
				panic(rec)
			}
			got = err
		}()
		fn()
	}()
	return got
}
