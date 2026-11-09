// 外部 agent 当工人的端到端测试：同进程的假 agent 走真的执行器、控制面、Broker 和续接。

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
	"github.com/Shenchangxin/yoyo/internal/acpworker"
	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/vault"
)

// fakeAgents 每次连接起一个同进程的假 agent。探测、健康检查和工人都连到这里，不会启动真进程。
type fakeAgents struct {
	script acptest.Script
	mu     sync.Mutex
	agents []*acptest.Agent
}

func (f *fakeAgents) dial(ctx context.Context, opts acp.Options) (*acp.Client, error) {
	r, w, ag := acptest.Pipe(f.script)
	f.mu.Lock()
	f.agents = append(f.agents, ag)
	f.mu.Unlock()
	return acp.Connect(ctx, r, w, opts)
}

// prompted 返回收到过 session/prompt 的假 agent。
func (f *fakeAgents) prompted() []*acptest.Agent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*acptest.Agent
	for _, ag := range f.agents {
		if slices.Contains(ag.Calls(), "session/prompt") {
			out = append(out, ag)
		}
	}
	return out
}

// notes 收集假 agent 在回合里看到的结果，回合跑在别的 goroutine 里。
type notes struct {
	mu    sync.Mutex
	items []string
}

func (n *notes) add(format string, args ...any) {
	n.mu.Lock()
	n.items = append(n.items, fmt.Sprintf(format, args...))
	n.mu.Unlock()
}

func (n *notes) all() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.items)
}

