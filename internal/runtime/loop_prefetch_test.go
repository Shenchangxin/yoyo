// 锁住只读预取只落一次轨迹，以及 git_status 在非仓库目录不算错误。
// 也锁住默认策略不强制询问 shell，ask 模式会问，bypass 不问。

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

// prefetchStreamer 把消息里的工具调用按块结束推给流，用来触发只读预取。
type prefetchStreamer struct {
	msg Message
}

// Chat 直接返回预设消息。
func (p prefetchStreamer) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	return p.msg, nil
}

// ChatStream 对每个工具调用发出块开始、参数增量和块结束，然后返回同一条消息。
func (p prefetchStreamer) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	if emit != nil {
		for i, tc := range p.msg.ToolCalls {
			_ = emit(StreamChunk{Type: ChunkBlockStart, Index: i, BlockType: BlockToolCall})
			_ = emit(StreamChunk{Type: ChunkToolCallDelta, Index: i, ID: tc.ID, Name: tc.Name, ArgumentsDelta: tc.Arguments})
			if err := emit(StreamChunk{Type: ChunkBlockEnd, Index: i, Block: &ContentBlock{Type: BlockToolCall, ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}); err != nil {
				return Message{}, err
			}
		}
	}
	return p.msg, nil
}

// TestPrefetchReadonlyEmitsOnce 确认 read_file 预取之后，轨迹里这个 id 的工具调用和结果各只有一条。
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

// seqClient 第一次走 prefetchStreamer，之后走 ScriptedClient。
type seqClient struct {
	first prefetchStreamer
	rest  *ScriptedClient
	n     int
}

// Chat 第一次转发到预取流，之后转发到脚本客户端。
func (s *seqClient) Chat(ctx context.Context, req ChatRequest) (Message, error) {
	s.n++
	if s.n == 1 {
		return s.first.Chat(ctx, req)
	}
	return s.rest.Chat(ctx, req)
}

// ChatStream 第一次走预取流，之后改用脚本客户端的 Chat。
func (s *seqClient) ChatStream(ctx context.Context, req ChatRequest, emit func(StreamChunk) error) (Message, error) {
	s.n++
	if s.n == 1 {
		return s.first.ChatStream(ctx, req, emit)
	}
	return s.rest.Chat(ctx, req)
}

// TestGitNotRepoIsNotError 确认没有 git 仓库时 git_status 不返回错误，正文写明 not a git repository。
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

// TestPolicyRequiresDoesNotForceAskDefault 确认默认策略不强制询问 shell，ask 模式会问，bypass 不问。
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
