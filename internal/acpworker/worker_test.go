// worker 的测试：经同进程的假 agent 跑完整的回合，看日志、续接、取消、超时、崩溃、要求登录、换模型、权限、空闲回收和进程上限。

package acpworker

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acp/acptest"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// dialer 每次连接都起一个新的同进程假 agent。script 按第几次连接给脚本。
type dialer struct {
	mu     sync.Mutex
	script func(n int) acptest.Script
	agents []*acptest.Agent
}

func scripted(s acptest.Script) *dialer {
	return &dialer{script: func(int) acptest.Script { return s }}
}

func (d *dialer) dial(ctx context.Context, opts acp.Options) (*acp.Client, error) {
	d.mu.Lock()
	r, w, a := acptest.Pipe(d.script(len(d.agents) + 1))
	d.agents = append(d.agents, a)
	d.mu.Unlock()
	return acp.Connect(ctx, r, w, opts)
}

func (d *dialer) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.agents)
}

func (d *dialer) agent(i int) *acptest.Agent {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.agents[i]
}

// record 收集回调的参数，回调可能在别的 goroutine 里。
type record[T any] struct {
	mu    sync.Mutex
	items []T
}

func (r *record[T]) add(v T) {
	r.mu.Lock()
	r.items = append(r.items, v)
	r.mu.Unlock()
}

func (r *record[T]) list() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]T(nil), r.items...)
}

func newWorker(t *testing.T, d *dialer, edit func(*Config)) *Worker {
	t.Helper()
	return newWorkerIn(t, newPool(maxLive), d, edit)
}

// newWorkerIn 在池 p 里建一个连到 d 的工人，测试结束时关掉。
func newWorkerIn(t *testing.T, p *pool, d *dialer, edit func(*Config)) *Worker {
	t.Helper()
	cfg := Config{
		Spec:    superagent.WorkerSpec{TicketID: "T-1", Agent: "fake", Dir: t.TempDir(), System: "按工单做事。"},
		Dial:    d.dial,
		Version: "test",
	}
	if edit != nil {
		edit(&cfg)
	}
	w := open(cfg, p)
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// run 发一条消息，等这一轮跑完，返回回合结束。
func run(w *Worker, text string) dsh.TurnEnd {
	w.Followup(dsh.UserMessage{Text: text})
	w.WhenIdle()
	return lastEnd(w)
}

func lastEnd(w *Worker) dsh.TurnEnd {
	events := w.Session().Events()
	for i := len(events) - 1; i >= 0; i-- {
		if end, ok := events[i].Data.(dsh.TurnEnd); ok && events[i].Type == dsh.EventTurnEnd {
			return end
		}
	}
	return dsh.TurnEnd{}
}

func turnStarts(w *Worker) int {
	n := 0
	for _, ev := range w.Session().Events() {
		if ev.Type == dsh.EventTurnStart {
			n++
		}
	}
	return n
}

func connected(w *Worker) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.client != nil
}

// eventually 等 cond 成立，最多两秒。
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func waitDone(t *testing.T, a *acptest.Agent) {
	t.Helper()
	select {
	case <-a.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("the agent connection is still open")
	}
}

// echoTurns 记下每一轮的提示文字，回 reply。
func echoTurns(texts *record[string], reply string) func(*acptest.Turn) acp.StopReason {
	return func(t *acptest.Turn) acp.StopReason {
		texts.add(t.Text())
		if reply != "" {
			t.Say(reply)
		}
		return acp.StopEndTurn
	}
}

// slowTurn 开始后通知 started，然后一直等到被取消。
func slowTurn(started chan<- struct{}) func(*acptest.Turn) acp.StopReason {
	return func(t *acptest.Turn) acp.StopReason {
		started <- struct{}{}
		if t.WaitCancel(time.Minute) {
			return acp.StopCancelled
		}
		return acp.StopEndTurn
	}
}

