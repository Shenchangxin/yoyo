// 锁住投影配对、recall 分页后的尾部、检查点不再回放全文、溢出同一轮恢复，以及连续两次溢出即停。

package runtime

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// pairingOK 检查助手的 tool call 都有非空的 role=tool，且没有插在下一轮助手之前的未配对 id。
func pairingOK(msgs []Message) error {
	pending := map[string]bool{}
	for i, m := range msgs {
		switch m.Role {
		case RoleAssistant:
			if len(pending) > 0 {
				return fmt.Errorf("msg %d: unpaired before next assistant: %v", i, pending)
			}
			for _, tc := range m.ToolCalls {
				if tc.ID == "" {
					return fmt.Errorf("msg %d: empty tool id", i)
				}
				pending[tc.ID] = true
			}
		case RoleTool:
			if !pending[m.ToolCallID] {
				return fmt.Errorf("msg %d: orphan tool %s", i, m.ToolCallID)
			}
			delete(pending, m.ToolCallID)
			if strings.TrimSpace(m.Content) == "" {
				return fmt.Errorf("msg %d: empty tool body %s", i, m.ToolCallID)
			}
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf("unpaired at end: %v", pending)
	}
	return nil
}

// TestC1ShapePairingFuzz 用固定种子随机抄本，确认 Shape 不改条数，Legalize 之后配对完整。
func TestC1ShapePairingFuzz(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	loop := DefaultLoop()
	loop.CompactionKeep = 6
	loop.CompactionTokens = 900
	for i := 0; i < 1000; i++ {
		msgs := randomTranscript(rng)
		orig := copyMessages(msgs)
		out, _ := Shape(msgs, ShapeOpts{Loop: loop})
		if len(msgs) != len(orig) {
			t.Fatalf("i=%d mutated length", i)
		}
		out = Legalize(out)
		if err := pairingOK(out); err != nil {
			t.Fatalf("i=%d %v\n%+v", i, err, out)
		}
	}
}

// TestIllegalTranscriptFixtures 确认缺结果、空正文、孤立 tool、配对中插入用户，以及空 id，经过 Legalize 后都能配对。
func TestIllegalTranscriptFixtures(t *testing.T) {
	fixtures := [][]Message{
		{
			{Role: RoleSystem, Content: "s"},
			{Role: RoleUser, Content: "u"},
			{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "a", Name: "read_file"}, {ID: "b", Name: "grep"}}},
			{Role: RoleTool, ToolCallID: "a", Name: "read_file", Content: "one"},
		},
		{
			{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "z", Name: "shell"}}},
			{Role: RoleTool, ToolCallID: "z", Name: "shell", Content: ""},
		},
		{
			{Role: RoleTool, ToolCallID: "orphan", Name: "grep", Content: "lost"},
			{Role: RoleUser, Content: "next"},
		},
		{
			{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "a", Name: "read_file"}, {ID: "b", Name: "grep"}}},
			{Role: RoleTool, ToolCallID: "a", Name: "read_file", Content: "one"},
			{Role: RoleUser, Content: "interrupt"},
			{Role: RoleTool, ToolCallID: "b", Name: "grep", Content: "two"},
		},
		{
			{Role: RoleAssistant, ToolCalls: []ToolCall{{Name: "read_file", Arguments: "{}"}}},
		},
	}
	for i, fx := range fixtures {
		out := Legalize(copyMessages(fx))
		if err := pairingOK(out); err != nil {
			t.Fatalf("fixture %d: %v\n%+v", i, err, out)
		}
	}
}

// randomTranscript 生成带随机缺结果和孤立 tool 的抄本，供配对模糊测试使用。
func randomTranscript(rng *rand.Rand) []Message {
	n := 4 + rng.Intn(18)
	msgs := []Message{{Role: RoleSystem, Content: "sys"}}
	id := 0
	for i := 0; i < n; i++ {
		switch rng.Intn(5) {
		case 0:
			msgs = append(msgs, Message{Role: RoleUser, Content: "u"})
		case 1:
			msgs = append(msgs, Message{Role: RoleAssistant, Content: "a"})
		case 2:
			k := 1 + rng.Intn(3)
			calls := make([]ToolCall, 0, k)
			for j := 0; j < k; j++ {
				id++
				calls = append(calls, ToolCall{ID: fmt.Sprintf("c%d", id), Name: "read_file", Arguments: "{}"})
			}
			msgs = append(msgs, Message{Role: RoleAssistant, ToolCalls: calls})
			for _, tc := range calls {
				if rng.Intn(4) == 0 {
					continue
				}
				body := "ok"
				if rng.Intn(5) == 0 {
					body = strings.Repeat("x", rng.Intn(1200)+8)
				}
				if rng.Intn(8) == 0 {
					body = ""
				}
				msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Name, Content: body})
			}
		case 3:
			id++
			msgs = append(msgs, Message{Role: RoleTool, ToolCallID: fmt.Sprintf("orphan%d", id), Name: "grep", Content: "orphan"})
		default:
			msgs = append(msgs, Message{Role: RoleUser, Content: strings.Repeat("t", rng.Intn(40)+1)})
		}
	}
	return msgs
}

