package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

func TestUniqueReplace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("foo foo"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("str_replace", `{"path":"a.txt","old_str":"foo","new_str":"bar"}`)
	if res.Err == nil {
		t.Fatal("expected non-unique failure")
	}
	res = tools.Call("str_replace", `{"path":"a.txt","old_str":"foo","new_str":"bar","replace_all":true}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
}

func TestGlobAndGrep(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	g := tools.Call("glob", `{"pattern":"*.go"}`)
	if g.Err != nil || !strings.Contains(g.Content, "hello.go") {
		t.Fatalf("%+v", g)
	}
	gr := tools.Call("grep", `{"pattern":"package"}`)
	if gr.Err != nil || !strings.Contains(gr.Content, "hello.go") {
		t.Fatalf("%+v", gr)
	}
}

func TestCompactStubsTools(t *testing.T) {
	loop := DefaultLoop()
	loop.CompactionTokens = 200
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u"},
		{Role: RoleTool, Content: strings.Repeat("x", 8000)},
		{Role: RoleAssistant, Content: "ok"},
	}
	out, note := Compact(msgs, loop)
	if note == "" {
		t.Fatalf("expected stub, tokens=%d", messagesTokens(out))
	}
}

func TestCompactDoesNotMutateInput(t *testing.T) {
	loop := DefaultLoop()
	loop.CompactionTokens = 200
	orig := strings.Repeat("x", 8000)
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleTool, Content: orig},
	}
	_, _ = Compact(msgs, loop)
	if msgs[1].Content != orig {
		t.Fatal("compact mutated live history")
	}
}

func TestReadFileOverflow(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("line\n", 80_000)
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("read_file", `{"path":"big.txt"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, ".yoyo/overflow") {
		t.Fatalf("expected overflow path, got %s", res.Content[:min(200, len(res.Content))])
	}
}

func TestBudgetStopsLoop(t *testing.T) {
	dir := t.TempDir()
	loop := DefaultLoop()
	loop.MaxBudgetUSD = 0.5
	meter := &Meter{USDPerMTok: 1e6}
	_, err := Run(context.Background(), RunRequest{
		User:      "Write hello.txt containing hello",
		Workspace: dir,
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    HeuristicSolver{},
		Loop:      loop,
		Meter:     meter,
	})
	var b ErrBudget
	if !errors.As(err, &b) {
		t.Fatalf("want budget stop, got %v", err)
	}
}

func TestFileHookDeniesTool(t *testing.T) {
	dir := t.TempDir()
	client := &ScriptedClient{Steps: []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "1", Name: "write_file", Arguments: `{"path":"x.txt","content":"no"}`}}},
		{Role: RoleAssistant, Content: "ok"},
	}}
	_, err := Run(context.Background(), RunRequest{
		User:      "write",
		Tools:     &WorkspaceTools{Workspace: dir},
		Client:    client,
		Loop:      DefaultLoop(),
		FileHooks: []FileHook{{Match: "write_file", Deny: true, Reason: "blocked"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "x.txt")); err == nil {
		t.Fatal("write should have been denied")
	}
}

func TestResolveGitBashDrivePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("msys drive paths are a Windows Git-Bash jail bug")
	}
	dir := t.TempDir()
	inner := filepath.Join(dir, "server.log")
	if err := os.WriteFile(inner, []byte("listening"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir, ChatOverlay: true}
	msys := gitBashPath(dir)
	got, err := tools.resolve(msys + "/server.log")
	if err != nil {
		t.Fatal(err)
	}
	if !capability.WithinWorkspace(dir, got) {
		t.Fatalf("escaped %s", got)
	}
	res := tools.Call("shell", `{"command":"cd `+msys+` && cat server.log"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "listening") {
		t.Fatalf("%q", res.Content)
	}
	if strings.Contains(res.Content, dir+string(filepath.Separator)+"c"+string(filepath.Separator)+"Users") {
		t.Fatalf("doubled path %q", res.Content)
	}
}

func TestWindowsPosixShellPrefersGit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}
	sh := windowsPosixShell()
	if sh == "" {
		t.Skip("no Git bash installed")
	}
	low := strings.ToLower(sh)
	if strings.Contains(low, `\windowsapps\`) || strings.Contains(low, `\system32\`) {
		t.Fatalf("wsl stub %s", sh)
	}
}

func gitBashPath(p string) string {
	p = filepath.ToSlash(p)
	if len(p) >= 2 && p[1] == ':' {
		return "/" + strings.ToLower(p[:1]) + p[2:]
	}
	return p
}

func TestReadFileDeniesDiagPlane(t *testing.T) {
	home := t.TempDir()
	logDir := filepath.Join(home, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(logDir, "yoyo.log")
	if err := os.WriteFile(secret, []byte("sk-live-secretvalue"), 0o600); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: home, Home: home}
	res := tools.Call("read_file", `{"path":"logs/yoyo.log"}`)
	if res.Err == nil {
		t.Fatal("expected diagnostic plane deny")
	}
	if strings.Contains(res.Content, "sk-live") {
		t.Fatalf("secret leaked: %s", res.Content)
	}
}
