package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestDesktopCheckpointUsesStructuredSummary(t *testing.T) {
	dir := t.TempDir()
	st := trace.NewStore(dir)
	loop := ApplyChatHorizon(DefaultLoop())
	body := strings.Repeat("x", 3000)
	var hist []Message
	hist = append(hist,
		Message{Role: RoleSystem, Content: "sys"},
		Message{Role: RoleUser, Content: "build the game"},
		Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "w", Name: "write_file",
			Arguments: `{"path":"main.js","content":"` + body + `"}`,
		}}},
		Message{Role: RoleTool, ToolCallID: "w", Name: "write_file", Content: "wrote main.js"},
	)
	for i := 0; i < 40; i++ {
		id := "r" + itoa(i)
		hist = append(hist,
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: id, Name: "read_file", Arguments: `{"path":"main.js"}`}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: strings.Repeat("z", 4000)},
		)
	}
	summary := "## Objective\nbuild the game\n## Plan\n1. [in_progress] finish verify\n## Files\nmain.js changed\n## Decisions\nkeep the engine\n## Errors\nverify.js input step\n## Next\nfix the cited product\n"
	var sawBody bool
	client := ClientFunc(func(_ context.Context, req ChatRequest) (Message, error) {
		for _, m := range req.Messages {
			for _, tc := range m.ToolCalls {
				if strings.Contains(tc.Arguments, body) {
					sawBody = true
				}
			}
		}
		return Message{Role: RoleAssistant, Content: summary}, nil
	})
	out := forceCheckpoint(st, "s", hist, loop, NewSpill(filepath.Join(dir, "spill")), client, "m", 128000, "runtime", nil, CompactOpts{Trigger: "budget"})
	joined := messagesBlob(out)
	if sawBody {
		t.Fatal("summarizer was sent the raw write body")
	}
	if !strings.Contains(joined, "finish verify") || !strings.Contains(joined, "fix the cited product") {
		t.Fatalf("structured summary missing:\n%s", joined[:min(400, len(joined))])
	}
	if strings.Contains(joined, body) {
		t.Fatal("checkpoint kept the dropped write body")
	}
}

func TestReplayFacbSession(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".yoyo", "sessions", "facb2e21ba0b31a3.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Skip(path)
	}
	defer f.Close()
	var evs []trace.Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		var ev trace.Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			t.Fatalf("line %d: %v", len(evs)+1, err)
		}
		evs = append(evs, ev)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(evs) < 900 {
		t.Fatalf("events %d", len(evs))
	}
	msgs := MessagesFromEvents(evs)
	loop := ApplyChatHorizon(DefaultLoop())
	projected, _ := projectChat(msgs, ShapeOpts{Loop: loop, ModelWindow: 1_000_000})
	original := map[string]string{}
	for _, m := range msgs {
		if m.Role != RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			original[tc.ID] = tc.Arguments
		}
	}
	for _, m := range projected {
		if m.Role != RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if strings.Contains(tc.Arguments, `"_elided"`) && !strings.Contains(original[tc.ID], `"_elided"`) {
				t.Fatalf("projection invented a placeholder for %s", tc.ID)
			}
		}
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "main.js")
	const body = "function tick() {\n  return 1;\n}\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	refused := 0
	for _, ev := range evs {
		if ev.Type != trace.TypeToolCall {
			continue
		}
		name, _ := ev.Payload["name"].(string)
		args, _ := ev.Payload["arguments"].(string)
		if name != "str_replace" || !strings.Contains(args, `"_elided"`) {
			continue
		}
		rewritten := strings.ReplaceAll(args, "game-adaptations/xiyou-tianming-guiling/build/app/main.js", "main.js")
		res := tools.Call("str_replace", rewritten)
		if res.Err == nil {
			t.Fatalf("elided str_replace was executed: %s", args[:min(180, len(args))])
		}
		refused++
	}
	if refused == 0 {
		t.Fatal("session had no elided str_replace to refuse")
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("replay deleted file content:\n%s", got)
	}
}