// openBuddyApp 在 root 上打开应用。外部 agent 都连 f，密钥只进临时目录。
func openBuddyApp(t *testing.T, root string, f *fakeAgents) *App {
	t.Helper()
	a, err := Open(root, filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	a.Vault = vault.NewFileOnly(filepath.Join(root, "test-vault"))
	a.buddyDial = f.dial
	return a
}

// fakeCommand 造一个找得到、但跑不起来的命令文件。连接都走假 agent，它永远不会被执行。
func fakeCommand(t *testing.T) string {
	t.Helper()
	name := "fake-agent"
	if goruntime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("not a program"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// probeOnline 立刻探测 id，等它在线。
func probeOnline(t *testing.T, m *buddy.Manager, id string) {
	t.Helper()
	m.Prepare(id)
	waitFor(t, id+" online", func() bool {
		if m.Online(id) {
			return true
		}
		_, _ = m.Probe(context.Background(), id)
		return m.Online(id)
	})
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ticketFor 找 trigger 开出的工单。
func ticketFor(rt *superagent.Runtime, trigger string) (superagent.TicketRow, bool) {
	for _, row := range rt.View().Tickets {
		if row.Origin == trigger {
			return row, true
		}
	}
	return superagent.TicketRow{}, false
}

// reportCount 数工单上 state 为 want 的报告。
func reportCount(row superagent.TicketRow, want string) int {
	n := 0
	for _, asg := range row.Assignments {
		for _, line := range asg.Lines {
			if line.State == want {
				n++
			}
		}
	}
	return n
}

// callControl 用会话里交来的 HTTP 控制面调 name，返回结果正文和是否出错。
func callControl(s acptest.Session, name string, args map[string]any) (string, bool, error) {
	i := slices.IndexFunc(s.McpServers, func(srv acp.McpServer) bool { return srv.Type == "http" })
	if i < 0 {
		return "", false, errors.New("no http control plane")
	}
	srv := s.McpServers[i]
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": args},
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	for _, h := range srv.Headers {
		req.Header.Set(h.Name, h.Value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	var out struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false, fmt.Errorf("status %d: %w", resp.StatusCode, err)
	}
	if out.Error != nil {
		return "", false, errors.New(out.Error.Message)
	}
	var text []string
	for _, c := range out.Result.Content {
		text = append(text, c.Text)
	}
	return strings.Join(text, "\n"), out.Result.IsError, nil
}

// TestBuddyApproverMapsKinds 锁住外部 agent 的权限请求按种类换成 Broker 检查：orchestrator_* 和 think 直接放行；
// 写只能落在工单目录，只读目录不能写；危险命令先被确定性名单挡下；切换模式总要问人；「每次问我」时其余请求都问人。
func TestBuddyApproverMapsKinds(t *testing.T) {
	dir := t.TempDir()
	shared := t.TempDir()
	var asked []capability.Request
	a := &App{Caps: capability.NewBroker(capability.AutoPolicy{}, func(_ context.Context, req capability.Request) (capability.Decision, error) {
		asked = append(asked, req)
		return capability.Deny, nil
	})}
	a.Caps.ApplyAuthMode("ticket-T1", capability.AuthFull)
	spec := superagent.WorkerSpec{TicketID: "T1", Dir: dir, ReadRoots: []string{shared}}
	ctx := context.Background()
	approve := a.buddyApprover(spec, "yoyo")
	for _, c := range []struct {
		name string
		req  acpworker.Approval
		want bool
	}{
		{"control tool", acpworker.Approval{Kind: "other", Name: "orchestrator_report"}, true},
		{"think", acpworker.Approval{Kind: "think"}, true},
		{"edit inside", acpworker.Approval{Kind: "edit", Paths: []string{filepath.Join(dir, "a.go")}}, true},
		{"edit outside", acpworker.Approval{Kind: "edit", Paths: []string{filepath.Join(t.TempDir(), "x")}}, false},
		{"edit read-only root", acpworker.Approval{Kind: "edit", Paths: []string{filepath.Join(shared, "x")}}, false},
		{"read read-only root", acpworker.Approval{Kind: "read", Paths: []string{filepath.Join(shared, "x")}}, true},
		{"destructive shell", acpworker.Approval{Kind: "execute", Command: "rm -rf /"}, false},
		{"plain shell", acpworker.Approval{Kind: "execute", Command: "go test ./..."}, true},
		{"shell without command", acpworker.Approval{Kind: "execute"}, false},
		{"fetch", acpworker.Approval{Kind: "fetch", Input: json.RawMessage(`{"url":"https://example.com"}`)}, true},
		{"switch mode", acpworker.Approval{Kind: "switch_mode", Title: "bypassPermissions"}, false},
		{"other kind", acpworker.Approval{Kind: "other", Name: "custom"}, true},
	} {
		ok, err := approve(ctx, c.req)
		if ok != c.want || ok != (err == nil) {
			t.Errorf("%s: ok=%v err=%v, want %v", c.name, ok, err, c.want)
		}
	}
	if len(asked) != 1 || asked[0].Level != capability.HighRisk || !asked[0].ForceAsk {
		t.Fatalf("only switching modes should ask, asked %+v", asked)
	}

	asked = nil
	ask := a.buddyApprover(spec, "ask")
	if ok, _ := ask(ctx, acpworker.Approval{Kind: "think"}); !ok {
		t.Fatal("think needs no person")
	}
	if ok, _ := ask(ctx, acpworker.Approval{Kind: "edit", Paths: []string{filepath.Join(dir, "a.go")}}); ok {
		t.Fatal("ask mode approved without a person")
	}
	if len(asked) != 1 || !asked[0].ForceAsk || asked[0].Action != "buddy.edit" || asked[0].SessionID != "ticket-T1" {
		t.Fatalf("ask mode asked %+v", asked)
	}

	check := a.buddyWriteCheck(spec)
	if err := check(ctx, filepath.Join(dir, "b.txt")); err != nil {
		t.Fatal(err)
	}
	if err := check(ctx, filepath.Join(shared, "b.txt")); err == nil {
		t.Fatal("fs write outside the ticket dir passed")
	}
	if ok, err := (&App{}).buddyApprover(spec, "yoyo")(ctx, acpworker.Approval{Kind: "edit", Paths: []string{filepath.Join(dir, "a.go")}}); ok || !errors.Is(err, errNoBroker) {
		t.Fatalf("no broker: %v %v", ok, err)
	}
}

// fakeCaps 声明能加载会话、能连 HTTP 的 MCP 服务器。
var fakeCaps = map[string]any{"loadSession": true, "mcpCapabilities": map[string]any{"http": true}}

// TestBuddyWorkerRunsTicketEndToEnd 锁住外部 agent 接一张点名的工单：第一轮崩溃后重连并加载原会话；
// 写进工单目录的文件落盘，目录外的写入和越界的权限请求被拒，危险命令被确定性名单挡下；
// 它经控制面报告完成；应用重启后，同一张工单的后续工作加载原会话，不新开。
func TestBuddyWorkerRunsTicketEndToEnd(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "escape.txt")
	seen := &notes{}
	var prompts sync.Map
	first := &fakeAgents{script: acptest.Script{Caps: fakeCaps, OnPrompt: func(turn *acptest.Turn) acp.StopReason {
		if _, crashed := prompts.LoadOrStore("crashed", true); !crashed {
			turn.Crash(1, "boom")
			return acp.StopEndTurn
		}
		dir := turn.Session.Cwd
		inside := filepath.Join(dir, "hello.txt")
		seen.add("write inside: %v", turn.WriteFile(inside, "hi from fake"))
		seen.add("write outside refused: %v", turn.WriteFile(outside, "x") != nil)
		for _, call := range []map[string]any{
			{"toolCallId": "p1", "title": "Edit escape", "kind": "edit", "locations": []any{map[string]any{"path": outside}}},
			{"toolCallId": "p2", "title": "rm -rf /", "kind": "execute", "rawInput": map[string]any{"command": "rm -rf /"}},
			{"toolCallId": "p3", "title": "Edit hello", "kind": "edit", "locations": []any{map[string]any{"path": inside}}},
		} {
			out, err := turn.Permission(call, acptest.DefaultPermOptions())
			seen.add("%s: %s %s %v", call["toolCallId"], out.Outcome, out.OptionID, err)
		}
		text, isErr, err := callControl(turn.Session, "orchestrator_report", map[string]any{"state": "ready", "summary": "写好了 hello.txt"})
		seen.add("report: error=%v %v %s", isErr, err, text)
		turn.Say("写好了。")
		return acp.StopEndTurn
	}}}
	a := openBuddyApp(t, root, first)
	closed := false
	defer func() {
		if !closed {
			_ = a.Close()
		}
	}()
	m := a.Buddies()
	if _, err := m.Add(buddy.CustomAgent{ID: "fake", Name: "Fake", Command: fakeCommand(t)}); err != nil {
		t.Fatal(err)
	}
	probeOnline(t, m, "fake")
	rt := a.orchestrator()
	trigger := "desk:s1:t1"
	if status, _ := rt.Receive(superagent.Trigger{
		ID: trigger, Source: "desk", Threads: []string{"desk:s1"}, Primary: "desk:s1",
		Human: true, Addressed: true, Sender: "operator", Text: "@Fake 写个 hello.txt", Channel: "desk",
	}); status != "stored" {
		t.Fatalf("receive %q", status)
	}
	if err := rt.DecideBriefFor(trigger, "accept", "hello", "在工单目录写 hello.txt，写好后报告完成。", nil, "fake", true); err != nil {
		t.Fatal(err)
	}
	var row superagent.TicketRow
	want := []string{
		"write inside: <nil>",
		"write outside refused: true",
		"p1: selected reject <nil>",
		"p2: selected reject <nil>",
		"p3: selected allow <nil>",
	}
	waitFor(t, "ready report", func() bool {
		var ok bool
		row, ok = ticketFor(rt, trigger)
		return ok && reportCount(row, "ready") > 0 && len(seen.all()) > len(want)
	})
	got := seen.all()
	if len(got) < len(want)+1 || !slices.Equal(got[:len(want)], want) || !strings.HasPrefix(got[len(want)], "report: error=false <nil>") {
		t.Fatalf("fake agent saw:\n%s", strings.Join(got, "\n"))
	}
	body, err := os.ReadFile(filepath.Join(row.Directory, "hello.txt"))
	if err != nil || string(body) != "hi from fake" {
		t.Fatalf("hello.txt in %s: %q %v", row.Directory, body, err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside write landed: %v", err)
	}
	workers := first.prompted()
	if len(workers) != 2 {
		t.Fatalf("want a crashed worker and its reconnect, got %d", len(workers))
	}
	again := workers[1].Calls()
	if !slices.Contains(again, "session/load") || slices.Contains(again, "session/new") {
		t.Fatalf("reconnect after the crash should load the session: %v", again)
	}
	sid := workers[0].Sessions()
	if len(sid) != 1 {
		t.Fatalf("first worker sessions %+v", sid)
	}
	wrote := false
	for _, line := range rt.WorkerTranscript(row.ID) {
		if line.Name == "write_file" && strings.Contains(line.Text, "wrote ") && strings.Contains(line.Text, "hello.txt") {
			wrote = true
		}
	}
	if !wrote {
		t.Fatalf("worker log lacks the write: %+v", rt.WorkerTranscript(row.ID))
	}

	closed = true
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}

	resumed := &notes{}
	second := &fakeAgents{script: acptest.Script{Caps: fakeCaps, OnPrompt: func(turn *acptest.Turn) acp.StopReason {
		resumed.add("session %s", turn.SessionID)
		text, isErr, err := callControl(turn.Session, "orchestrator_report", map[string]any{"state": "ready", "summary": "加好了第二行"})
		resumed.add("report: error=%v %v %s", isErr, err, text)
		return acp.StopEndTurn
	}}}
	b := openBuddyApp(t, root, second)
	defer func() { _ = b.Close() }()
	rt2 := b.orchestrator()
	follow := "desk:s2:t1"
	if status, _ := rt2.Receive(superagent.Trigger{
		ID: follow, Source: "desk", Threads: []string{"desk:s2"}, Primary: "desk:s2",
		Human: true, Addressed: true, Sender: "operator", Text: "再加一行", Channel: "desk",
	}); status != "stored" {
		t.Fatalf("receive follow-up %q", status)
	}
	if err := rt2.Act("assign", row.ID, follow, "在 hello.txt 末尾再加一行。", ""); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "second ready report", func() bool {
		r, ok := ticketFor(rt2, trigger)
		return ok && reportCount(r, "ready") > 1 && len(resumed.all()) > 1
	})
	after := second.prompted()
	if len(after) != 1 {
		t.Fatalf("want one resumed worker, got %d", len(after))
	}
	calls := after[0].Calls()
	if !slices.Contains(calls, "session/load") || slices.Contains(calls, "session/new") {
		t.Fatalf("restart should load the old session: %v", calls)
	}
	if got := resumed.all(); len(got) < 2 || got[0] != "session "+sid[0].ID || !strings.HasPrefix(got[1], "report: error=false <nil>") {
		t.Fatalf("resumed worker saw:\n%s", strings.Join(got, "\n"))
	}
}
