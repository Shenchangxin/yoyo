package api

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// evalsDir 从当前工作目录向上两级定位 evals 的绝对路径。定位失败时让测试 Fatal。
func evalsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(filepath.Join(wd, "..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// TestRPCHealthAndUnknown 锁住 health 的 ok 为真，并且方法 nope 返回错误。
func TestRPCHealthAndUnknown(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ok := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "health"})
	if ok.Error != nil {
		t.Fatal(ok.Error)
	}
	m, _ := ok.Result.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("%+v", ok.Result)
	}
	bad := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "nope"})
	if bad.Error == nil {
		t.Fatal("expected method error")
	}
}

// TestRPCHarnessLineage 锁住 harness.get 的 active 非空，lineage 至少有一个带 hash 的节点。
func TestRPCHarnessLineage(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "harness.get"})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	m, _ := got.Result.(map[string]any)
	if m["active"] == "" || m["active"] == nil {
		t.Fatalf("active %+v", m)
	}
	nodes, ok := m["lineage"].([]app.LineageNode)
	if !ok || len(nodes) == 0 {
		t.Fatalf("lineage %+v", m["lineage"])
	}
	if nodes[0].Hash == "" {
		t.Fatalf("node %+v", nodes[0])
	}
}

// TestRPCThreadLifecycle 走 thread.start、thread.pin、thread.isolate.set、用前 8 位 id 做 session.dump，再 thread.delete。隔离打开后 Isolate 为真且 Worktree 非空。
func TestRPCThreadLifecycle(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	created := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "thread.start", Params: jsonRaw(`{"workspace":"` + filepath.ToSlash(ws) + `"}`)})
	if created.Error != nil {
		t.Fatal(created.Error)
	}
	m, _ := created.Result.(app.SessionMeta)
	if m.ID == "" {
		t.Fatalf("%+v", created.Result)
	}
	pin := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "thread.pin", Params: jsonRaw(`{"session":"` + m.ID + `","pinned":true}`)})
	if pin.Error != nil {
		t.Fatal(pin.Error)
	}
	iso := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 4, Method: "thread.isolate.set", Params: jsonRaw(`{"session":"` + m.ID + `","isolate":true}`)})
	if iso.Error != nil {
		t.Fatal(iso.Error)
	}
	im, _ := iso.Result.(app.SessionMeta)
	if !im.Isolate || im.Worktree == "" {
		t.Fatalf("isolate %+v", iso.Result)
	}
	dump := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 5, Method: "session.dump", Params: jsonRaw(`{"session":"` + m.ID[:8] + `"}`)})
	if dump.Error != nil {
		t.Fatal(dump.Error)
	}
	dm, _ := dump.Result.(app.SessionDump)
	if dm.ID != m.ID {
		t.Fatalf("dump %+v", dump.Result)
	}
	del := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "thread.delete", Params: jsonRaw(`{"session":"` + m.ID + `"}`)})
	if del.Error != nil {
		t.Fatal(del.Error)
	}
}

// jsonRaw 把字符串收成 json.RawMessage。
func jsonRaw(s string) json.RawMessage { return json.RawMessage(s) }

// TestRPCWorkspacePreview 锁住 workspace.preview 对 n.go 给出 lang 为 go，正文含 package n。
func TestRPCWorkspacePreview(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "n.go"), []byte("package n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]string{"workspace": ws, "path": "n.go"})
	if err != nil {
		t.Fatal(err)
	}
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 9, Method: "workspace.preview", Params: params})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	m, _ := got.Result.(map[string]any)
	if m["lang"] != "go" {
		t.Fatalf("%+v", got.Result)
	}
	text, _ := m["text"].(string)
	if !strings.Contains(text, "package n") {
		t.Fatalf("text %+v", m)
	}
}

// TestRPCCanvasHTTP 锁住 canvas.http 的 GET canvas-projects 有结果，POST 能创建标题为 Board 的项目。
func TestRPCCanvasHTTP(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 11, Method: "canvas.http", Params: jsonRaw(`{"method":"GET","path":"canvas-projects"}`)})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	if got.Result == nil {
		t.Fatal("canvas.http empty result")
	}
	created := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 12, Method: "canvas.http", Params: jsonRaw(`{"method":"POST","path":"canvas-projects","body":{"title":"Board"}}`)})
	if created.Error != nil {
		t.Fatalf("create canvas: %v", created.Error)
	}
}

// TestLineClientHealth 用管道把 LineClient 接到 ServeRPC，调用 health，结果的 ok 为真。
func TestLineClientHealth(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	pr, pw := io.Pipe()
	qr, qw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ServeRPC(ctx, a, pr, qw) }()
	cli := NewLineClient(qr, pw)
	v, err := cli.Call("health", nil)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := v.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("%+v", v)
	}
}

// TestConfigSetMergesPartial 锁住 config.set 只改 model 和 theme，SearchURL 和 CrashResume 还在。
func TestConfigSetMergesPartial(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.Config.SearchURL = "https://keep.example/{q}"
	a.Config.CrashResume = true
	if err := a.SaveConfig(); err != nil {
		t.Fatal(err)
	}
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 20, Method: "config.set", Params: jsonRaw(`{"model":"gpt-4.1-mini","theme":"dark"}`)})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	if a.Config.Model != "gpt-4.1-mini" {
		t.Fatalf("model %q", a.Config.Model)
	}
	if a.Config.SearchURL != "https://keep.example/{q}" {
		t.Fatalf("search wiped %q", a.Config.SearchURL)
	}
	if !a.Config.CrashResume {
		t.Fatal("crash_resume wiped")
	}
	if a.Config.Theme != "dark" {
		t.Fatalf("theme %q", a.Config.Theme)
	}
}