// priorTurn 是一轮已经做完的旧日志。
func priorTurn(said string) []dsh.Event {
	s := dsh.NewSession()
	s.Append(dsh.EventTurnStart, 1, 0, nil)
	s.Append(dsh.EventUser, 1, 0, dsh.UserMessage{Text: "做三步"})
	s.Append(dsh.EventAssistant, 1, 1, dsh.Assistant{Text: said, Finish: "stop"})
	s.Append(dsh.EventTurnEnd, 1, 0, dsh.TurnEnd{Kind: "completed"})
	return s.Events()
}

var closeCaps = map[string]any{"loadSession": true, "sessionCapabilities": map[string]any{"close": map[string]any{}}}

func TestWorkerToolsTurnLandsInJournal(t *testing.T) {
	var usage record[superagent.Usage]
	var pointers record[string]
	d := scripted(acptest.Script{OnPrompt: acptest.ToolsTurn})
	w := newWorker(t, d, func(c *Config) {
		c.Spec.OnUsage = usage.add
		c.Spec.OnPointer = pointers.add
	})
	if end := run(w, "把 a 改成 b"); end.Kind != "completed" {
		t.Fatalf("turn end = %+v", end)
	}
	path := filepath.Join(w.spec.Dir, "a.txt")
	wantLines(t, superagent.TranscriptLines(w.Session().Events()),
		[3]string{"user", "", "把 a 改成 b"},
		[3]string{"assistant", "", "先看一下。"},
		[3]string{"tool_call", "edit_file", "*"},
		[3]string{"tool_result", "edit_file", "wrote " + path + "\ndiff " + path + ": 1 -> 1 lines"},
		[3]string{"tool_call", "update_plan", "*"},
		[3]string{"tool_result", "update_plan", "1. [complete] 改 a.txt\n2. [in_progress] 跑测试"},
		[3]string{"assistant", "", "改好了。"},
	)
	if got := usage.list(); len(got) != 1 || got[0] != (superagent.Usage{Used: 1200, Size: 200000, Cost: 0.02, Currency: "USD"}) {
		t.Fatalf("usage = %+v", got)
	}
	if got := pointers.list(); len(got) != 1 || got[0] != "sess-1" {
		t.Fatalf("pointers = %v", got)
	}
	if s := d.agent(0).Sessions(); len(s) != 1 || s[0].Cwd != w.spec.Dir {
		t.Fatalf("sessions = %+v", s)
	}
}

func TestWorkerInstructionsLeadFirstPromptWithoutSystemPrompt(t *testing.T) {
	var texts record[string]
	d := scripted(acptest.Script{OnPrompt: echoTurns(&texts, "好")})
	w := newWorker(t, d, nil)
	run(w, "第一条")
	run(w, "第二条")
	got := texts.list()
	if len(got) != 2 || got[0] != "<yoyo-instructions>\n按工单做事。\n</yoyo-instructions>\n\n第一条" || got[1] != "第二条" {
		t.Fatalf("prompts = %q", got)
	}
	if s := d.agent(0).Sessions(); len(s) != 1 || s[0].System != nil {
		t.Fatalf("sessions = %+v", s)
	}
}

func TestWorkerUsesSystemPromptAndExtraDirsWhenSupported(t *testing.T) {
	var texts record[string]
	d := scripted(acptest.Script{
		Caps:     map[string]any{"sessionCapabilities": map[string]any{"systemPrompt": map[string]any{}, "additionalDirectories": map[string]any{}}},
		OnPrompt: echoTurns(&texts, ""),
	})
	shared, before := t.TempDir(), t.TempDir()
	w := newWorker(t, d, func(c *Config) {
		c.Spec.Roots = []string{shared}
		c.Spec.ReadRoots = []string{before}
	})
	run(w, "开始")
	if got := texts.list(); len(got) != 1 || got[0] != "开始" {
		t.Fatalf("prompts = %q", got)
	}
	s := d.agent(0).Sessions()[0]
	if s.System == nil || s.System.Mode != "append" || s.System.Text != "按工单做事。" {
		t.Fatalf("system prompt = %+v", s.System)
	}
	if !slices.Equal(s.Dirs, []string{shared, before}) {
		t.Fatalf("additional directories = %v", s.Dirs)
	}
}

