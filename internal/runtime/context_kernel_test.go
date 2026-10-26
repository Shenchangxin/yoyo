// 锁住工具配对、spill 先于截断、技能加载不改前缀字节、检查点从尾部重建，以及聊天投影保留写操作参数。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestLegalizePairsToolCalls 确认缺了结果的 tool call 会被补上配对。
func TestLegalizePairsToolCalls(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "a", Name: "read_file", Arguments: "{}"}, {ID: "b", Name: "grep", Arguments: "{}"}}},
		{Role: RoleTool, ToolCallID: "a", Name: "read_file", Content: "one"},
	}
	out := Legalize(msgs)
	ids := map[string]bool{}
	for _, m := range out {
		if m.Role == RoleTool {
			ids[m.ToolCallID] = true
		}
	}
	if !ids["a"] || !ids["b"] {
		t.Fatalf("missing pair: %+v", out)
	}
}

// TestSnipDoesNotBreakPairing 确认剪旧回合之后，再 Legalize 不会留下未配对的 tool call。
func TestSnipDoesNotBreakPairing(t *testing.T) {
	loop := DefaultLoop()
	loop.CompactionKeep = 2
	loop.CompactionTokens = 50
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "old"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}},
		{Role: RoleTool, ToolCallID: "t1", Name: "read_file", Content: strings.Repeat("x", 4000)},
		{Role: RoleUser, Content: "new"},
		{Role: RoleAssistant, Content: "ok"},
	}
	out, _ := Shape(msgs, ShapeOpts{Loop: loop})
	out = Legalize(out)
	pending := map[string]bool{}
	for _, m := range out {
		if m.Role == RoleAssistant {
			for _, tc := range m.ToolCalls {
				pending[tc.ID] = true
			}
		}
		if m.Role == RoleTool {
			delete(pending, m.ToolCallID)
		}
	}
	if len(pending) != 0 {
		t.Fatalf("unpaired tool_calls after snip: %v\n%+v", pending, out)
	}
}

// TestSpillBeforeCapIsLossless 确认预览被截成占位时，spill 里仍是全文。
func TestSpillBeforeCapIsLossless(t *testing.T) {
	dir := t.TempDir()
	sp := NewSpill(filepath.Join(dir, "spill"))
	full := strings.Repeat("TAIL", 20_000)
	preview, _ := ingestToolResult(sp, "big", "shell", full, 0, false)
	if !strings.Contains(preview, "elided") {
		t.Fatalf("expected preview stub, got %s", preview[:min(120, len(preview))])
	}
	got, err := sp.Get("big")
	if err != nil || got != full {
		t.Fatalf("spill lost bytes: err=%v len=%d", err, len(got))
	}
}

// TestAssemblePrefixStableWhenSkillsLoad 确认同一材料的前缀字节稳定，加载技能正文只出现在前缀之后。
func TestAssemblePrefixStableWhenSkillsLoad(t *testing.T) {
	loop := DefaultLoop()
	skills := []artifact.Skill{{Name: "b", Description: "two", Body: "BBB"}, {Name: "a", Description: "one", Body: "AAA"}}
	p1 := AssemblePrefix(loop, nil, artifact.Playbook{}, skills, "tabs", "YOYO.md")
	p2 := AssemblePrefix(loop, nil, artifact.Playbook{}, skills, "tabs", "YOYO.md")
	if p1 != p2 {
		t.Fatal("prefix not deterministic")
	}
	full := Assemble(loop, nil, artifact.Playbook{}, skills, "tabs", "YOYO.md", []string{"## Skill: a\nAAA"})
	if !strings.HasPrefix(full, p1) {
		t.Fatal("loaded skills mutated prefix bytes")
	}
	if !strings.Contains(full[len(p1):], "AAA") {
		t.Fatal("dynamic tail missing loaded skill")
	}
}

