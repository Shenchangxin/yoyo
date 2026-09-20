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
