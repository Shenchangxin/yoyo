package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type prefetchStreamer struct {
	msg Message
}

func (p prefetchStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return p.msg, nil
}

func (p prefetchStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	if emit != nil {
		for _, tc := range p.msg.ToolCalls {
			if err := emit(StreamDelta{Tool: tc, ToolDone: true}); err != nil {
				return Message{}, err
			}
		}
	}
	return p.msg, nil
}

func TestPrefetchReadonlyEmitsOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := trace.NewStore(t.TempDir())
	client := prefetchStreamer{msg: Message{
		Role: RoleAssistant,
		ToolCalls: []ToolCall{{
			ID: "c1", Name: "read_file", Arguments: `{"path":"a.txt"}`,
		}},
	}}
	second := &ScriptedClient{Steps: []Message{{Role: RoleAssistant, Content: "done"}}}
	seq := &seqClient{first: client, rest: second}
	_, err := Run(context.Background(), RunRequest{
		SessionID: "s",
		User:      "read a.txt",
		Workspace: dir,
		Loop:      DefaultLoop(),
		Client:    seq,
		Trace:     st,
		Tools:     &WorkspaceTools{Workspace: dir, Spill: NewSpill(filepath.Join(dir, "spill"))},
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := st.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	calls, results := 0, 0
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeToolCall:
			if id, _ := ev.Payload["id"].(string); id == "c1" {
				calls++
			}
		case trace.TypeToolResult:
			if id, _ := ev.Payload["id"].(string); id == "c1" {
				results++
			}
		}
	}
	if calls != 1 || results != 1 {
		t.Fatalf("calls=%d results=%d", calls, results)
	}
}

type seqClient struct {
	first prefetchStreamer
	rest  *ScriptedClient
	n     int
}

func (s *seqClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	s.n++
	if s.n == 1 {
		return s.first.Chat(ctx, req)
	}
	return s.rest.Chat(ctx, req)
}

func (s *seqClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamDelta) error) (Message, error) {
	s.n++
	if s.n == 1 {
		return s.first.ChatStream(ctx, req, emit)
	}
	return s.rest.Chat(ctx, req)
}

func TestGitNotRepoIsNotError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	res := (&WorkspaceTools{Workspace: dir}).Call("git_status", `{}`)
	if res.Err != nil {
		t.Fatalf("err %v content %q", res.Err, res.Content)
	}
	if strings.HasPrefix(res.Content, "ERROR:") {
		t.Fatalf("content %q", res.Content)
	}
	if !strings.Contains(res.Content, "not a git repository") {
		t.Fatalf("content %q", res.Content)
	}
}

func TestPolicyRequiresDoesNotForceAskDefault(t *testing.T) {
	p := DefaultPolicy()
	if policyRequires(p, capability.Shell) {
		t.Fatal("default policy must not ForceAsk shell")
	}
	p.Mode = "ask"
	if !policyRequires(p, capability.Shell) {
		t.Fatal("ask mode should ForceAsk shell")
	}
	p.Mode = "bypass"
	if policyRequires(p, capability.Shell) {
		t.Fatal("bypass ForceAsk")
	}
}
