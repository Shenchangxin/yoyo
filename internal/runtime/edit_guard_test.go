package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrReplaceMissingNewStrDoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.js")
	body := "function tick() {\n  return 1;\n}\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	raw := `{"path":"main.js","old_str":"function tick() {\n  return 1;\n}\n","_elided":[{"field":"new_str","runes":80}],"_hint":"on disk"}`
	res := tools.Call("str_replace", raw)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "placeholder") {
		t.Fatalf("%+v", res)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("file changed:\n%s", got)
	}
}

func TestWriteFileAppendParts(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	if res := tools.Call("write_file", `{"path":"a.js","content":"const a = 1;\n","mode":"create"}`); res.Err != nil {
		t.Fatal(res.Err)
	}
	if res := tools.Call("write_file", `{"path":"a.js","content":"const b = 2;\n","mode":"append"}`); res.Err != nil {
		t.Fatal(res.Err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "a.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "const a = 1;\nconst b = 2;\n" {
		t.Fatalf("%q", got)
	}
}

func TestWriteFileRejectsHugePart(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	body := strings.Repeat("a", writeFilePartMax+1)
	raw := `{"path":"big.js","content":"` + body + `"}`
	res := tools.Call("write_file", raw)
	if res.Err == nil || !strings.Contains(res.Err.Error(), fmt.Sprintf("%d", writeFilePartMax)) {
		t.Fatalf("%+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "big.js")); !os.IsNotExist(err) {
		t.Fatal("nothing should have been written")
	}
	ok := strings.Repeat("b", writeFilePartMax)
	if res := tools.Call("write_file", `{"path":"ok.js","content":"`+ok+`"}`); res.Err != nil {
		t.Fatal(res.Err)
	}
}

func TestWriteFileAllowsTypicalMarkdownPart(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	body := strings.Repeat("文", 7000) // ~21 KB UTF-8, used to trip the 12 KB gate
	raw := `{"path":"SOURCE_BIBLE.md","content":"` + body + `"}`
	if res := tools.Call("write_file", raw); res.Err != nil {
		t.Fatal(res.Err)
	}
}

func TestSoftGuardStopsWhenNothingMoved(t *testing.T) {
	reset, stop := softGuard(ChatSoftGuardTurns, ChatStallWindow)
	if reset || !stop {
		t.Fatalf("reset=%v stop=%v", reset, stop)
	}
	reset, stop = softGuard(ChatSoftGuardTurns, 0)
	if !reset || stop {
		t.Fatalf("recent progress must extend, reset=%v stop=%v", reset, stop)
	}
	reset, stop = softGuard(ChatSoftGuardTurns-1, ChatStallWindow)
	if reset || stop {
		t.Fatalf("under the cap must keep going, reset=%v stop=%v", reset, stop)
	}
}

func TestTrimTruncatedReplyDropsCutToolCall(t *testing.T) {
	msg, nudge := trimTruncatedReply(Message{
		Role:      RoleAssistant,
		Truncated: true,
		ToolCalls: []ToolCall{
			{ID: "ok", Name: "read_file", Arguments: `{"path":"a.js"}`},
			{ID: "cut", Name: "write_file", Arguments: `{"path":"b.js","content":"partial`},
		},
	})
	if nudge == "" || len(msg.ToolCalls) != 1 || msg.ToolCalls[0].ID != "ok" {
		t.Fatalf("msg=%+v nudge=%q", msg.ToolCalls, nudge)
	}
}
