// client_test.go 锁住 ACP 客户端的协议规则：版本协商、流式回复、取消时答复挂起的权限、未知方法、换行转义、要求登录、续接退化顺序、加载回放标记，以及进程崩溃时带上 stderr 尾部。

package acp_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
)

// TestMain 让测试二进制在子进程里扮演假 agent。
func TestMain(m *testing.M) {
	if acptest.RunIfHelper() {
		return
	}
	os.Exit(m.Run())
}

// collector 收集会话更新。
type collector struct {
	mu      sync.Mutex
	text    strings.Builder
	replays int
	updates []acp.SessionUpdate
}

// update 记下一条更新。
func (c *collector) update(n acp.SessionNotification, replay bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if replay {
		c.replays++
		return
	}
	c.updates = append(c.updates, n.Update)
	if n.Update.Kind == acp.UpdateAgentChunk && n.Update.Chunk != nil {
		c.text.WriteString(n.Update.Chunk.Text)
	}
}

// said 返回收到的助手正文。
func (c *collector) said() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.text.String()
}

// pipeClient 用同进程假 agent 建客户端。
func pipeClient(t *testing.T, script acptest.Script, h acp.Handler) (*acp.Client, *acptest.Agent, error) {
	t.Helper()
	r, w, agent := acptest.Pipe(script)
	c, err := acp.Connect(context.Background(), r, w, acp.Options{Handler: h, Version: "test"})
	if c != nil {
		t.Cleanup(func() { _ = c.Close() })
	}
	return c, agent, err
}

// TestHandshakeRejectsOtherVersion 锁住对方报的协议版本不是 1 时握手失败，错误是 ErrProtocol。
func TestHandshakeRejectsOtherVersion(t *testing.T) {
	_, _, err := pipeClient(t, acptest.Script{Version: 2}, acp.Handler{})
	if !errors.Is(err, acp.ErrProtocol) {
		t.Fatalf("err = %v", err)
	}
}

// TestPromptStreamsChunks 锁住一轮提示里的正文按顺序到达，Prompt 在拿到 stopReason 后返回。
func TestPromptStreamsChunks(t *testing.T) {
	var got collector
	c, _, err := pipeClient(t, acptest.Script{OnPrompt: func(turn *acptest.Turn) acp.StopReason {
		for _, part := range []string{"你", "好", "，", turn.Text()} {
			turn.Say(part)
		}
		return acp.StopEndTurn
	}}, acp.Handler{Update: got.update})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	stop, err := c.Prompt(context.Background(), sess.SessionID, []acp.ContentBlock{acp.TextBlock("世界")})
	if err != nil || stop != acp.StopEndTurn {
		t.Fatalf("stop %q err %v", stop, err)
	}
	if got.said() != "你好，世界" {
		t.Fatalf("said %q", got.said())
	}
}

