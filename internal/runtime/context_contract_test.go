// 锁住热前缀只往后接、检查点会换掉热前缀、动态块在历史末尾、recall 按页返回，以及窗口为 0 时不做模型摘要。

package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestPrefixStableAcrossToolRounds 确认连续工具回合里，后一轮热前缀仍以前一轮的字节为开头。
func TestPrefixStableAcrossToolRounds(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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

// TestCheckpointBreaksPrefixOnce 确认检查点之后的热前缀和检查点之前不同。
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

// TestToolUnlockDoesNotMutateSchemaUntilCheckpoint 确认 tool_search 在提交解锁之前不改变已公布 schema 的 token。
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

// TestRecallIsPaged 确认不传 offset 和 limit 时，recall_context 不倒出全文，并带上 recall id。
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

// TestHotFileHydrate 确认检查点补水会带上刚写过的文件路径和磁盘上的正文。
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

// TestHarborShapeUnchanged 确认 Compact 不调用模型；AllowLLMCompact 为真但 window 为 0 时，检查点也不做模型摘要。
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

// TestDynamicLivesAtTail 确认动态块在消息末尾，再次设置时旧块去掉、新块仍在末尾，用户消息留在原处。
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

// Chat 记下每次请求的热前缀，再交给脚本客户端。
func (p *prefixClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	p.hots = append(p.hots, hotPrefixBytes(req.Messages))
	return p.ScriptedClient.Chat(ctx, req)
}

// ClientFunc 把函数适配成 Client，测试用它替换模型调用。
type ClientFunc func(ctx context.Context, req ChatRequest) (Message, error)

// Chat 把请求转给这个函数值。
func (f ClientFunc) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return f(ctx, req)
}

// TestRetainKeepTokensKeepsOperatorGoal 确认按 token 保留尾部时，开头的操作者目标还在，并且总量不超过放宽后的上限。
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

// TestCheckpointEmitsStartThenComplete 确认检查点事件先是 checkpoint_start，最后一条是 checkpoint。
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