// TestCheckpointRebuildsFromTail 确认检查点之前的用户句不再出现，摘要、保留尾和新用户句还在。
func TestCheckpointRebuildsFromTail(t *testing.T) {
	dir := t.TempDir()
	st := trace.NewStore(dir)
	_ = st.Append(trace.Event{Type: trace.TypeUser, SessionID: "s", Payload: map[string]any{"text": "old user"}})
	_ = st.Append(trace.Event{Type: trace.TypeAssistant, SessionID: "s", Payload: map[string]any{"text": "old asst"}})
	tail := []Message{{Role: RoleUser, Content: "kept user"}, {Role: RoleAssistant, Content: "kept asst"}}
	if err := PersistCheckpoint(st, "s", "test", "## Objective\nkeep going", tail, true); err != nil {
		t.Fatal(err)
	}
	_ = st.Append(trace.Event{Type: trace.TypeUser, SessionID: "s", Payload: map[string]any{"text": "new"}})
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	msgs := MessagesFromEvents(evs)
	joined := ""
	for _, m := range msgs {
		joined += m.Content + "\n"
	}
	if strings.Contains(joined, "old user") {
		t.Fatalf("pre-checkpoint leaked: %s", joined)
	}
	if !strings.Contains(joined, "keep going") || !strings.Contains(joined, "kept user") || !strings.Contains(joined, "new") {
		t.Fatalf("missing checkpoint pieces: %s", joined)
	}
}

// TestSchemaCapDefersExtras 确认额外工具默认不进 tools 数组，tool_search 命中后才出现。
func TestSchemaCapDefersExtras(t *testing.T) {
	extra := map[string]ExtraTool{}
	for i := 0; i < 12; i++ {
		name := "mcp__s__t" + itoa(i)
		extra[name] = ExtraTool{JSON: fn(name, "d", map[string]any{"type": "object"})}
	}
	tools := &WorkspaceTools{Extra: extra}
	js := AllToolJSON(tools)
	count := 0
	for _, j := range js {
		name, _ := j.Function["name"].(string)
		if strings.HasPrefix(name, "mcp__") {
			count++
		}
	}
	if count != 0 {
		t.Fatalf("expected deferred extras, advertised %d", count)
	}
	res := tools.Call("tool_search", `{"query":"t1"}`)
	if res.Err != nil || !strings.Contains(res.Content, "mcp__s__t1") {
		t.Fatalf("%+v", res)
	}
	js = AllToolJSON(tools)
	found := false
	for _, j := range js {
		name, _ := j.Function["name"].(string)
		if name == "mcp__s__t1" {
			found = true
		}
	}
	if !found {
		t.Fatal("tool_search did not enable matching extra")
	}
}

// TestCopyAndRemoveSpill 确认 CopyTree 复制 spill 文件，RemoveSessionContext 删掉源目录。
func TestCopyAndRemoveSpill(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	src := filepath.Join(home, "sessions", "a", "spill")
	sp := NewSpill(src)
	sp.Put("x", "secret-full")
	dst := filepath.Join(home, "sessions", "b", "spill")
	if err := CopyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "x.txt"))
	if err != nil || string(got) != "secret-full" {
		t.Fatalf("%q %v", got, err)
	}
	RemoveSessionContext(home, ws, "a")
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("spill dir not removed")
	}
}

// TestHarborIgnoresModelWindow 确认不填 ModelWindow 时预算是 24000，填了窗口之后预算更大且仍小于窗口。
func TestHarborIgnoresModelWindow(t *testing.T) {
	loop := DefaultLoop()
	b := effectiveBudget(ShapeOpts{Loop: loop})
	if b != 24_000 {
		t.Fatalf("harbor budget %d", b)
	}
	chat := effectiveBudget(ShapeOpts{Loop: loop, ModelWindow: 128_000})
	if chat <= b || chat >= 128_000 {
		t.Fatalf("chat budget %d", chat)
	}
}

// TestChatPromptKeepsWriteArguments 确认聊天投影和原始消息都保留 write_file 的完整参数。
func TestChatPromptKeepsWriteArguments(t *testing.T) {
	body := strings.Repeat("x", 4000)
	origArgs := `{"path":"a.ts","content":"` + body + `"}`
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "w1", Name: "write_file", Arguments: origArgs}}},
		{Role: RoleTool, ToolCallID: "w1", Name: "write_file", Content: "wrote a.ts"},
	}
	req := &RunRequest{SoftHorizon: true, Loop: ApplyChatHorizon(DefaultLoop()), ModelWindow: 1_000_000, Tools: &WorkspaceTools{}}
	k := newContextKernel(req)
	out, _, _ := k.prompt(msgs, false)
	if msgs[2].ToolCalls[0].Arguments != origArgs {
		t.Fatal("live messages must keep full write")
	}
	shaped := ""
	for _, m := range out {
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			shaped = m.ToolCalls[0].Arguments
		}
	}
	if shaped != origArgs {
		t.Fatalf("chat prompt must keep the model's own write arguments, got %s", shaped[:min(180, len(shaped))])
	}
}