func TestWorkerGivesControlPlaneThroughBridge(t *testing.T) {
	d := scripted(acptest.Script{})
	w := newWorker(t, d, func(c *Config) {
		c.MCPURL = "http://127.0.0.1:4321/mcp"
		c.MCPToken = "tok-secret"
		c.Exe = filepath.Join("bin", "yoyo.exe")
	})
	run(w, "hi")
	servers := d.agent(0).Sessions()[0].McpServers
	if len(servers) != 1 {
		t.Fatalf("mcp servers = %+v", servers)
	}
	m := servers[0]
	if m.Name != "yoyo" || m.Command != filepath.Join("bin", "yoyo.exe") || len(m.Args) != 0 {
		t.Fatalf("bridge = %+v", m)
	}
	env := map[string]string{}
	for _, e := range m.Env {
		env[e.Name] = e.Value
	}
	if env["YOYO_MCP_BRIDGE"] != "1" || env["YOYO_MCP_URL"] != "http://127.0.0.1:4321/mcp" || env["YOYO_MCP_TOKEN"] != "tok-secret" {
		t.Fatalf("bridge env = %v", env)
	}
}

func TestMCPServersPreferHTTP(t *testing.T) {
	caps := acp.AgentCapabilities{McpCapabilities: acp.McpCapabilities{HTTP: true}}
	got := mcpServers(caps, "http://127.0.0.1:1/mcp", "tok", "yoyo.exe")
	if len(got) != 1 || got[0].Type != "http" || got[0].URL != "http://127.0.0.1:1/mcp" || got[0].Command != "" {
		t.Fatalf("servers = %+v", got)
	}
	if len(got[0].Headers) != 1 || got[0].Headers[0] != (acp.HTTPHeader{Name: "Authorization", Value: "Bearer tok"}) {
		t.Fatalf("headers = %+v", got[0].Headers)
	}
	if len(mcpServers(caps, "", "tok", "x")) != 0 || len(mcpServers(caps, "http://x", "", "x")) != 0 {
		t.Fatal("no control plane without both url and token")
	}
}

func TestWorkerResumesThenLoadsThenStartsNew(t *testing.T) {
	caps := map[string]any{"loadSession": true, "sessionCapabilities": map[string]any{"resume": map[string]any{}}}
	for _, tc := range []struct {
		name       string
		script     acptest.Script
		want, not  []string
		newSession bool
	}{
		{"resume", acptest.Script{Caps: caps}, []string{"session/resume"}, []string{"session/load", "session/new"}, false},
		{"load", acptest.Script{Caps: caps, ResumeFails: true}, []string{"session/resume", "session/load"}, []string{"session/new"}, false},
		{"new", acptest.Script{Caps: caps, ResumeFails: true, LoadFails: true}, []string{"session/resume", "session/load", "session/new"}, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var texts record[string]
			var pointers record[string]
			script := tc.script
			script.Replay = []map[string]any{chunk("old history")}
			script.OnPrompt = echoTurns(&texts, "接着做")
			d := scripted(script)
			w := newWorker(t, d, func(c *Config) {
				c.Spec.Pointer = "sess-old"
				c.Spec.OnPointer = pointers.add
			})
			w.Restore(priorTurn("上次做到第二步。"), 1)
			if end := run(w, "继续"); end.Kind != "completed" {
				t.Fatalf("turn end = %+v", end)
			}
			calls := d.agent(0).Calls()
			for _, m := range tc.want {
				if !slices.Contains(calls, m) {
					t.Fatalf("calls %v lack %s", calls, m)
				}
			}
			for _, m := range tc.not {
				if slices.Contains(calls, m) {
					t.Fatalf("calls %v include %s", calls, m)
				}
			}
			text := texts.list()[0]
			if tc.newSession {
				if got := pointers.list(); len(got) != 1 || got[0] != "sess-1" {
					t.Fatalf("pointers = %v", got)
				}
				for _, part := range []string{"<yoyo-instructions>", "原来的会话接不上了", "上次的摘要：上次做到第二步。", "继续"} {
					if !strings.Contains(text, part) {
						t.Fatalf("first prompt lacks %q:\n%s", part, text)
					}
				}
			} else {
				if got := pointers.list(); len(got) != 0 {
					t.Fatalf("pointer changed on %s: %v", tc.name, got)
				}
				if text != "继续" {
					t.Fatalf("prompt = %q", text)
				}
			}
			if turn := w.Session().Events()[len(w.Session().Events())-1].Turn; turn != 2 {
				t.Fatalf("turn after restore = %d, want 2", turn)
			}
			for _, line := range superagent.TranscriptLines(w.Session().Events()) {
				if strings.Contains(line.Text, "old history") {
					t.Fatal("replayed history leaked into the journal")
				}
			}
		})
	}
}

