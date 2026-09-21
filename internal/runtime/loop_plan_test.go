package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanOpen(t *testing.T) {
	if PlanOpen("") || PlanOpen("just a paragraph") {
		t.Fatal("empty")
	}
	open := "勘察后列出步骤\n\n1. [in_progress] 勘察现状\n2. [pending] 前端重写\n"
	if !PlanOpen(open) {
		t.Fatal("open plan")
	}
	done := "1. [complete] a\n2. [completed] b\n"
	if PlanOpen(done) {
		t.Fatal("done plan")
	}
	if PlanOpen("1. [done] a\n2. [finished] b\n") {
		t.Fatal("done aliases")
	}
}

func TestPlanContinueIfOpen(t *testing.T) {
	tools := &WorkspaceTools{}
	res := tools.Call("update_plan", `{"plan":[{"step":"a","status":"in_progress"},{"step":"b","status":"pending"}]}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	req := RunRequest{Tools: tools}
	c, at := 0, -1
	if got := planContinueIfOpen(req, false, 3, &c, &at); got != planContinueNudge {
		t.Fatalf("first %q", got)
	}
	if c != 1 || at != 3 {
		t.Fatalf("c=%d at=%d", c, at)
	}
	if got := planContinueIfOpen(req, false, 3, &c, &at); got != "" {
		t.Fatalf("no-progress must stop, got %q", got)
	}
	if got := planContinueIfOpen(req, false, 4, &c, &at); got != planContinueNudge {
		t.Fatalf("progress %q", got)
	}
	if planContinueIfOpen(req, true, 5, &c, &at) != "" {
		t.Fatal("plan mode")
	}
	done := &WorkspaceTools{}
	if res := done.Call("update_plan", `{"plan":[{"step":"a","status":"complete"}]}`); res.Err != nil {
		t.Fatal(res.Err)
	}
	c, at = 0, -1
	if planContinueIfOpen(RunRequest{Tools: done}, false, 1, &c, &at) != "" {
		t.Fatal("complete plan")
	}
}

func TestPlanContinueNudgeIsLanguageNeutral(t *testing.T) {
	tools := &WorkspaceTools{}
	if res := tools.Call("update_plan", `{"plan":[{"step":"勘察现状","status":"in_progress"}]}`); res.Err != nil {
		t.Fatal(res.Err)
	}
	req := RunRequest{User: "完善该项目", Tools: tools}
	got := planContinueIfOpen(req, false, 1, nil, nil)
	if got != planContinueNudge {
		t.Fatalf("%q", got)
	}
}

func TestRunContinuesOpenPlanInsteadOfEndTurn(t *testing.T) {
	dir := t.TempDir()
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"inspect","status":"in_progress"},{"step":"write","status":"pending"}]}`,
		}}},
		{Role: RoleAssistant, Content: "plan ready, please confirm"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "w1", Name: "write_file",
			Arguments: `{"path":"out.txt","content":"shipped"}`,
		}}},
		{Role: RoleAssistant, Content: "still need the rest"},
		{Role: RoleAssistant, Content: "stopping now"},
	}}
	out, err := Run(context.Background(), RunRequest{
		User:      "build the thing",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      DefaultLoop(),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "shipped" {
		t.Fatalf("wrote %q", b)
	}
	if !strings.Contains(out, "stopping now") {
		t.Fatalf("should stop after a continue that produced no tools, got %q", out)
	}
	if client.i < 5 {
		t.Fatalf("expected continue past the confirm turn, steps used %d", client.i)
	}
}

func TestRunDoesNotContinueCompletePlan(t *testing.T) {
	dir := t.TempDir()
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"inspect","status":"complete"}]}`,
		}}},
		{Role: RoleAssistant, Content: "all done"},
		{Role: RoleAssistant, Content: "should not run"},
	}}
	out, err := Run(context.Background(), RunRequest{
		User:      "build the thing",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      DefaultLoop(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "all done" {
		t.Fatalf("%q", out)
	}
	if client.i > 2 {
		t.Fatalf("continued a complete plan, steps %d", client.i)
	}
}

type planRecordingClient struct {
	ScriptedClient
	seen [][]Message
}

func (c *planRecordingClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	cp := make([]Message, len(req.Messages))
	copy(cp, req.Messages)
	c.seen = append(c.seen, cp)
	return c.ScriptedClient.Chat(ctx, req)
}

func TestOpenPlanUsesContinueNudgeAtToolCap(t *testing.T) {
	dir := t.TempDir()
	loop := DefaultLoop()
	loop.MaxToolMessages = 1
	client := &planRecordingClient{ScriptedClient: ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"inspect","status":"in_progress"},{"step":"write","status":"pending"}]}`,
		}}},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "w1", Name: "write_file",
			Arguments: `{"path":"out.txt","content":"shipped"}`,
		}}},
		{Role: RoleAssistant, Content: "stopping now"},
		{Role: RoleAssistant, Content: "should not run"},
	}}}
	out, err := Run(context.Background(), RunRequest{
		User:      "build the thing",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      loop,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "should not run" {
		t.Fatalf("open plan must continue past the first wrap-up, got %q", out)
	}
	if len(client.seen) < 2 {
		t.Fatalf("chats %d", len(client.seen))
	}
	second := strings.Join(contentsOf(client.seen[1], RoleUser), "\n")
	if !strings.Contains(second, planContinueNudge) {
		t.Fatalf("expected continue nudge, got %q", second)
	}
	if strings.Contains(second, toolBudgetNudge) {
		t.Fatalf("stop nudge leaked onto an open plan: %q", second)
	}
	b, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "shipped" {
		t.Fatalf("wrote %q", b)
	}
}

func contentsOf(msgs []Message, role Role) []string {
	var out []string
	for _, m := range msgs {
		if m.Role == role && strings.TrimSpace(m.Content) != "" {
			out = append(out, m.Content)
		}
	}
	return out
}

func TestPlanModeDoesNotForceContinue(t *testing.T) {
	dir := t.TempDir()
	loop := DefaultLoop()
	loop.PlanMode = true
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: "p1", Name: "update_plan",
			Arguments: `{"plan":[{"step":"inspect","status":"in_progress"},{"step":"write","status":"pending"}]}`,
		}}},
		{Role: RoleAssistant, Content: "here is the plan"},
		{Role: RoleAssistant, Content: "should not run"},
	}}
	out, err := Run(context.Background(), RunRequest{
		User:      "plan it",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir, PlanMode: true},
		Client:    client,
		Loop:      loop,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "here is the plan" {
		t.Fatalf("%q", out)
	}
}

