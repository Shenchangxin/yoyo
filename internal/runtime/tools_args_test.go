package runtime

import (
	"encoding/json"
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
	if !toolArgsTruncated(raw) {
		t.Fatal("truncated payload must be flagged")
	}
}

func TestWriteFileSavesTruncatedNewFile(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	raw := "{\"path\": \"game.html\", \"content\": \"<!DOCTYPE html>\\n<html>\\nfunction startGame(){\\npartial"
	res := tools.Call("write_file", raw)
	if res.Err != nil {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Content, "partial") || !strings.Contains(res.Content, "game.html") {
		t.Fatalf("expected partial-write hint: %s", res.Content)
	}
	if !strings.Contains(res.Content, "startGame") {
		t.Fatalf("hint should include last complete marker: %s", res.Content)
	}
	b, err := os.ReadFile(filepath.Join(dir, "game.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<!DOCTYPE html>") || !strings.Contains(string(b), "function startGame") {
		t.Fatalf("recovered prefix not on disk: %s", b)
	}
}

func TestWriteFileRefusesTruncatedOverwriteOfLargerFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.html")
	full := strings.Repeat("x", 400)
	if err := os.WriteFile(path, []byte(full), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	raw := "{\"path\": \"game.html\", \"content\": \"<!DOCTYPE html>\\n<html>\\nfunction startGame(){\\npartial"
	res := tools.Call("write_file", raw)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "truncated") {
		t.Fatalf("%+v", res)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != full {
		t.Fatalf("existing file clobbered")
	}
}

func TestWriteFileAcceptsFencedJSON(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	raw := "```json\n{\"path\":\"notes.md\",\"content\":\"# hi\"}\n```"
	res := tools.Call("write_file", raw)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "# hi" {
		t.Fatalf("%q", b)
	}
	if res.FileChange == nil || !res.FileChange.Created || !strings.Contains(res.FileChange.Patch, "+# hi") {
		t.Fatalf("file change %+v", res.FileChange)
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

func TestUnwrapNestedWriteArguments(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	raw := `{"arguments":{"path":"reports/_part_head.html","content":"<h1>ok</h1>"}}`
	res := tools.Call("write_file", raw)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "reports", "_part_head.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<h1>ok</h1>") {
		t.Fatalf("%s", b)
	}
}

func TestWriteFileRejectsContextStub(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	stub := "[elided content 687 chars path=tools/md2canvas.py — on disk, read_file that path; do not rewrite from memory]\nprint(1)\n"
	res := tools.Call("write_file", `{"path":"tools/md2canvas.py","content":`+mustJSONString(stub)+`}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "context stub") {
		t.Fatalf("%+v", res)
	}
}

func mustJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestStrReplaceRejectsContextStub(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.html")
	if err := os.WriteFile(path, []byte("const GROUND = H - 78;"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	stubs := []string{
		"[elided new_str 9627 chars path=game.html — on disk, read_file that path; do not rewrite from memory]\nconst GROUND = H - 78;",
		"(omitted 9627-char new_str already on disk at game.html; read_file that path. Do not paste this placeholder as new_str.)\nconst GROUND = H - 78;",
	}
	for _, stub := range stubs {
		res := tools.Call("str_replace", `{"path":"game.html","old_str":"const GROUND = H - 78;","new_str":`+mustJSONString(stub)+`}`)
		if res.Err == nil || !strings.Contains(res.Err.Error(), "context stub") {
			t.Fatalf("%+v", res)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "elided") || strings.Contains(string(b), "omitted") {
		t.Fatalf("file mutated: %s", b)
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
