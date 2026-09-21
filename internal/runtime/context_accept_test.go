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

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

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

func TestC2RecallKeepsTailOfHugeDump(t *testing.T) {
	dir := t.TempDir()
	sp := NewSpill(filepath.Join(dir, "spill"))
	tools := &WorkspaceTools{Workspace: dir, Spill: sp}
	full := strings.Repeat("HEAD", 2000) + strings.Repeat("x", 2<<20) + "UNIQUE_TAIL_MARKER"
	preview, _ := ingestToolResult(sp, "big", "shell", full, 0, false)
	if !strings.Contains(preview, "elided") {
		t.Fatalf("expected preview stub")
	}
	res := tools.Call("recall_context", `{"id":"big"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "UNIQUE_TAIL_MARKER") {
		t.Fatalf("recall lost tail, len=%d", len(res.Content))
	}
	st := trace.NewStore(t.TempDir())
	req := RunRequest{SessionID: "s", Tools: tools, Trace: st, Loop: DefaultLoop()}
	out := dispatchTools(context.Background(), req, []ToolCall{{
		ID: "r1", Name: "recall_context", Arguments: `{"id":"big"}`,
	}}, "s:r1", true)
	if len(out) != 1 {
		t.Fatalf("dispatch %d", len(out))
	}
	if strings.HasPrefix(out[0].Content, "[elided") {
		t.Fatalf("recall restubbed: %s", out[0].Content[:min(120, len(out[0].Content))])
	}
	if !strings.Contains(out[0].Content, "UNIQUE_TAIL_MARKER") {
		t.Fatalf("dispatch recall lost tail, len=%d", len(out[0].Content))
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	var jsonl string
	for _, ev := range evs {
		if ev.Type != trace.TypeToolResult {
			continue
		}
		jsonl, _ = ev.Payload["content"].(string)
	}
	if jsonl != out[0].Content {
		t.Fatalf("jsonl diverged from live")
	}
}

func TestIngestStubsOnlyOverBudget(t *testing.T) {
	sp := NewSpill(t.TempDir())
	small := strings.Repeat("a", 400)
	got, _ := ingestToolResult(sp, "s", "read_file", small, 8000, false)
	if got != small || strings.Contains(got, "elided") {
		t.Fatalf("small result stubbed: %q", got[:min(80, len(got))])
	}
	big := strings.Repeat("b", 12_000) + "TAIL_MARK"
	preview, _ := ingestToolResult(sp, "b", "read_file", big, 8000, false)
	if !strings.HasPrefix(preview, "[elided") {
		t.Fatalf("expected stub, got %s", preview[:min(80, len(preview))])
	}
	if !strings.Contains(preview, "TAIL_MARK") {
		t.Fatal("over-budget preview lost tail")
	}
}

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
	_ = CompactHistory(st, "s", MessagesFromEvents(evs), loop, sp, nil, "", 0, nil)
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

func TestC8PlaybookStableAcrossCheckpoint(t *testing.T) {
	pb := artifact.Playbook{ID: "p", Bullets: []artifact.PlaybookBullet{
		{ID: "b1", Text: "prefer grep", Helpful: 3},
		{ID: "b2", Text: "small diffs", Helpful: 1},
	}}
	loop := DefaultLoop()
	before := AssemblePrefix(loop, nil, pb, nil, "tabs", "YOYO.md")
	st := trace.NewStore(t.TempDir())
	hist := []Message{{Role: RoleUser, Content: "old"}, {Role: RoleAssistant, Content: "ok"}}
	_ = CompactHistory(st, "s", hist, loop, nil, nil, "", 0, nil)
	after := AssemblePrefix(loop, nil, pb, nil, "tabs", "YOYO.md")
	if before != after {
		t.Fatal("playbook prefix mutated by checkpoint")
	}
	if !strings.Contains(before, "prefer grep") {
		t.Fatal(before)
	}
}

type overflowThenOK struct {
	n    int
	last ChatRequest
}

func (o *overflowThenOK) Chat(_ context.Context, req ChatRequest) (Message, error) {
	o.last = req
	o.n++
	if o.n == 1 {
		return Message{}, fmt.Errorf("openai: 400: context_length_exceeded")
	}
	return Message{Role: RoleAssistant, Content: "recovered"}, nil
}

type alwaysOverflow struct{ n int }

func (a *alwaysOverflow) Chat(_ context.Context, _ ChatRequest) (Message, error) {
	a.n++
	return Message{}, fmt.Errorf("context_length_exceeded")
}

func TestSecondTurnKeepsBothUsersAndSkipsEmptyShape(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	c := &recordingClient{}
	ws := t.TempDir()
	loop := DefaultLoop()
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s", User: "first", Workspace: ws, Loop: loop, Client: c, Trace: st,
		Tools: &WorkspaceTools{Workspace: ws, Depth: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Run(context.Background(), RunRequest{
		SessionID: "s", User: "second", Workspace: ws, Loop: loop, Client: c, Trace: st,
		History: MessagesFromEvents(evs),
		Tools:   &WorkspaceTools{Workspace: ws, Depth: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	var users []string
	for _, m := range c.last.Messages {
		if m.Role == RoleUser {
			users = append(users, m.Content)
		}
	}
	if len(users) < 2 || users[len(users)-1] != "second" {
		t.Fatalf("users %+v", users)
	}
	nFirst := 0
	for _, u := range users {
		if u == "first" {
			nFirst++
		}
	}
	if nFirst != 1 {
		t.Fatalf("first user count %d in %+v", nFirst, users)
	}
	evs, _ = st.Read("s")
	emptyShape := 0
	for _, ev := range evs {
		if ev.Type != trace.TypeCompact {
			continue
		}
		note, _ := ev.Payload["note"].(string)
		kind, _ := ev.Payload["kind"].(string)
		if kind == "shape" && note == "" {
			emptyShape++
		}
	}
	if emptyShape != 0 {
		t.Fatalf("empty shape events=%d", emptyShape)
	}
	rounds := map[string]int{}
	usersJSONL := 0
	for _, ev := range evs {
		if ev.Type == trace.TypeUser {
			usersJSONL++
		}
		if ev.Type != trace.TypeAssistant {
			continue
		}
		id, _ := ev.Payload["id"].(string)
		if id == "" {
			t.Fatalf("assistant missing round id: %+v", ev.Payload)
		}
		rounds[id]++
	}
	if usersJSONL != 2 {
		t.Fatalf("jsonl users=%d", usersJSONL)
	}
	if len(rounds) != 2 {
		t.Fatalf("assistant rounds=%v", rounds)
	}
}

func TestResumeContinuesWithoutNewUser(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	c := &recordingClient{}
	ws := t.TempDir()
	loop := DefaultLoop()
	if _, err := Run(context.Background(), RunRequest{
		SessionID: "s", User: "first", Workspace: ws, Loop: loop, Client: c, Trace: st,
		Tools: &WorkspaceTools{Workspace: ws, Depth: 1},
	}); err != nil {
		t.Fatal(err)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), RunRequest{
		SessionID: "s", User: "", Workspace: ws, Loop: loop, Client: c, Trace: st,
		History: MessagesFromEvents(evs),
		Tools:   &WorkspaceTools{Workspace: ws, Depth: 1},
	}); err != nil {
		t.Fatal(err)
	}
	var users []string
	for _, m := range c.last.Messages {
		if m.Role == RoleUser {
			users = append(users, m.Content)
		}
	}
	if len(users) != 1 || users[0] != "first" {
		t.Fatalf("resume users %+v", users)
	}
	evs, _ = st.Read("s")
	usersJSONL := 0
	for _, ev := range evs {
		if ev.Type == trace.TypeUser {
			usersJSONL++
		}
	}
	if usersJSONL != 1 {
		t.Fatalf("jsonl users=%d", usersJSONL)
	}
}

func TestResumeEmptyHistoryErrors(t *testing.T) {
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s", User: "", Workspace: t.TempDir(), Loop: DefaultLoop(),
		Client: &recordingClient{},
	})
	if err == nil || !strings.Contains(err.Error(), "nothing to continue") {
		t.Fatalf("got %v", err)
	}
}

func TestLiveEventsStampTSAndUniqueUserIDs(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	c := &recordingClient{}
	ws := t.TempDir()
	loop := DefaultLoop()
	var live []trace.Event
	on := func(ev trace.Event) { live = append(live, ev) }
	if _, err := Run(context.Background(), RunRequest{
		SessionID: "s", User: "ok", Workspace: ws, Loop: loop, Client: c, Trace: st, OnEvent: on,
		Tools: &WorkspaceTools{Workspace: ws, Depth: 1},
	}); err != nil {
		t.Fatal(err)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), RunRequest{
		SessionID: "s", User: "ok", Workspace: ws, Loop: loop, Client: c, Trace: st, OnEvent: on,
		History:  MessagesFromEvents(evs),
		RoundSeq: MaxAssistantRound(evs),
		Tools:    &WorkspaceTools{Workspace: ws, Depth: 1},
	}); err != nil {
		t.Fatal(err)
	}
	var users []trace.Event
	for _, ev := range live {
		if ev.TS.IsZero() {
			t.Fatalf("zero ts type=%s", ev.Type)
		}
		if ev.Type == trace.TypeUser {
			users = append(users, ev)
		}
	}
	if len(users) != 2 {
		t.Fatalf("live users=%d", len(users))
	}
	id0, _ := users[0].Payload["id"].(string)
	id1, _ := users[1].Payload["id"].(string)
	if id0 == "" || id1 == "" || id0 == id1 {
		t.Fatalf("user ids %q %q", id0, id1)
	}
	rounds := map[string]int{}
	for _, ev := range live {
		if ev.Type != trace.TypeAssistant {
			continue
		}
		id, _ := ev.Payload["id"].(string)
		if id != "" {
			rounds[id]++
		}
	}
	if len(rounds) != 2 {
		t.Fatalf("assistant rounds=%v", rounds)
	}
}

type recordingClient struct {
	last ChatRequest
}

func (c *recordingClient) Chat(_ context.Context, req ChatRequest) (Message, error) {
	c.last = req
	return Message{Role: RoleAssistant, Content: "ok"}, nil
}

func TestC9OverflowRecoversSameTurn(t *testing.T) {
	dir := t.TempDir()
	st := trace.NewStore(dir)
	client := &overflowThenOK{}
	out, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      "hello",
		Workspace: t.TempDir(),
		Loop:      DefaultLoop(),
		Client:    client,
		Trace:     st,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "recovered" {
		t.Fatalf("%q", out)
	}
	if client.last.CacheKey == "" || client.last.CacheKey == "s" {
		t.Fatalf("cache key %q", client.last.CacheKey)
	}
	evs, _ := st.Read("s")
	saw := false
	for _, ev := range evs {
		if ev.Type == trace.TypeCompact {
			if kind, _ := ev.Payload["kind"].(string); kind == "checkpoint" {
				saw = true
			}
		}
	}
	if !saw {
		t.Fatal("overflow did not persist a checkpoint")
	}
}

func TestOverflowCircuitBreaker(t *testing.T) {
	client := &alwaysOverflow{}
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      "hello",
		Loop:      DefaultLoop(),
		Client:    client,
	})
	if err == nil || !strings.Contains(err.Error(), "circuit breaker") {
		t.Fatalf("err=%v", err)
	}
	if client.n != 2 {
		t.Fatalf("attempts %d", client.n)
	}
}

func TestIsContextOverflowDoesNotMatchParamName(t *testing.T) {
	if IsContextOverflow(fmt.Errorf("invalid max_tokens")) {
		t.Fatal("bare max_tokens should not trip overflow recovery")
	}
	if !IsContextOverflow(fmt.Errorf("openai: 400 context_length_exceeded")) {
		t.Fatal("expected overflow")
	}
}

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

func TestCompactThenResumeDoesNotArmSensitiveSolver(t *testing.T) {
	st := trace.NewStore(t.TempDir())
	sp := NewSpill(t.TempDir())
	fat := strings.Repeat("blob ", 20_000)
	hist := []Message{
		{Role: RoleUser, Content: "old"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}},
		{Role: RoleTool, ToolCallID: "t1", Name: "read_file", Content: fat},
	}
	loop := DefaultLoop()
	loop.AllowLLMCompact = false
	_ = CompactHistory(st, "s", hist, loop, sp, nil, "", 0, nil)
	evs, _ := st.Read("s")
	rebuilt := MessagesFromEvents(evs)
	work := t.TempDir()
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      "Write hello.txt containing hello",
		Workspace: work,
		History:   rebuilt,
		Loop:      loop,
		Tools:     &WorkspaceTools{Workspace: work, Depth: 1},
		Client:    PromptSensitiveSolver{},
		Trace:     st,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(work, "hello.txt")); err == nil {
		t.Fatal("PromptSensitiveSolver wrote without placeholder pin")
	}
	work2 := t.TempDir()
	_, err = Run(context.Background(), RunRequest{
		SessionID: "s2",
		User:      "Write hello.txt containing hello",
		Workspace: work2,
		History:   rebuilt,
		Loop:      loop,
		Tools:     &WorkspaceTools{Workspace: work2, Depth: 1},
		Client:    HeuristicSolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(work2, "hello.txt"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("%q %v", b, err)
	}
}
