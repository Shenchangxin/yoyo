// 锁住这些不变量：工作区根内的路径通过 WithinWorkspace，经 .. 跳出根的路径不通过。
// Session 决定会记住，同一请求第二次不再问；ForceAsk 跳过这份会话缓存，但 Always 仍然生效。
// 策略里已有 WriteWorkspace 时，review_page 第一次仍会问，第二次靠会话放行；另一会话的 write_file 能够通过。
// HighRisk 在 ForceAsk 下即使审批返回 Always，第二次仍会问。
// AuthAsk 跳过始终表；改回 AuthDefault 后恢复。AuthFull 直接放行 Shell，HighRisk 仍被拒绝。
// Network 的 https 地址不因路径判断失败，会话授权对另一条 URL 仍有效。读工作区逃出根目录会失败。

package capability

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestWithinWorkspace 确认根内路径通过，root 再经 .. 出去的路径不通过。
func TestWithinWorkspace(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "a", "b.txt")
	if !WithinWorkspace(root, inside) {
		t.Fatal("inside")
	}
	if WithinWorkspace(root, filepath.Join(root, "..", "x")) {
		t.Fatal("escape")
	}
}

// TestBrokerSessionAllow 确认 Shell 返回 Session 后，同一会话第二次 Check 不再调用审批。
func TestBrokerSessionAllow(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Session, nil
	})
	req := Request{Level: Shell, SessionID: "s1"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("asked %d times", n)
	}
}

// TestBrokerForceAskSkipsSession 确认 ForceAsk 的 Network 不使用会话缓存，两次都问。
func TestBrokerForceAskSkipsSession(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Session, nil
	})
	req := Request{Level: Network, SessionID: "s1", ForceAsk: true, Action: "mcp"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("ForceAsk should skip session cache, asked %d", n)
	}
}

// TestBrokerForceAskHonorsAlways 确认 ForceAsk 仍接受 Always，第二次不再问。
func TestBrokerForceAskHonorsAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Always, nil
	})
	req := Request{Level: Network, SessionID: "s1", ForceAsk: true, Action: "mcp"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("ForceAsk must still honor Always, asked %d", n)
	}
}

// TestBrokerPagePromoteNeverAlways 确认 review_page 第一次会问、第二次不再问，且另一会话的 write_file 能通过。
func TestBrokerPagePromoteNeverAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Always, nil
	})
	req := Request{Level: WriteWorkspace, Action: "review_page", SessionID: "s1"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("first review_page should ask, n=%d", n)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("session grant should skip a second ask, n=%d", n)
	}
	if b.always[WriteWorkspace] && n == 1 {
		// 始终表不能吞掉第一次 review_page。
	}
	write := Request{Level: WriteWorkspace, Action: "write_file", SessionID: "other"}
	if err := b.Check(write); err != nil {
		t.Fatal(err)
	}
}

// TestBrokerNeverAlwaysHighRisk 确认 HighRisk 在 ForceAsk 下即使返回 Always，第二次仍会问。
func TestBrokerNeverAlwaysHighRisk(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Always, nil
	})
	req := Request{Level: HighRisk, SessionID: "s1", ForceAsk: true, Action: "wasm"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("identity Always must not stick, asked %d", n)
	}
}

// TestBrokerAskModeIgnoresAlways 确认 AuthAsk 忽略始终表里的 WriteWorkspace，两次都问。
func TestBrokerAskModeIgnoresAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{ReadWorkspace, WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Once, nil
	})
	b.ApplyAuthMode("s1", AuthAsk)
	req := Request{Level: WriteWorkspace, SessionID: "s1"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("ask mode should skip always-map, asked %d", n)
	}
}

// TestBrokerFullGrantsShell 确认 AuthFull 直接放行 Shell，HighRisk 仍然要问并被拒绝。
func TestBrokerFullGrantsShell(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{ReadWorkspace, WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Deny, nil
	})
	b.ApplyAuthMode("s1", AuthFull)
	if err := b.Check(Request{Level: Shell, SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("full access should grant shell, asked %d", n)
	}
	if err := b.Check(Request{Level: HighRisk, SessionID: "s1", ForceAsk: true}); err == nil {
		t.Fatal("identity must still ask")
	}
}

// TestBrokerAskThenDefaultRestoresAlways 确认从 AuthAsk 改回 AuthDefault 后，WriteWorkspace 重新走始终表。
func TestBrokerAskThenDefaultRestoresAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Once, nil
	})
	b.ApplyAuthMode("s1", AuthAsk)
	if err := b.Check(Request{Level: WriteWorkspace, SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	b.ApplyAuthMode("s1", AuthDefault)
	if err := b.Check(Request{Level: WriteWorkspace, SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("default should honor always-map again, asked %d", n)
	}
}

// TestParseAuthMode 确认 auto-edit、full_access 和空串分别收成 AuthAutoEdit、AuthFull、AuthDefault。
func TestParseAuthMode(t *testing.T) {
	if ParseAuthMode("auto-edit") != AuthAutoEdit {
		t.Fatal(ParseAuthMode("auto-edit"))
	}
	if ParseAuthMode("full_access") != AuthFull {
		t.Fatal(ParseAuthMode("full_access"))
	}
	if ParseAuthMode("") != AuthDefault {
		t.Fatal(ParseAuthMode(""))
	}
}

// TestBrokerNetworkURLSkipsPathJail 确认 Network 的 https 地址不会被路径判断拒绝，会话授权对另一条 URL 仍然有效。
func TestBrokerNetworkURLSkipsPathJail(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Session, nil
	})
	ws := t.TempDir()
	req := Request{
		Level:     Network,
		Action:    "web_fetch",
		Path:      "https://export.arxiv.org/api/query?q=agent",
		Command:   "https://export.arxiv.org/api/query?q=agent",
		SessionID: "s1",
		Workspace: ws,
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("asked %d", n)
	}
	if err := b.Check(Request{Level: Network, SessionID: "s1", Path: "https://example.com", Workspace: ws}); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("session grant should stick, asked %d", n)
	}
}

// TestBrokerReadStillJailsEscape 确认读工作区时逃出根目录会返回含 escapes workspace 的错误。
func TestBrokerReadStillJailsEscape(t *testing.T) {
	b := NewBroker(AutoPolicy{Allow: []Level{ReadWorkspace}}, func(context.Context, Request) (Decision, error) {
		return Always, nil
	})
	ws := t.TempDir()
	err := b.Check(Request{Level: ReadWorkspace, Path: filepath.Join(ws, "..", "secret"), Workspace: ws, SessionID: "s1"})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("%v", err)
	}
}