func TestWorkerCancelDropsQueuedInput(t *testing.T) {
	started := make(chan struct{}, 4)
	d := scripted(acptest.Script{OnPrompt: slowTurn(started)})
	w := newWorker(t, d, nil)
	w.Followup(dsh.UserMessage{Text: "慢慢做"})
	<-started
	w.Followup(dsh.UserMessage{Text: "排队的"})
	w.Cancel(dsh.CancelCause{Kind: "user", Reason: "换方向"}, false)
	w.WhenIdle()
	if end := lastEnd(w); end.Kind != "aborted" || end.Reason != "user: 换方向" {
		t.Fatalf("turn end = %+v", end)
	}
	if n := turnStarts(w); n != 1 {
		t.Fatalf("turns = %d, want the queued input dropped", n)
	}
	if !slices.Contains(d.agent(0).Calls(), "session/cancel") {
		t.Fatalf("calls = %v, want session/cancel", d.agent(0).Calls())
	}
	if w.Status() != "idle" {
		t.Fatalf("status = %s", w.Status())
	}
}

func TestWorkerCancelKeepsInboxForNextTurn(t *testing.T) {
	started := make(chan struct{}, 4)
	var texts record[string]
	slow := slowTurn(started)
	d := scripted(acptest.Script{OnPrompt: func(t *acptest.Turn) acp.StopReason {
		if strings.Contains(t.Text(), "慢") {
			return slow(t)
		}
		return echoTurns(&texts, "收到")(t)
	}})
	w := newWorker(t, d, nil)
	w.Followup(dsh.UserMessage{Text: "慢慢做"})
	<-started
	w.Followup(dsh.UserMessage{Text: "补一句"})
	w.Cancel(dsh.CancelCause{Kind: "steer"}, true)
	w.WhenIdle()
	if n := turnStarts(w); n != 2 {
		t.Fatalf("turns = %d, want 2", n)
	}
	if end := lastEnd(w); end.Kind != "completed" {
		t.Fatalf("turn end = %+v", end)
	}
	if got := texts.list(); len(got) != 1 || got[0] != "补一句" {
		t.Fatalf("prompts = %q", got)
	}
}

func TestWorkerTurnTimeout(t *testing.T) {
	started := make(chan struct{}, 4)
	d := scripted(acptest.Script{OnPrompt: slowTurn(started)})
	w := newWorker(t, d, func(c *Config) { c.TurnTimeout = 80 * time.Millisecond })
	if end := run(w, "很久"); end.Kind != "aborted" || end.Reason != superagent.ReasonTurnTimeout {
		t.Fatalf("turn end = %+v", end)
	}
	if got := w.Caps(); got != (superagent.WorkerCaps{Nudges: 1, External: true}) {
		t.Fatalf("caps = %+v", got)
	}
	if got := open(Config{TurnTimeout: 3 * time.Minute}, newPool(1)).Caps().TurnMinutes; got != 3 {
		t.Fatalf("TurnMinutes = %d", got)
	}
}

