// 锁住替换必须唯一、glob 和 grep 能命中、超长读取会溢出、诊断面路径被拒绝，以及 Git Bash 盘符路径留在工作区内。

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

// TestUniqueReplace 锁住 old_str 匹配多次且未设 replace_all 时失败，设了之后成功。
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

// TestGlobAndGrep 锁住 *.go 能找到 hello.go，grep package 也能命中。
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

// TestCompactStubsTools 锁住压缩预算很小时，超长工具结果会被收成带说明的摘要。
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

// TestCompactDoesNotMutateInput 锁住 Compact 不改调用方切片里的原文。
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

// TestReadFileMissingQAPath 锁住缺失的 qa/verification.json 说明路径不存在，并指出那不是 CSS 选择器。
func TestReadFileMissingQAPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "game-adaptations", "tianming-guiling", "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := &WorkspaceTools{Workspace: dir}
	res := tools.Call("read_file", `{"path":"game-adaptations/tianming-guiling/qa/verification.json"}`)
	if res.Err == nil {
		t.Fatal("expected missing path")
	}
	msg := res.Err.Error()
	if !strings.Contains(msg, "path does not exist") || !strings.Contains(msg, "not a CSS selector") {
		t.Fatalf("%s", msg)
	}
}

// TestReadFileOverflow 锁住超长文件的读取结果指向 .yoyo/overflow。
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

// TestBudgetStopsLoop 锁住费用超过 MaxBudgetUSD 时 Run 返回 ErrBudget。
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

// TestFileHookDeniesTool 锁住 FileHook 拒绝 write_file 时，目标文件不会出现。
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

// TestResolveGitBashDrivePath 在 Windows 上锁住 /c/... 这种 Git Bash 路径经 resolve 后仍在工作区内，shell 读取不会把盘符再拼一次。
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

// TestWindowsPosixShellPrefersGit 在 Windows 上锁住找到的 POSIX shell 不落在 WindowsApps 或 system32。
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

// gitBashPath 把 Windows 盘符路径收成 /c/... 这种 Git Bash 形式。
func gitBashPath(p string) string {
	p = filepath.ToSlash(p)
	if len(p) >= 2 && p[1] == ':' {
		return "/" + strings.ToLower(p[:1]) + p[2:]
	}
	return p
}

// TestReadFileDeniesDiagPlane 锁住读取 YOYO_HOME 下 logs 里的文件被拒绝，秘密不会出现在 Content 里。
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

// TestSealedGameWorldWrite 锁住 game-worlds 下的 index.html 和 playable/runtime.js 被拒绝，DESIGN.md 可以写。
func TestSealedGameWorldWrite(t *testing.T) {
	dir := t.TempDir()
	tools := &WorkspaceTools{Workspace: dir}
	html := tools.Call("write_file", `{"path":"game-worlds/world-x/index.html","content":"<html><body>nope</body></html>"}`)
	if html.Err == nil || !strings.Contains(html.Err.Error(), "not a supported product") {
		t.Fatalf("root html should be refused: %+v", html)
	}
	rt := tools.Call("write_file", `{"path":"game-worlds/world-x/playable/runtime.js","content":"window.x=1"}`)
	if rt.Err == nil || !strings.Contains(rt.Err.Error(), "sealed") {
		t.Fatalf("runtime.js should be refused: %+v", rt)
	}
	ok := tools.Call("write_file", `{"path":"game-worlds/world-x/DESIGN.md","content":"# spec\n"}`)
	if ok.Err != nil {
		t.Fatal(ok.Err)
	}
}

// TestToolSearchMatchesAnyNamedGameTool 锁住查询里只要有一个已登记的扩展工具名就会命中它。
func TestToolSearchMatchesAnyNamedGameTool(t *testing.T) {
	tools := &WorkspaceTools{
		Extra: map[string]ExtraTool{
			"game_analyze_scene": {JSON: ToolJSON{Function: map[string]any{"name": "game_analyze_scene", "description": "Build a scene packet"}}},
			"video_route":        {JSON: ToolJSON{Function: map[string]any{"name": "video_route", "description": "Open a workshop"}}},
		},
	}
	res := tools.Call("tool_search", `{"query":"game_analyze_scene game_plan_rules game_compile"}`)
	if res.Err != nil || !strings.Contains(res.Content, "game_analyze_scene") {
		t.Fatalf("%+v", res)
	}
}
