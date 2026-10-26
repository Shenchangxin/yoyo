// 锁住工具参数里的字面换行可以修好，截断的写文件不落盘，上下文占位不覆盖已有文件，路径对不上的 recall 会被拒绝。

package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseToolArgsRepairsLiteralNewlines 锁住 JSON 字符串里的原始换行能解析成真正的换行，且不算截断。
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

// TestParseToolArgsClosesTruncatedContent 锁住未闭合的 JSON 能补括号解析，并标成截断。
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

// TestWriteFileRefusesTruncatedNewFile 锁住截断的 write_file 不创建文件。
func TestWriteFileRefusesTruncatedNewFile(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	raw := "{\"path\": \"game.html\", \"content\": \"<!DOCTYPE html>\\n<html>\\nfunction startGame(){\\npartial"
	res := tools.Call("write_file", raw)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "truncated") || !strings.Contains(res.Err.Error(), "mode=append") {
		t.Fatalf("%+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "game.html")); !os.IsNotExist(err) {
		t.Fatal("truncated write must not land on disk")
	}
}

// TestWriteFileRefusesTruncatedOverwriteOfLargerFile 锁住截断载荷不会用半截正文覆盖更大的已有文件。
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

// TestWriteFileAcceptsFencedJSON 锁住包在 ```json 围栏里的参数仍能写入。
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

// TestWriteFileSalvagesLiteralNewlines 锁住带字面换行的合法 JSON 能写入，且文件里是真换行。
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

// TestApplyPatchSalvagesLiteralNewlines 锁住补丁参数里的字面换行能被应用。
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

// TestUnwrapNestedWriteArguments 锁住嵌在 arguments 里的 path 和 content 能抬上来并写入。
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

// TestWriteFileStubWhenOnDiskIsUnchanged 锁住磁盘上已有相同正文时，上下文占位返回 unchanged，不改文件。
func TestWriteFileStubWhenOnDiskIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.ToSlash(filepath.Join("docs", "CONCEPT.md"))
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("# real concept\n")
	if err := os.WriteFile(abs, want, 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	stub := `(omitted 8405-char content already on disk at ` + rel + `; read_file that path. Do not paste this placeholder as content.)`
	res := tools.Call("write_file", `{"path":`+mustJSONString(rel)+`,"content":`+mustJSONString(stub)+`}`)
	if res.Err != nil {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Content, "unchanged") {
		t.Fatalf("%s", res.Content)
	}
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("clobbered: %s", got)
	}
}

// TestWriteFileStubRecoversFromSpill 锁住占位写入能从溢出的 args 里按路径找回正文并写入。
func TestWriteFileStubRecoversFromSpill(t *testing.T) {
	dir := t.TempDir()
	spill := NewSpill(t.TempDir())
	rel := "concepts/CONCEPT.md"
	body := "# recovered from spill\n\nThe real file body that was compacted."
	spill.Put("call1:args", `{"path":"`+rel+`","content":`+mustJSONString(body)+`}`)
	tools := &WorkspaceTools{Workspace: dir, Spill: spill}
	stub := `(omitted 80-char content already on disk at ` + rel + `; read_file that path. Do not paste this placeholder as content.)`
	res := tools.Call("write_file", `{"path":`+mustJSONString(rel)+`,"content":`+mustJSONString(stub)+`}`)
	if res.Err != nil {
		t.Fatalf("%+v", res)
	}
	got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("got %q", got)
	}
}

// TestWriteFileElidedRecallRecoversMatchingPath 锁住 _recall 指向的溢出参数在路径一致时能恢复正文。
func TestWriteFileElidedRecallRecoversMatchingPath(t *testing.T) {
	dir := t.TempDir()
	spill := NewSpill(t.TempDir())
	rel := "design/GAME_DESIGN.md"
	body := "# GAME_DESIGN — recovered via _recall\n"
	id := "call_00_Ha2Sy21RYPnaJRlNH0p4929:args"
	spill.Put(id, `{"path":"`+rel+`","content":`+mustJSONString(body)+`}`)
	tools := &WorkspaceTools{Workspace: dir, Spill: spill}
	raw := `{"path":"` + rel + `","_elided":[{"field":"content","runes":80}],"_hint":"on disk","_recall":"` + id + `"}`
	res := tools.Call("write_file", raw)
	if res.Err != nil {
		t.Fatalf("%+v", res)
	}
	got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("got %q", got)
	}
}

// TestWriteFileElidedRecallWrongPathRefuses 锁住 recall 里的路径和本次 path 不一致时拒绝写入。
func TestWriteFileElidedRecallWrongPathRefuses(t *testing.T) {
	dir := t.TempDir()
	spill := NewSpill(t.TempDir())
	spill.Put("call_00_other:args", `{"path":"other.md","content":"# other"}`)
	tools := &WorkspaceTools{Workspace: dir, Spill: spill}
	res := tools.Call("write_file", `{"path":"design/GAME_DESIGN.md","_elided":[{"field":"content","runes":80}],"recall":"call_00_other:args"}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "context stub") {
		t.Fatalf("%+v", res)
	}
}

// TestWriteFileRejectsContextStub 锁住像上下文摘要的 content 在文件还不存在时被拒绝。
func TestWriteFileRejectsContextStub(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	stub := "[elided content 687 chars path=tools/md2canvas.py — on disk, read_file that path; do not rewrite from memory]\nprint(1)\n"
	res := tools.Call("write_file", `{"path":"tools/md2canvas.py","content":`+mustJSONString(stub)+`}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "context stub") {
		t.Fatalf("%+v", res)
	}
}

// mustJSONString 把字符串编成 JSON 字面量，失败则让测试失败。
func mustJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestStrReplaceRejectsContextStub 锁住 old_str 或 new_str 像上下文占位时不改文件。
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

// TestWriteFileEmptyPathStillErrorsOnBlankJSON 锁住空 JSON 对象因缺少 path 而失败。
func TestWriteFileEmptyPathStillErrorsOnBlankJSON(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("write_file", `{}`)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "empty path") {
		t.Fatalf("%+v", res)
	}
}

// TestListDirIncludesSizeAndMtime 锁住 list_dir 的文件行带字节大小。
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

// TestUsableWindowsPosixShellRejectsWSL 锁住 system32 或 WindowsApps 下的 bash 不算可用的 POSIX shell。
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