// TestRPCPacksListAndEnable 锁住 packs.list 含 superpowers 和 novel-to-game，且目录默认不启用。对未安装的包调用 packs.enable 后，installed 仍为假，enabled 也不为真。
func TestRPCPacksListAndEnable(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "packs.list"})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	raw, _ := json.Marshal(got.Result)
	if !strings.Contains(string(raw), `"id":"superpowers"`) {
		t.Fatalf("%s", raw)
	}
	if !strings.Contains(string(raw), `"id":"novel-to-game"`) {
		t.Fatalf("catalog missing domain pack: %s", raw)
	}
	if strings.Contains(string(raw), `"enabled":true`) {
		t.Fatal("catalog must default off")
	}
	en := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "packs.enable", Params: jsonRaw(`{"id":"superpowers","scope":"workspace","enabled":true}`)})
	if en.Error != nil {
		t.Fatal(en.Error)
	}
	again := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "packs.list"})
	raw2, _ := json.Marshal(again.Result)
	if !strings.Contains(string(raw2), `"installed":false`) {
		t.Fatalf("%s", raw2)
	}
	if strings.Contains(string(raw2), `"enabled":true`) {
		t.Fatal("uninstalled pack must not report enabled")
	}
}

// TestRPCPagesProfilesAndPause 锁住 pages.save 之后 pages.list 能看到 Brief，profiles.list 含 researcher，app.pause.set 之后 health 的 paused 为真。
func TestRPCPagesProfilesAndPause(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	created := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "pages.save", Params: jsonRaw(`{"title":"Brief","content":"hello pages"}`)})
	if created.Error != nil {
		t.Fatal(created.Error)
	}
	list := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "pages.list"})
	if list.Error != nil {
		t.Fatal(list.Error)
	}
	raw, _ := json.Marshal(list.Result)
	if !strings.Contains(string(raw), "Brief") || !strings.Contains(string(raw), "hello pages") {
		t.Fatalf("%s", raw)
	}
	pro := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "profiles.list"})
	if pro.Error != nil {
		t.Fatal(pro.Error)
	}
	pr, _ := json.Marshal(pro.Result)
	if !strings.Contains(string(pr), "researcher") {
		t.Fatalf("%s", pr)
	}
	pause := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 4, Method: "app.pause.set", Params: jsonRaw(`{"paused":true}`)})
	if pause.Error != nil {
		t.Fatal(pause.Error)
	}
	if !a.IsPaused() {
		t.Fatal("paused")
	}
	health := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 5, Method: "health"})
	h, _ := json.Marshal(health.Result)
	if !strings.Contains(string(h), `"paused":true`) {
		t.Fatalf("%s", h)
	}
}

// TestRPCEvolveOverlayRejectsPromote 锁住 evolve.overlay 带 promote，或同时带 sealed、baselines 和 k 时，错误信息里能看到被拒绝的参数。
func TestRPCEvolveOverlayRejectsPromote(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: "evolve.overlay", Params: jsonRaw(`{"session":"s","promote":true}`)})
	if got.Error == nil || !strings.Contains(got.Error.Message, "promote") {
		t.Fatalf("%+v", got.Error)
	}
	got = Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 2, Method: "evolve.overlay", Params: jsonRaw(`{"session":"s","sealed":true,"baselines":2,"k":3}`)})
	if got.Error == nil || !strings.Contains(got.Error.Message, "sealed") {
		t.Fatalf("%+v", got.Error)
	}
}

// TestRPCEvolveOverlayKeepBase 用 heuristic 求解器、reuse_turn 为假跑 evolve.overlay，结果 KeptBase 为真且 Adopted 为假，active、canary、staging 和会话的 FollowActive 都不改。
func TestRPCEvolveOverlayKeepBase(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	active := a.ActiveHash()
	canary := a.Refs.GetOrEmpty(artifact.RefCanary)
	staging := a.Refs.GetOrEmpty(artifact.RefStaging)
	sess, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"session":    sess.ID,
		"task":       "Write a file named hello.txt with the word hello",
		"solver":     "heuristic",
		"reuse_turn": false,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := Dispatch(context.Background(), a, RPCRequest{JSONRPC: "2.0", ID: 3, Method: "evolve.overlay", Params: params})
	if got.Error != nil {
		t.Fatal(got.Error)
	}
	res, ok := got.Result.(evolve.OverlayResult)
	if !ok {
		t.Fatalf("type %T", got.Result)
	}
	if res.Adopted || !res.KeptBase {
		t.Fatalf("want keep_base: %+v", res)
	}
	if res.R != evolve.OverlayR || res.P != evolve.OverlayP {
		t.Fatalf("R=%d P=%d", res.R, res.P)
	}
	meta, err := a.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if meta.HarnessPolicy != session.FollowActive {
		t.Fatalf("policy %s", meta.HarnessPolicy)
	}
	if a.ActiveHash() != active || a.Refs.GetOrEmpty(artifact.RefCanary) != canary || a.Refs.GetOrEmpty(artifact.RefStaging) != staging {
		t.Fatal("overlay rpc moved global refs")
	}
}