// TestC3CompactDropsPreCheckpointBodies 确认检查点重建不再带回检查点前的整段工具正文，尾部用户句还在。
func TestC3CompactDropsPreCheckpointBodies(t *testing.T) {
	dir := t.TempDir()
	st := trace.NewStore(dir)
	sp := NewSpill(filepath.Join(dir, "spill"))
	fat := strings.Repeat("SECRET_BODY ", 40_000)
	_ = st.Append(trace.Event{Type: trace.TypeUser, SessionID: "s", Payload: map[string]any{"text": "old"}})
	_ = st.Append(trace.Event{Type: trace.TypeAssistant, SessionID: "s", Payload: map[string]any{"text": ""}})
	_ = st.Append(trace.Event{Type: trace.TypeToolCall, SessionID: "s", Payload: map[string]any{"id": "t1", "name": "shell", "arguments": "{}"}})
	_ = st.Append(trace.Event{Type: trace.TypeToolResult, SessionID: "s", Payload: map[string]any{"id": "t1", "name": "shell", "content": fat}})
	_ = st.Append(trace.Event{Type: trace.TypeUser, SessionID: "s", Payload: map[string]any{"text": "keep going"}})
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	loop := DefaultLoop()
	loop.AllowLLMCompact = false
	loop.CompactionKeep = 4
	loop.CompactionTokens = 1_200
	_ = CompactHistoryOpts(st, "s", MessagesFromEvents(evs), loop, sp, nil, "", 0, nil, CompactOpts{Trigger: "user"})
	evs, err = st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, m := range MessagesFromEvents(evs) {
		joined += m.Content
	}
	if strings.Contains(joined, fat) {
		t.Fatal("pre-checkpoint tool body replayed in full")
	}
	if !strings.Contains(joined, "keep going") {
		t.Fatalf("lost tail: %s", joined[:min(200, len(joined))])
	}
}

// TestC6SchemaCapDropsTokensAndWritesCatalog 确认未启用的额外工具不进 tools 数组，schema token 更少，并写出 MCP 索引。
func TestC6SchemaCapDropsTokensAndWritesCatalog(t *testing.T) {
	ws := t.TempDir()
	extra := map[string]ExtraTool{}
	var names []string
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("mcp__s__tool%02d", i)
		names = append(names, name)
		extra[name] = ExtraTool{JSON: fn(name, strings.Repeat("desc", 80), map[string]any{"type": "object"})}
	}
	capped := AllToolJSON(&WorkspaceTools{Workspace: ws, Extra: extra})
	enabled := &WorkspaceTools{Workspace: ws, Extra: extra, ExtraEnabled: names}
	all := AllToolJSON(enabled)
	if toolsJSONTokens(capped) >= toolsJSONTokens(all) {
		t.Fatalf("schemaCap did not drop tokens: %d vs %d", toolsJSONTokens(capped), toolsJSONTokens(all))
	}
	mcp := 0
	for _, j := range capped {
		name, _ := j.Function["name"].(string)
		if strings.HasPrefix(name, "mcp__") {
			mcp++
		}
	}
	if mcp != 0 {
		t.Fatalf("advertised %d extras", mcp)
	}
	res := (&WorkspaceTools{Workspace: ws, Extra: extra}).Call("tool_search", `{"query":"tool19"}`)
	if res.Err != nil || !strings.Contains(res.Content, "mcp__s__tool19") {
		t.Fatalf("%+v", res)
	}
	if _, err := os.Stat(filepath.Join(ws, ".yoyo", "mcp", "INDEX.md")); err != nil {
		t.Fatal(err)
	}
}

// TestC7LedgerLabelsWindowSeparately 确认 Window 和 Budget 分开填，预算不会被写成模型窗口。
func TestC7LedgerLabelsWindowSeparately(t *testing.T) {
	rep := ShapeReport{Budget: 24_000}
	fillLedger(&rep, "prefix-bytes", "dyn", nil, 1_000_000)
	if rep.Window != 1_000_000 {
		t.Fatalf("window %d", rep.Window)
	}
	if rep.Budget == rep.Window {
		t.Fatal("budget must not masquerade as the model window")
	}
	if rep.PrefixTokens <= 0 {
		t.Fatal("prefix tokens")
	}
}

// TestIsContextOverflowDoesNotMatchParamName 确认单独的 max_tokens 不算溢出，context_length_exceeded 算。
func TestIsContextOverflowDoesNotMatchParamName(t *testing.T) {
	if IsContextOverflow(fmt.Errorf("invalid max_tokens")) {
		t.Fatal("bare max_tokens should not trip overflow recovery")
	}
	if !IsContextOverflow(fmt.Errorf("openai: 400 context_length_exceeded")) {
		t.Fatal("expected overflow")
	}
}

// TestPromptCacheKeyOnPayload 确认请求体和 X-Prompt-Cache-Key 都带上 CacheKey。
func TestPromptCacheKeyOnPayload(t *testing.T) {
	var gotBody, gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		gotHeader = r.Header.Get("X-Prompt-Cache-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()
	c := NewOpenAIClient(srv.URL, "k")
	_, err := c.Chat(context.Background(), ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
		CacheKey: "sess-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"prompt_cache_key":"sess-1"`) {
		t.Fatalf("body %s", gotBody)
	}
	if gotHeader != "sess-1" {
		t.Fatalf("header %q", gotHeader)
	}
}

// TestBindSpillMirrorsIntoWorkspace 确认 BindSpill 把正文镜像到工作区 spill，发现索引列出该 id。
func TestBindSpillMirrorsIntoWorkspace(t *testing.T) {
	home := t.TempDir()
	ws := t.TempDir()
	sp := BindSpill(home, ws, "sid")
	sp.Put("dump", "GREPPABLE_SECRET")
	b, err := os.ReadFile(filepath.Join(ws, ".yoyo", "context", "sid", "spill", "dump.txt"))
	if err != nil || string(b) != "GREPPABLE_SECRET" {
		t.Fatalf("%q %v", b, err)
	}
	WriteDiscoverIndex(ws, "sid", sp)
	idx, err := os.ReadFile(filepath.Join(ws, ".yoyo", "context", "sid", "INDEX.md"))
	if err != nil || !strings.Contains(string(idx), "dump") {
		t.Fatalf("%s %v", idx, err)
	}
}