func TestWorkerCrashReconnectsAndLoads(t *testing.T) {
	d := &dialer{script: func(n int) acptest.Script {
		if n == 1 {
			return acptest.Script{OnPrompt: func(t *acptest.Turn) acp.StopReason {
				t.Say("about to fail")
				t.Crash(3, "boom")
				return acp.StopEndTurn
			}}
		}
		return acptest.Script{}
	}}
	var runs record[error]
	w := newWorker(t, d, func(c *Config) { c.OnRun = runs.add })
	end := run(w, "做")
	if end.Kind != "error" || !strings.HasPrefix(end.Reason, "agent exited") {
		t.Fatalf("crash turn end = %+v", end)
	}
	if connected(w) {
		t.Fatal("a dead connection was kept")
	}
	if end := run(w, "再来"); end.Kind != "completed" {
		t.Fatalf("turn after crash = %+v", end)
	}
	if d.count() != 2 {
		t.Fatalf("dials = %d, want 2", d.count())
	}
	if calls := d.agent(1).Calls(); !slices.Contains(calls, "session/load") || slices.Contains(calls, "session/new") {
		t.Fatalf("reconnect calls = %v, want session/load of the old session", calls)
	}
	if got := runs.list(); len(got) != 2 || got[0] != nil || got[1] != nil {
		t.Fatalf("OnRun = %v", got)
	}
}

func TestWorkerAuthRequiredTellsBuddy(t *testing.T) {
	d := scripted(acptest.Script{NeedAuth: true})
	var auth record[string]
	var runs record[error]
	w := newWorker(t, d, func(c *Config) {
		c.OnAuth = auth.add
		c.OnRun = runs.add
		c.Redact = func(s string) string { return strings.ReplaceAll(strings.ToLower(s), "required", "[redacted]") }
	})
	if end := run(w, "做"); end.Kind != "error" || end.Reason != superagent.ReasonAuthRequired {
		t.Fatalf("turn end = %+v", end)
	}
	if got := auth.list(); len(got) != 1 || !strings.Contains(got[0], "[redacted]") {
		t.Fatalf("OnAuth = %q", got)
	}
	if got := runs.list(); len(got) != 0 {
		t.Fatalf("OnRun = %v, want login failures left to OnAuth", got)
	}
	waitDone(t, d.agent(0))
}

func TestWorkerSwitchesToChosenModel(t *testing.T) {
	models := []acp.ModelInfo{{ModelID: "m1", Name: "One"}, {ModelID: "m2", Name: "Two"}}
	for model, want := range map[string]bool{"m2": true, "m1": false, "": false} {
		d := scripted(acptest.Script{Models: models})
		w := newWorker(t, d, func(c *Config) { c.Model = model })
		if end := run(w, "hi"); end.Kind != "completed" {
			t.Fatalf("model %q: turn end = %+v", model, end)
		}
		if got := slices.Contains(d.agent(0).Calls(), "session/set_model"); got != want {
			t.Fatalf("model %q: set_model sent = %v, want %v", model, got, want)
		}
	}
}

// permTurn 要一次执行命令的权限，再把得到的选择说出来。
func permTurn(options []acp.PermissionOption) func(*acptest.Turn) acp.StopReason {
	return func(t *acptest.Turn) acp.StopReason {
		out, err := t.Permission(map[string]any{"toolCallId": "p1", "title": "rm -rf build", "kind": "execute", "rawInput": map[string]any{"command": "rm -rf build"}}, options)
		if err != nil {
			t.Say("permission error: " + err.Error())
			return acp.StopEndTurn
		}
		t.Say("outcome " + out.Outcome + " " + out.OptionID)
		return acp.StopEndTurn
	}
}

