package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

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

func TestSpillBeforeCapIsLossless(t *testing.T) {
	dir := t.TempDir()
	sp := NewSpill(filepath.Join(dir, "spill"))
	full := strings.Repeat("TAIL", 20_000)
	preview, _ := ingestToolResult(sp, "big", "shell", full)
	if !strings.Contains(preview, "elided") {
		t.Fatalf("expected preview stub, got %s", preview[:min(120, len(preview))])
	}
	got, err := sp.Get("big")
	if err != nil || got != full {
		t.Fatalf("spill lost bytes: err=%v len=%d", err, len(got))
	}
}

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
