package app

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/im"
)

type imCapture struct {
	mu   sync.Mutex
	sent []string
}

func (c *imCapture) push(_, _, text string) error {
	c.mu.Lock()
	c.sent = append(c.sent, text)
	c.mu.Unlock()
	return nil
}

func (c *imCapture) all() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.sent...)
}

func boundApp(t *testing.T) (*App, *imCapture) {
	t.Helper()
	a := &App{Home: &home.Dir{Root: t.TempDir()}}
	cap := &imCapture{}
	a.imh.loaded = true
	a.imh.home = IMHome{Account: "lark-1", Chat: "oc_1", Sender: "u1", Platform: "lark"}
	a.imh.push = cap.push
	a.imh.sync = true
	return a, cap
}

func TestParseIMCommand(t *testing.T) {
	cmd, args, tail, ok := parseIMCommand("@yoyo /答 ab3 先改登录  再说")
	if !ok || cmd != "answer" || len(args) < 1 || args[0] != "ab3" || tail != "先改登录  再说" {
		t.Fatalf("answer = %q %q %q %v", cmd, args, tail, ok)
	}
	if _, _, _, ok := parseIMCommand("状态"); ok {
		t.Fatal("a command needs a slash")
	}
	if _, _, _, ok := parseIMCommand("/随便聊聊"); ok {
		t.Fatal("unknown words are not commands")
	}
	cmd, args, _, ok = parseIMCommand("／批准 k2 总是")
	if !ok || cmd != "approve" || len(args) != 2 || args[1] != "总是" {
		t.Fatalf("full-width slash = %q %q %v", cmd, args, ok)
	}
}

func TestIMBindNeedsTheCode(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}}
	var got []string
	in := im.Inbound{AccountID: "lark-1", Platform: "lark", Kind: "direct", ChatID: "oc_1", SenderID: "u1", Text: "/绑定 000000",
		Reply: func(s string) error { got = append(got, s); return nil }}
	if !a.imCommand(in, in) || a.IMHomeState().Bound {
		t.Fatal("a wrong code must not bind")
	}
	st := a.IMPair()
	if len(st.Code) != 6 {
		t.Fatalf("code = %q", st.Code)
	}
	in.Text = "/绑定 " + st.Code
	a.imCommand(in, in)
	if !a.IMHomeState().Bound || a.IMHomeState().Code != "" {
		t.Fatalf("bind = %+v", a.IMHomeState())
	}
	again := &App{Home: &home.Dir{Root: dir}}
	if !again.IMHomeState().Bound {
		t.Fatal("the binding should survive a restart")
	}
	other := in
	other.SenderID = "u2"
	other.Text = "/状态"
	if !a.imCommand(other, other) || !strings.Contains(got[len(got)-1], "绑定的人") {
		t.Fatalf("a stranger should be refused, got %q", got)
	}
	if a.isIMHome(other) || !a.isIMHome(in) {
		t.Fatal("home check")
	}
}

func TestIMCodesAreStable(t *testing.T) {
	a, _ := boundApp(t)
	x := a.imIssue(imCode{Kind: "approval", Target: "o1"})
	if y := a.imIssue(imCode{Kind: "approval", Target: "o1"}); y != x {
		t.Fatalf("same target got %q then %q", x, y)
	}
	if _, c, ok := a.imLookup(strings.ToUpper(x), "approval"); !ok || c.Target != "o1" {
		t.Fatal("lookup")
	}
	if _, _, ok := a.imLookup(x, "ticket"); ok {
		t.Fatal("kind must match")
	}
	s := a.imIssue(imCode{Kind: "speak", Target: "cal:1", Class: "calendar"})
	if code, c, ok := a.imLookup("", "speak"); !ok || code != s || c.Class != "calendar" {
		t.Fatal("an empty speak code means the last note")
	}
}

func TestIMQuietHoursHoldThenFlush(t *testing.T) {
	a, cap := boundApp(t)
	a.Config.Desk = DeskConfig{QuietStart: "23:00", QuietEnd: "08:00"}
	night := time.Date(2026, 11, 27, 2, 0, 0, 0, time.Local)
	morning := time.Date(2026, 11, 27, 8, 30, 0, 0, time.Local)
	a.imSendAt("speak", "第一条", "", night)
	a.imSendAt("brief", "第二条", "", night)
	a.imSendAt("approval", "已经批过的", "gone", night)
	if len(cap.all()) != 0 || a.IMHomeState().Held != 3 {
		t.Fatalf("quiet hours should hold, sent %q", cap.all())
	}
	a.imFlush(night.Add(time.Hour))
	if len(cap.all()) != 0 {
		t.Fatal("still quiet")
	}
	a.imFlush(morning)
	sent := cap.all()
	if len(sent) != 1 || !strings.Contains(sent[0], "第一条") || !strings.Contains(sent[0], "第二条") || strings.Contains(sent[0], "已经批过的") {
		t.Fatalf("flush = %q", sent)
	}
	if a.IMHomeState().Held != 0 {
		t.Fatal("held should be empty")
	}
	a.imSendAt("speak", "白天", "", morning)
	if got := cap.all(); len(got) != 2 || got[1] != "白天" {
		t.Fatalf("daytime should send now, %q", got)
	}
}

func TestIMUnboundSendsNothing(t *testing.T) {
	a := &App{Home: &home.Dir{Root: t.TempDir()}}
	cap := &imCapture{}
	a.imh.push = cap.push
	a.imh.sync = true
	if a.imSend("speak", "x", "") || len(cap.all()) != 0 {
		t.Fatal("no home, no send")
	}
}

func TestApprovalTextHidesSecrets(t *testing.T) {
	o := capability.Offer{ID: "o1", Request: capability.Request{Action: "shell", Command: "export API_KEY=sk-123 && deploy"}}
	text := approvalIMText(o, "k2", true)
	if strings.Contains(text, "sk-123") || !strings.Contains(text, "/批准 k2") {
		t.Fatalf("approval text = %q", text)
	}
	o.Request.Command = "go test ./..."
	if text := approvalIMText(o, "k2", true); !strings.Contains(text, "go test ./...") {
		t.Fatalf("plain command should show, %q", text)
	}
}

func TestBriefIMText(t *testing.T) {
	if briefIMText("晨间简报", []BriefCard{{ID: "quiet", Title: "今天没有值得写的事"}}, true) != "" {
		t.Fatal("a quiet brief is not sent")
	}
	text := briefIMText("晨间简报 · 11月27日", []BriefCard{{ID: "a", Title: "评审今天到期", Lede: "下午三点前要交"}, {ID: "b", Title: "CI 卡住了"}}, true)
	if !strings.HasPrefix(text, "晨间简报 · 11月27日") || !strings.Contains(text, "· 评审今天到期\n  下午三点前要交") || !strings.Contains(text, "· CI 卡住了") {
		t.Fatalf("brief text = %q", text)
	}
}

func TestIMTicketOncePerReport(t *testing.T) {
	a, cap := boundApp(t)
	a.imTicket("tkt-1", "working", "在改", "", "handoff:tkt-1:a")
	if len(cap.all()) != 0 {
		t.Fatal("progress stays on the desktop")
	}
	a.imTicket("tkt-1", "blocked", "要选一个方案", "用 A 还是 B？", "handoff:tkt-1:b")
	a.imTicket("tkt-1", "blocked", "要选一个方案", "用 A 还是 B？", "handoff:tkt-1:b")
	sent := cap.all()
	if len(sent) != 1 || !strings.Contains(sent[0], "在等你") || !strings.Contains(sent[0], "/答 ") {
		t.Fatalf("question = %q", sent)
	}
}