func TestPermissionBridge(t *testing.T) {
	allow := func(context.Context, Approval) (bool, error) { return true, nil }
	rejectOnly := []acp.PermissionOption{{OptionID: "no", Name: "No", Kind: acp.PermRejectOnce}}
	for _, tc := range []struct {
		name    string
		options []acp.PermissionOption
		approve func(context.Context, Approval) (bool, error)
		said    string
		denied  bool
	}{
		{"nil approve denies", acptest.DefaultPermOptions(), nil, "outcome selected reject", true},
		{"approved picks allow_once", acptest.DefaultPermOptions(), allow, "outcome selected allow", false},
		{"error denies", acptest.DefaultPermOptions(), func(context.Context, Approval) (bool, error) { return true, errors.New("broker down") }, "outcome selected reject", true},
		{"no allow option rejects", rejectOnly, allow, "outcome selected no", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var seen record[Approval]
			d := scripted(acptest.Script{OnPrompt: permTurn(tc.options)})
			w := newWorker(t, d, func(c *Config) {
				if tc.approve != nil {
					c.Approve = func(ctx context.Context, a Approval) (bool, error) {
						seen.add(a)
						return tc.approve(ctx, a)
					}
				}
			})
			run(w, "清理")
			lines := superagent.TranscriptLines(w.Session().Events())
			wantLines(t, lines,
				[3]string{"user", "", "清理"},
				[3]string{"tool_call", "shell", `{"command":"rm -rf build"}`},
				[3]string{"tool_result", "shell", "*"},
				[3]string{"assistant", "", tc.said},
			)
			if lines[2].Error != tc.denied {
				t.Fatalf("result = %+v, denied %v", lines[2], tc.denied)
			}
			if tc.approve != nil {
				got := seen.list()
				if len(got) != 1 || got[0].Kind != "execute" || got[0].Name != "shell" || got[0].Command != "rm -rf build" || got[0].Label() != "rm -rf build" {
					t.Fatalf("approvals = %+v", got)
				}
			}
		})
	}
}

