package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestPrefixStableAcrossToolRounds(t *testing.T) {
	dir := t.TempDir()
	var steps []Message
	for i := 0; i < 8; i++ {
		id := "r" + itoa(i)
		steps = append(steps, Message{Role: RoleAssistant, ToolCalls: []ToolCall{{
			ID: id, Name: "read_file", Arguments: `{"path":"a.go"}`,
		}}})
	}
	steps = append(steps, Message{Role: RoleAssistant, Content: "done"})
	client := &prefixClient{ScriptedClient: ScriptedClient{Steps: steps}}
	_, err := Run(context.Background(), RunRequest{
		SessionID:   "s",
		User:        "inspect a.go",
		Workspace:   dir,
		Tools:       &WorkspaceTools{Workspace: dir},
		Client:      client,
		Loop:        DefaultLoop(),
		SoftHorizon: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.hots) < 8 {
		t.Fatalf("rounds %d", len(client.hots))
	}
	for i := 1; i < len(client.hots); i++ {
		if !strings.HasPrefix(client.hots[i], client.hots[i-1]) {
			t.Fatalf("prefix broke at %d prev=%d next=%d", i, len(client.hots[i-1]), len(client.hots[i]))
		}
	}
}

func TestCheckpointBreaksPrefixOnce(t *testing.T) {
	dir := t.TempDir()
	st := trace.NewStore(dir)
	sp := NewSpill(filepath.Join(dir, "spill"))
	hist := []Message{{Role: RoleUser, Content: "old"}, {Role: RoleAssistant, Content: "ok"}}
	before := hotPrefixBytes(append([]Message{{Role: RoleSystem, Content: "sys"}}, hist...))
	dummy := append([]Message{{Role: RoleSystem, Content: "sys"}}, hist...)
	_ = forceCheckpoint(st, "s", dummy, DefaultLoop(), sp, nil, "", 0, "test", nil, CompactOpts{Trigger: "user"})
	evs, _ := st.Read("s")
	after := hotPrefixBytes(MessagesFromEvents(evs))
	if after == before {
		t.Fatal("checkpoint must rewrite the hot prefix")
	}
}

func TestToolUnlockDoesNotMutateSchemaUntilCheckpoint(t *testing.T) {
	tools := &WorkspaceTools{
		ChatOverlay: true,
		Advertised:  ApplyChatToolMenu(nil),
		Extra: map[string]ExtraTool{
			"mcp__s__wide": {JSON: fn("mcp__s__wide", "d", map[string]any{"type": "object"})},
		},
	}
	before := AllToolJSON(tools)
	_ = tools.Call("tool_search", `{"query":"wide"}`)
	after := AllToolJSON(tools)
	if toolsJSONTokens(before) != toolsJSONTokens(after) {
		t.Fatal("tool_search mutated advertised schema before checkpoint")
	}
	tools.CommitToolUnlocks()
	if toolsJSONTokens(AllToolJSON(tools)) <= toolsJSONTokens(before) {
		t.Fatal("checkpoint unlock did not advertise extra")
	}
}

func TestRecallIsPaged(t *testing.T) {
	dir := t.TempDir()
	sp := NewSpill(filepath.Join(dir, "spill"))
	full := strings.Repeat("HEADLINE\n", 400) + "UNIQUE_TAIL_MARKER"
	sp.Put("big", full)
	tools := &WorkspaceTools{Workspace: dir, Spill: sp}
	res := tools.Call("recall_context", `{"id":"big"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(res.Content) >= len(full) {
		t.Fatalf("recall dumped full spill %d", len(res.Content))
	}
	if !strings.Contains(res.Content, "recall id=big") {
		t.Fatalf("%s", res.Content[:min(120, len(res.Content))])
	}
}

func TestHotFileHydrate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hot.go"), []byte("package hot\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	msgs := []Message{
		{Role: RoleUser, Content: "edit"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "w1", Name: "write_file", Arguments: `{"path":"hot.go","content":"x"}`}}},
		{Role: RoleTool, ToolCallID: "w1", Name: "write_file", Content: "wrote hot.go"},
	}
	extra, n := hydrateAfterCheckpoint(dir, msgs, SessionNotes{Files: []string{"hot.go"}}, nil)
	if n == 0 || extra.Role != RoleUser {
		t.Fatal("expected hydrate")
	}
	if !strings.Contains(extra.Content, "hot.go") || !strings.Contains(extra.Content, "package hot") {
		t.Fatalf("%s", extra.Content)
	}
}

func TestHarborShapeUnchanged(t *testing.T) {
	loop := DefaultLoop()
	loop.AllowLLMCompact = true
	orig := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleTool, ToolCallID: "t1", Name: "read_file", Content: strings.Repeat("x", 4000)},
	}
	calls := 0
	client := ClientFunc(func(ctx context.Context, req ChatRequest) (Message, error) {
		calls++
		return Message{Role: RoleAssistant, Content: "## Objective\nx\n## Files\n## Decisions\n## Errors\n## Next\n"}, nil
	})
	out, note := Compact(orig, loop)
	if calls != 0 {
		t.Fatal("Harbor Compact must not call LLM")
	}
	if note == "" && messagesTokens(out) > 24_000 {
		t.Fatal("expected deterministic shape")
	}
	_ = forceCheckpoint(nil, "", orig, loop, nil, client, "m", 0, "eval", nil, CompactOpts{})
	if calls != 0 {
		t.Fatal("Harbor checkpoint with window=0 must not LLM compact")
	}
}

func TestDynamicLivesAtTail(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleDeveloper, Content: "pins"},
		{Role: RoleUser, Content: "u"},
	}
	out := setDynamic(msgs, "plan v1")
	if out[len(out)-1].Role != RoleMemory {
		t.Fatalf("dynamic not at tail: %+v", out)
	}
	out = append(out, Message{Role: RoleAssistant, Content: "ok"})
	out = setDynamic(out, "plan v2")
	if out[len(out)-1].Role != RoleMemory {
		t.Fatal("dynamic must move to tail")
	}
	if out[2].Role != RoleUser {
		t.Fatal("user displaced")
	}
}

type prefixClient struct {
	ScriptedClient
	hots []string
}

func (p *prefixClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	p.hots = append(p.hots, hotPrefixBytes(req.Messages))
	return p.ScriptedClient.Chat(ctx, req)
}

type ClientFunc func(ctx context.Context, req ChatRequest) (Message, error)

func (f ClientFunc) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return f(ctx, req)
}

func TestRetainKeepTokensKeepsOperatorGoal(t *testing.T) {
	goal := "ship the parser in loop.go"
	var msgs []Message
	msgs = append(msgs, Message{Role: RoleUser, Content: goal})
	for i := 0; i < 40; i++ {
		id := "r" + itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: id, Name: "read_file", Arguments: `{"path":"n.go"}`}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: strings.Repeat("x", 80)},
		)
	}
	msgs = append(msgs, Message{Role: RoleUser, Content: "also run tests"})
	out := retainKeepTokens(msgs, 200)
	joined := ""
	for _, m := range out {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "ship the parser") {
		t.Fatalf("lost operator goal:\n%s", joined)
	}
	if messagesTokens(out) > 400 {
		t.Fatalf("keep budget ignored: tokens=%d", messagesTokens(out))
	}
}

func TestCheckpointEmitsStartThenComplete(t *testing.T) {
	dir := t.TempDir()
	st := trace.NewStore(dir)
	sp := NewSpill(filepath.Join(dir, "spill"))
	hist := []Message{{Role: RoleUser, Content: "old"}, {Role: RoleAssistant, Content: "ok"}}
	dummy := append([]Message{{Role: RoleSystem, Content: "sys"}}, hist...)
	_ = forceCheckpoint(st, "s", dummy, DefaultLoop(), sp, nil, "", 0, "test", nil, CompactOpts{Trigger: "user"})
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, ev := range evs {
		if ev.Type == trace.TypeCompact {
			kinds = append(kinds, payloadStr(ev.Payload, "kind"))
		}
	}
	if len(kinds) < 2 || kinds[0] != "checkpoint_start" || kinds[len(kinds)-1] != "checkpoint" {
		t.Fatalf("phases %v", kinds)
	}
}