// TestCancelAnswersPendingPermission 锁住取消这一轮时，挂着的权限请求被答成 cancelled，Prompt 以 cancelled 结束。
func TestCancelAnswersPendingPermission(t *testing.T) {
	asked := make(chan struct{})
	outcome := make(chan acp.PermissionOutcome, 1)
	c, _, err := pipeClient(t, acptest.Script{OnPrompt: func(turn *acptest.Turn) acp.StopReason {
		out, _ := turn.Permission(map[string]any{"toolCallId": "p1", "title": "rm", "kind": "execute"}, acptest.DefaultPermOptions())
		outcome <- out
		turn.WaitCancel(2 * time.Second)
		return acp.StopCancelled
	}}, acp.Handler{Permission: func(ctx context.Context, _ acp.PermissionRequest) (acp.PermissionOutcome, error) {
		close(asked)
		<-ctx.Done()
		return acp.Selected("allow"), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan acp.StopReason, 1)
	go func() {
		stop, _ := c.Prompt(ctx, sess.SessionID, []acp.ContentBlock{acp.TextBlock("删掉")})
		stopped <- stop
	}()
	select {
	case <-asked:
	case <-time.After(3 * time.Second):
		t.Fatal("permission never asked")
	}
	cancel()
	select {
	case out := <-outcome:
		if out.Outcome != "cancelled" {
			t.Fatalf("outcome %+v", out)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("permission never answered")
	}
	select {
	case stop := <-stopped:
		if stop != acp.StopCancelled {
			t.Fatalf("stop %q", stop)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("prompt never returned")
	}
}

// TestUnknownMethodAnswered 锁住客户端不认识的请求回 -32601，没接的文件读写也一样。
func TestUnknownMethodAnswered(t *testing.T) {
	codes := make(chan int, 2)
	c, _, err := pipeClient(t, acptest.Script{OnPrompt: func(turn *acptest.Turn) acp.StopReason {
		for _, method := range []string{"terminal/create", "fs/read_text_file"} {
			err := turn.Call(method, map[string]any{"sessionId": turn.SessionID, "path": "/x"}, nil)
			var rpcErr *acp.RPCError
			if errors.As(err, &rpcErr) {
				codes <- rpcErr.Code
			} else {
				codes <- 0
			}
		}
		return acp.StopEndTurn
	}}, acp.Handler{})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	if _, err := c.Prompt(context.Background(), sess.SessionID, []acp.ContentBlock{acp.TextBlock("x")}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if code := <-codes; code != acp.CodeMethodNotFound {
			t.Fatalf("code %d", code)
		}
	}
}

// TestNewlinesStayEscaped 锁住正文里的换行被转义，一条消息只占一行，往返后原样还原。
func TestNewlinesStayEscaped(t *testing.T) {
	var buf bytes.Buffer
	conn := acp.NewConn(strings.NewReader(""), &buf, acp.ConnOptions{})
	_ = conn.Notify("session/update", map[string]any{"text": "第一行\n第二行\r\n"})
	if strings.Count(buf.String(), "\n") != 1 || !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("wire %q", buf.String())
	}
	var got collector
	c, _, err := pipeClient(t, acptest.Script{}, acp.Handler{Update: got.update})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	if _, err := c.Prompt(context.Background(), sess.SessionID, []acp.ContentBlock{acp.TextBlock("a\nb")}); err != nil {
		t.Fatal(err)
	}
	if got.said() != "echo: a\nb" {
		t.Fatalf("said %q", got.said())
	}
}

// TestAuthRequired 锁住没登录时 session/new 的错误认作 ErrAuthRequired，登录后能建会话。
func TestAuthRequired(t *testing.T) {
	c, _, err := pipeClient(t, acptest.Script{NeedAuth: true, AuthMethods: []acp.AuthMethod{{ID: "login", Name: "Log in"}, {ID: "term", Name: "T", Type: "terminal", Args: []string{"--login"}}}}, acp.Handler{})
	if err != nil {
		t.Fatal(err)
	}
	methods := c.Init().AuthMethods
	if len(methods) != 2 || methods[0].Kind() != "agent" || methods[1].Kind() != "terminal" {
		t.Fatalf("methods %+v", methods)
	}
	if _, err := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()}); !errors.Is(err, acp.ErrAuthRequired) {
		t.Fatalf("err = %v", err)
	}
	if err := c.Authenticate(context.Background(), "login"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
}

// TestReopenFallsBack 锁住续接的退化顺序：能 resume 就 resume，失败退到 load，再失败就新建。
func TestReopenFallsBack(t *testing.T) {
	caps := map[string]any{"loadSession": true, "sessionCapabilities": map[string]any{"resume": map[string]any{}}}
	old := acp.LoadSessionRequest{SessionID: "sess-old", Cwd: t.TempDir()}
	fresh := acp.NewSessionRequest{Cwd: old.Cwd}
	cases := []struct {
		script acptest.Script
		how    string
	}{
		{acptest.Script{Caps: caps}, "resume"},
		{acptest.Script{Caps: caps, ResumeFails: true}, "load"},
		{acptest.Script{Caps: caps, ResumeFails: true, LoadFails: true}, "new"},
		{acptest.Script{Caps: map[string]any{}}, "new"},
	}
	for _, tc := range cases {
		c, _, err := pipeClient(t, tc.script, acp.Handler{})
		if err != nil {
			t.Fatal(err)
		}
		id, how, err := c.Reopen(context.Background(), old, fresh)
		if err != nil || how != tc.how {
			t.Fatalf("want %s got %s %v", tc.how, how, err)
		}
		if (how == "new") == (id == old.SessionID) {
			t.Fatalf("%s returned session %s", how, id)
		}
	}
}

// TestLoadMarksReplay 锁住 session/load 期间的更新标成回放，之后的一轮不是回放。
func TestLoadMarksReplay(t *testing.T) {
	var got collector
	c, _, err := pipeClient(t, acptest.Script{Replay: []map[string]any{
		{"sessionUpdate": acp.UpdateUserChunk, "content": map[string]any{"type": "text", "text": "旧问题"}},
		{"sessionUpdate": acp.UpdateAgentChunk, "content": map[string]any{"type": "text", "text": "旧回答"}},
	}}, acp.Handler{Update: got.update})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.LoadSession(context.Background(), acp.LoadSessionRequest{SessionID: "sess-9", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Prompt(context.Background(), "sess-9", []acp.ContentBlock{acp.TextBlock("新问题")}); err != nil {
		t.Fatal(err)
	}
	got.mu.Lock()
	replays := got.replays
	got.mu.Unlock()
	if replays != 2 || got.said() != "echo: 新问题" {
		t.Fatalf("replays %d said %q", replays, got.said())
	}
}

// TestCrashCarriesStderrTail 锁住 agent 进程中途退出时 Prompt 返回 ExitError，带退出码和打码后的 stderr 尾部。
func TestCrashCarriesStderrTail(t *testing.T) {
	cmd, args, env := acptest.Command("crash", nil)
	c, err := acp.Dial(context.Background(), acp.Options{Command: cmd, Args: args, Env: env, Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	sess, err := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Prompt(context.Background(), sess.SessionID, []acp.ContentBlock{acp.TextBlock("go")})
	var exit *acp.ExitError
	if !errors.As(err, &exit) || exit.Code != 3 || !strings.Contains(exit.Stderr, "boom") || !errors.Is(err, acp.ErrClosed) {
		t.Fatalf("err = %v", err)
	}
	if !strings.HasPrefix(err.Error(), "agent exited: 3") {
		t.Fatalf("message %q", err.Error())
	}
}

// TestDialHelperRoundTrip 锁住子进程版假 agent 能完成握手、读能力和一轮提示，Close 后进程不在了。
func TestDialHelperRoundTrip(t *testing.T) {
	var got collector
	cmd, args, env := acptest.Command("models", map[string]any{"loadSession": true, "mcpCapabilities": map[string]any{"http": true}})
	c, err := acp.Dial(context.Background(), acp.Options{Command: cmd, Args: args, Env: env, Handler: acp.Handler{Update: got.update}})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Caps().LoadSession || !c.Caps().McpCapabilities.HTTP || c.Caps().McpCapabilities.SSE {
		t.Fatalf("caps %+v", c.Caps())
	}
	sess, err := c.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil || sess.Models == nil || len(sess.Models.AvailableModels) != 2 {
		t.Fatalf("session %+v %v", sess, err)
	}
	if _, err := c.Prompt(context.Background(), sess.SessionID, []acp.ContentBlock{acp.TextBlock("hi")}); err != nil {
		t.Fatal(err)
	}
	if got.said() != "echo: hi" {
		t.Fatalf("said %q", got.said())
	}
	_ = c.Close()
	select {
	case <-c.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("connection still open after Close")
	}
}

// TestRingRedactsAndTrims 锁住 stderr 缓冲只留最后的部分，取出时打码。
func TestRingRedactsAndTrims(t *testing.T) {
	r := acp.NewRing(64)
	_, _ = r.Write([]byte(strings.Repeat("x", 100)))
	_, _ = r.Write([]byte("\nkey sk-abcdefghijklmnop leaked\n"))
	tail := r.Tail(200)
	if strings.Contains(tail, "sk-abcdefghijklmnop") || !strings.Contains(tail, "[REDACTED]") || strings.Contains(tail, "\n") {
		t.Fatalf("tail %q", tail)
	}
	if len(r.Text()) > 64+len("[REDACTED] KEY") {
		t.Fatalf("ring kept %d bytes", len(r.Text()))
	}
}