func TestPermissionWaitShowsOnTicket(t *testing.T) {
	old := attentionDelay
	attentionDelay = 10 * time.Millisecond
	t.Cleanup(func() { attentionDelay = old })
	for _, tc := range []struct {
		name  string
		delay time.Duration
		want  []string
	}{
		{"slow approval", 150 * time.Millisecond, []string{"等批准：rm -rf build", ""}},
		{"instant approval", 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var notes record[string]
			d := scripted(acptest.Script{OnPrompt: permTurn(acptest.DefaultPermOptions())})
			w := newWorker(t, d, func(c *Config) {
				c.Spec.OnAttention = notes.add
				c.Approve = func(context.Context, Approval) (bool, error) {
					time.Sleep(tc.delay)
					return true, nil
				}
			})
			run(w, "清理")
			if got := notes.list(); !slices.Equal(got, tc.want) {
				t.Fatalf("attention = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPermissionCancelledWithTurn(t *testing.T) {
	asked := make(chan struct{}, 1)
	d := scripted(acptest.Script{OnPrompt: permTurn(acptest.DefaultPermOptions())})
	w := newWorker(t, d, func(c *Config) {
		c.Approve = func(ctx context.Context, _ Approval) (bool, error) {
			asked <- struct{}{}
			<-ctx.Done()
			return false, ctx.Err()
		}
	})
	w.Followup(dsh.UserMessage{Text: "清理"})
	<-asked
	w.Cancel(dsh.CancelCause{Kind: "user"}, false)
	w.WhenIdle()
	if end := lastEnd(w); end.Kind != "aborted" || end.Reason != "user" {
		t.Fatalf("turn end = %+v", end)
	}
	lines := superagent.TranscriptLines(w.Session().Events())
	if last := lines[len(lines)-1]; last.Text != "outcome cancelled" {
		t.Fatalf("agent saw %q, want a cancelled outcome", last.Text)
	}
}

func TestWorkerIdleSuspendKeepsSession(t *testing.T) {
	var pointers record[string]
	d := scripted(acptest.Script{Caps: closeCaps})
	w := newWorker(t, d, func(c *Config) {
		c.IdleTimeout = 30 * time.Millisecond
		c.Spec.OnPointer = pointers.add
	})
	run(w, "一")
	eventually(t, "the idle process to stop", func() bool { return !connected(w) && w.pool.size() == 0 })
	waitDone(t, d.agent(0))
	if slices.Contains(d.agent(0).Calls(), "session/close") {
		t.Fatal("idle suspend closed the session")
	}
	if end := run(w, "二"); end.Kind != "completed" {
		t.Fatalf("turn after suspend = %+v", end)
	}
	if d.count() != 2 || !slices.Contains(d.agent(1).Calls(), "session/load") {
		t.Fatalf("dials = %d, second calls = %v", d.count(), d.agent(1).Calls())
	}
	if got := pointers.list(); len(got) != 1 {
		t.Fatalf("pointers = %v, want the session kept", got)
	}
}

func TestPoolStopsLeastRecentlyUsedIdleWorker(t *testing.T) {
	p := newPool(1)
	d1, d2 := scripted(acptest.Script{}), scripted(acptest.Script{})
	w1 := newWorkerIn(t, p, d1, nil)
	w2 := newWorkerIn(t, p, d2, nil)
	run(w1, "一")
	if p.size() != 1 {
		t.Fatalf("live = %d", p.size())
	}
	run(w2, "二")
	eventually(t, "the older idle worker to stop", func() bool { return !connected(w1) })
	waitDone(t, d1.agent(0))
	if !connected(w2) || p.size() != 1 {
		t.Fatalf("w2 connected %v, live %d", connected(w2), p.size())
	}
}

func TestPoolNeverStopsBusyWorkers(t *testing.T) {
	p := newPool(1)
	started := make(chan struct{}, 4)
	d1, d2 := scripted(acptest.Script{OnPrompt: slowTurn(started)}), scripted(acptest.Script{})
	w1 := newWorkerIn(t, p, d1, nil)
	w2 := newWorkerIn(t, p, d2, nil)
	w1.Followup(dsh.UserMessage{Text: "慢慢做"})
	<-started
	run(w2, "二")
	time.Sleep(30 * time.Millisecond)
	if !connected(w1) || p.size() != 2 {
		t.Fatalf("busy worker stopped: connected %v, live %d", connected(w1), p.size())
	}
	w1.Cancel(dsh.CancelCause{Kind: "user"}, false)
	w1.WhenIdle()
}

func TestCloseEndsSessionOnce(t *testing.T) {
	d := scripted(acptest.Script{Caps: closeCaps})
	var closes record[bool]
	w := newWorker(t, d, func(c *Config) { c.OnClose = func() { closes.add(true) } })
	run(w, "一")
	_ = w.Close()
	_ = w.Close()
	if n := len(closes.list()); n != 1 {
		t.Fatalf("OnClose ran %d times", n)
	}
	if !slices.Contains(d.agent(0).Calls(), "session/close") {
		t.Fatalf("calls = %v, want session/close", d.agent(0).Calls())
	}
	w.Followup(dsh.UserMessage{Text: "二"})
	w.WhenIdle()
	if turnStarts(w) != 1 || w.pool.size() != 0 || d.count() != 1 {
		t.Fatalf("closed worker ran again: turns %d, live %d, dials %d", turnStarts(w), w.pool.size(), d.count())
	}
}

func TestCloseAllHaltsWithoutReconnecting(t *testing.T) {
	started := make(chan struct{}, 4)
	d := scripted(acptest.Script{Caps: closeCaps, OnPrompt: slowTurn(started)})
	w := newWorker(t, d, nil)
	w.Followup(dsh.UserMessage{Text: "慢慢做"})
	<-started
	w.pool.closeAll()
	w.WhenIdle()
	if end := lastEnd(w); end.Kind != "aborted" || end.Reason != "shutdown" {
		t.Fatalf("turn end = %+v", end)
	}
	w.Followup(dsh.UserMessage{Text: "再来"})
	w.WhenIdle()
	if d.count() != 1 || turnStarts(w) != 1 {
		t.Fatalf("worker ran after CloseAll: dials %d, turns %d", d.count(), turnStarts(w))
	}
	waitDone(t, d.agent(0))
	if slices.Contains(d.agent(0).Calls(), "session/close") {
		t.Fatal("CloseAll closed the session")
	}
	fresh := newWorkerIn(t, w.pool, scripted(acptest.Script{}), nil)
	if end := run(fresh, "新的"); end.Kind != "completed" {
		t.Fatalf("a worker opened after CloseAll = %+v", end)
	}
}
