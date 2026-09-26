package runtime

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestNotesSkipControlUsers(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "fix the parser in loop.go"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "1", Name: "read_file", Arguments: `{"path":"loop.go"}`}}},
		{Role: RoleTool, ToolCallID: "1", Name: "read_file", Content: "ok"},
		{Role: RoleUser, Content: toolBudgetNudge},
		{Role: RoleUser, Content: planContinueNudge},
		{Role: RoleUser, Content: planContinueNudgeZH},
		{Role: RoleUser, Content: "Context checkpoint (untrusted working memory; pins were reassembled separately):\n## Objective\nwrong"},
		{Role: RoleUser, Content: mentionUserPrefix + "\nREADME.md"},
		{Role: RoleUser, Content: steerUserPrefix + "\nignore this"},
	}
	n := NotesFromMessages(msgs)
	if n.Objective != "fix the parser in loop.go" {
		t.Fatalf("objective %q", n.Objective)
	}
	evs := []trace.Event{
		{Type: trace.TypeUser, Source: "user", Payload: map[string]any{"text": "fix the parser in loop.go"}},
		{Type: trace.TypeUser, Source: "steer", Payload: map[string]any{"text": "do something else"}},
		{Type: trace.TypeUser, Source: "runtime", Payload: map[string]any{"text": toolBudgetNudge}},
	}
	got := ExtractNotes(evs)
	if got.Objective != "fix the parser in loop.go" {
		t.Fatalf("extract objective %q", got.Objective)
	}
}

func TestAttachMentionPutsOperatorFirst(t *testing.T) {
	got := AttachMention("帮我查论文", "## @skill:arxiv-watcher\nUse scripts")
	if !strings.HasPrefix(got, "帮我查论文") {
		t.Fatalf("%q", got)
	}
	if !strings.Contains(got, mentionUserPrefix) || !strings.Contains(got, "arxiv-watcher") {
		t.Fatalf("%q", got)
	}
	if AttachMention("hello", "") != "hello" {
		t.Fatal("empty inject")
	}
}

func TestWriteNotesKeepsOldObjective(t *testing.T) {
	sp := NewSpill(t.TempDir())
	WriteNotes(sp, SessionNotes{Objective: "ship the patch", Files: []string{"a.go"}})
	WriteNotes(sp, SessionNotes{Objective: "", Files: []string{"b.go"}})
	got := ParseNotes(ReadNotes(sp))
	if got.Objective != "ship the patch" {
		t.Fatalf("objective %q", got.Objective)
	}
	joined := strings.Join(got.Files, " ")
	if !strings.Contains(joined, "a.go") || !strings.Contains(joined, "b.go") {
		t.Fatalf("files %+v", got.Files)
	}
}

func TestResumeDoesNotReplaceObjective(t *testing.T) {
	sp := NewSpill(t.TempDir())
	WriteNotes(sp, SessionNotes{Objective: "完善 RBAC 控制台"})
	WriteNotes(sp, NotesFromMessages([]Message{
		{Role: RoleUser, Content: "完善 RBAC 控制台"},
		{Role: RoleUser, Content: "继续"},
	}))
	got := ParseNotes(ReadNotes(sp))
	if got.Objective != "完善 RBAC 控制台" {
		t.Fatalf("objective %q", got.Objective)
	}
	evs := []trace.Event{
		{Type: trace.TypeUser, Source: "user", Payload: map[string]any{"text": "完善 RBAC 控制台"}},
		{Type: trace.TypeUser, Source: "user", Payload: map[string]any{"text": "继续"}},
	}
	if ExtractNotes(evs).Objective != "完善 RBAC 控制台" {
		t.Fatalf("extract %q", ExtractNotes(evs).Objective)
	}
}

func TestObjectiveStickyAcrossFollowUps(t *testing.T) {
	n := NotesFromMessages([]Message{
		{Role: RoleUser, Content: "ship the parser"},
		{Role: RoleUser, Content: "also fix the tests"},
	})
	if n.Objective != "ship the parser" {
		t.Fatalf("objective %q", n.Objective)
	}
	joined := strings.Join(n.Decisions, " ")
	if !strings.Contains(joined, "also fix the tests") {
		t.Fatalf("decisions %+v", n.Decisions)
	}
}

func TestCollapseDoubledObjective(t *testing.T) {
	once := "本项目是一个基于golang开发的RBAC权限管理系统，请完善前端界面。"
	n := NotesFromMessages([]Message{{Role: RoleUser, Content: once + once}})
	if n.Objective != once {
		t.Fatalf("objective %q", n.Objective)
	}
	if got := collapseDoubledText("haha"); got != "haha" {
		t.Fatalf("short %q", got)
	}
}

func TestNotePathsSkipGlobs(t *testing.T) {
	n := NotesFromMessages([]Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{Name: "glob", Arguments: `{"glob":"**/*.go"}`},
			{Name: "read_file", Arguments: `{"path":"api/api.go"}`},
			{Name: "list_dir", Arguments: `{"path":"."}`},
		}},
	})
	if len(n.Files) != 1 || n.Files[0] != "api/api.go" {
		t.Fatalf("files %+v", n.Files)
	}
}