// TestShapeKeepsWriteArguments 确认 Shape 不改调用方消息，投影里的写操作参数也保持原样。
func TestShapeKeepsWriteArguments(t *testing.T) {
	body := strings.Repeat("x", 4000)
	origArgs := `{"path":"a.ts","content":"` + body + `"}`
	orig := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "w1", Name: "write_file", Arguments: origArgs}}},
		{Role: RoleTool, ToolCallID: "w1", Name: "write_file", Content: "wrote a.ts"},
	}
	out, _ := Shape(orig, ShapeOpts{Loop: DefaultLoop()})
	if orig[2].ToolCalls[0].Arguments != origArgs {
		t.Fatal("live transcript must keep full write")
	}
	if out[2].ToolCalls[0].Arguments != origArgs {
		t.Fatal("shaped prompt must keep the model's own write arguments")
	}
}

// TestForceFitStubsCallArgsBeforeResults 确认强制适配换掉的是工具结果，写操作参数仍在。
func TestForceFitStubsCallArgsBeforeResults(t *testing.T) {
	body := strings.Repeat("y", 500)
	origArgs := make([]string, 6)
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
	}
	for i := 0; i < 6; i++ {
		id := "w" + itoa(i)
		args := `{"path":"f` + itoa(i) + `.ts","content":"` + body + `"}`
		origArgs[i] = args
		msgs = append(msgs,
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: id, Name: "write_file", Arguments: args}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "write_file", Content: strings.Repeat("z", 2000)},
		)
	}
	loop := DefaultLoop()
	loop.CompactionTokens = 2500
	out, rep := Shape(msgs, ShapeOpts{Loop: loop})
	if msgs[2].ToolCalls[0].Arguments != origArgs[0] {
		t.Fatal("live transcript must keep mid-size writes")
	}
	if !strings.Contains(rep.Note, "force") {
		t.Fatalf("force path must stub tool results, layers %s", rep.Note)
	}
	for _, m := range out {
		if m.Role != RoleAssistant {
			continue
		}
		for _, tc := range m.ToolCalls {
			if tc.Name == "write_file" && strings.Count(tc.Arguments, "y") < 500 {
				t.Fatalf("write args were stubbed: %s", tc.Arguments)
			}
		}
	}
	stubbed := 0
	for _, m := range out {
		if m.Role == RoleTool && alreadyStubbed(m.Content) {
			stubbed++
		}
	}
	if stubbed == 0 {
		t.Fatal("force path must stub tool results, not arguments")
	}
}

// TestMicrocompactKeepsLastPathOnChatKeep 确认 keepLast 为 16 时留下该路径最后一次读，为 4 时旧读仍换成占位。
func TestMicrocompactKeepsLastPathOnChatKeep(t *testing.T) {
	aBody := strings.Repeat("A", 800)
	bBody := strings.Repeat("B", 800)
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "a1", Name: "read_file", Arguments: `{"path":"api.go"}`}}},
		{Role: RoleTool, ToolCallID: "a1", Name: "read_file", Content: aBody},
	}
	for i := 0; i < 20; i++ {
		id := "b" + itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: id, Name: "read_file", Arguments: `{"path":"other.go"}`}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: bBody},
		)
	}
	chat := copyMessages(msgs)
	microcompact(chat, 16, nil)
	got := ""
	for _, m := range chat {
		if m.ToolCallID == "a1" {
			got = m.Content
		}
	}
	if alreadyStubbed(got) || !strings.Contains(got, "AAA") {
		t.Fatalf("chat must keep last read of api.go, got %q", capRunes(got, 80))
	}
	harbor := copyMessages(msgs)
	microcompact(harbor, 4, nil)
	got = ""
	for _, m := range harbor {
		if m.ToolCallID == "a1" {
			got = m.Content
		}
	}
	if !alreadyStubbed(got) {
		t.Fatalf("harbor microkeep 4 must still stub the old read")
	}
}
