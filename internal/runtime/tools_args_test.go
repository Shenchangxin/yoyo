package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseToolArgsRepairsLiteralNewlines(t *testing.T) {
	raw := "{\"path\": \"reports/out.md\", \"content\": \"# Title\n\n- item A\n- item B\"}"
	args := parseToolArgs(raw)
	if str(args["path"]) != "reports/out.md" {
		t.Fatalf("path=%v", args["path"])
	}
	content := str(args["content"])
	if !strings.Contains(content, "# Title") || !strings.Contains(content, "- item B") {
		t.Fatalf("content=%q", content)
	}
}

func TestParseToolArgsClosesTruncatedContent(t *testing.T) {
	raw := "{\"path\": \"reports/out.md\", \"content\": \"# Title\n\npartial"
	args := parseToolArgs(raw)
	if str(args["path"]) != "reports/out.md" {
		t.Fatalf("path=%v args=%v", args["path"], args)
	}
	if !strings.Contains(str(args["content"]), "# Title") {
		t.Fatalf("content=%q", args["content"])
	}
}

func TestWriteFileSalvagesLiteralNewlines(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	raw := "{\"path\": \"reports/out.md\", \"content\": \"# Agent survey\n\n- 2609.13406\"}"
	res := tools.Call("write_file", raw)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "reports", "out.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "# Agent survey") || !strings.Contains(got, "2609.13406") {
		t.Fatalf("%s", got)
	}
}

func TestApplyPatchSalvagesLiteralNewlines(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	raw := "{\"patch\": \"*** Begin Patch\n*** Add File: notes.md\n+# hello\n+world\n*** End Patch\"}"
	res := tools.Call("apply_patch", raw)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "# hello") {
		t.Fatalf("%s", b)
	}
}

func TestWriteFileEmptyPathStillErrorsOnBlankJSON(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("write_file", `{}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "empty path") {
		t.Fatalf("%+v", res)
	}
}

func TestListDirIncludesSizeAndMtime(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("list_dir", `{"path":"."}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "a.txt") || !strings.Contains(res.Content, "5") {
		t.Fatalf("%s", res.Content)
	}
	if !strings.Contains(res.Content, "T") {
		t.Fatalf("missing mtime: %s", res.Content)
	}
}

func TestUsableWindowsPosixShellRejectsWSL(t *testing.T) {
	if usableWindowsPosixShell(`C:\Windows\System32\bash.exe`) {
		t.Fatal("system32")
	}
	if usableWindowsPosixShell(`C:\Windows\Sysnative\bash.exe`) {
		t.Fatal("sysnative")
	}
	if usableWindowsPosixShell(`C:\Windows\SysWOW64\bash.exe`) {
		t.Fatal("syswow64")
	}
}