func TestSoftHorizonIgnoresTurnAndToolCaps(t *testing.T) {
	dir := t.TempDir()
	loop := DefaultLoop()
	loop.MaxTurns = 2
	loop.MaxToolMessages = 1
	var steps []Message
	for i := 0; i < 5; i++ {
		steps = append(steps, Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: fmt.Sprintf("w%d", i), Name: "write_file",
			Arguments: fmt.Sprintf(`{"path":"f%d.txt","content":"%d"}`, i, i),
		}}})
	}
	steps = append(steps, Message{Role: RoleAssistant, Content: "finished the long task"})
	client := &planRecordingClient{ScriptedClient: ScriptedClient{Steps: steps}}
	out, err := Run(context.Background(), RunRequest{
		User:        "build the thing",
		Workspace:   dir,
		Tools:       &WorkspaceTools{Workspace: dir},
		Client:      client,
		Loop:        loop,
		SoftHorizon: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "finished the long task" {
		t.Fatalf("%q", out)
	}
	for i := 0; i < 5; i++ {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("f%d.txt", i)))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != fmt.Sprintf("%d", i) {
			t.Fatalf("f%d.txt %q", i, b)
		}
	}
	for i, msgs := range client.seen {
		joined := strings.Join(contentsOf(msgs, RoleUser), "\n")
		if strings.Contains(joined, toolBudgetNudge) {
			t.Fatalf("chat injected stop-tools on turn %d: %q", i, joined)
		}
	}
	if client.i < 6 {
		t.Fatalf("stopped early, steps %d", client.i)
	}
}

func TestHardHorizonStillStopsAtMaxTurns(t *testing.T) {
	dir := t.TempDir()
	loop := DefaultLoop()
	loop.MaxTurns = 2
	var steps []Message
	for i := 0; i < 5; i++ {
		steps = append(steps, Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: fmt.Sprintf("w%d", i), Name: "write_file",
			Arguments: fmt.Sprintf(`{"path":"f%d.txt","content":"%d"}`, i, i),
		}}})
	}
	steps = append(steps, Message{Role: RoleAssistant, Content: "should not run"})
	_, err := Run(context.Background(), RunRequest{
		User:      "build the thing",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    &ScriptedClient{Steps: steps},
		Loop:      loop,
	})
	if err == nil || !strings.Contains(err.Error(), "max turns") {
		t.Fatalf("harbor must keep a hard turn cap, err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "f2.txt")); !os.IsNotExist(statErr) {
		t.Fatal("hard cap should not reach the third write")
	}
}
